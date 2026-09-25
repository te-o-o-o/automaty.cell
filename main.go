package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"image"
	"image/color"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
)

// options holds every setting of a run. The CLI fills it from os.Args and the
// web server from query parameters, through the same flags.
type options struct {
	w, h, scale, gens         int
	seed                      int64
	symmetry                  int
	density                   float64
	rule                      string
	wrap                      bool
	delay                     int
	cyclic                    bool
	states, threshold, radius int
	neighborhood              string
	palette, palFrom, palTo   string
	colors                    string
	gif                       bool // web only: GIF instead of PNG
	out, serve                string
	listRules                 bool
}

func newFlagSet(o *options) *flag.FlagSet {
	fs := flag.NewFlagSet("cellgen", flag.ContinueOnError)
	fs.IntVar(&o.w, "w", 380, "grid width in cells")
	fs.IntVar(&o.h, "h", 380, "grid height in cells")
	fs.IntVar(&o.scale, "scale", 2, "pixels per cell")
	fs.IntVar(&o.gens, "gens", 100, "number of generations to run")
	fs.Int64Var(&o.seed, "seed", 1, "random seed")
	fs.Float64Var(&o.density, "density", 0.3, "initial fraction of live cells")
	fs.IntVar(&o.symmetry, "symmetry", 1, "mirror the random start: 1 (none), 2, 4 or 8 (8 needs a square grid)")
	fs.StringVar(&o.out, "o", "out.png", "output file")
	fs.StringVar(&o.rule, "rule", "B3/S23", "rule in B/S (B3/S23) or Generations S/B/C (345/2/4) notation, or a preset name (see -list-rules)")
	fs.BoolVar(&o.listRules, "list-rules", false, "list preset rules and exit")
	fs.BoolVar(&o.wrap, "wrap", true, "toroidal edges; -wrap=false makes cells beyond the edge dead")
	fs.IntVar(&o.delay, "delay", 5, "GIF frame delay in 1/100 s")
	fs.BoolVar(&o.cyclic, "cyclic", false, "run a cyclic cellular automaton instead of -rule")
	fs.IntVar(&o.states, "states", 14, "cyclic: number of states (2-256)")
	fs.IntVar(&o.threshold, "threshold", 3, "cyclic: neighbours in the next state needed to advance")
	fs.StringVar(&o.neighborhood, "neighborhood", "moore", "cyclic: moore or vonneumann")
	fs.IntVar(&o.radius, "radius", 1, "cyclic: neighbourhood radius")
	fs.StringVar(&o.palette, "palette", "age", "colour gradient: "+strings.Join(paletteNames(), ", "))
	fs.StringVar(&o.colors, "colors", "", "custom gradient of 2 to 8 colours, RRGGBB,RRGGBB,… (replaces -palette)")
	fs.StringVar(&o.palFrom, "palette-from", "", "custom gradient start colour, RRGGBB (with -palette-to)")
	fs.StringVar(&o.palTo, "palette-to", "", "custom gradient end colour, RRGGBB (with -palette-from)")
	fs.StringVar(&o.serve, "serve", "", "serve the web page on this address (e.g. :8080) instead of writing a file")
	return fs
}

func main() {
	var o options
	fs := newFlagSet(&o)
	if err := fs.Parse(os.Args[1:]); err != nil {
		os.Exit(2) // the flag package already printed the error and usage
	}

	switch {
	case o.listRules:
		for _, p := range Presets {
			fmt.Printf("%-18s %s\n", p.Name, p.Rule)
		}
	case o.serve != "":
		fail(1, serve(o.serve))
	default:
		o.gif = strings.EqualFold(filepath.Ext(o.out), ".gif")
		var buf bytes.Buffer
		if err := o.generate(&buf); err != nil {
			fail(2, err)
		}
		if err := os.WriteFile(o.out, buf.Bytes(), 0o644); err != nil {
			fail(1, err)
		}
	}
}

// generate runs the automaton and writes a GIF of every generation, or a PNG
// of the last one, to w.
func (o *options) generate(w io.Writer) error {
	g, step, pal, err := o.setup()
	if err != nil {
		return err
	}
	if o.gif {
		// ponytail: all frames held in memory (W*H*scale² bytes each), stream if it gets too big
		frames := []*image.Paletted{Render(g, o.scale, pal)}
		for i := 1; i < o.gens; i++ {
			g = step(g)
			frames = append(frames, Render(g, o.scale, pal))
		}
		return WriteGIF(w, frames, o.delay)
	}
	for i := 1; i < o.gens; i++ {
		g = step(g)
	}
	return WritePNG(w, Render(g, o.scale, pal))
}

// setup checks the options and returns the starting grid, the function that
// computes the next generation, and the palette.
func (o *options) setup() (g *Grid, step func(*Grid) *Grid, pal color.Palette, err error) {
	if o.w < 1 || o.h < 1 || o.scale < 1 || o.gens < 1 {
		return nil, nil, nil, errors.New("want w, h, scale and gens >= 1")
	}
	switch {
	case o.symmetry != 1 && o.symmetry != 2 && o.symmetry != 4 && o.symmetry != 8:
		return nil, nil, nil, fmt.Errorf("symmetry %d: want 1, 2, 4 or 8", o.symmetry)
	case o.symmetry == 8 && o.w != o.h:
		return nil, nil, nil, errors.New("symmetry 8 needs a square grid (w = h)")
	}
	rule := o.rule
	for _, p := range Presets {
		if strings.EqualFold(rule, p.Name) {
			rule = p.Rule
		}
	}

	gr, ok := gradients[o.palette]
	if !ok {
		return nil, nil, nil, fmt.Errorf("unknown palette %q (want one of %s)", o.palette, strings.Join(paletteNames(), ", "))
	}
	if o.palFrom != "" || o.palTo != "" {
		from, err1 := parseHex(o.palFrom)
		to, err2 := parseHex(o.palTo)
		if err := errors.Join(err1, err2); err != nil {
			return nil, nil, nil, fmt.Errorf("-palette-from and -palette-to: %w", err)
		}
		gr = gradient{black, []color.RGBA{from, to}}
	}
	if o.colors != "" {
		var stops []color.RGBA
		for _, hex := range strings.Split(o.colors, ",") {
			c, err := parseHex(hex)
			if err != nil {
				return nil, nil, nil, fmt.Errorf("-colors: %w", err)
			}
			stops = append(stops, c)
		}
		if len(stops) < 2 || len(stops) > 8 {
			return nil, nil, nil, fmt.Errorf("-colors: want 2 to 8 colours, got %d", len(stops))
		}
		gr = gradient{black, stops}
	}

	g = NewGrid(o.w, o.h, o.wrap)
	if o.cyclic {
		c := Cyclic{States: o.states, Threshold: o.threshold, Radius: o.radius}
		switch o.neighborhood {
		case "moore":
		case "vonneumann":
			c.VonNeumann = true
		default:
			return nil, nil, nil, fmt.Errorf("unknown neighborhood %q (want moore or vonneumann)", o.neighborhood)
		}
		if c.States < 2 || c.States > 256 || c.Threshold < 1 || c.Radius < 1 || c.Radius >= min(o.w, o.h) {
			return nil, nil, nil, errors.New("cyclic: want 2 <= states <= 256, threshold >= 1, 1 <= radius < grid size")
		}
		g.RandomizeStates(o.seed, c.States)
		step = func(g *Grid) *Grid { return g.StepCyclic(c) }
		// Every state is a live colour, spread evenly.
		pal = gr.palette(c.States, false, func(s int) float64 { return float64(s) / float64(c.States-1) })
	} else {
		r, err := ParseRule(rule)
		if err != nil {
			return nil, nil, nil, err
		}
		g.Randomize(o.seed, o.density)
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

	g.Mirror(o.symmetry)
	return g, step, pal, nil
}

func fail(code int, err error) {
	fmt.Fprintln(os.Stderr, "cellgen:", err)
	os.Exit(code)
}
