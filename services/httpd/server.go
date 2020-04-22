package main

import (
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
	"net/http"
)

type server struct {
	logger *zap.Logger
	router *echo.Echo
	config *config
}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}
