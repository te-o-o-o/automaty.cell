package main

import (
	"errors"
	"flag"
	"fmt"
	"image"
	"image/color"
	"math"
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
	rule := flag.String("rule", "B3/S23", "rule in B/S (B3/S23) or Generations S/B/C (345/2/4) notation, or a preset name (see -list-rules)")
	listRules := flag.Bool("list-rules", false, "list preset rules and exit")
	wrap := flag.Bool("wrap", true, "toroidal edges; -wrap=false makes cells beyond the edge dead")
	delay := flag.Int("delay", 5, "GIF frame delay in 1/100 s")
	cyclic := flag.Bool("cyclic", false, "run a cyclic cellular automaton instead of -rule")
	states := flag.Int("states", 14, "cyclic: number of states (2-256)")
	threshold := flag.Int("threshold", 3, "cyclic: neighbours in the next state needed to advance")
	neighborhood := flag.String("neighborhood", "moore", "cyclic: moore or vonneumann")
	radius := flag.Int("radius", 1, "cyclic: neighbourhood radius")
	palette := flag.String("palette", "age", "colour gradient: age, bw, fire, ocean, viridis or mono")
	palFrom := flag.String("palette-from", "", "custom gradient start colour, RRGGBB (with -palette-to)")
	palTo := flag.String("palette-to", "", "custom gradient end colour, RRGGBB (with -palette-from)")
	flag.Parse()

	if *listRules {
		for _, p := range Presets {
			fmt.Printf("%-18s %s\n", p.Name, p.Rule)
		}
		return
	}
	for _, p := range Presets {
		if strings.EqualFold(*rule, p.Name) {
			*rule = p.Rule
		}
	}

	gr, ok := gradients[*palette]
	if !ok {
		fail(2, fmt.Errorf("unknown palette %q (want age, bw, fire, ocean, viridis or mono)", *palette))
	}
	if *palFrom != "" || *palTo != "" {
		from, err1 := parseHex(*palFrom)
		to, err2 := parseHex(*palTo)
		if err := errors.Join(err1, err2); err != nil {
			fail(2, fmt.Errorf("-palette-from and -palette-to: %w", err))
		}
		gr = gradient{black, []color.RGBA{from, to}}
	}

	g := NewGrid(*w, *h, *wrap)
	var step func(*Grid) *Grid
	var pal color.Palette
	if *cyclic {
		c := Cyclic{States: *states, Threshold: *threshold, Radius: *radius}
		switch *neighborhood {
		case "moore":
		case "vonneumann":
			c.VonNeumann = true
		default:
			fail(2, fmt.Errorf("unknown neighborhood %q (want moore or vonneumann)", *neighborhood))
		}
		if c.States < 2 || c.States > 256 || c.Threshold < 1 || c.Radius < 1 || c.Radius >= min(*w, *h) {
			fail(2, fmt.Errorf("cyclic: want 2 <= states <= 256, threshold >= 1, 1 <= radius < grid size"))
		}
		g.RandomizeStates(*seed, c.States)
		step = func(g *Grid) *Grid { return g.StepCyclic(c) }
		// Every state is a live colour, spread evenly.
		pal = gr.palette(c.States, false, func(s int) float64 { return float64(s) / float64(c.States-1) })
	} else {
		r, err := ParseRule(*rule)
		if err != nil {
			fail(2, err)
		}
		g.Randomize(*seed, *density)
		step = func(g *Grid) *Grid { return g.Step(r) }
		if r.States > 0 {
			// Generations: alive first, then dying states evenly.
			pal = gr.palette(r.States, true, func(s int) float64 {
				return float64(s-1) / float64(max(r.States-2, 1))
			})
		} else {
			// B/S ages 1-255, log scale: the first few generations matter most.
			pal = gr.palette(256, true, func(s int) float64 { return math.Log(float64(s)) / math.Log(255) })
		}
	}

	var err error
	if strings.EqualFold(filepath.Ext(*out), ".gif") {
		// ponytail: all frames held in memory (W*H*scale² bytes each), stream if it gets too big
		frames := []*image.Paletted{Render(g, *scale, pal)}
		for i := 1; i < *gens; i++ {
			g = step(g)
			frames = append(frames, Render(g, *scale, pal))
		}
		err = WriteGIF(*out, frames, *delay)
	} else {
		for i := 1; i < *gens; i++ {
			g = step(g)
		}
		err = WritePNG(*out, Render(g, *scale, pal))
	}
	if err != nil {
		fail(1, err)
	}
}

func fail(code int, err error) {
	fmt.Fprintln(os.Stderr, "cellgen:", err)
	os.Exit(code)
}
