package main

import (
	"github.com/labstack/echo/v4"
	"html/template"
	"io"
)

type Templates struct {
	templates *template.Template
}

func (t *Templates) Load(pattern string) (err error) {
	t.templates, err = template.ParseGlob(pattern)
	if err != nil {
		return err
	}
	return nil
}

func (t *Templates) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}
