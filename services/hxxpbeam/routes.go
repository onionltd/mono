package main

import (
	"github.com/labstack/echo/v4"
	"github.com/onionltd/mono/pkg/echo/middleware/auth"
	serverutils "github.com/onionltd/mono/pkg/echo/server"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func (s *server) routes() {
	s.router.GET("/", s.handleHello())
	s.router.GET("/png/:id/:address", s.handlePNG())
	s.router.GET("/json/:id/:address", s.handleJSON())
	s.router.GET("/health", serverutils.HandleHealthCheck())
	s.router.GET("/metrics",
		echo.WrapHandler(promhttp.Handler()),
		auth.KeyAuthWithConfig(
			string(s.config.PromMetricsAuth),
		),
	)
}
