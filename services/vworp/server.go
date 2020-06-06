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
	"sort"
	"strconv"
	"strings"
	"time"
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
	sortAddressesV3First := func(addrs []string) {
		sort.SliceStable(addrs, func(i, j int) bool {
			return len(addrs[i]) > len(addrs[j])
		})
	}
	oops := func(c echo.Context, code int, showSubmitForm bool) error {
		return s.handleOops(&code, false)(c)
	}
	return func(c echo.Context) error {
		serviceID := c.Param("id")
		fingerprint := c.Param("fp")

		service, err := s.linksMonitor.GetService(serviceID)
		if err != nil {
			return oops(c, http.StatusNotFound, false)
		}

		key := links.NewKey(fingerprint)
		link := &links.Link{}
		if err := s.badgerDB.View(badgerutil.Load(key, link)); err != nil {
			if errors.Is(err, badger.ErrKeyNotFound) {
				return oops(c, http.StatusNotFound, false)
			}
			s.logger.Error("failed to read the database", zap.Error(err))
			return oops(c, http.StatusInternalServerError, false)
		}

		if link.ServiceID() != service.ID {
			return oops(c, http.StatusNotFound, false)
		}

		online, _ := s.linksMonitor.GetOnlineLinks(serviceID)
		sortAddressesV3First(online)

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
	oops := func(c echo.Context, code int) error {
		return c.Redirect(http.StatusSeeOther, fmt.Sprintf("/links/oops/%d", code))
	}
	return func(c echo.Context) error {
		u, err := url.Parse(
			strings.TrimSpace(c.FormValue("link")),
		)
		if err != nil || u.Scheme == "" || u.Host == "" {
			return oops(c, http.StatusBadRequest)
		}

		// Force root directory, if not present.
		if u.Path == "" {
			u.Path = "/"
		}

		service, err := s.linksMonitor.GetServiceByURL(
			fmt.Sprintf("%s://%s", u.Scheme, u.Host),
		)
		if err != nil {
			return oops(c, http.StatusNotFound)
		}

		// Someone just pasted a vworp! link
		if service.ID == "vworp" {
			return oops(c, http.StatusNotAcceptable)
		}

		path := (&url.URL{
			Path:     u.Path,
			RawQuery: u.RawQuery,
			Fragment: u.Fragment,
		}).String()

		link, err := links.NewLink(service.ID, path)
		if err != nil {
			s.logger.Error("failed to create a new link", zap.Error(err))
			return oops(c, http.StatusInternalServerError)
		}

		if err := s.badgerDB.Update(badgerutil.Store(link)); err != nil {
			s.logger.Error("failed to update the database", zap.Error(err))
			return oops(c, http.StatusInternalServerError)
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
	oops := func(c echo.Context, code int, showSubmitForm bool) error {
		return s.handleOops(&code, false)(c)
	}
	return func(c echo.Context) error {
		fingerprint := c.Param("fp")

		key := links.NewKey(fingerprint)
		link := &links.Link{}
		if err := s.badgerDB.View(badgerutil.Load(key, link)); err != nil {
			if errors.Is(err, badger.ErrKeyNotFound) {
				return oops(c, http.StatusNotFound, false)
			}
			s.logger.Error("failed to read the database", zap.Error(err))
			return oops(c, http.StatusInternalServerError, false)
		}

		service, err := s.linksMonitor.GetService(link.ServiceID())
		if err != nil {
			return oops(c, http.StatusNotFound, false)
		}

		pageContent := pageData{}
		pageContent.Section = queryParamsToSectionName(c.QueryParams())
		pageContent.Service = service
		pageContent.Link = link
		pageContent.ServerAddress = c.Request().Host

		return c.Render(http.StatusOK, "links_view", pageContent)
	}
}

func (s *server) handleOops(oopsID *int, showSubmitForm bool) echo.HandlerFunc {
	type oopsPageContent struct {
		OopsMessage    string
		ShowSubmitForm bool
	}
	newOopsPageContent := func(c echo.Context, code int, showSubmitForm bool) oopsPageContent {
		oopsSet := s.oopsSet[c.Path()]
		return oopsPageContent{
			OopsMessage: oopsSet.Get(code),
		}
	}
	deduceStatusCode := func(oopsID int) int {
		if http.StatusText(oopsID) == "" {
			return http.StatusOK
		}
		return oopsID
	}
	return func(c echo.Context) error {
		var code int
		if oopsID != nil {
			code = *oopsID
		} else {
			code, _ = strconv.Atoi(c.Param("id"))
		}
		return c.Render(deduceStatusCode(code), "oops", newOopsPageContent(c, code, showSubmitForm))
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

func (s *server) handleBackupBadgerDB() echo.HandlerFunc {
	newBackupFilename := func() string {
		return fmt.Sprintf("backup-%s.dat", time.Now().Format("2006-01-02_1504"))
	}
	setContentDisposition := func(resp *echo.Response, filename string) {
		v := fmt.Sprintf("attachment; filename=\"%s\"", filename)
		resp.Header().Set("Content-Disposition", v)
	}
	return func(c echo.Context) error {
		since := c.QueryParam("since")
		if since == "" {
			since = "0"
		}
		tsSince, err := strconv.ParseUint(since, 10, 64)
		if err != nil {
			return c.String(http.StatusBadRequest, "query parameter `since` is not a timestamp")
		}

		// Set Content-Disposition header
		setContentDisposition(c.Response(), newBackupFilename())

		if _, err := s.badgerDB.Backup(c.Response().Writer, tsSince); err != nil {
			s.logger.Error("failed to backup the badger database", zap.Error(err))
			return err
		}
		return nil
	}
}

func (s *server) handlePage(name string) echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.Render(http.StatusOK, name, nil)
	}
}
