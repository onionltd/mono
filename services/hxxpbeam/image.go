package main

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
)

func newImage(color color.RGBA) ([]byte, error) {
	img := image.NewRGBA(image.Rectangle{image.Point{0, 0}, image.Point{1, 1}})
	img.Set(0, 0, color)
	imgData := bytes.NewBuffer([]byte{})
	if err := png.Encode(imgData, img); err != nil {
		return []byte{}, err
	}
	return imgData.Bytes(), nil
}
