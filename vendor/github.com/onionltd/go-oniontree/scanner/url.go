package scanner

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
)

func Normalize(u string) (string, error) {
	r, err := url.Parse(u)
	if err != nil {
		return "", err
	}
	if r.Scheme == "" || r.Host == "" {
		return "", errors.New("invalid url format")
	}
	return r.Scheme + "://" + r.Host, nil
}

func ParseHostPort(u string) (string, error) {
	schemeToPortNumber := func(s string) string {
		switch s {
		case "", "http":
			return "80"
		case "https":
			return "443"
		}
		return ""
	}
	r, err := url.Parse(u)
	if err != nil {
		return "", err
	}
	host := r.Host
	if r.Port() == "" {
		port := schemeToPortNumber(r.Scheme)
		host = fmt.Sprintf("%s", strings.Join([]string{host, port}, ":"))
	}
	return host, nil
}
