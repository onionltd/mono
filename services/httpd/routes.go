package main

import (
	"github.com/labstack/echo/v4"
	basicauthmw "github.com/onionltd/mono/pkg/echo/middleware/basicauth"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func (s *server) routes() {
	s.router.File("/robots.txt", s.config.WWWDir+"/robots.txt")
	s.router.Static("/", s.config.WWWDir)
	s.router.GET("/metrics",
		echo.WrapHandler(promhttp.Handler()),
		basicauthmw.WithConfig(
			s.config.PromMetricsAuth.Username(),
			s.config.PromMetricsAuth.Password(),
		),
	)
}
