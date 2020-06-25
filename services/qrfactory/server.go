package main

import (
	"bytes"
	"errors"
	"github.com/disintegration/imaging"
	"github.com/labstack/echo/v4"
	"github.com/skip2/go-qrcode"
	"go.uber.org/zap"
	"image"
	"image/color"
	"image/png"
	"net/http"
)

type server struct {
	logger *zap.Logger
	router *echo.Echo
	config *config
	icons  map[string]image.Image
}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

func (s *server) handleHello() echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"qrfactory": "1.0",
		})
	}
}

func (s *server) handleQR() echo.HandlerFunc {
	const (
		smallSize   = 159
		mediumSize  = 235
		largeSize   = 310
		defaultSize = mediumSize
	)
	parseHexColor := func(s string) (c color.RGBA, err error) {
		errInvalidFormat := errors.New("invalid format")
		c.A = 0xff

		if s[0] != '#' {
			return c, errInvalidFormat
		}

		hexToByte := func(b byte) byte {
			switch {
			case b >= '0' && b <= '9':
				return b - '0'
			case b >= 'a' && b <= 'f':
				return b - 'a' + 10
			case b >= 'A' && b <= 'F':
				return b - 'A' + 10
			}
			err = errInvalidFormat
			return 0
		}

		switch len(s) {
		case 7:
			c.R = hexToByte(s[1])<<4 + hexToByte(s[2])
			c.G = hexToByte(s[3])<<4 + hexToByte(s[4])
			c.B = hexToByte(s[5])<<4 + hexToByte(s[6])
		case 4:
			c.R = hexToByte(s[1]) * 17
			c.G = hexToByte(s[2]) * 17
			c.B = hexToByte(s[3]) * 17
		default:
			err = errInvalidFormat
		}
		return
	}
	getIcon := func(c echo.Context) image.Image {
		v, ok := s.icons[c.QueryParam("icon")]
		if !ok {
			return nil
		}
		return v
	}
	getQRSize := func(c echo.Context) int {
		switch c.QueryParam("size") {
		case "small":
			return smallSize
		case "medium":
			return mediumSize
		case "large":
			return largeSize
		default:
			return defaultSize
		}
	}
	getForegroundColor := func(c echo.Context) string {
		v := c.QueryParam("fg")
		if v == "" {
			return "000"
		}
		return v
	}
	getBackgroundColor := func(c echo.Context) string {
		v := c.QueryParam("bg")
		if v == "" {
			return "FFF"
		}
		return v
	}
	generateQRCode := func(text string, size int, fgColor color.Color, bgColor color.Color) (image.Image, error) {
		qr, err := qrcode.New(text, qrcode.High)
		if err != nil {
			return nil, err
		}
		qr.BackgroundColor = bgColor
		qr.ForegroundColor = fgColor
		return qr.Image(size), nil
	}
	oneFourthOf := func(size int) int {
		return (size / 100) * 25
	}
	addOverlayLogo := func(bgImg image.Image, logoImg image.Image, size int) (image.Image, error) {
		logoImgResized := imaging.Fit(logoImg, size, size, imaging.Lanczos)
		return imaging.OverlayCenter(
			bgImg,
			logoImgResized,
			1,
		), nil
	}
	drawImage := func(c echo.Context, code int, img image.Image) error {
		imgData := bytes.NewBuffer([]byte{})
		if err := png.Encode(imgData, img); err != nil {
			s.logger.Error("failed to encode PNG image", zap.Error(err))
			return c.String(http.StatusInternalServerError, "oops, that didn't work")
		}
		return c.Blob(code, "image/png", imgData.Bytes())
	}
	return func(c echo.Context) error {
		text := c.QueryParam("text")
		if text == "" {
			return c.String(http.StatusBadRequest, "text content for the QR code not provided")
		}

		qrSize := getQRSize(c)

		fgColor, err := parseHexColor(getForegroundColor(c))
		if err != nil {
			return c.String(http.StatusBadRequest, "invalid foreground color")
		}

		bgColor, err := parseHexColor(getBackgroundColor(c))
		if err != nil {
			return c.String(http.StatusBadRequest, "invalid background color")
		}

		qrImage, err := generateQRCode(
			text,
			qrSize,
			fgColor,
			bgColor,
		)
		if err != nil {
			s.logger.Error("failed to generate the QR code", zap.Error(err))
			return c.String(http.StatusBadRequest, "text is too long")
		}

		if icon := getIcon(c); icon != nil {
			qrImage, err = addOverlayLogo(qrImage, icon, oneFourthOf(qrSize))
			if err != nil {
				s.logger.Error("failed to overlay a logo", zap.Error(err))
				return err
			}
		}

		return drawImage(c, http.StatusOK, qrImage)
	}
}

func (s *server) handleHome() echo.HandlerFunc {
	type pageData struct {
		Icons map[string]image.Image
	}
	return func(c echo.Context) error {
		pageContent := pageData{}
		pageContent.Icons = s.icons
		return c.Render(http.StatusOK, "home", pageContent)
	}
}
