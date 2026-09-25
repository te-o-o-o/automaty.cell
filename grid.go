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
