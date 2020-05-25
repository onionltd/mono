package auth

import (
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func KeyAuthWithConfig(key string) echo.MiddlewareFunc {
	if key == "" {
		return noOp()
	}
	return middleware.KeyAuth(func(authKey string, c echo.Context) (bool, error) {
		if authKey == key {
			return true, nil
		}
		return false, nil
	})
}
