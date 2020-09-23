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
	zaputil "github.com/onionltd/mono/pkg/utils/zap"
	"github.com/oniontree-org/go-oniontree"
	"github.com/oniontree-org/go-oniontree/scanner"
	"github.com/oniontree-org/go-oniontree/scanner/evtcache"
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

	ot, err := setupOnionTree(cfg)
	if err != nil {
		return err
	}

	templates, err := setupTemplates(templatesLogger, cfg)
	if err != nil {
		return err
	}

	db, err := setupBadger(cfg)
	if err != nil {
		return err
	}
	defer db.Close()

	scanr := setupScanner(cfg)
	cache := setupEventCache()
	router := setupRouter(httpdLogger, templates)

	server := server{
		logger:   httpdLogger,
		config:   cfg,
		router:   router,
		cache:    cache,
		badgerDB: db,
		ot:       ot,
		oopsSet:  oopsies,
		captcha:  setupCaptcha(),
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

	wg := sync.WaitGroup{}
	wg.Add(3)

	go func() {
		defer wg.Done()
		if err := scanr.Start(context.Background(), cfg.OnionTreeDir, eventCh); err != nil {
			rootLogger.Error("scanner error", zap.Error(err))
			die()
		}
	}()
	go func() {
		defer wg.Done()
		if err := cache.ReadEvents(context.Background(), eventCh); err != nil {
			rootLogger.Error("cache error", zap.Error(err))
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

	// Setup prometheus metrics
	p := prometheus.NewPrometheus("httpd", nil)
	p.RequestCounterURLLabelMappingFunc = func(c echo.Context) string {
		return c.Request().RequestURI
	}
	p.Use(e)
	return e
}

func setupBadger(cfg *config) (*badger.DB, error) {
	opts := badger.DefaultOptions(cfg.BadgerDBDir)
	opts = opts.WithValueLogLoadingMode(options.FileIO)
	return badger.Open(opts)
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
