package main

type config struct {
	Listen       string `long:"listen" description:"Listen on address" default:":8080" env:"HTTP_LISTEN"`
	WWWDir       string `long:"www" description:"WWW resources directory" required:"yes" env:"WWW_PATH"`
	TemplatesDir string `long:"templates" description:"Templates directory" required:"yes" env:"TEMPLATES_PATH"`
	OnionTreeDir string `long:"oniontree" description:"OnionTree directory" required:"yes" env:"ONIONTREE_PATH"`
	DatabaseDir  string `long:"database" description:"Database directory" required:"yes" env:"BADGERDB_PATH"`
	LogLevel     string `long:"log-level" description:"Set log level" env:"LOG_LEVEL"`
}
