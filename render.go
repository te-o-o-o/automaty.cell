package main

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg" // masks may be JPEG; PNG is registered by main.go
)

// Render draws the grid with palette pal, each cell scale×scale pixels.
// Paletted works for both PNG and GIF frames.
func Render(g *Grid, scale int, pal color.Palette) *image.Paletted {
	img := image.NewPaletted(image.Rect(0, 0, g.W*scale, g.H*scale), pal)
	last := uint8(len(pal) - 1)
	for y := 0; y < g.H; y++ {
		for x := 0; x < g.W; x++ {
			idx := min(g.Cells[y*g.W+x], last)
			if idx == 0 {
				continue // index 0 is the zero value
			}
			for py := 0; py < scale; py++ {
				for px := 0; px < scale; px++ {
					img.SetColorIndex(x*scale+px, y*scale+py, idx)
				}
			}
		}
	}
	return img
}

// loadMask decodes a PNG or JPEG image and samples it on a w×h grid: a cell is
// inside where the image is opaque if it has transparent parts (a cut-out
// logo), else where it is light (a white silhouette on black).
func loadMask(data []byte, w, h int) ([]bool, error) {
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("mask: %w", err)
	}
	if cfg.Width*cfg.Height > 25_000_000 { // decoding would take gigabytes
		return nil, errors.New("mask: image larger than 25 megapixels")
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("mask: %w", err)
	}
	b := img.Bounds()
	light, opaque := make([]bool, w*h), make([]bool, w*h)
	transparent := false
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := img.At(b.Min.X+x*b.Dx()/w, b.Min.Y+y*b.Dy()/h)
			_, _, _, a := c.RGBA()
			i := y*w + x
			opaque[i] = a > 0x7fff
			light[i] = color.GrayModel.Convert(c).(color.Gray).Y >= 128
			transparent = transparent || !opaque[i]
		}
	}
	if transparent {
		return opaque, nil
	}
	return light, nil
}
