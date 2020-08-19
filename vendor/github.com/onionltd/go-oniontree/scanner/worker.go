package scanner

import (
	"context"
	"fmt"
	"golang.org/x/net/proxy"
	"runtime/debug"
	"time"
)

type WorkerConfig struct {
	PingInterval      time.Duration
	PingTimeout       time.Duration
	PingPauseInterval time.Duration
	PingRetryInterval time.Duration
	PingRetryAttempts int
}

type Worker struct {
	config WorkerConfig
	cancel context.CancelFunc
}

var DefaultWorkerConfig = WorkerConfig{
	PingInterval:      1 * time.Minute,
	PingTimeout:       50 * time.Second,
	PingPauseInterval: 5 * time.Minute,
	PingRetryInterval: 10 * time.Second,
	PingRetryAttempts: 3,
}

func (w *Worker) Start(ctx context.Context, url string, outputCh chan<- Event) error {
	ctx, w.cancel = context.WithCancel(ctx)

	emitStatusEvent := func(err error) {
		var status Status

		switch err {
		case nil:
			status = StatusOnline
		default:
			status = StatusOffline
		}

		select {
		case outputCh <- workerStatus{
			Status: status,
			URL:    url,
			Error:  err,
		}:
		}
	}

	host, err := ParseHostPort(url)
	if err != nil {
		return err
	}

	connect := func(host string) error {
		if err := workerConnSem.Acquire(ctx, 1); err != nil {
			return err
		}
		defer workerConnSem.Release(1)

		ctxReq, cancel := context.WithTimeout(ctx, w.config.PingTimeout)
		defer cancel()

		conn, err := proxy.Dial(ctxReq, "tcp", host)
		if err != nil {
			return err
		}
		_ = conn.Close()
		return nil
	}

	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("%+v\n", r)
			debug.PrintStack()
			return
		}
	}()

	failedAttempts := int(0)
	sleepTime := time.Duration(0)
	for {
		select {
		case <-time.After(sleepTime):
			sleepTime = w.config.PingInterval

			// Connect to the host and inform process about the result
			err := connect(host)

			select {
			case <-ctx.Done():
				continue
			default:
			}

			if err != nil {
				// Handle StatusOffline.
				// It may be that StatusOffline is only temporary due to network conditions.
				// To take this into account, the worker issues another request after WORKER_PING_RETRY_INTERVAL,
				// this is repeated until WORKER_PING_RETRY_ATTEMPTS is reached, after which worker pauses
				// its operation for WORKER_PING_PAUSE.
				failedAttempts++

				if failedAttempts < w.config.PingRetryAttempts {
					sleepTime = w.config.PingRetryInterval
					continue
				}

				sleepTime = w.config.PingPauseInterval
			}

			failedAttempts = 0
			emitStatusEvent(err)

		case <-ctx.Done():
			emitStatusEvent(context.Canceled)
			return nil
		}
	}
}

func (w *Worker) Stop() {
	w.cancel()
}

func newWorker(cfg WorkerConfig) *Worker {
	return &Worker{
		config: cfg,
	}
}
