package main

import (
	"github.com/labstack/echo/v4"
	"net/http"
)

func (s *server) solveCaptcha() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		checkSolution := func(id, solution string) bool {
			return s.captcha.Verify(id, solution)
		}
		solveRedirect := func(c echo.Context) error {
			formValues, err := c.FormParams()
			if err != nil {
				// TODO: redirect to /sorry/oops?
				return c.Redirect(http.StatusSeeOther, "/")
			}

			captchaID, err := s.captcha.Generate()
			if err != nil {
				// TODO: redirect to /sorry/oops?
				return c.Redirect(http.StatusSeeOther, "/")
			}

			formValues.Set("cid", captchaID)
			formValues.Set("continue", c.Request().RequestURI)
			// Remove captcha specific fields to prevent duplication.
			formValues.Del("solution")
			return c.Redirect(http.StatusSeeOther, "/sorry?"+formValues.Encode())
		}
		wrongSolutionRedirect := func(c echo.Context) error {
			formValues, err := c.FormParams()
			if err != nil {
				// TODO: redirect to /sorry/oops?
				return c.Redirect(http.StatusSeeOther, "/")
			}

			captchaID, err := s.captcha.Generate()
			if err != nil {
				// TODO: redirect to /sorry/oops?
				return c.Redirect(http.StatusSeeOther, "/")
			}

			formValues.Set("cid", captchaID)
			// Remove captcha specific fields to prevent duplication.
			formValues.Del("solution")
			return c.Redirect(http.StatusSeeOther, "/sorry?"+formValues.Encode())
		}
		return func(c echo.Context) error {
			continueURL := c.FormValue("continue")
			if continueURL == "" {
				return solveRedirect(c)
			}

			id := c.FormValue("cid")
			solution := c.FormValue("solution")
			if !checkSolution(id, solution) {
				return wrongSolutionRedirect(c)
			}

			return next(c)
		}
	}
}
