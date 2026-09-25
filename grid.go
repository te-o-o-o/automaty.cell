package main

import (
	"fmt"
	"math/rand"
	"strings"
)

// Rule holds, for each neighbour count 0..8, whether a dead cell is born
// and whether a live cell survives.
type Rule struct {
	Birth   [9]bool
	Survive [9]bool
}

// Life is the classic Game of Life, B3/S23.
var Life = Rule{
	Birth:   [9]bool{3: true},
	Survive: [9]bool{2: true, 3: true},
}

type Grid struct {
	W, H int
	// Age is row-major, len W*H: 0 = dead, n = alive for n generations
	// (saturates at 255).
	Age []uint8
}

func NewGrid(w, h int) *Grid {
	return &Grid{W: w, H: h, Age: make([]uint8, w*h)}
}

func (g *Grid) Alive(x, y int) bool { return g.Age[y*g.W+x] > 0 }

// Set makes (x, y) a newborn cell, or kills it.
func (g *Grid) Set(x, y int, alive bool) {
	g.Age[y*g.W+x] = 0
	if alive {
		g.Age[y*g.W+x] = 1
	}
}

// Randomize fills the grid so that roughly density of the cells are alive.
// The same seed always gives the same grid.
func (g *Grid) Randomize(seed int64, density float64) {
	r := rand.New(rand.NewSource(seed))
	for i := range g.Age {
		g.Age[i] = 0
		if r.Float64() < density {
			g.Age[i] = 1
		}
	}
}

// neighbours counts live cells around (x, y), wrapping around the edges.
func (g *Grid) neighbours(x, y int) int {
	n := 0
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			if dx == 0 && dy == 0 {
				continue
			}
			nx := (x + dx + g.W) % g.W
			ny := (y + dy + g.H) % g.H
			if g.Alive(nx, ny) {
				n++
			}
		}
	}
	return n
}

// Step returns the next generation under rule r.
func (g *Grid) Step(r Rule) *Grid {
	next := NewGrid(g.W, g.H)
	for y := 0; y < g.H; y++ {
		for x := 0; x < g.W; x++ {
			n, i := g.neighbours(x, y), y*g.W+x
			switch age := g.Age[i]; {
			case age > 0 && r.Survive[n]:
				next.Age[i] = max(age, age+1) // +1, saturating at 255
			case age == 0 && r.Birth[n]:
				next.Age[i] = 1
			}
		}
	}
	return next
}

// ParseRule reads Golly-style B/S notation, e.g. "B3/S23" or "s23/b3".
func ParseRule(s string) (Rule, error) {
	var r Rule
	parts := strings.Split(strings.ToUpper(s), "/")
	if len(parts) != 2 {
		return r, fmt.Errorf("rule %q: want B<digits>/S<digits>", s)
	}
	seen := map[byte]bool{}
	for _, p := range parts {
		if p == "" || (p[0] != 'B' && p[0] != 'S') || seen[p[0]] {
			return r, fmt.Errorf("rule %q: want B<digits>/S<digits>", s)
		}
		seen[p[0]] = true
		counts := &r.Birth
		if p[0] == 'S' {
			counts = &r.Survive
		}
		for _, c := range p[1:] {
			if c < '0' || c > '8' {
				return r, fmt.Errorf("rule %q: neighbour count %q not in 0-8", s, c)
			}
			counts[c-'0'] = true
		}
	}
	return r, nil
}
