package errors

import (
	"fmt"
	"github.com/labstack/echo/v4"
	"net/http"
)

func DefaultErrorHandler(err error, c echo.Context) {
	code := http.StatusInternalServerError
	if he, ok := err.(*echo.HTTPError); ok {
		code = he.Code
	} else {
		code = http.StatusInternalServerError
	}
	if !c.Response().Committed {
		if c.Request().Method == http.MethodHead {
			_ = c.NoContent(code)
		} else {
			_ = c.String(code, fmt.Sprintf("%d %s", code, http.StatusText(code)))
		}
	}
}
