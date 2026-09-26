package main

import (
	"fmt"
	"math"
	"math/rand"
	"strconv"
	"strings"
)

// Rule holds, for each neighbour count 0..8, whether a dead cell is born
// and whether a live cell survives.
//
// States is 0 for a B/S rule, where live cells track their age. Otherwise it
// is a Generations rule with that many states: 0 is dead, 1 is alive, and
// 2..States-1 are dying cells that advance one state per generation.
type Rule struct {
	Birth   [9]bool
	Survive [9]bool
	States  int
}

// Presets are well-known rules, usable by name with -rule.
var Presets = []struct{ Name, Rule string }{
	{"life", "B3/S23"},
	{"highlife", "B36/S23"},
	{"daynight", "B3678/S34678"},
	{"diamoeba", "B5678/S45678"},
	{"seeds", "B2/S"},
	{"maze", "B3/S12345"},
	{"coral", "B3/S45678"},
	{"anneal", "B4678/S35678"},
	{"replicator", "B1357/S1357"},
	{"morley", "B368/S245"},
	{"2x2", "B36/S125"},
	{"lifewithoutdeath", "B3/S012345678"},
	{"starwars", "345/2/4"},
	{"brian", "/2/3"},
	{"frogs", "345/2/6"},
	{"belzhab", "2/23/8"},
	// Larger than Life: big neighbourhoods, soft organic shapes.
	{"bosco", "R5,C0,M1,S34..58,B34..45,NM"},
	{"majority", "R4,C0,M1,S41..81,B41..81,NM"},
	{"majorly", "R7,C0,M1,S113..225,B113..225,NM"},
	{"waffle", "R7,C0,M1,S100..200,B75..170,NM"},
	{"globe", "R8,C0,M0,S163..223,B74..252,NM"},
}

// Cyclic is a cyclic cellular automaton: a cell in state k moves to state
// k+1 (mod States) when at least Threshold neighbours within Radius are
// already in state k+1.
type Cyclic struct {
	States, Threshold, Radius int
	VonNeumann                bool // false: Moore (square) neighbourhood
}

type Grid struct {
	W, H int
	Wrap bool // true: toroidal edges, false: cells beyond the edge are dead
	// Cells is row-major, len W*H. For B/S rules a cell holds its age:
	// 0 = dead, n = alive for n generations (saturates at 255). For
	// Generations and Cyclic rules it holds the state.
	Cells []uint8
}

func NewGrid(w, h int, wrap bool) *Grid {
	return &Grid{W: w, H: h, Wrap: wrap, Cells: make([]uint8, w*h)}
}

// Randomize fills the grid so that roughly density of the cells are alive.
// The same seed always gives the same grid.
func (g *Grid) Randomize(seed int64, density float64) {
	r := rand.New(rand.NewSource(seed))
	for i := range g.Cells {
		g.Cells[i] = 0
		if r.Float64() < density {
			g.Cells[i] = 1
		}
	}
}

// NoiseField returns fractal Perlin noise in [0, 1] for every cell, in blobs
// about scale cells wide; the same seed gives the same field. It tiles: the
// lattice fits a whole number of times in the grid and wraps around, so the
// right edge continues the left one and the bottom the top, as wrapped
// automata and seamless mosaics need.
func NoiseField(seed int64, w, h int, scale float64) []float64 {
	var perm [512]int
	for i, v := range rand.New(rand.NewSource(seed)).Perm(256) {
		perm[i], perm[i+256] = v, v
	}
	grad := func(hash int, dx, dy float64) float64 {
		switch hash & 3 {
		case 0:
			return dx + dy
		case 1:
			return -dx + dy
		case 2:
			return dx - dy
		}
		return -dx - dy
	}
	fade := func(t float64) float64 { return t * t * t * (t*(t*6-15) + 10) }
	lerp := func(t, a, b float64) float64 { return a + t*(b-a) }
	mod := func(a, m int) int { return (a%m + m) % m }
	// noise is Perlin noise (about -1..1) whose lattice repeats every px by py.
	noise := func(x, y float64, px, py int) float64 {
		xi, yi := int(math.Floor(x)), int(math.Floor(y))
		x, y = x-math.Floor(x), y-math.Floor(y)
		u, v := fade(x), fade(y)
		hash := func(ix, iy int) int { return perm[perm[mod(ix, px)&255]+mod(iy, py)&255] }
		return lerp(v,
			lerp(u, grad(hash(xi, yi), x, y), grad(hash(xi+1, yi), x-1, y)),
			lerp(u, grad(hash(xi, yi+1), x, y-1), grad(hash(xi+1, yi+1), x-1, y-1)))
	}
	// Lattice cells across the grid: about one per blob, but a whole number.
	px, py := max(1, int(math.Round(float64(w)/scale))), max(1, int(math.Round(float64(h)/scale)))
	field := make([]float64, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			n, amp := 0.0, 1.0
			for octave := 0; octave < 3; octave++ { // big blobs with finer detail
				ox, oy := px<<octave, py<<octave
				n += amp * noise(float64(x*ox)/float64(w), float64(y*oy)/float64(h), ox, oy)
				amp /= 2
			}
			field[y*w+x] = min(1, max(0, (n/1.75+1)/2))
		}
	}
	return field
}

// RandomizeNoise is Randomize in Perlin islands: cells are only born in the
// upper half of the noise, twice as dense, so the overall density stays about
// the same and the islands are clear enough to survive the first generations.
func (g *Grid) RandomizeNoise(seed int64, density, scale float64) {
	field := NoiseField(seed, g.W, g.H, scale)
	r := rand.New(rand.NewSource(seed))
	for i := range g.Cells {
		g.Cells[i] = 0
		if field[i] > 0.5 && r.Float64() < 2*density {
			g.Cells[i] = 1
		}
	}
}

// RandomizeStatesNoise is RandomizeStates in Perlin blobs: states follow the
// noise, wrapping around three times, so waves start from smooth gradients.
func (g *Grid) RandomizeStatesNoise(seed int64, n int, scale float64) {
	for i, f := range NoiseField(seed, g.W, g.H, scale) {
		g.Cells[i] = uint8(int(f*float64(n)*3) % n)
	}
}

// RandomizeStates gives every cell a uniformly random state in 0..n-1.
func (g *Grid) RandomizeStates(seed int64, n int) {
	r := rand.New(rand.NewSource(seed))
	for i := range g.Cells {
		g.Cells[i] = uint8(r.Intn(n))
	}
}

// symmetries maps each -symmetry mode to the transforms of a w×h grid whose
// cells share one value: mirrors (2: left/right, 4: also top/bottom, 8: also
// the diagonals) and rotations (r2: half turn, r4: quarter turns). 8 and r4
// need a square grid. Rules treat every direction alike, so a symmetric start
// stays symmetric.
var symmetries = map[string][]func(x, y, w, h int) (int, int){
	"1": nil,
	"2": {flipX},
	"4": {flipX, flipY, turn2},
	"8": {flipX, flipY, turn2,
		func(x, y, w, h int) (int, int) { return y, x },
		func(x, y, w, h int) (int, int) { return w - 1 - y, x },
		func(x, y, w, h int) (int, int) { return y, h - 1 - x },
		func(x, y, w, h int) (int, int) { return w - 1 - y, h - 1 - x }},
	"r2": {turn2},
	"r4": {turn2,
		func(x, y, w, h int) (int, int) { return w - 1 - y, x },
		func(x, y, w, h int) (int, int) { return y, h - 1 - x }},
}

func flipX(x, y, w, h int) (int, int) { return w - 1 - x, y }
func flipY(x, y, w, h int) (int, int) { return x, h - 1 - y }
func turn2(x, y, w, h int) (int, int) { return w - 1 - x, h - 1 - y }

// Symmetrize makes the grid symmetric under a -symmetry mode: every cell takes
// the value of the first cell of its group (the smallest x, then y), which
// keeps its own value, so the copy can be done in place.
func (g *Grid) Symmetrize(mode string) {
	for x := 0; x < g.W; x++ {
		for y := 0; y < g.H; y++ {
			sx, sy := x, y
			for _, t := range symmetries[mode] {
				tx, ty := t(x, y, g.W, g.H)
				if tx < sx || tx == sx && ty < sy {
					sx, sy = tx, ty
				}
			}
			g.Cells[y*g.W+x] = g.Cells[sy*g.W+sx]
		}
	}
}

// shapes are the -shape start areas: outside it, cells start dead (state 0).
var shapes = []string{"all", "disc", "ring", "cross", "frame", "stripes"}

// KeepShape kills the cells outside shape, measured from the grid's centre
// in units of half its smaller side.
func (g *Grid) KeepShape(shape string) {
	r := float64(min(g.W, g.H)) / 2
	cx, cy := float64(g.W-1)/2, float64(g.H-1)/2
	border := max(1, min(g.W, g.H)/10)
	for y := 0; y < g.H; y++ {
		for x := 0; x < g.W; x++ {
			dx, dy := math.Abs(float64(x)-cx)/r, math.Abs(float64(y)-cy)/r
			d := math.Hypot(dx, dy)
			var keep bool
			switch shape {
			case "disc":
				keep = d <= 0.4
			case "ring":
				keep = d >= 0.55 && d <= 0.75
			case "cross":
				keep = dx <= 0.12 || dy <= 0.12
			case "frame":
				keep = x < border || y < border || x >= g.W-border || y >= g.H-border
			case "stripes":
				keep = x*8/g.W%2 == 0
			default: // all
				keep = true
			}
			if !keep {
				g.Cells[y*g.W+x] = 0
			}
		}
	}
}

// Step returns the next generation under rule r.
func (g *Grid) Step(r Rule) *Grid {
	next := NewGrid(g.W, g.H, g.Wrap)
	g.StepInto(next, r)
	return next
}

// StepInto writes the next generation under rule r into next, a grid of the
// same size (so callers can swap two grids instead of allocating one per
// generation). Neighbours are counted directly: the rows above and below and
// the columns left and right are found once, not per neighbour.
func (g *Grid) StepInto(next *Grid, r Rule) {
	W, H := g.W, g.H
	generations := r.States > 0
	for y := 0; y < H; y++ {
		up, down := (y-1+H)%H, (y+1)%H
		if !g.Wrap && y == 0 {
			up = -1 // beyond the edge: dead
		}
		if !g.Wrap && y == H-1 {
			down = -1
		}
		for x := 0; x < W; x++ {
			left, right := (x-1+W)%W, (x+1)%W
			hasLeft, hasRight := g.Wrap || x > 0, g.Wrap || x < W-1
			n := 0
			for _, row := range [3]int{up, y, down} {
				if row < 0 {
					continue
				}
				cells := g.Cells[row*W : row*W+W]
				if hasLeft && alive(cells[left], generations) {
					n++
				}
				if row != y && alive(cells[x], generations) {
					n++
				}
				if hasRight && alive(cells[right], generations) {
					n++
				}
			}

			i := y*W + x
			next.Cells[i] = 0
			switch v := g.Cells[i]; {
			case v == 0:
				if r.Birth[n] {
					next.Cells[i] = 1
				}
			case !generations: // B/S: survivors age
				if r.Survive[n] {
					next.Cells[i] = max(v, v+1) // +1, saturating at 255
				}
			case v == 1 && r.Survive[n]:
				next.Cells[i] = 1
			default: // Generations: start or keep dying, back to 0 after the last state
				next.Cells[i] = uint8((int(v) + 1) % r.States)
			}
		}
	}
}

// alive reports whether a cell counts as a live neighbour: any age for B/S
// rules, only state 1 for Generations (dying cells don't count).
func alive(v uint8, generations bool) bool {
	return v == 1 || v > 0 && !generations
}

// StepCyclic returns the next generation under cyclic rule c.
func (g *Grid) StepCyclic(c Cyclic) *Grid {
	next := NewGrid(g.W, g.H, g.Wrap)
	g.StepCyclicInto(next, c)
	return next
}

// StepCyclicInto writes the next generation under cyclic rule c into next.
// The neighbourhood's offsets are listed once; cells at least Radius away
// from the edges use them as plain index steps, the others wrap or skip.
func (g *Grid) StepCyclicInto(next *Grid, c Cyclic) {
	W, H, rad := g.W, g.H, c.Radius
	type offset struct{ dx, dy, di int }
	var offsets []offset
	for dy := -rad; dy <= rad; dy++ {
		for dx := -rad; dx <= rad; dx++ {
			if dx == 0 && dy == 0 || c.VonNeumann && max(dx, -dx)+max(dy, -dy) > rad {
				continue
			}
			offsets = append(offsets, offset{dx, dy, dy*W + dx})
		}
	}
	for y := 0; y < H; y++ {
		for x := 0; x < W; x++ {
			i := y*W + x
			k := g.Cells[i]
			succ := uint8((int(k) + 1) % c.States)
			n := 0
			if x >= rad && x < W-rad && y >= rad && y < H-rad {
				for _, o := range offsets {
					if g.Cells[i+o.di] == succ {
						n++
					}
				}
			} else {
				for _, o := range offsets {
					nx, ny := x+o.dx, y+o.dy
					if g.Wrap {
						nx, ny = (nx+W)%W, (ny+H)%H
					} else if nx < 0 || ny < 0 || nx >= W || ny >= H {
						continue
					}
					if g.Cells[ny*W+nx] == succ {
						n++
					}
				}
			}
			next.Cells[i] = k
			if n >= c.Threshold {
				next.Cells[i] = succ
			}
		}
	}
}

// LtL is a Larger than Life rule: B/S over a big square neighbourhood, with
// ranges of neighbour counts. Written as in Golly: R5,C0,M1,S34..58,B34..45,NM
// (radius, states, middle cell counted or not, survival and birth ranges,
// Moore neighbourhood). With States 0 or 2 it is two-state, and live cells
// age like B/S rules; with more, they die one state at a time like Generations.
type LtL struct {
	Radius, States         int
	Middle                 bool
	SMin, SMax, BMin, BMax int
}

// IsLtL reports whether a rule is written in Larger than Life notation.
func IsLtL(rule string) bool { return strings.HasPrefix(strings.ToUpper(rule), "R") }

// ParseLtL reads Larger than Life notation, e.g. R5,C0,M1,S34..58,B34..45,NM.
func ParseLtL(s string) (LtL, error) {
	var r LtL
	var seen string
	bad := func(why string) (LtL, error) { return LtL{}, fmt.Errorf("rule %q: %s", s, why) }
	for _, part := range strings.Split(strings.ToUpper(s), ",") {
		if part == "" {
			return bad("empty part")
		}
		key, val := part[0], part[1:]
		seen += string(key)
		var err error
		switch key {
		case 'R':
			r.Radius, err = strconv.Atoi(val)
		case 'C':
			r.States, err = strconv.Atoi(val)
		case 'M':
			r.Middle = val == "1"
			if val != "0" && val != "1" {
				return bad("M must be 0 or 1")
			}
		case 'S', 'B':
			lo, hi, ok := strings.Cut(val, "..")
			if !ok {
				return bad("want S<min>..<max> and B<min>..<max>")
			}
			var a, b int
			if a, err = strconv.Atoi(lo); err == nil {
				b, err = strconv.Atoi(hi)
			}
			if key == 'S' {
				r.SMin, r.SMax = a, b
			} else {
				r.BMin, r.BMax = a, b
			}
		case 'N':
			if val != "M" {
				return bad("only the Moore neighbourhood (NM) is supported")
			}
		default:
			return bad(fmt.Sprintf("unknown part %q", part))
		}
		if err != nil {
			return bad(fmt.Sprintf("part %q: %v", part, err))
		}
	}
	switch {
	case !strings.Contains(seen, "R") || !strings.Contains(seen, "S") || !strings.Contains(seen, "B"):
		return bad("want at least R, S and B")
	case r.Radius < 1 || r.Radius > 20:
		return bad("radius must be 1 to 20")
	case r.States == 1 || r.States < 0 || r.States > 256:
		return bad("C must be 0, or 2 to 256")
	case r.BMin < 1: // birth on 0 neighbours would flash the whole empty grid
		return bad("birth must need at least 1 neighbour")
	}
	return r, nil
}

// StepLtLInto writes the next generation under Larger than Life rule r into
// next. Neighbour counts come from a summed-area table of the live cells over
// the grid padded by the radius (wrapped, or dead beyond the edges), so every
// count costs the same whatever the radius.
// ponytail: the table is allocated every generation; keep one if GC shows up
func (g *Grid) StepLtLInto(next *Grid, r LtL) {
	W, H, R := g.W, g.H, r.Radius
	generations := r.States > 2
	PW, PH := W+2*R, H+2*R
	sum := make([]int32, (PW+1)*(PH+1)) // sum[(y)*(PW+1)+x]: live cells above and left of (x, y)
	for py := 0; py < PH; py++ {
		y := py - R
		for px := 0; px < PW; px++ {
			x := px - R
			var v int32
			if g.Wrap {
				x, y := (x%W+W)%W, (y%H+H)%H
				if alive(g.Cells[y*W+x], generations) {
					v = 1
				}
			} else if x >= 0 && y >= 0 && x < W && y < H && alive(g.Cells[y*W+x], generations) {
				v = 1
			}
			i := (py+1)*(PW+1) + px + 1
			sum[i] = v + sum[i-1] + sum[i-PW-1] - sum[i-PW-2]
		}
	}
	side := 2*R + 1
	for y := 0; y < H; y++ {
		for x := 0; x < W; x++ {
			// the window is padded (x..x+2R, y..y+2R), centred on the cell
			n := int(sum[(y+side)*(PW+1)+x+side] - sum[y*(PW+1)+x+side] - sum[(y+side)*(PW+1)+x] + sum[y*(PW+1)+x])
			i := y*W + x
			v := g.Cells[i]
			if !r.Middle && alive(v, generations) {
				n--
			}
			next.Cells[i] = 0
			switch {
			case v == 0:
				if n >= r.BMin && n <= r.BMax {
					next.Cells[i] = 1
				}
			case !generations: // two states: survivors age
				if n >= r.SMin && n <= r.SMax {
					next.Cells[i] = max(v, v+1)
				}
			case v == 1 && n >= r.SMin && n <= r.SMax:
				next.Cells[i] = 1
			default: // dying one state at a time
				next.Cells[i] = uint8((int(v) + 1) % r.States)
			}
		}
	}
}

// ParseRule reads Golly-style B/S notation, e.g. "B3/S23" or "s23/b3", or
// Generations S/B/C notation, e.g. "345/2/4".
func ParseRule(s string) (Rule, error) {
	var r Rule
	parts := strings.Split(strings.ToUpper(s), "/")
	if len(parts) == 3 {
		c, err := strconv.Atoi(parts[2])
		if err != nil || c < 2 || c > 256 {
			return r, fmt.Errorf("rule %q: state count %q not in 2-256", s, parts[2])
		}
		r.States = c
		if err := parseCounts(s, parts[0], &r.Survive); err != nil {
			return r, err
		}
		return r, parseCounts(s, parts[1], &r.Birth)
	}
	if len(parts) != 2 {
		return r, fmt.Errorf("rule %q: want B<digits>/S<digits> or S/B/C", s)
	}
	seen := map[byte]bool{}
	for _, p := range parts {
		if p == "" || (p[0] != 'B' && p[0] != 'S') || seen[p[0]] {
			return r, fmt.Errorf("rule %q: want B<digits>/S<digits> or S/B/C", s)
		}
		seen[p[0]] = true
		counts := &r.Birth
		if p[0] == 'S' {
			counts = &r.Survive
		}
		if err := parseCounts(s, p[1:], counts); err != nil {
			return r, err
		}
	}
	return r, nil
}

// parseCounts sets counts[n] for each digit n in digits.
func parseCounts(rule, digits string, counts *[9]bool) error {
	for _, c := range digits {
		if c < '0' || c > '8' {
			return fmt.Errorf("rule %q: neighbour count %q not in 0-8", rule, c)
		}
		counts[c-'0'] = true
	}
	return nil
}
