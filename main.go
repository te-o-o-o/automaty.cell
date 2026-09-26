package main

import (
	"archive/zip"
	"bytes"
	"errors"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/gif"
	"image/png"
	"io"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

// options holds every setting of a run. The CLI fills it from os.Args and the
// web server from query parameters, through the same flags.
type options struct {
	w, h, scale, gens         int
	seed                      int64
	symmetry, shape, at       string
	pingpong                  bool
	density, noise            float64
	rule                      string
	wrap                      bool
	delay                     int
	cyclic                    bool
	states, threshold, radius int
	neighborhood              string
	palette, colors           string
	gif, zip                  bool   // GIF or ZIP of PNG frames instead of one PNG
	mask                      string // CLI: image file whose light (or opaque) areas stay alive
	maskData                  []byte // web: the same image, uploaded
	out, serve                string
	listRules                 bool
}

func newFlagSet(o *options) *flag.FlagSet {
	fs := flag.NewFlagSet("automaty.cell", flag.ContinueOnError)
	fs.IntVar(&o.w, "w", 380, "grid width in cells")
	fs.IntVar(&o.h, "h", 380, "grid height in cells")
	fs.IntVar(&o.scale, "scale", 2, "pixels per cell")
	fs.IntVar(&o.gens, "gens", 100, "number of generations to run")
	fs.Int64Var(&o.seed, "seed", 1, "random seed")
	fs.Float64Var(&o.density, "density", 0.3, "initial fraction of live cells")
	fs.Float64Var(&o.noise, "noise", 0, "start in Perlin noise blobs about this many cells wide (0: plain random)")
	fs.StringVar(&o.symmetry, "symmetry", "1", "symmetric start: 1 (none), mirrors 2, 4 or 8, rotations r2 or r4 (8 and r4 need a square grid)")
	fs.StringVar(&o.shape, "shape", "all", "start area, dead outside: "+strings.Join(shapes, ", "))
	fs.StringVar(&o.at, "at", "", "changes during the run, space-separated GEN:FLAG=VALUE (e.g. '80:rule=bosco 150:palette=candy'), FLAG one of "+strings.Join(liveFlags, ", "))
	fs.StringVar(&o.out, "o", "out.png", "output file: .png (last generation), .gif (animation) or .zip (one PNG per generation)")
	fs.StringVar(&o.rule, "rule", "B3/S23", "rule in B/S (B3/S23) or Generations S/B/C (345/2/4) notation, or a preset name (see -list-rules)")
	fs.BoolVar(&o.listRules, "list-rules", false, "list preset rules and exit")
	fs.BoolVar(&o.wrap, "wrap", true, "toroidal edges; -wrap=false makes cells beyond the edge dead")
	fs.IntVar(&o.delay, "delay", 5, "GIF frame delay in 1/100 s")
	fs.BoolVar(&o.pingpong, "pingpong", false, "GIF or ZIP: play forward then backward, a loop without a jump")
	fs.StringVar(&o.mask, "mask", "", "PNG or JPEG image: cells only live in its light areas (or opaque ones, if it has transparency)")
	fs.BoolVar(&o.cyclic, "cyclic", false, "run a cyclic cellular automaton instead of -rule")
	fs.IntVar(&o.states, "states", 14, "cyclic: number of states (2-256)")
	fs.IntVar(&o.threshold, "threshold", 1, "cyclic: neighbours in the next state needed to advance")
	fs.StringVar(&o.neighborhood, "neighborhood", "vonneumann", "cyclic: moore or vonneumann")
	fs.IntVar(&o.radius, "radius", 1, "cyclic: neighbourhood radius")
	fs.StringVar(&o.palette, "palette", "age", "colour gradient: "+strings.Join(paletteNames(), ", "))
	fs.StringVar(&o.colors, "colors", "", "custom gradient of 2 to 8 colours, RRGGBB,RRGGBB,… (replaces -palette)")
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
		o.zip = strings.EqualFold(filepath.Ext(o.out), ".zip")
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
		frames := []*image.Paletted{Render(g, o.scale, pal())}
		for i := 1; i < o.gens; i++ {
			g = step(g)
			frames = append(frames, Render(g, o.scale, pal()))
		}
		if o.pingpong {
			frames = pingpong(frames)
		}
		anim := &gif.GIF{Image: frames, Delay: make([]int, len(frames))}
		for i := range anim.Delay {
			anim.Delay[i] = o.delay // 1/100 s per frame
		}
		return gif.EncodeAll(w, anim)
	}
	if o.zip {
		// One PNG per generation, for video mapping tools that read image
		// sequences. Only the encoded PNGs are kept, to replay them backward.
		var frames [][]byte
		for i := 0; i < o.gens; i++ {
			if i > 0 {
				g = step(g)
			}
			var buf bytes.Buffer
			if err := png.Encode(&buf, Render(g, o.scale, pal())); err != nil {
				return err
			}
			frames = append(frames, buf.Bytes())
		}
		if o.pingpong {
			frames = pingpong(frames)
		}
		z := zip.NewWriter(w)
		for i, f := range frames {
			// Stored, not deflated: PNGs are already compressed.
			fw, err := z.CreateHeader(&zip.FileHeader{Name: fmt.Sprintf("frame%05d.png", i+1), Method: zip.Store})
			if err != nil {
				return err
			}
			if _, err := fw.Write(f); err != nil {
				return err
			}
		}
		return z.Close()
	}
	for i := 1; i < o.gens; i++ {
		g = step(g)
	}
	return png.Encode(w, Render(g, o.scale, pal()))
}

// resolvePreset returns the rule a preset name stands for, or rule itself.
func resolvePreset(rule string) string {
	for _, p := range Presets {
		if strings.EqualFold(rule, p.Name) {
			return p.Rule
		}
	}
	return rule
}

// pingpong appends the frames again, backward, without repeating the ends,
// so the sequence loops without a jump.
func pingpong[T any](frames []T) []T {
	for i := len(frames) - 2; i > 0; i-- {
		frames = append(frames, frames[i])
	}
	return frames
}

// setup checks the options and returns the starting grid, the function that
// computes the next generation, and the palette of the last generation
// computed (-at may change it on the way).
func (o *options) setup() (g *Grid, step func(*Grid) *Grid, pal func() color.Palette, err error) {
	if o.w < 1 || o.h < 1 || o.scale < 1 || o.gens < 1 {
		return nil, nil, nil, errors.New("want w, h, scale and gens >= 1")
	}
	switch {
	case symmetries[o.symmetry] == nil && o.symmetry != "1":
		return nil, nil, nil, fmt.Errorf("symmetry %q: want 1, 2, 4, 8, r2 or r4", o.symmetry)
	case (o.symmetry == "8" || o.symmetry == "r4") && o.w != o.h:
		return nil, nil, nil, fmt.Errorf("symmetry %s needs a square grid (w = h)", o.symmetry)
	case !slices.Contains(shapes, o.shape):
		return nil, nil, nil, fmt.Errorf("shape %q: want one of %s", o.shape, strings.Join(shapes, ", "))
	}
	if o.noise < 0 {
		return nil, nil, nil, errors.New("want noise >= 0")
	}
	cur, err := o.automaton()
	if err != nil {
		return nil, nil, nil, err
	}
	keys, err := o.keyframes()
	if err != nil {
		return nil, nil, nil, err
	}

	g = NewGrid(o.w, o.h, o.wrap)
	switch {
	case o.cyclic && o.noise > 0:
		g.RandomizeStatesNoise(o.seed, o.states, o.noise)
	case o.cyclic:
		g.RandomizeStates(o.seed, o.states)
	case o.noise > 0:
		g.RandomizeNoise(o.seed, o.density, o.noise)
	default:
		g.Randomize(o.seed, o.density)
	}
	g.KeepShape(o.shape)
	g.Symmetrize(o.symmetry)

	step = cur.step
	if len(keys) > 0 {
		gen := 0
		step = func(g *Grid) *Grid {
			gen++
			for len(keys) > 0 && keys[0].gen == gen {
				convert(g.Cells, cur, keys[0].automaton)
				cur, keys = keys[0].automaton, keys[1:]
				g.Wrap = cur.wrap
			}
			return cur.step(g)
		}
	}
	pal = func() color.Palette { return cur.pal }

	if o.mask != "" || o.maskData != nil {
		data := o.maskData
		if data == nil {
			if data, err = os.ReadFile(o.mask); err != nil {
				return nil, nil, nil, err
			}
		}
		inside, err := loadMask(data, o.w, o.h)
		if err != nil {
			return nil, nil, nil, err
		}
		// Cells outside the mask are dead at the start and stay dead.
		keep := func(g *Grid) *Grid {
			for i, in := range inside {
				if !in {
					g.Cells[i] = 0
				}
			}
			return g
		}
		keep(g)
		inner := step
		step = func(g *Grid) *Grid { return keep(inner(g)) }
	}
	return g, step, pal, nil
}

// automaton is what the options make of the rule: the step function, the
// palette, the edges, and what cells hold (states: 0 when they age, as in B/S).
type automaton struct {
	step   func(*Grid) *Grid
	pal    color.Palette
	wrap   bool
	cyclic bool
	states int
}

func (o *options) automaton() (a automaton, err error) {
	gr, ok := gradients[o.palette]
	if !ok {
		return a, fmt.Errorf("unknown palette %q (want one of %s)", o.palette, strings.Join(paletteNames(), ", "))
	}
	if o.colors != "" {
		var stops []color.RGBA
		for _, hex := range strings.Split(o.colors, ",") {
			c, err := parseHex(hex)
			if err != nil {
				return a, fmt.Errorf("-colors: %w", err)
			}
			stops = append(stops, c)
		}
		if len(stops) < 2 || len(stops) > 8 {
			return a, fmt.Errorf("-colors: want 2 to 8 colours, got %d", len(stops))
		}
		gr = gradient{black, stops}
	}

	a.wrap = o.wrap
	switch {
	case o.cyclic:
		c := Cyclic{States: o.states, Threshold: o.threshold, Radius: o.radius}
		switch o.neighborhood {
		case "moore":
		case "vonneumann":
			c.VonNeumann = true
		default:
			return a, fmt.Errorf("unknown neighborhood %q (want moore or vonneumann)", o.neighborhood)
		}
		if c.States < 2 || c.States > 256 || c.Threshold < 1 || c.Radius < 1 || c.Radius >= min(o.w, o.h) {
			return a, errors.New("cyclic: want 2 <= states <= 256, threshold >= 1, 1 <= radius < grid size")
		}
		a.cyclic, a.states = true, c.States
		a.step = swapping(func(g, next *Grid) { g.StepCyclicInto(next, c) })
		// Every state is a live colour, spread evenly; a one-colour gradient
		// (bw) fades from its background, or every state would look the same.
		if len(gr.stops) == 1 {
			gr.stops = []color.RGBA{gr.bg, gr.stops[0]}
		}
		a.pal = gr.palette(c.States, false, func(s int) float64 { return float64(s) / float64(c.States-1) })
	default: // B/S, Generations or Larger than Life
		rule := resolvePreset(o.rule)
		if IsLtL(rule) {
			r, err := ParseLtL(rule)
			if err != nil {
				return a, err
			}
			if r.Radius >= min(o.w, o.h) {
				return a, errors.New("Larger than Life: radius must be smaller than the grid")
			}
			a.step = swapping(func(g, next *Grid) { g.StepLtLInto(next, r) })
			if r.States > 2 {
				a.states = r.States
			}
		} else {
			r, err := ParseRule(rule)
			if err != nil {
				return a, err
			}
			a.step = swapping(func(g, next *Grid) { g.StepInto(next, r) })
			a.states = r.States
		}
		if states := a.states; states > 0 {
			// Generations: alive first, then dying states evenly.
			a.pal = gr.palette(states, true, func(s int) float64 {
				return float64(s-1) / float64(max(states-2, 1))
			})
		} else {
			// B/S ages 1-255, log scale: the first few generations matter most.
			a.pal = gr.palette(256, true, func(s int) float64 { return math.Log(float64(s)) / math.Log(255) })
		}
	}
	return a, nil
}

// liveFlags are the options -at may change during a run.
var liveFlags = []string{"rule", "cyclic", "states", "threshold", "radius", "neighborhood", "palette", "colors", "wrap"}

type keyframe struct {
	gen int
	automaton
}

// keyframes parses -at, space-separated GEN:FLAG=VALUE changes, into the
// automaton each generation switches to, in order. Each change builds on the
// ones before it.
func (o *options) keyframes() ([]keyframe, error) {
	type change struct {
		gen       int
		flag, val string
	}
	var changes []change
	for _, f := range strings.Fields(o.at) {
		gen, fv, ok := strings.Cut(f, ":")
		flag, val, ok2 := strings.Cut(fv, "=")
		n, err := strconv.Atoi(gen)
		if !ok || !ok2 || err != nil || n < 1 {
			return nil, fmt.Errorf("-at %q: want GEN:FLAG=VALUE with GEN >= 1", f)
		}
		if !slices.Contains(liveFlags, flag) {
			return nil, fmt.Errorf("-at %q: %s can't change during a run (want one of %s)", f, flag, strings.Join(liveFlags, ", "))
		}
		changes = append(changes, change{n, flag, val})
	}
	slices.SortStableFunc(changes, func(a, b change) int { return a.gen - b.gen })

	cur := *o
	fs := newFlagSet(&cur) // resets cur to the defaults…
	cur = *o               // …so put the options back
	var keys []keyframe
	for _, c := range changes {
		if err := fs.Set(c.flag, c.val); err != nil {
			return nil, fmt.Errorf("-at %d:%s=%s: %w", c.gen, c.flag, c.val, err)
		}
		switch c.flag { // or the option before would hide the change
		case "palette":
			cur.colors = ""
		case "rule":
			cur.cyclic = false
		}
		a, err := cur.automaton()
		if err != nil {
			return nil, fmt.Errorf("-at %d:%s=%s: %w", c.gen, c.flag, c.val, err)
		}
		keys = append(keys, keyframe{c.gen, a})
	}
	return keys, nil
}

// convert rewrites cells for the automaton to, when it holds them differently
// from from: cyclic states wrap round, dying Generations states beyond to's
// die, and everything else is alive (1) or dead. Cyclic cells count as alive
// in odd states, which keeps their bands.
func convert(cells []uint8, from, to automaton) {
	if from.cyclic == to.cyclic && from.states == to.states {
		return
	}
	for i, v := range cells {
		alive := v != 0 && (from.states == 0 || v == 1)
		if from.cyclic {
			alive = v%2 == 1
		}
		switch {
		case to.cyclic:
			cells[i] = uint8(int(v) % to.states)
		case !from.cyclic && from.states > 0 && to.states > 0 && int(v) < to.states:
			// Generations to Generations: the state stays.
		case alive:
			cells[i] = 1
		default:
			cells[i] = 0
		}
	}
}

// swapping turns an in-place stepper into a step function that alternates
// between two grids instead of allocating one per generation. The grid it
// returns is only valid until the call after next, which is how generate and
// activity use it.
func swapping(into func(g, next *Grid)) func(*Grid) *Grid {
	var spare *Grid
	return func(g *Grid) *Grid {
		if spare == nil {
			spare = NewGrid(g.W, g.H, g.Wrap)
		}
		into(g, spare)
		g, spare = spare, g
		return g
	}
}

func fail(code int, err error) {
	fmt.Fprintln(os.Stderr, "automaty.cell:", err)
	os.Exit(code)
}
