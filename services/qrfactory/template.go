package main

import (
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
	"html/template"
	"io"
)

type Templates struct {
	logger    *zap.Logger
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
	err := t.templates.ExecuteTemplate(w, name, data)
	if err != nil {
		t.logger.Error("template error", zap.Error(err))
	}
	return err
}
