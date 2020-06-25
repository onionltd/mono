package main

import baseconfig "github.com/onionltd/mono/pkg/services/config"

type config struct {
	baseconfig.BaseConfig

	WWWDir           string `long:"www" description:"WWW resources directory" required:"yes" env:"WWW_PATH"`
	TemplatesPattern string `long:"templates" description:"A glob pattern to match files" required:"yes" env:"TEMPLATES_GLOB"`
	IconsPattern     string `long:"icons" description:"A glob pattern to match files" required:"yes" env:"ICONS_GLOB"`
}
