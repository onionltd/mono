package main

import (
	"github.com/labstack/echo/v4"
	"github.com/oniontree-org/go-oniontree"
	"github.com/oniontree-org/go-oniontree/scanner"
	"github.com/oniontree-org/go-oniontree/scanner/evtcache"
	"go.uber.org/zap"
	"image/color"
	"net/http"
	"net/url"
)

type server struct {
	logger *zap.Logger
	router *echo.Echo
	config *config
	cache  *evtcache.Cache
	ot     *oniontree.OnionTree
}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

func (s *server) handleHello() echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"hxxpbeam": "2.0",
		})
	}
}

func (s *server) handleJSON() echo.HandlerFunc {
	type response struct {
		Status string `json:"status,omitempty"`
		Error  string `json:"error,omitempty"`
	}
	return func(c echo.Context) error {
		serviceID := c.Param("id")
		address := c.Param("address")

		address, err := url.PathUnescape(address)
		if err != nil {
			return c.JSON(http.StatusBadRequest, response{Error: err.Error()})
		}

		service, err := s.ot.GetService(serviceID)
		if err != nil {
			if _, ok := err.(*oniontree.ErrIdNotExists); ok {
				return c.JSON(http.StatusNotFound, response{Error: "service not found"})
			}
			return c.JSON(http.StatusInternalServerError, response{Error: "oops, something is wrong"})
		}

		// Find if given address belongs to the service
		found := false
		for i := range service.URLs {
			if service.URLs[i] == address {
				found = true
				break
			}
		}
		if !found {
			return c.JSON(http.StatusNotFound, response{Error: "address does not belong to the service"})
		}

		status := scanner.StatusOffline
		if addrs, ok := s.cache.GetAddresses(serviceID); ok {
			if v, ok := addrs[address]; ok {
				status = v
			}
		}

		return c.JSON(http.StatusOK, response{Status: status.String()})
	}
}

func (s *server) handlePNG() echo.HandlerFunc {
	// Set response headers so that client never caches the response.
	preventClientCaching := func(c echo.Context) {
		resp := c.Response()
		resp.Header().Set("Cache-Control", "no-store, must-revalidate")
		resp.Header().Set("Expires", "0")
	}
	statusToColor := func(status string) color.RGBA {
		switch status {
		case "offline":
			// #FF6347
			return color.RGBA{R: 0xFF, G: 0x64, B: 0x47, A: 0xFF}
		case "online":
			// #ADFF2F
			return color.RGBA{R: 0xAD, G: 0xFF, B: 0x2F, A: 0xFF}
		default:
			// #D3D3D3
			return color.RGBA{R: 0xD3, G: 0xD3, B: 0xD3, A: 0xFF}
		}
	}
	drawImage := func(c echo.Context, code int, status string) error {
		preventClientCaching(c)
		contentType := "image/png"
		b, err := newImage(statusToColor(status))
		if err != nil {
			return c.Blob(http.StatusInternalServerError, contentType, b)
		}
		return c.Blob(code, contentType, b)
	}
	return func(c echo.Context) error {
		serviceID := c.Param("id")
		address := c.Param("address")

		address, err := url.PathUnescape(address)
		if err != nil {
			return drawImage(c, http.StatusBadRequest, "error")
		}

		service, err := s.ot.GetService(serviceID)
		if err != nil {
			if _, ok := err.(*oniontree.ErrIdNotExists); ok {
				return drawImage(c, http.StatusNotFound, "error")
			}
			return drawImage(c, http.StatusInternalServerError, "error")
		}

		// Find if given address belongs to the service
		found := false
		for i := range service.URLs {
			if service.URLs[i] == address {
				found = true
				break
			}
		}
		if !found {
			return drawImage(c, http.StatusNotFound, "error")
		}

		status := scanner.StatusOffline
		if addrs, ok := s.cache.GetAddresses(serviceID); ok {
			if v, ok := addrs[address]; ok {
				status = v
			}
		}

		return drawImage(c, http.StatusOK, status.String())
	}
}
