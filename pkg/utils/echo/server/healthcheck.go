package server

import (
	"github.com/labstack/echo/v4"
	"net/http"
)

func HandleHealthCheck() echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"status": "ok",
		})
	}
}
