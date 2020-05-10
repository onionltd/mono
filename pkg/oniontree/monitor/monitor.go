package monitor

import (
	"context"
	"errors"
	"github.com/fsnotify/fsnotify"
	"github.com/onionltd/mono/pkg/oniontree/monitor/speed"
	"github.com/onionltd/oniontree-tools/pkg/oniontree"
	"github.com/onionltd/oniontree-tools/pkg/types/service"
	"go.uber.org/zap"
	"golang.org/x/sync/semaphore"
	"path"
	"sync"
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

func (m *Monitor) Start(dir string) error {
	ot, err := oniontree.Open(dir)
	if err != nil {
		return err
	}
	m.ot = ot

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()
	if err := watcher.Add(path.Join(dir, "unsorted")); err != nil {
		return err
	}
	if err := watcher.Add(path.Join(dir, "tagged/dead")); err != nil {
		return err
	}

	listServices := func() ([]string, error) {
		services, err := m.ot.List()
		if err != nil {
			return nil, err
		}
		tag, err := m.ot.GetTag("dead")
		if err != nil {
			if err != oniontree.ErrIdNotExists {
				return nil, err
			}
			return services, nil
		}
		deadServices := tag.Services
		filtered := make([]string, 0, len(services)-len(deadServices))
		for i := range services {
			isDead := false
			for _, deadService := range deadServices {
				if services[i] == deadService {
					isDead = true
					break
				}
			}
			if !isDead {
				filtered = append(filtered, services[i])
			}
		}
		return filtered, nil
	}

	m.procsEventCh = make(chan interface{}, procsChCapacity)
	workerConnSem = semaphore.NewWeighted(m.config.WorkerTCPConnectionsMax)

	m.logger.Info("started", zap.String("dir", dir), zap.Reflect("config", m.config))

	defer func() {
		if r := recover(); r != nil {
			m.logger.Error("panicking", zap.Reflect("reason", r))
		}
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

	eventCh := make(chan struct{}, 1)
	eventCh <- struct{}{}

	go func() {
		for {
			select {
			case e := <-watcher.Events:
				m.logger.Debug("fsnotify event", zap.String("event", e.String()))
				eventCh <- struct{}{}

			case <-watcher.Errors:
				m.logger.Error("fsnotify error", zap.Error(err))

			case <-m.stopCh:
				m.logger.Debug("fsnotify stopped watching")
				return
			}
		}
	}()

	var (
		speedMeasurement *speed.Measurement
		speedHeartbeat   speed.Measurement
		speedProcEvents  speed.Measurement
	)

	for {
		select {
		case <-eventCh:
			speedMeasurement = &speedHeartbeat
			speedMeasurement.Start()

			serviceIDs, err := listServices()
			if err != nil {
				m.logger.Warn("failed to read list of services", zap.Error(err))
				break
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
			// Restart the loop, I'm not interested in the speed measurement.
			// Measuring the speed in this case would taint the average value.
			if !ok {
				continue
			}

			speedMeasurement = &speedProcEvents
			speedMeasurement.Start()

			switch e := event.(type) {
			case processStatusEvent:
				if writer := m.logger.Check(zap.DebugLevel, "event"); writer != nil {
					writer.Write(zap.Reflect("event", e))
				}
				m.updateOnlineLinks(e)
				m.updateLinksDB(e)
			case processStoppedEvent:
				m.deleteOnlineLinks(e)
			}

		case <-m.stopCh:
			m.logger.Info("stopped", zap.String("reason", "stop request"))
			return nil
		}

		speedMeasurement.Stop()

		if ce := m.logger.Check(zap.DebugLevel, "average loop speed"); ce != nil {
			ce.Write(
				zap.String("heartbeat", speedHeartbeat.Average().String()),
				zap.String("events", speedProcEvents.Average().String()),
			)
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
		deadCh: make(chan int, 1),
		logger: logger,
		config: cfg,
	}
}
