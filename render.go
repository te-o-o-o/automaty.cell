package main

import (
	"image"
	"image/color"
	"image/gif"
	"image/png"
	"math"
	"os"
)

// A palette maps a cell's age to a colour: index 0 is dead, index i is age i,
// and ages beyond the last index use the last colour.
var palettes = map[string]color.Palette{
	"bw":  {color.Black, color.White},
	"age": agePalette(),
}

// agePalette fades newborn cells (pale yellow) through orange and magenta to
// deep blue for old ones. Log scale: the first few generations matter most.
func agePalette() color.Palette {
	stops := []color.RGBA{
		{255, 246, 192, 255},
		{255, 138, 61, 255},
		{194, 24, 91, 255},
		{40, 53, 147, 255},
	}
	p := color.Palette{color.RGBA{11, 11, 20, 255}}
	for age := 1; age <= 255; age++ {
		t := math.Log(float64(age)) / math.Log(255) * float64(len(stops)-1)
		i := min(int(t), len(stops)-2)
		f := t - float64(i)
		a, b := stops[i], stops[i+1]
		lerp := func(x, y uint8) uint8 { return uint8(float64(x) + (float64(y)-float64(x))*f + 0.5) }
		p = append(p, color.RGBA{lerp(a.R, b.R), lerp(a.G, b.G), lerp(a.B, b.B), 255})
	}
	return p
}

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
