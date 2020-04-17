package onions

import (
	"context"
	urlutils "github.com/onionltd/mono/pkg/utils/url"
	"go.uber.org/zap"
	"golang.org/x/net/proxy"
	"time"
)

const (
	pingInterval time.Duration = 1 * time.Minute
	pingTimeout  time.Duration = 25 * time.Second
)

// statusCount must must be an odd number!
const statusCount = 7

type Worker struct {
	logger        *zap.Logger
	status        [statusCount]Status
	statusCounter int
	stopCh        chan int
	outputCh      chan<- Link

	// This context is used to cancel undergoing HTTP request.
	ctxReq       context.Context
	ctxReqCancel context.CancelFunc
}

func (w *Worker) Start(url string) {
	w.logger = w.logger.With(zap.String("workerID", url))
	w.logger.Info("started")

	// FIXME: handle this error!
	host, _ := urlutils.ParseHostPort(url)

	// If worker exits, notify process that link is offline.
	defer func() {
		select {
		case w.outputCh <- Link{
			URL:    url,
			Status: StatusOffline,
		}:
		}
	}()

	timeout := time.Duration(0)
	for {
		select {
		case <-time.After(timeout):
			timeout = pingInterval
			// Ping URL and forward the result back to the process
			status := w.ping(w.ctxReq, host)

			select {
			case <-w.ctxReq.Done():
				w.logger.Debug("HTTP request canceled")
				continue
			default:
			}

			select {
			case w.outputCh <- Link{
				URL:    url,
				Status: status,
			}:
			}
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
	w.status[w.statusCounter] = status
	w.statusCounter = (w.statusCounter + 1) % statusCount
	return
}

func (w *Worker) testHost(ctx context.Context, host string) error {
	if err := connSem.Acquire(ctx, 1); err != nil {
		return err
	}
	defer connSem.Release(1)
	ctx, _ = context.WithTimeout(ctx, pingTimeout)
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

func (w *Worker) Stop() {
	// Cancel context to stop an HTTP request in progress.
	w.ctxReqCancel()
	close(w.stopCh)
}

func NewWorker(logger *zap.Logger, outputCh chan<- Link) *Worker {
	ctx, cancel := context.WithCancel(context.Background())
	return &Worker{
		logger:       logger,
		stopCh:       make(chan int),
		outputCh:     outputCh,
		ctxReq:       ctx,
		ctxReqCancel: cancel,
	}
}
