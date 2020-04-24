package monitor

import (
	"context"
	urlutils "github.com/onionltd/mono/pkg/utils/url"
	"github.com/onionltd/oniontree-tools/pkg/oniontree"
	"go.uber.org/zap"
)

const workersChCapacity = 256

type Process struct {
	logger   *zap.Logger
	config   WorkerConfig
	reloadCh chan int
	stopCh   chan int
	outputCh chan<- Link
	ot       *oniontree.OnionTree

	workersCh chan Link
	workers   map[string]*Worker
}

func (p *Process) Start(serviceID string) {
	p.logger = p.logger.With(zap.String("processID", serviceID))
	p.logger.Info("started")

	// Do a proper clean up
	defer func() {
		p.logger.Debug("cleaning up running workers")
		for url := range p.workers {
			p.destroyWorker(url)

			select {
			case link, ok := <-p.workersCh:
				if !ok {
					continue
				}
				link.ServiceID = serviceID
				select {
				case p.outputCh <- link:
				}
			}
		}
		// Sent termination event to notify the monitor that this process is no longer running.
		select {
		case p.outputCh <- Link{
			ServiceID: serviceID,
		}:
		}
		close(p.workersCh)
	}()

	p.workersCh = make(chan Link, workersChCapacity)

	for {
		select {
		case <-p.reloadCh:
			service, err := p.ot.Get(serviceID)
			if err != nil {
				p.logger.Warn("failed to read a service file")
				continue
			}

			urls := service.URLs
			workers := make(map[string]*Worker)

			for i, _ := range urls {
				url, err := urlutils.Normalize(urls[i])
				if err != nil {
					p.logger.Warn("failed to normalize URL",
						zap.String("url", urls[i]),
						zap.Error(err))
					continue
				}
				workers[url] = nil
			}

			// Find obsolete workers and preserve state of other running workers
			obsoleteWorkers := []string{}

			for url := range p.workers {
				if _, ok := workers[url]; !ok {
					obsoleteWorkers = append(obsoleteWorkers, url)
					continue
				}
				workers[url] = p.workers[url]
			}

			for i := range obsoleteWorkers {
				//p.logger.Debug("destroy obsolete worker", zap.String("workerID", obsoleteWorkers[i]))
				p.destroyWorker(obsoleteWorkers[i])
			}

			// Start new workers if not running.
			for url := range workers {
				if workers[url] != nil {
					continue
				}
				p.startNewWorker(url)
			}

		// Read from workers channel and forward the data to monitor.
		// Before doing so, include service ID in Link data.
		case link, ok := <-p.workersCh:
			if !ok {
				continue
			}
			link.ServiceID = serviceID
			select {
			case p.outputCh <- link:
			}
		case <-p.stopCh:
			p.logger.Info("stopped", zap.String("reason", "stop request"))
			return
		}
	}
}

func (p *Process) Stop() {
	close(p.stopCh)
}

func (p *Process) Reload(ctx context.Context) {
	select {
	case p.reloadCh <- 1:
	case <-ctx.Done():
		p.logger.Warn("reload request canceled")
		return
	}
}

// startNewWorker starts a new worker and stores the object in internal map.
// If there's an already running workers for the same URL, only the reference to it is
// thrown away and not the entire worker! If not cautious this may lead to memory leaks!
func (p *Process) startNewWorker(url string) {
	worker := NewWorker(p.logger.Named("worker"), p.config, p.workersCh)
	go worker.Start(url)
	p.workers[url] = worker
}

func (p *Process) destroyWorker(url string) {
	worker := p.workers[url]
	delete(p.workers, url)
	worker.Stop()
}

func NewProcess(logger *zap.Logger, ot *oniontree.OnionTree, cfg WorkerConfig, outputCh chan<- Link) *Process {
	return &Process{
		workers:  make(map[string]*Worker),
		ot:       ot,
		logger:   logger,
		reloadCh: make(chan int),
		stopCh:   make(chan int),
		outputCh: outputCh,
		config:   cfg,
	}
}
