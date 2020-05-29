package main

import (
	"context"
	"fmt"
	"github.com/dgraph-io/badger/v2"
	"github.com/dgraph-io/badger/v2/options"
	"github.com/jessevdk/go-flags"
	"github.com/labstack/echo-contrib/prometheus"
	"github.com/labstack/echo/v4"
	"github.com/mojocn/base64Captcha"
	captcha "github.com/onionltd/mono/pkg/base64captcha"
	echoerrors "github.com/onionltd/mono/pkg/echo/errors"
	loggermw "github.com/onionltd/mono/pkg/echo/middleware/logger"
	"github.com/onionltd/mono/pkg/oniontree/monitor"
	zaputil "github.com/onionltd/mono/pkg/utils/zap"
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
	templatesLogger := rootLogger.Named("templates")

	templates, err := setupTemplates(templatesLogger, cfg)
	if err != nil {
		return err
	}

	db, err := setupBadger(cfg)
	if err != nil {
		return err
	}
	defer db.Close()

	mon := setupMonitor(rootLogger.Named("monitor"), cfg)

	router := setupRouter(httpdLogger, templates)

	// Setup prometheus metrics
	setupRouterMetrics(router)

	server := server{
		logger:       httpdLogger,
		config:       cfg,
		linksMonitor: mon,
		router:       router,
		badgerDB:     db,
		oopsSet:      oopsies,
		captcha:      setupCaptcha(),
	}
	server.routes()

	// Handle termination signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)

	go func() {
		<-sigCh
		rootLogger.Warn("received a termination signal")
		ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
		_ = router.Shutdown(ctx)
		_ = mon.Stop(ctx)
	}()

	wg := sync.WaitGroup{}
	wg.Add(2)

	go func() {
		defer wg.Done()
		if err := mon.Start(cfg.OnionTreeDir); err != nil {
			rootLogger.Error("monitor error", zap.Error(err))
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

func setupTemplates(logger *zap.Logger, cfg *config) (*Templates, error) {
	t := &Templates{
		logger: logger,
	}
	if err := t.Load(cfg.TemplatesDir); err != nil {
		return nil, err
	}
	return t, nil
}

func setupRouter(logger *zap.Logger, t *Templates) *echo.Echo {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	e.Renderer = t
	e.Use(loggermw.WithConfig(logger))
	e.HTTPErrorHandler = echoerrors.DefaultErrorHandler
	return e
}

func setupRouterMetrics(e *echo.Echo) {
	p := prometheus.NewPrometheus("httpd", nil)
	p.RequestCounterURLLabelMappingFunc = func(c echo.Context) string {
		return c.Request().RequestURI
	}
	p.Use(e)
}

func setupBadger(cfg *config) (*badger.DB, error) {
	opts := badger.DefaultOptions(cfg.BadgerDBDir)
	opts = opts.WithValueLogLoadingMode(options.FileIO)
	return badger.Open(opts)
}

func setupMonitor(logger *zap.Logger, cfg *config) *monitor.Monitor {
	monitorCfg := monitor.DefaultMonitorConfig
	monitorCfg.WorkerTCPConnectionsMax = cfg.MonitorConnectionsMax
	monitorCfg.WorkerConfig.PingTimeout = cfg.MonitorPingTimeout
	monitorCfg.WorkerConfig.PingInterval = cfg.MonitorPingInterval
	return monitor.NewMonitor(logger, monitorCfg)
}

func setupCaptcha() *captcha.Captcha {
	return captcha.NewCaptcha(base64Captcha.DefaultDriverDigit, base64Captcha.DefaultMemStore)
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
