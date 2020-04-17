package onions

import (
	"context"
	"errors"
	"github.com/onionltd/oniontree-tools/pkg/oniontree"
	"github.com/onionltd/oniontree-tools/pkg/types/service"
	"go.uber.org/zap"
	"sync"
	"time"
)

const monitorRefreshInterval time.Duration = 10 * time.Second
const procsChCapacity = 512

type Monitor struct {
	logger      *zap.Logger
	stopCh      chan int
	deadCh      chan int
	ot          *oniontree.OnionTree
	onlineLinks sync.Map
	linksDB     sync.Map

	procsCh chan Link
	procs   map[string]*Process
}

func (m *Monitor) Start(path string) error {
	ot, err := oniontree.Open(path)
	if err != nil {
		return err
	}
	m.ot = ot

	m.logger.Info("started", zap.String("path", path))

	defer func() {
		m.logger.Debug("cleaning up running processes")
		for url := range m.procs {
			m.destroyProcess(url)

			// Drain the channel (wait for process termination event)
		drain_loop:
			for {
				select {
				case link, ok := <-m.procsCh:
					if !ok {
						continue
					}
					if link.URL == "" {
						m.onlineLinks.Delete(link.ServiceID)
						break drain_loop
					}
				}
			}
		}
		close(m.procsCh)
		m.deadCh <- 1
	}()

	m.procsCh = make(chan Link, procsChCapacity)
	timeout := time.Duration(0)

	for {
		select {
		case <-time.After(timeout):
			timeout = monitorRefreshInterval
			serviceIDs, err := m.ot.List()
			if err != nil {
				m.logger.Warn("failed to read list of services", zap.Error(err))
				continue
			}

			procs := make(map[string]*Process)

			for i := range serviceIDs {
				procs[serviceIDs[i]] = nil
			}

			// Find obsolete processes and preserve state of other running processes
			obsoleteProcs := []string{}
			for serviceID := range m.procs {
				if _, ok := procs[serviceID]; !ok {
					obsoleteProcs = append(obsoleteProcs, serviceID)
					continue
				}
				procs[serviceID] = m.procs[serviceID]
			}

			for i := range obsoleteProcs {
				//m.logger.Debug("destroy obsolete process", zap.String("processID", obsoleteProcs[i]))
				m.destroyProcess(obsoleteProcs[i])
			}

			// Start new worker if not running.
			for serviceID := range procs {
				if procs[serviceID] != nil {
					// Process is already running
					m.reloadRunningProcess(serviceID)
					continue
				}
				m.startNewProcess(serviceID)
				m.reloadRunningProcess(serviceID)
			}

		case link, ok := <-m.procsCh:
			m.logger.Debug("update link", zap.Reflect("link", link), zap.Bool("ok", ok))
			if !ok {
				continue
			}
			// Handle events sent by a process.
			// There are two types of event:
			// 1. sent to update link status (online/offline)
			// 2. sent when a process terminates
			//
			// When process terminates, emitted event holds only a service ID.
			if link.URL != "" {
				m.updateOnlineLinks(link)
				m.updateLinksDB(link)
			} else {
				// TODO: wrap inside a method!
				m.onlineLinks.Delete(link.ServiceID)
			}
		case <-m.stopCh:
			m.logger.Info("stopped", zap.String("reason", "stop request"))
			return nil
		}
	}
}

func (m *Monitor) Stop(ctx context.Context) error {
	close(m.stopCh)
	select {
	case <-m.deadCh:
	case <-ctx.Done():
		m.logger.Warn("stop request canceled")
		return ctx.Err()
	}
	return nil
}

func (m *Monitor) GetOnlineLinks(serviceID string) (online []string, ok bool) {
	online = []string{}
	val, ok := m.onlineLinks.Load(serviceID)
	if !ok {
		return
	}
	for link := range val.(map[string]string) {
		online = append(online, link)
	}
	return
}

func (m *Monitor) GetService(serviceID string) (service.Service, error) {
	return m.ot.Get(serviceID)
}

var ErrUrlNotFound = errors.New("url not found")

func (m *Monitor) GetServiceByURL(url string) (service service.Service, err error) {
	val, ok := m.linksDB.Load(url)
	if !ok {
		err = ErrUrlNotFound
		return
	}
	serviceID := val.(string)
	service, err = m.ot.Get(serviceID)
	if err != nil {
		return
	}
	return
}

func (m *Monitor) updateOnlineLinks(link Link) {
	val, _ := m.onlineLinks.LoadOrStore(link.ServiceID, make(map[string]string))
	urls := val.(map[string]string)

	if link.Status == StatusOffline {
		delete(urls, link.URL)
	} else {
		urls[link.URL] = link.URL
	}
	m.onlineLinks.Store(link.ServiceID, urls)
}

// updateLinksDB stores a link in links database. The database contains data which can be used
// for fast link -> serviceID translation. Data in links database are never deleted.
func (m *Monitor) updateLinksDB(link Link) {
	m.linksDB.Store(link.URL, link.ServiceID)
}

func (m *Monitor) startNewProcess(serviceID string) {
	proc := NewProcess(m.logger.Named("process"), m.ot, m.procsCh)
	go proc.Start(serviceID)
	m.procs[serviceID] = proc
}

func (m *Monitor) destroyProcess(serviceID string) {
	proc := m.procs[serviceID]
	delete(m.procs, serviceID)
	proc.Stop()
}

// reloadRunningProcess send a reload event to a running process. Reload event causes the process to reload
// it's configuration from a service file.
func (m *Monitor) reloadRunningProcess(serviceID string) {
	proc := m.procs[serviceID]
	proc.Reload(context.TODO())
}

func NewMonitor(logger *zap.Logger) *Monitor {
	return &Monitor{
		procs:  make(map[string]*Process),
		stopCh: make(chan int),
		deadCh: make(chan int),
		logger: logger,
	}
}
