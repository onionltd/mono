package main

import (
	"errors"
	"fmt"
	"github.com/labstack/echo/v4"
	"github.com/onionltd/mono/services/vworp/onions"
	"github.com/onionltd/oniontree-tools/pkg/oniontree"
	"github.com/onionltd/oniontree-tools/pkg/types/service"
	"go.uber.org/zap"
	"net/http"
	"net/url"
	"strings"
)

type server struct {
	logger       *zap.Logger
	linksMonitor *onions.Monitor
	router       *echo.Echo
	config       *config
}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

func (s *server) handleServiceRedirect() echo.HandlerFunc {
	type pageData struct {
		Service service.Service
		ID      string
		Mirror  string
		URL     string
	}
	return func(c echo.Context) error {
		pathTokens := strings.Split(c.Request().URL.String(), "/")[2:]
		serviceID := pathTokens[0]
		serviceURL := ""

		if len(pathTokens) > 1 {
			serviceURL = strings.Join(pathTokens[1:], "/")
			serviceURL, _ = url.QueryUnescape(serviceURL)
		}

		online, ok := s.linksMonitor.GetOnlineLinks(serviceID)
		if !ok {
			return c.Render(http.StatusNotFound, "service_notfound", nil)
		}

		service, err := s.linksMonitor.GetService(serviceID)
		if err != nil {
			return c.Render(http.StatusNotFound, "service_notfound", nil)
		}

		pageContent := pageData{}
		pageContent.ID = serviceID
		pageContent.Service = service
		pageContent.URL = serviceURL

		if len(online) == 0 {
			return c.Render(http.StatusOK, "service_offline", pageContent)
		}

		pageContent.Mirror = online[0]

		return c.Render(http.StatusOK, "service_online", pageContent)
	}
}

func (s *server) handleServiceNewURL() echo.HandlerFunc {
	type pageData struct {
		Service service.Service
		Link    string
		Error   string
	}
	errParseURI := errors.New("uri parser failed")
	toErrorMessage := func(err error) string {
		switch err {
		case errParseURI:
			return "This doesn't look like a valid link."
		case onions.ErrUrlNotFound, oniontree.ErrIdNotExists:
			return "Your link does not belong to any service vworp! can recognize."
		default:
			return "Hmm... Something has broken but don't worry it's not your fault."
		}
	}
	return func(c echo.Context) error {
		pageContent := pageData{}
		link := strings.TrimSpace(c.FormValue("link"))
		u, err := url.ParseRequestURI(link)
		if err != nil {
			err = errParseURI
			pageContent.Error = toErrorMessage(err)
			return c.Render(http.StatusOK, "service_newlink", pageContent)
		}

		serviceURL := fmt.Sprintf("%s://%s", u.Scheme, u.Host)
		service, err := s.linksMonitor.GetServiceByURL(serviceURL)
		if err != nil {
			pageContent.Error = toErrorMessage(err)
			return c.Render(http.StatusOK, "service_newlink", pageContent)
		}

		pageContent.Link = (&url.URL{
			Scheme:   "http",
			Host:     c.Request().Host,
			Path:     "to/" + service.ID + u.Path,
			RawQuery: u.RawQuery,
		}).String()
		pageContent.Service = service

		return c.Render(http.StatusOK, "service_newlink", pageContent)
	}
}

func (s *server) handleHome() echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.Render(http.StatusOK, "home", nil)
	}
}

func (s *server) handleHealthCheck() echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.String(http.StatusOK, "")
	}
}

func (s *server) handleRedirectHome() echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.Redirect(http.StatusMovedPermanently, "/")
	}
}
