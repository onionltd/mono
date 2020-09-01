package main

import baseconfig "github.com/onionltd/mono/pkg/config"

type config struct {
	baseconfig.BaseConfig

	WWWDir string `long:"www" description:"WWW resources directory" required:"yes" env:"WWW_PATH"`
}
