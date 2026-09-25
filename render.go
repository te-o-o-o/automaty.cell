package main

import (
	"image"
	"image/color"
	"image/gif"
	"image/png"
	"os"
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

func WritePNG(path string, img image.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := png.Encode(f, img); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

// WriteGIF writes frames as a looping animation, delay in 1/100 s per frame.
func WriteGIF(path string, frames []*image.Paletted, delay int) error {
	anim := &gif.GIF{Image: frames, Delay: make([]int, len(frames))}
	for i := range anim.Delay {
		anim.Delay[i] = delay
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := gif.EncodeAll(f, anim); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}
