package main

import (
	"fmt"
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

// Life is the classic Game of Life, B3/S23.
var Life = Rule{
	Birth:   [9]bool{3: true},
	Survive: [9]bool{2: true, 3: true},
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

type Grid struct {
	W, H int
	Wrap bool // true: toroidal edges, false: cells beyond the edge are dead
	// Cells is row-major, len W*H. For B/S rules a cell holds its age:
	// 0 = dead, n = alive for n generations (saturates at 255). For
	// Generations rules it holds the state.
	Cells []uint8
}

func NewGrid(w, h int, wrap bool) *Grid {
	return &Grid{W: w, H: h, Wrap: wrap, Cells: make([]uint8, w*h)}
}

// Set makes (x, y) a newborn cell, or kills it.
func (g *Grid) Set(x, y int, alive bool) {
	g.Cells[y*g.W+x] = 0
	if alive {
		g.Cells[y*g.W+x] = 1
	}
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

// neighbours counts the cells around (x, y) for which live is true.
func (g *Grid) neighbours(x, y int, live func(uint8) bool) int {
	n := 0
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			if dx == 0 && dy == 0 {
				continue
			}
			nx, ny := x+dx, y+dy
			if g.Wrap {
				nx, ny = (nx+g.W)%g.W, (ny+g.H)%g.H
			} else if nx < 0 || ny < 0 || nx >= g.W || ny >= g.H {
				continue
			}
			if live(g.Cells[ny*g.W+nx]) {
				n++
			}
		}
	}
	return n
}

// Step returns the next generation under rule r.
func (g *Grid) Step(r Rule) *Grid {
	live := func(v uint8) bool { return v > 0 }
	if r.States > 0 {
		live = func(v uint8) bool { return v == 1 } // dying cells don't count
	}
	next := NewGrid(g.W, g.H, g.Wrap)
	for y := 0; y < g.H; y++ {
		for x := 0; x < g.W; x++ {
			n, i := g.neighbours(x, y, live), y*g.W+x
			switch v := g.Cells[i]; {
			case v == 0:
				if r.Birth[n] {
					next.Cells[i] = 1
				}
			case r.States == 0: // B/S: survivors age
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
	return next
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
