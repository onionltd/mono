package monitor

import (
	"context"
	"errors"
	"github.com/onionltd/oniontree-tools/pkg/oniontree"
	"github.com/onionltd/oniontree-tools/pkg/types/service"
	"go.uber.org/zap"
	"golang.org/x/sync/semaphore"
	"sync"
	"time"
)

const procsChCapacity = 512

type Monitor struct {
	logger      *zap.Logger
	config      MonitorConfig
	stopCh      chan int
	deadCh      chan int
	ot          *oniontree.OnionTree
	onlineLinks sync.Map
	linksDB     sync.Map

	procsEventCh chan interface{}
	procs        map[string]*Process
}

func (m *Monitor) Start(path string) error {
	ot, err := oniontree.Open(path)
	if err != nil {
		return err
	}
	m.ot = ot

	workerConnSem = semaphore.NewWeighted(m.config.WorkerTCPConnectionsMax)

	m.logger.Info("started", zap.String("path", path), zap.Reflect("config", m.config))

	defer func() {
		m.logger.Debug("cleaning up running processes")
		for url := range m.procs {
			m.destroyProcess(url)

			// Wait for process stopped event
			done := false
			for !done {
				select {
				case event, ok := <-m.procsEventCh:
					if !ok {
						continue
					}
					switch e := event.(type) {
					case processStoppedEvent:
						m.deleteOnlineLinks(e)
						done = true
					}
				}
			}
		}
		close(m.procsEventCh)
		m.deadCh <- 1
	}()

	m.procsEventCh = make(chan interface{}, procsChCapacity)
	timeout := time.Duration(0)

	for {
		select {
		case <-time.After(timeout):
			timeout = m.config.MonitorHeartbeat
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

		case event, ok := <-m.procsEventCh:
			if !ok {
				continue
			}
			switch e := event.(type) {
			case processStatusEvent:
				m.logger.Debug("update link", zap.Reflect("event", e))
				m.updateOnlineLinks(e)
				m.updateLinksDB(e)
			case processStoppedEvent:
				m.deleteOnlineLinks(e)
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

func (m *Monitor) updateOnlineLinks(event processStatusEvent) {
	val, _ := m.onlineLinks.LoadOrStore(event.ServiceID, make(map[string]string))
	urls := val.(map[string]string)

	if event.Status == StatusOffline {
		delete(urls, event.URL)
	} else {
		urls[event.URL] = event.URL
	}
	m.onlineLinks.Store(event.ServiceID, urls)
}

func (m *Monitor) deleteOnlineLinks(event processStoppedEvent) {
	m.onlineLinks.Delete(event.ServiceID)
}

// updateLinksDB stores a link in links database. The database contains data which can be used
// for fast link -> serviceID translation. Data in links database are never deleted.
func (m *Monitor) updateLinksDB(event processStatusEvent) {
	m.linksDB.Store(event.URL, event.ServiceID)
}

func (m *Monitor) startNewProcess(serviceID string) {
	proc := NewProcess(m.logger.Named("process"), m.ot, m.config.WorkerConfig, m.procsEventCh)
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

func NewMonitor(logger *zap.Logger, cfg MonitorConfig) *Monitor {
	return &Monitor{
		procs:  make(map[string]*Process),
		stopCh: make(chan int),
		deadCh: make(chan int),
		logger: logger,
		config: cfg,
	}
}
