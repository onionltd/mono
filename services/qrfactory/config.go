package main

import baseconfig "github.com/onionltd/mono/pkg/services/config"

type config struct {
	baseconfig.BaseConfig

	ImagesPattern string `long:"images" description:"A glob pattern to match files" required:"yes" env:"IMAGES_GLOB"`
}
