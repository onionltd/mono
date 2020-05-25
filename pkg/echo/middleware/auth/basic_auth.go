package auth

import (
	"crypto/subtle"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func BasicAuthWithConfig(username, password string) echo.MiddlewareFunc {
	if username == "" || password == "" {
		return noOp()
	}
	return middleware.BasicAuth(func(authUsername, authPassword string, c echo.Context) (bool, error) {
		if subtle.ConstantTimeCompare([]byte(authUsername), []byte(username)) == 1 &&
			subtle.ConstantTimeCompare([]byte(authPassword), []byte(password)) == 1 {
			return true, nil
		}
		return false, nil
	})
}
