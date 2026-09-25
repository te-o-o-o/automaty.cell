package main

import "testing"

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
	for _, bad := range []string{"", "B3", "B3/S23/X", "B9/S23", "B3/B23", "X3/S23"} {
		if _, err := ParseRule(bad); err == nil {
			t.Errorf("ParseRule(%q): want error", bad)
		}
	}
}
