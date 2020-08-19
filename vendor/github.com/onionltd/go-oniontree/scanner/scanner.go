package scanner

import (
	"context"
	"fmt"
	"github.com/onionltd/go-oniontree"
	"github.com/onionltd/go-oniontree/watcher"
	"golang.org/x/sync/semaphore"
	"runtime/debug"
)

type ScannerConfig struct {
	// WorkerConnectionsMax limits number of parallel outbound TCP connections.
	WorkerTCPConnectionsMax int64
	// WorkerConfig is a configuration passed to workers.
	WorkerConfig WorkerConfig
}

type Scanner struct {
	config ScannerConfig
	ot     *oniontree.OnionTree
	cancel context.CancelFunc
}

var DefaultScannerConfig = ScannerConfig{
	WorkerTCPConnectionsMax: 256,
	WorkerConfig:            DefaultWorkerConfig,
}

func (m *Scanner) Start(ctx context.Context, dir string, outputCh chan<- Event) error {
	ot, err := oniontree.Open(dir)
	if err != nil {
		return err
	}
	m.ot = ot

	ctx, m.cancel = context.WithCancel(ctx)
	pCtx, pCtxCancel := context.WithCancel(context.Background())

	emitEvent := func(event Event) {
		if e, ok := event.(processStatus); ok {
			event = ScanEvent{
				Status:    e.Status,
				URL:       e.URL,
				ServiceID: e.ServiceID,
				Directory: dir,
				Error:     e.Error,
			}
		}
		//fmt.Printf("%s :: %+v\n", reflect.TypeOf(event), event)
		select {
		case outputCh <- event:
		}
	}
	loadServices := func() (map[string]*Process, error) {
		serviceIDs, err := m.ot.ListServices()
		if err != nil {
			return nil, err
		}

		procs := make(map[string]*Process, len(serviceIDs))
		for i := range serviceIDs {
			procs[serviceIDs[i]] = nil
		}

		deadServiceIDs, err := m.ot.ListServicesWithTag("dead")
		if err != nil {
			if err != oniontree.ErrTagNotExists {
				return nil, err
			}
			return procs, nil
		}

		for i := range deadServiceIDs {
			delete(procs, deadServiceIDs[i])
		}
		return procs, nil
	}

	workerConnSem = semaphore.NewWeighted(m.config.WorkerTCPConnectionsMax)

	const procsChCapacity = 512
	procsEventCh := make(chan Event, procsChCapacity)

	procs, err := loadServices()
	if err != nil {
		return err
	}

	processExists := func(serviceID string) bool {
		if p, ok := procs[serviceID]; ok {
			return p != nil
		}
		return false
	}
	destroyRunningProcess := func(serviceID string) {
		if !processExists(serviceID) {
			return
		}
		procs[serviceID].Stop()
		delete(procs, serviceID)
	}
	startNewProcess := func(serviceID string) {
		if processExists(serviceID) {
			return
		}
		procs[serviceID] = newProcess(m.ot, m.config.WorkerConfig)
		procsEventCh <- ProcessStarted{
			ServiceID: serviceID,
		}
		go func() {
			err := procs[serviceID].Start(pCtx, serviceID, procsEventCh)
			procsEventCh <- ProcessStopped{
				ServiceID: serviceID,
				Error:     err,
			}
		}()
	}
	reloadRunningProcess := func(serviceID string) {
		if !processExists(serviceID) {
			return
		}
		procs[serviceID].Reload(pCtx)
	}

	for serviceID := range procs {
		startNewProcess(serviceID)
	}

	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("%+v\n", r)
			debug.PrintStack()
			return
		}

		m.cancel()
		// Cancel context for all processes
		pCtxCancel()

		for i := 0; i < len(procs); {
			select {
			case event := <-procsEventCh:
				if _, ok := event.(ProcessStopped); ok {
					i++
				}
				emitEvent(event)
			}
		}

		close(outputCh)
	}()

	watcherEventCh := make(chan watcher.Event)
	watcherErrCh := make(chan error, 1)

	w := watcher.NewWatcher(ot)
	go func() {
		if err := w.Watch(ctx, watcherEventCh); err != nil {
			watcherErrCh <- err
		}
	}()

	for {
		select {
		case e, ok := <-watcherEventCh:
			if !ok {
				continue
			}

			switch event := e.(type) {
			case watcher.ServiceAdded:
				startNewProcess(event.ID)

			case watcher.ServiceUpdated:
				reloadRunningProcess(event.ID)

			case watcher.ServiceRemoved:
				destroyRunningProcess(event.ID)

			case watcher.ServiceTagged:
				if event.Tag != "dead" {
					continue
				}
				destroyRunningProcess(event.ID)

			case watcher.ServiceUntagged:
				if event.Tag != "dead" {
					continue
				}
				startNewProcess(event.ID)
			}

		case err := <-watcherErrCh:
			return err

		case event := <-procsEventCh:
			if e, ok := event.(ProcessStopped); ok {
				destroyRunningProcess(e.ServiceID)
			}
			emitEvent(event)

		case <-ctx.Done():
			return nil
		}
	}
}

func (m *Scanner) Stop() {
	m.cancel()
}

func NewScanner(cfg ScannerConfig) *Scanner {
	return &Scanner{
		config: cfg,
	}
}
