package main

import (
	"context"
	"fmt"
	"github.com/jessevdk/go-flags"
	"github.com/labstack/echo/v4"
	loggermw "github.com/onionltd/mono/pkg/echo/middleware/logger"
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

	router := setupRouter(httpdLogger)

	server := server{
		logger: httpdLogger,
		config: cfg,
		router: router,
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
	}()

	wg := sync.WaitGroup{}
	wg.Add(1)

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

func setupRouter(logger *zap.Logger) *echo.Echo {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	e.Use(loggermw.WithConfig(logger))
	e.HTTPErrorHandler = func(err error, c echo.Context) {
		code := http.StatusInternalServerError
		if he, ok := err.(*echo.HTTPError); ok {
			code = he.Code
		}
		_ = c.String(code, fmt.Sprintf("%d %s", code, http.StatusText(code)))
	}
	return e
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
