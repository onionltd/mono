package scanner

import (
	"context"
	"fmt"
	"github.com/oniontree-org/go-oniontree"
	"runtime/debug"
)

type Process struct {
	workerConfig WorkerConfig
	reloadCh     chan int
	ot           *oniontree.OnionTree
	cancel       context.CancelFunc
}

func (p *Process) Start(ctx context.Context, serviceID string, outputCh chan<- Event) error {
	ctx, p.cancel = context.WithCancel(ctx)
	wCtx, wCtxCancel := context.WithCancel(context.Background())

	emitEvent := func(event Event) {
		if e, ok := event.(workerStatus); ok {
			event = processStatus{
				Status:    e.Status,
				URL:       e.URL,
				ServiceID: serviceID,
				Error:     e.Error,
			}
		}
		//fmt.Printf("%s :: %+v\n", reflect.TypeOf(event), event)
		select {
		case outputCh <- event:
		}
	}
	loadWorkers := func() (map[string]*Worker, error) {
		service, err := p.ot.GetService(serviceID)
		if err != nil {
			return nil, err
		}

		urls := make(map[string]*Worker, len(service.URLs))
		for _, url := range service.URLs {
			url, err := Normalize(url)
			if err != nil {
				continue
			}
			urls[url] = nil
		}

		return urls, nil
	}

	const workersChCapacity = 256
	workersEventCh := make(chan Event, workersChCapacity)

	workers, err := loadWorkers()
	if err != nil {
		return err
	}

	workerExists := func(url string) bool {
		if w, ok := workers[url]; ok {
			return w != nil
		}
		return false
	}
	// destroyWorker stops and destroys a running worker go routine.
	destroyRunningWorker := func(url string) {
		if !workerExists(url) {
			return
		}
		workers[url].Stop()
		delete(workers, url)
	}
	// startNewWorker creates and starts a new worker go routine.
	startNewWorker := func(url string) {
		if workerExists(url) {
			return
		}
		workers[url] = newWorker(p.workerConfig)
		workersEventCh <- WorkerStarted{
			URL:       url,
			ServiceID: serviceID,
		}
		go func() {
			err := workers[url].Start(wCtx, url, workersEventCh)
			workersEventCh <- WorkerStopped{
				URL:       url,
				ServiceID: serviceID,
				Error:     err,
			}
		}()
	}

	for url := range workers {
		startNewWorker(url)
	}

	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("%+v\n", r)
			debug.PrintStack()
			return
		}

		// Release resources used by the context
		p.cancel()
		// Cancel context for all workers
		wCtxCancel()

		for i := 0; i < len(workers); {
			select {
			case event := <-workersEventCh:
				if _, ok := event.(WorkerStopped); ok {
					i++
				}
				emitEvent(event)
			}
		}
	}()

	for {
		select {
		case <-p.reloadCh:
			newWorkers, err := loadWorkers()
			if err != nil {
				return err
			}

			// Find obsolete workers.
			obsoleteWorkers := []string{}
			for url := range workers {
				if _, ok := newWorkers[url]; !ok {
					obsoleteWorkers = append(obsoleteWorkers, url)
					continue
				}
				newWorkers[url] = workers[url]
			}

			// Destroy obsolete workers
			for i := range obsoleteWorkers {
				destroyRunningWorker(obsoleteWorkers[i])
			}

			// Start new workers if not running.
			for url := range newWorkers {
				if newWorkers[url] != nil {
					continue
				}
				startNewWorker(url)
			}

		// Read from workers event channel and forward the data to the scanner.
		case event := <-workersEventCh:
			if e, ok := event.(WorkerStopped); ok {
				destroyRunningWorker(e.URL)
			}
			emitEvent(event)

		case <-ctx.Done():
			return nil
		}
	}
}

func (p *Process) Stop() {
	p.cancel()
}

func (p *Process) Reload(ctx context.Context) {
	select {
	case p.reloadCh <- 1:
	case <-ctx.Done():
	}
}

func newProcess(ot *oniontree.OnionTree, cfg WorkerConfig) *Process {
	return &Process{
		ot:           ot,
		reloadCh:     make(chan int),
		workerConfig: cfg,
	}
}
