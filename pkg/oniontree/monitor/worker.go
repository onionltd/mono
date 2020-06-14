package monitor

import (
	"context"
	urlutils "github.com/onionltd/mono/pkg/utils/url"
	"go.uber.org/zap"
	"golang.org/x/net/proxy"
	"time"
)

type Worker struct {
	logger  *zap.Logger
	config  WorkerConfig
	stopCh  chan int
	eventCh chan<- interface{}

	// This context is used to cancel undergoing HTTP request.
	ctxReq       context.Context
	ctxReqCancel context.CancelFunc
}

func (w *Worker) Start(url string) {
	w.logger = w.logger.With(zap.String("workerID", url))
	w.logger.Info("started", zap.Reflect("config", w.config))

	// FIXME: handle this error!
	host, _ := urlutils.ParseHostPort(url)

	// If worker exits, notify process that link is offline.
	defer func() {
		if r := recover(); r != nil {
			w.logger.Error("panicking", zap.Reflect("reason", r))
		}
		w.sendEvent(WorkerStatusEvent{
			Status: StatusOffline,
			URL:    url,
		})
	}()

	failedAttempts := int(0)
	sleepTime := time.Duration(0)
	for {
		select {
		case <-time.After(sleepTime):
			sleepTime = w.config.PingInterval
			// Ping URL and forward the result back to the process
			status := w.ping(w.ctxReq, host)

			select {
			case <-w.ctxReq.Done():
				w.logger.Debug("HTTP request canceled")
				continue
			default:
			}

			if status == StatusOffline {
				// Handle StatusOffline.
				// It may be that StatusOffline is only temporary due to network conditions.
				// To take this into account, the worker issues another request after WORKER_PING_RETRY_INTERVAL,
				// this is repeated until WORKER_PING_RETRY_ATTEMPTS is reached, after which worker pauses
				// its operation for WORKER_PING_PAUSE.
				failedAttempts++

				if failedAttempts <= w.config.PingRetryAttempts {
					sleepTime = w.config.PingRetryInterval
					continue
				}

				sleepTime = w.config.PingPause
				w.logger.Info("too many failed ping attempts, pausing the worker",
					zap.String("sleep", sleepTime.String()))
			}

			failedAttempts = 0

			w.sendEvent(WorkerStatusEvent{
				Status: status,
				URL:    url,
			})

		case <-w.stopCh:
			w.logger.Info("stopped", zap.String("reason", "stop request"))
			return
		}
	}
}

func (w *Worker) ping(ctx context.Context, host string) (status Status) {
	err := w.testHost(ctx, host)
	switch err {
	case nil:
		status = StatusOnline
	default:
		status = StatusOffline
	}
	return
}

func (w *Worker) testHost(ctx context.Context, host string) error {
	if err := workerConnSem.Acquire(ctx, 1); err != nil {
		return err
	}
	defer workerConnSem.Release(1)
	ctx, _ = context.WithTimeout(ctx, w.config.PingTimeout)
	conn, err := proxy.Dial(ctx, "tcp", host)
	if err != nil {
		w.logger.Debug("failed to establish TCP connection",
			zap.Error(err),
		)
		return err
	}
	_ = conn.Close()
	return nil
}

func (w *Worker) sendEvent(e interface{}) {
	select {
	case w.eventCh <- e:
	}
}

func (w *Worker) Stop() {
	// Cancel context to stop an HTTP request in progress.
	w.ctxReqCancel()
	close(w.stopCh)
}

func NewWorker(logger *zap.Logger, cfg WorkerConfig, outputCh chan<- interface{}) *Worker {
	ctx, cancel := context.WithCancel(context.Background())
	return &Worker{
		logger:       logger,
		stopCh:       make(chan int),
		eventCh:      outputCh,
		ctxReq:       ctx,
		ctxReqCancel: cancel,
		config:       cfg,
	}
}
