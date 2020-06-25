package main

import (
	"github.com/labstack/echo/v4"
	"github.com/onionltd/mono/pkg/echo/middleware/auth"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func (s *server) routes() {
	s.router.GET("/", s.handleHome())
	s.router.GET("/metrics",
		echo.WrapHandler(promhttp.Handler()),
		auth.KeyAuthWithConfig(
			string(s.config.PromMetricsAuth),
		),
	)
	s.router.GET("/qr/generate", s.handleQR())

	s.router.File("/robots.txt", s.config.WWWDir+"/robots.txt")

	s.router.Static("/static", s.config.WWWDir)
}
