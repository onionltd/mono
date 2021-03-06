package main

import (
	"context"
	"fmt"
	"github.com/jessevdk/go-flags"
	prometheusmw "github.com/labstack/echo-contrib/prometheus"
	"github.com/labstack/echo/v4"
	echoerrors "github.com/onionltd/mono/pkg/echo/errors"
	loggermw "github.com/onionltd/mono/pkg/echo/middleware/logger"
	zaputil "github.com/onionltd/mono/pkg/utils/zap"
	"github.com/oniontree-org/go-oniontree"
	"github.com/oniontree-org/go-oniontree/scanner"
	"github.com/oniontree-org/go-oniontree/scanner/evtcache"
	"github.com/oniontree-org/go-oniontree/scanner/evtmetrics"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"
)

func run() error {
	cfg, err := setupConfig()
	if err != nil {
		return err
	}

	rootLogger, err := setupLogger(cfg)
	if err != nil {
		return err
	}
	httpdLogger := rootLogger.Named("httpd")

	ot, err := setupOnionTree(cfg)
	if err != nil {
		return err
	}

	scanr := setupScanner(cfg)
	cache := setupEventCache()
	metrics := setupEventMetrics()
	router := setupRouter(httpdLogger)

	server := server{
		logger: httpdLogger,
		config: cfg,
		router: router,
		cache:  cache,
		ot:     ot,
	}
	server.routes()

	// Handle termination signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)

	go func() {
		<-sigCh
		rootLogger.Warn("received a termination signal")

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = router.Shutdown(ctx)

		scanr.Stop()
	}()

	eventCh := make(chan scanner.Event)
	eventCopyCh := make(chan scanner.Event)

	wg := sync.WaitGroup{}
	wg.Add(4)

	go func() {
		defer wg.Done()
		if err := scanr.Start(context.Background(), cfg.OnionTreeDir, eventCh); err != nil {
			rootLogger.Error("scanner error", zap.Error(err))
			die()
		}
	}()
	go func() {
		defer wg.Done()
		if err := cache.ReadEvents(context.Background(), eventCh, eventCopyCh); err != nil {
			rootLogger.Error("cache error", zap.Error(err))
			die()
		}
	}()
	go func() {
		defer wg.Done()
		if err := metrics.ReadEvents(context.Background(), eventCopyCh, nil); err != nil {
			rootLogger.Error("metrics error", zap.Error(err))
			die()
		}
	}()
	go func() {
		defer wg.Done()
		if err := router.Start(cfg.Listen); err != nil {
			if err != http.ErrServerClosed {
				rootLogger.Error("http server error", zap.Error(err))
				die()
			}
		}
	}()

	wg.Wait()
	return nil
}

func setupConfig() (*config, error) {
	cfg := &config{}
	parser := flags.NewParser(cfg, flags.HelpFlag)
	if _, err := parser.Parse(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func setupLogger(cfg *config) (*zap.Logger, error) {
	return zaputil.DefaultConfigWithLogLevel(cfg.LogLevel).Build()
}

func setupOnionTree(cfg *config) (*oniontree.OnionTree, error) {
	return oniontree.Open(cfg.OnionTreeDir)
}

func setupRouter(logger *zap.Logger) *echo.Echo {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	e.Use(loggermw.WithConfig(logger))
	e.HTTPErrorHandler = echoerrors.DefaultErrorHandler

	// Setup prometheus metrics
	p := prometheusmw.NewPrometheus("httpd", nil)
	p.RequestCounterURLLabelMappingFunc = func(c echo.Context) string {
		return c.Request().RequestURI
	}
	p.Use(e)
	return e
}

func setupScanner(cfg *config) *scanner.Scanner {
	scannerCfg := scanner.DefaultScannerConfig
	scannerCfg.WorkerTCPConnectionsMax = cfg.MonitorConnectionsMax
	scannerCfg.WorkerConfig.PingTimeout = cfg.MonitorPingTimeout
	scannerCfg.WorkerConfig.PingInterval = cfg.MonitorPingInterval
	return scanner.NewScanner(scannerCfg)
}

func setupEventCache() *evtcache.Cache {
	return &evtcache.Cache{}
}

func setupEventMetrics() *evtmetrics.Metrics {
	m := evtmetrics.New()
	prometheus.MustRegister(m)
	return m
}

func die() {
	p, _ := os.FindProcess(os.Getpid())
	_ = p.Signal(os.Interrupt)
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}
