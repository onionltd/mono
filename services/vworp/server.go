package main

import (
	"errors"
	"fmt"
	"github.com/dgraph-io/badger/v2"
	"github.com/labstack/echo/v4"
	badgerutil "github.com/onionltd/mono/pkg/utils/badger"
	"github.com/onionltd/mono/services/vworp/badger/links"
	"github.com/onionltd/mono/services/vworp/onions"
	"github.com/onionltd/oniontree-tools/pkg/types/service"
	"go.uber.org/zap"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type server struct {
	logger       *zap.Logger
	linksMonitor *onions.Monitor
	router       *echo.Echo
	badgerDB     *badger.DB
	config       *config
	oopsSet      oopsSet
}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

func (s *server) handleRedirectPath() echo.HandlerFunc {
	type pageData struct {
		Service service.Service
		Online  bool
		Link    *links.Link
		Mirror  string
	}
	return func(c echo.Context) error {
		pathTokens := strings.Split(c.Request().URL.String(), "/")[2:]
		serviceID := pathTokens[0]
		path := ""

		if len(pathTokens) > 1 {
			path = strings.Join(pathTokens[1:], "/")
			path, _ = url.QueryUnescape(path)
		}

		online, ok := s.linksMonitor.GetOnlineLinks(serviceID)
		if !ok {
			return c.Redirect(http.StatusSeeOther, fmt.Sprintf("/to/oops/%d", http.StatusNotFound))
		}

		service, err := s.linksMonitor.GetService(serviceID)
		if err != nil {
			return c.Redirect(http.StatusSeeOther, fmt.Sprintf("/to/oops/%d", http.StatusNotFound))
		}

		link, err := links.NewLink(serviceID, path)
		if err != nil {
			s.logger.Error("failed to create a new link", zap.Error(err))
			return c.Redirect(http.StatusSeeOther, fmt.Sprintf("/to/oops/%d", http.StatusInternalServerError))
		}

		pageContent := pageData{}
		pageContent.Service = service
		pageContent.Link = link
		pageContent.Online = len(online) > 0

		if len(online) > 0 {
			pageContent.Mirror = online[0]
		}

		return c.Render(http.StatusOK, "redirect", pageContent)
	}
}

func (s *server) handleRedirectFingerprint() echo.HandlerFunc {
	type pageData struct {
		Service service.Service
		Online  bool
		Link    *links.Link
		Mirror  string
	}
	return func(c echo.Context) error {
		fingerprint := c.Param("fp")

		link := &links.Link{}
		if err := s.badgerDB.View(badgerutil.Load(badgerutil.Key(fingerprint), link)); err != nil {
			if errors.Is(err, badger.ErrKeyNotFound) {
				return c.Redirect(http.StatusSeeOther, fmt.Sprintf("/t/oops/%d", http.StatusNotFound))
			}
			s.logger.Error("failed to read the database", zap.Error(err))
			return c.Redirect(http.StatusSeeOther, fmt.Sprintf("/t/oops/%d", http.StatusInternalServerError))
		}

		online, ok := s.linksMonitor.GetOnlineLinks(link.ServiceID())
		if !ok {
			return c.Redirect(http.StatusSeeOther, fmt.Sprintf("/t/oops/%d", http.StatusNotFound))
		}

		service, err := s.linksMonitor.GetService(link.ServiceID())
		if err != nil {
			return c.Redirect(http.StatusSeeOther, fmt.Sprintf("/t/oops/%d", http.StatusNotFound))
		}

		pageContent := pageData{}
		pageContent.Service = service
		pageContent.Link = link
		pageContent.Online = len(online) > 0

		if len(online) > 0 {
			pageContent.Mirror = online[0]
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

		service, err := s.linksMonitor.GetServiceByURL(
			fmt.Sprintf("%s://%s", u.Scheme, u.Host),
		)
		if err != nil {
			return c.Redirect(http.StatusSeeOther, fmt.Sprintf("/links/oops/%d", http.StatusNotFound))
		}

		// Someone just pasted a vworp! link
		if service.ID == "vworp" {
			// TODO: parse the link and show an information
			// 	This is a legitimate feature which translates shortened links to full links.
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
		id := c.Param("fp")

		link := &links.Link{}
		if err := s.badgerDB.View(badgerutil.Load(badgerutil.Key(id), link)); err != nil {
			if errors.Is(err, badger.ErrKeyNotFound) {
				return c.Redirect(http.StatusSeeOther, fmt.Sprintf("%s/oops/%d", id, http.StatusNotFound))
			}
			s.logger.Error("failed to read the database", zap.Error(err))
			return c.Redirect(http.StatusSeeOther, fmt.Sprintf("%s/oops/%d", id, http.StatusInternalServerError))
		}

		service, err := s.linksMonitor.GetService(link.ServiceID())
		if err != nil {
			return c.Redirect(http.StatusSeeOther, fmt.Sprintf("%s/oops/%d", id, http.StatusNotFound))
		}

		pageContent := pageData{}
		pageContent.Section = queryParamsToSectionName(c.QueryParams())
		pageContent.Service = service
		pageContent.Link = link
		pageContent.ServerAddress = c.Request().Host

		return c.Render(http.StatusOK, "links_view", pageContent)
	}
}

func (s *server) handleLinksOops(oopsMessages oopsMessages) echo.HandlerFunc {
	type pageData struct {
		OopsMessage string
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
		pageContent.OopsMessage = idToOopsMessage(c.Param("id"))
		return c.Render(http.StatusOK, "links_oops", pageContent)
	}
}

func (s *server) handleHome() echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.Render(http.StatusOK, "home", nil)
	}
}

func (s *server) handleHealthCheck() echo.HandlerFunc {
	return func(c echo.Context) error {
		// TODO: return json?
		return c.String(http.StatusOK, "")
	}
}
