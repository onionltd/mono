package main

type config struct {
	Listen   string `long:"listen" description:"Listen on address" default:":8080" env:"HTTP_LISTEN"`
	WWWDir   string `long:"www" description:"WWW resources directory" required:"yes" env:"WWW_PATH"`
	LogLevel string `long:"log-level" description:"Set log level" default:"info" env:"LOG_LEVEL"`
}
