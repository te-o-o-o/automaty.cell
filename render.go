package main

import (
	"image"
	"image/color"
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
