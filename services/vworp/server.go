package main

import (
	"errors"
	"fmt"
	"github.com/dgraph-io/badger/v2"
	"github.com/labstack/echo/v4"
	captcha "github.com/onionltd/mono/pkg/base64captcha"
	"github.com/onionltd/mono/pkg/oniontree/monitor"
	badgerutil "github.com/onionltd/mono/pkg/utils/badger"
	"github.com/onionltd/mono/services/vworp/badger/links"
	"github.com/onionltd/oniontree-tools/pkg/types/service"
	"go.uber.org/zap"
	"html/template"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type server struct {
	logger       *zap.Logger
	linksMonitor *monitor.Monitor
	router       *echo.Echo
	captcha      *captcha.Captcha
	badgerDB     *badger.DB
	config       *config
	oopsSet      oopsSet
}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

func (s *server) handleRedirect() echo.HandlerFunc {
	type pageData struct {
		Service service.Service
		Online  bool
		Link    *links.Link
		Mirror  string
	}
	isPreview := func(values url.Values) bool {
		for key := range values {
			if key == "preview" {
				return true
			}
		}
		return false
	}
	return func(c echo.Context) error {
		serviceID := c.Param("id")
		fingerprint := c.Param("fp")

		service, err := s.linksMonitor.GetService(serviceID)
		if err != nil {
			return c.Redirect(http.StatusSeeOther, fmt.Sprintf("/to/oops/%d", http.StatusNotFound))
		}

		key := links.NewKey(fingerprint)
		link := &links.Link{}
		if err := s.badgerDB.View(badgerutil.Load(key, link)); err != nil {
			if errors.Is(err, badger.ErrKeyNotFound) {
				return c.Redirect(http.StatusSeeOther, fmt.Sprintf("/to/oops/%d", http.StatusNotFound))
			}
			s.logger.Error("failed to read the database", zap.Error(err))
			return c.Redirect(http.StatusSeeOther, fmt.Sprintf("/to/oops/%d", http.StatusInternalServerError))
		}

		if link.ServiceID() != service.ID {
			return c.Redirect(http.StatusSeeOther, fmt.Sprintf("/to/oops/%d", http.StatusNotFound))
		}

		online, _ := s.linksMonitor.GetOnlineLinks(serviceID)

		pageContent := pageData{}
		pageContent.Service = service
		pageContent.Link = link
		pageContent.Online = len(online) > 0

		if len(online) > 0 {
			pageContent.Mirror = online[0]
		}

		// If there'a an active mirror and preview is disabled, redirect immediately.
		if pageContent.Mirror != "" && !isPreview(c.QueryParams()) {
			dest := pageContent.Mirror + link.Path()
			return c.Redirect(http.StatusSeeOther, dest)
		}

		return c.Render(http.StatusOK, "redirect", pageContent)
	}
}

func (s *server) handleLinksNew() echo.HandlerFunc {
	return func(c echo.Context) error {
		u, err := url.Parse(
			strings.TrimSpace(c.FormValue("link")),
		)
		if err != nil || u.Scheme == "" || u.Host == "" {
			return c.Redirect(http.StatusSeeOther, fmt.Sprintf("/links/oops/%d", http.StatusBadRequest))
		}

		// Force root directory, if not present.
		if u.Path == "" {
			u.Path = "/"
		}

		service, err := s.linksMonitor.GetServiceByURL(
			fmt.Sprintf("%s://%s", u.Scheme, u.Host),
		)
		if err != nil {
			return c.Redirect(http.StatusSeeOther, fmt.Sprintf("/links/oops/%d", http.StatusNotFound))
		}

		// Someone just pasted a vworp! link
		if service.ID == "vworp" {
			return c.Redirect(http.StatusSeeOther, fmt.Sprintf("/links/oops/%d", http.StatusNotAcceptable))
		}

		path := (&url.URL{
			Path:     u.Path,
			RawQuery: u.RawQuery,
			Fragment: u.Fragment,
		}).String()

		link, err := links.NewLink(service.ID, path)
		if err != nil {
			s.logger.Error("failed to create a new link", zap.Error(err))
			return c.Redirect(http.StatusSeeOther, fmt.Sprintf("/links/oops/%d", http.StatusInternalServerError))
		}

		if err := s.badgerDB.Update(badgerutil.Store(link)); err != nil {
			s.logger.Error("failed to update the database", zap.Error(err))
			return c.Redirect(http.StatusSeeOther, fmt.Sprintf("/links/oops/%d", http.StatusInternalServerError))
		}

		return c.Redirect(http.StatusSeeOther, fmt.Sprintf("/links/%s?new", link.Fingerprint()))
	}
}

func (s *server) handleLinksView() echo.HandlerFunc {
	type pageData struct {
		Section       string
		Service       service.Service
		ServerAddress string
		Link          *links.Link
	}
	queryParamsToSectionName := func(values url.Values) string {
		sections := []string{"new"}
		for key := range values {
			for _, section := range sections {
				if key == section {
					return section
				}
			}
		}
		return ""
	}
	return func(c echo.Context) error {
		fingerprint := c.Param("fp")

		key := links.NewKey(fingerprint)
		link := &links.Link{}
		if err := s.badgerDB.View(badgerutil.Load(key, link)); err != nil {
			if errors.Is(err, badger.ErrKeyNotFound) {
				return c.Redirect(http.StatusSeeOther, fmt.Sprintf("%s/oops/%d", fingerprint, http.StatusNotFound))
			}
			s.logger.Error("failed to read the database", zap.Error(err))
			return c.Redirect(http.StatusSeeOther, fmt.Sprintf("%s/oops/%d", fingerprint, http.StatusInternalServerError))
		}

		service, err := s.linksMonitor.GetService(link.ServiceID())
		if err != nil {
			return c.Redirect(http.StatusSeeOther, fmt.Sprintf("%s/oops/%d", fingerprint, http.StatusNotFound))
		}

		pageContent := pageData{}
		pageContent.Section = queryParamsToSectionName(c.QueryParams())
		pageContent.Service = service
		pageContent.Link = link
		pageContent.ServerAddress = c.Request().Host

		return c.Render(http.StatusOK, "links_view", pageContent)
	}
}

func (s *server) handleLinksOops(oopsMessages oopsMessages, showSubmitForm bool) echo.HandlerFunc {
	type pageData struct {
		OopsMessage    string
		ShowSubmitForm bool
	}
	idToOopsMessage := func(val string) string {
		num, err := strconv.Atoi(val)
		if err != nil {
			num = 0
		}
		return oopsMessages.Get(num)
	}
	return func(c echo.Context) error {
		pageContent := pageData{}
		pageContent.ShowSubmitForm = showSubmitForm
		pageContent.OopsMessage = idToOopsMessage(c.Param("id"))
		return c.Render(http.StatusOK, "links_oops", pageContent)
	}
}

func (s *server) handleCaptcha() echo.HandlerFunc {
	type pageData struct {
		CaptchaBase64 template.URL
		QueryParams   url.Values
	}
	return func(c echo.Context) error {
		b64, err := s.captcha.GetImageData(c.QueryParam("cid"))
		if err != nil {
			return c.Redirect(http.StatusSeeOther, "/")
		}
		pageContent := pageData{}
		pageContent.CaptchaBase64 = template.URL(b64)
		pageContent.QueryParams = c.QueryParams()
		return c.Render(http.StatusOK, "captcha", pageContent)
	}
}

func (s *server) handlePage(name string) echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.Render(http.StatusOK, name, nil)
	}
}
