package main

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
)

var iconData []byte

func init() {
	const S = 22
	img := image.NewRGBA(image.Rect(0, 0, S, S))

	// Filled circle
	cx, cy, r := 11.0, 10.0, 9.0
	for y := 0; y < S; y++ {
		for x := 0; x < S; x++ {
			dx := float64(x) - cx + 0.5
			dy := float64(y) - cy + 0.5
			if dx*dx+dy*dy <= r*r {
				img.Set(x, y, color.White)
			}
		}
	}

	// Three horizontal text lines
	for li := 0; li < 3; li++ {
		ly := 4 + li*4
		for x := 6; x <= 16; x++ {
			for dy := 0; dy < 2; dy++ {
				img.Set(x, ly+dy, color.Transparent)
			}
		}
	}

	var buf bytes.Buffer
	png.Encode(&buf, img)
	iconData = buf.Bytes()
}
