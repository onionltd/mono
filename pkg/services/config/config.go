package config

type BaseConfig struct {
	Listen   string `long:"listen" description:"Listen on address" default:":8080" env:"HTTP_LISTEN"`
	LogLevel string `long:"log-level" description:"Set log level" default:"info" env:"LOG_LEVEL"`
}
