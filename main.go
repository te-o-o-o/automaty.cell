package main

import (
	"flag"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	w := flag.Int("w", 200, "grid width in cells")
	h := flag.Int("h", 200, "grid height in cells")
	scale := flag.Int("scale", 4, "pixels per cell")
	gens := flag.Int("gens", 100, "number of generations to run")
	seed := flag.Int64("seed", 1, "random seed")
	density := flag.Float64("density", 0.3, "initial fraction of live cells")
	out := flag.String("o", "out.png", "output file")
	rule := flag.String("rule", "B3/S23", "rule in B/S notation")
	delay := flag.Int("delay", 5, "GIF frame delay in 1/100 s")
	palette := flag.String("palette", "age", "colour palette: age or bw")
	flag.Parse()

	r, err := ParseRule(*rule)
	if err != nil {
		fmt.Fprintln(os.Stderr, "cellgen:", err)
		os.Exit(2)
	}

	pal, ok := palettes[*palette]
	if !ok {
		fmt.Fprintf(os.Stderr, "cellgen: unknown palette %q (want age or bw)\n", *palette)
		os.Exit(2)
	}

	g := NewGrid(*w, *h)
	g.Randomize(*seed, *density)

	if strings.EqualFold(filepath.Ext(*out), ".gif") {
		// ponytail: all frames held in memory (W*H*scale² bytes each), stream if it gets too big
		frames := []*image.Paletted{Render(g, *scale, pal)}
		for i := 1; i < *gens; i++ {
			g = g.Step(r)
			frames = append(frames, Render(g, *scale, pal))
		}
		err = WriteGIF(*out, frames, *delay)
	} else {
		for i := 1; i < *gens; i++ {
			g = g.Step(r)
		}
		err = WritePNG(*out, Render(g, *scale, pal))
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "cellgen:", err)
		os.Exit(1)
	}
}
