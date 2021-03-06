package logger

import (
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
	"net/http"
)

func WithConfig(logger *zap.Logger) echo.MiddlewareFunc {
	return func(h echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			err := h(c)
			statusCode := 0
			if v, ok := err.(*echo.HTTPError); ok {
				statusCode = v.Code
			} else {
				statusCode = c.Response().Status
			}

			if writer := logger.Check(zap.InfoLevel, ""); writer != nil {
				writer.Write(
					zap.Reflect("request", map[string]interface{}{
						"method":     c.Request().Method,
						"path":       c.Request().URL.RequestURI(),
						"user_agent": c.Request().UserAgent(),
					}),
					zap.Reflect("response", map[string]interface{}{
						"code":   statusCode,
						"status": http.StatusText(statusCode),
					}),
				)
			}
			return err
		}
	}
}
