package main

import (
	"context"
	"fmt"
	"github.com/dgraph-io/badger/v2"
	"github.com/jessevdk/go-flags"
	"github.com/labstack/echo/v4"
	"github.com/onionltd/mono/services/vworp/onions"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"net/http"
	"os"
	"os/signal"
	"strings"
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
	monitorLogger := rootLogger.Named("monitor")
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

	monitor := onions.NewMonitor(monitorLogger)
	router := setupRouter(httpdLogger, templates)

	server := server{
		logger:       httpdLogger,
		config:       cfg,
		linksMonitor: monitor,
		router:       router,
		badgerDB:     db,
		oopsSet:      oopsies,
	}
	server.routes()

	wg := sync.WaitGroup{}
	wg.Add(2)

	go func() {
		defer wg.Done()
		if err := monitor.Start(cfg.OnionTreeDir); err != nil {
			rootLogger.Error("monitor error", zap.Error(err))
		}
	}()
	go func() {
		defer wg.Done()
		if err := router.Start(cfg.Listen); err != nil {
			if err != http.ErrServerClosed {
				rootLogger.Error("http server error", zap.Error(err))
			}
		}
	}()

	// Handle termination signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)

	go func() {
		<-sigCh
		rootLogger.Warn("received a termination signal")
		ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
		_ = router.Shutdown(ctx)
		_ = monitor.Stop(ctx)
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
	translateLogLevel := func(s string) zapcore.Level {
		switch strings.ToLower(s) {
		case "debug":
			return zap.DebugLevel
		case "info":
			return zap.InfoLevel
		case "warning":
			return zap.WarnLevel
		case "error":
			return zap.ErrorLevel
		default:
			return zap.InfoLevel
		}
	}
	logLevel := translateLogLevel(cfg.LogLevel)
	return zap.Config{
		Encoding:         "console",
		Level:            zap.NewAtomicLevelAt(logLevel),
		OutputPaths:      []string{"stderr"},
		ErrorOutputPaths: []string{"stderr"},
		EncoderConfig: zapcore.EncoderConfig{
			MessageKey: "message",

			LevelKey:    "level",
			EncodeLevel: zapcore.CapitalLevelEncoder,

			TimeKey:    "time",
			EncodeTime: zapcore.ISO8601TimeEncoder,

			NameKey:    "name",
			EncodeName: zapcore.FullNameEncoder,
		},
	}.Build()
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
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// FIXME: use only a single log message
			logger.Info("request",
				zap.String("method", c.Request().Method),
				zap.String("url", c.Request().URL.Path),
				zap.String("user_agent", c.Request().UserAgent()),
			)
			logger.Info("response",
				zap.Int("status", c.Response().Status),
			)
			return next(c)
		}
	})
	e.HTTPErrorHandler = func(err error, c echo.Context) {
		code := http.StatusInternalServerError
		if he, ok := err.(*echo.HTTPError); ok {
			code = he.Code
		}
		_ = c.String(code, fmt.Sprintf("%d %s", code, http.StatusText(code)))
	}
	return e
}

func setupBadger(cfg *config) (*badger.DB, error) {
	return badger.Open(badger.DefaultOptions(cfg.DatabaseDir))
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}
