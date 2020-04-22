package main

import serverutils "github.com/onionltd/mono/pkg/utils/echo/server"

func (s *server) routes() {
	s.router.GET("/", s.handleHello())
	s.router.GET("/png/:id/:address", s.handlePNG())
	s.router.GET("/json/:id/:address", s.handleJSON())
	s.router.GET("/health", serverutils.HandleHealthCheck())
}
