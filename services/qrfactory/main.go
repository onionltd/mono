package main

import (
	"context"
	"fmt"
	"github.com/disintegration/imaging"
	"github.com/jessevdk/go-flags"
	"github.com/labstack/echo-contrib/prometheus"
	"github.com/labstack/echo/v4"
	echoerrors "github.com/onionltd/mono/pkg/echo/errors"
	loggermw "github.com/onionltd/mono/pkg/echo/middleware/logger"
	zaputil "github.com/onionltd/mono/pkg/utils/zap"
	"go.uber.org/zap"
	"image"
	"net/http"
	"os"
	"os/signal"
	"path"
	"path/filepath"
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
	httpdLogger := rootLogger.Named("httpd")
	templatesLogger := rootLogger.Named("templates")

	templates, err := setupTemplates(templatesLogger, cfg)
	if err != nil {
		return err
	}

	images, err := setupImages(cfg)
	if err != nil {
		return err
	}

	router := setupRouter(httpdLogger, templates)

	// Setup prometheus metrics
	setupRouterMetrics(router)

	server := server{
		logger: httpdLogger,
		config: cfg,
		router: router,
		images: images,
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

func setupTemplates(logger *zap.Logger, cfg *config) (*Templates, error) {
	t := &Templates{
		logger: logger,
	}
	if err := t.Load(cfg.TemplatesPattern); err != nil {
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

func setupImages(cfg *config) (map[string]image.Image, error) {
	files, err := filepath.Glob(cfg.ImagesPattern)
	if err != nil {
		return nil, err
	}
	images := make(map[string]image.Image)
	for i := range files {
		img, err := imaging.Open(files[i])
		if err != nil {
			return nil, err
		}
		name := path.Base(files[i])
		id := strings.TrimSuffix(name, filepath.Ext(name))
		images[id] = img
	}
	return images, nil
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
