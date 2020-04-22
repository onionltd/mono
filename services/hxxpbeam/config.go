package main

type config struct {
	Listen       string `long:"listen" description:"Listen on address" default:":8080" env:"HTTP_LISTEN"`
	OnionTreeDir string `long:"oniontree" description:"OnionTree directory" required:"yes" env:"ONIONTREE_PATH"`
	LogLevel     string `long:"log-level" description:"Set log level" default:"info" env:"LOG_LEVEL"`
}
