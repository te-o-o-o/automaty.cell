package main

import (
	"image/color"
	"testing"
)

// A glider moves one cell diagonally (down-right) every 4 generations.
func TestGliderMoves(t *testing.T) {
	glider := [][2]int{{1, 0}, {2, 1}, {0, 2}, {1, 2}, {2, 2}}

	g := NewGrid(10, 10, true)
	for _, p := range glider {
		g.Set(p[0], p[1], true)
	}
	for i := 0; i < 4; i++ {
		g = g.Step(Life)
	}

	want := NewGrid(10, 10, true)
	for _, p := range glider {
		want.Set(p[0]+1, p[1]+1, true)
	}
	for i := range g.Cells {
		if (g.Cells[i] > 0) != (want.Cells[i] > 0) {
			t.Fatalf("after 4 gens, cell (%d,%d) alive = %v, want %v",
				i%10, i/10, g.Cells[i] > 0, want.Cells[i] > 0)
		}
	}
}

// A block never changes, so its cells age by one each generation.
func TestBlockAges(t *testing.T) {
	g := NewGrid(6, 6, true)
	for _, p := range [][2]int{{2, 2}, {3, 2}, {2, 3}, {3, 3}} {
		g.Set(p[0], p[1], true)
	}
	for i := 0; i < 300; i++ {
		g = g.Step(Life)
		if i == 2 && g.Cells[2*6+2] != 4 {
			t.Fatalf("age after 3 gens = %d, want 4", g.Cells[2*6+2])
		}
	}
	if g.Cells[2*6+2] != 255 {
		t.Fatalf("age after 300 gens = %d, want 255 (saturated)", g.Cells[2*6+2])
	}
}

// A blinker on the top edge flips into row -1: it wraps to the bottom row
// on a torus, and is lost with dead borders.
func TestBorders(t *testing.T) {
	for _, tc := range []struct {
		wrap bool
		want int
	}{{true, 3}, {false, 2}} {
		g := NewGrid(5, 5, tc.wrap)
		for x := 1; x <= 3; x++ {
			g.Set(x, 0, true)
		}
		g = g.Step(Life)
		alive := 0
		for _, a := range g.Cells {
			if a > 0 {
				alive++
			}
		}
		if alive != tc.want {
			t.Errorf("wrap=%v: %d live cells, want %d", tc.wrap, alive, tc.want)
		}
	}
}

// In Generations rules, a live cell that doesn't survive dies one state at a
// time, and dying cells don't count as live neighbours.
func TestGenerations(t *testing.T) {
	r, _ := ParseRule("345/2/4")
	g := NewGrid(5, 5, false)
	g.Cells[2*5+1], g.Cells[2*5+3] = 1, 1 // two live cells around (2,2)
	g = g.Step(r)
	if got := g.Cells[2*5+2]; got != 1 {
		t.Fatalf("(2,2) with 2 live neighbours: state %d, want 1 (born)", got)
	}
	for _, want := range []uint8{2, 3, 0} {
		if got := g.Cells[2*5+1]; got != want {
			t.Fatalf("(1,2) state %d, want %d", got, want)
		}
		g = g.Step(r)
	}

	g = NewGrid(5, 5, false)
	g.Cells[2*5+1], g.Cells[2*5+3] = 2, 2 // two dying cells around (2,2)
	if got := g.Step(r).Cells[2*5+2]; got != 0 {
		t.Fatalf("(2,2) with 2 dying neighbours: state %d, want 0", got)
	}
}

func TestCyclic(t *testing.T) {
	c := Cyclic{States: 3, Threshold: 1, Radius: 1}
	g := NewGrid(5, 5, false)
	g.Cells[2*5+3] = 1                    // (3,2)=1 next to (2,2)=0: advances
	g.Cells[0], g.Cells[1] = 2, 0         // (0,0)=2 next to (1,0)=0: wraps 2 -> 0
	g.Cells[4*5+4], g.Cells[4*5+3] = 0, 2 // (4,4)=0 next to (3,4)=2 only: stays
	next := g.StepCyclic(c)
	for _, tc := range []struct {
		x, y int
		want uint8
	}{{2, 2, 1}, {0, 0, 0}, {4, 4, 0}} {
		if got := next.Cells[tc.y*5+tc.x]; got != tc.want {
			t.Errorf("(%d,%d) = %d, want %d", tc.x, tc.y, got, tc.want)
		}
	}

	// A diagonal neighbour counts for Moore, not for Von Neumann; and a
	// threshold of 2 isn't reached with a single neighbour.
	g = NewGrid(5, 5, false)
	g.Cells[3*5+3] = 1
	for _, tc := range []struct {
		c    Cyclic
		want uint8
	}{
		{Cyclic{States: 3, Threshold: 1, Radius: 1}, 1},
		{Cyclic{States: 3, Threshold: 1, Radius: 1, VonNeumann: true}, 0},
		{Cyclic{States: 3, Threshold: 1, Radius: 2, VonNeumann: true}, 1},
		{Cyclic{States: 3, Threshold: 2, Radius: 1}, 0},
	} {
		if got := g.StepCyclic(tc.c).Cells[2*5+2]; got != tc.want {
			t.Errorf("%+v: (2,2) = %d, want %d", tc.c, got, tc.want)
		}
	}
}

func TestParseRule(t *testing.T) {
	r, err := ParseRule("s23/b36")
	if err != nil || r != (Rule{Birth: [9]bool{3: true, 6: true}, Survive: [9]bool{2: true, 3: true}}) {
		t.Fatalf("got %v, %v", r, err)
	}
	if r, err := ParseRule("B2/S"); err != nil || r != (Rule{Birth: [9]bool{2: true}}) {
		t.Fatalf("B2/S: got %v, %v", r, err)
	}
	if r, _ := ParseRule("B3/S23"); r != Life {
		t.Fatalf("B3/S23 != Life")
	}
	r, err = ParseRule("345/2/4")
	if err != nil || r != (Rule{Birth: [9]bool{2: true}, Survive: [9]bool{3: true, 4: true, 5: true}, States: 4}) {
		t.Fatalf("345/2/4: got %v, %v", r, err)
	}
	if r, err := ParseRule("/2/3"); err != nil || r != (Rule{Birth: [9]bool{2: true}, States: 3}) {
		t.Fatalf("/2/3: got %v, %v", r, err)
	}
	for _, bad := range []string{"", "B3", "B3/S23/X", "B9/S23", "B3/B23", "X3/S23",
		"345/2/1", "345/2/", "9/2/3", "3/2/4/5", "B3/S23/4"} {
		if _, err := ParseRule(bad); err == nil {
			t.Errorf("ParseRule(%q): want error", bad)
		}
	}
}

func TestPalette(t *testing.T) {
	if p := gradients["bw"].palette(256, true, func(s int) float64 { return 1 }); len(p) != 2 {
		t.Errorf("bw palette: %d colours, want 2", len(p))
	}
	from, _ := parseHex("#000000")
	to, _ := parseHex("ff8000")
	gr := gradient{black, []color.RGBA{from, to}}
	p := gr.palette(3, false, func(s int) float64 { return float64(s) / 2 })
	want := color.Palette{from, color.RGBA{128, 64, 0, 255}, to}
	for i := range want {
		if p[i] != want[i] {
			t.Errorf("state %d: %v, want %v", i, p[i], want[i])
		}
	}
	for _, bad := range []string{"", "fff", "gg0000", "1234567"} {
		if _, err := parseHex(bad); err == nil {
			t.Errorf("parseHex(%q): want error", bad)
		}
	}
}
