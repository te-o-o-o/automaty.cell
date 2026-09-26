package main

import (
	"image/color"
	"math"
	"slices"
	"testing"
)

// Life is the classic Game of Life, B3/S23.
var Life = Rule{
	Birth:   [9]bool{3: true},
	Survive: [9]bool{2: true, 3: true},
}

// Set makes (x, y) a newborn cell, or kills it.
func (g *Grid) Set(x, y int, alive bool) {
	g.Cells[y*g.W+x] = 0
	if alive {
		g.Cells[y*g.W+x] = 1
	}
}

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

// A symmetric start (mirrors and rotations) stays symmetric as it evolves,
// with any rule.
func TestSymmetry(t *testing.T) {
	const n = 20
	for mode, wantSame := range map[string][]func(x, y int) (int, int){
		"2":  {func(x, y int) (int, int) { return n - 1 - x, y }},
		"4":  {func(x, y int) (int, int) { return n - 1 - x, y }, func(x, y int) (int, int) { return x, n - 1 - y }},
		"8":  {func(x, y int) (int, int) { return n - 1 - x, y }, func(x, y int) (int, int) { return y, x }},
		"r2": {func(x, y int) (int, int) { return n - 1 - x, n - 1 - y }},
		"r4": {func(x, y int) (int, int) { return n - 1 - y, x }},
	} {
		for _, rule := range []string{"B3/S23", "2/23/8"} {
			r, _ := ParseRule(rule)
			g := NewGrid(n, n, true)
			g.Randomize(1, 0.4)
			g.Symmetrize(mode)
			for i := 0; i < 30; i++ {
				g = g.Step(r)
			}
			for y := 0; y < n; y++ {
				for x := 0; x < n; x++ {
					for _, same := range wantSame {
						if sx, sy := same(x, y); g.Cells[y*n+x] != g.Cells[sy*n+sx] {
							t.Fatalf("symmetry %s, rule %s: (%d,%d) differs from (%d,%d)", mode, rule, x, y, sx, sy)
						}
					}
				}
			}
		}
	}
	// r2 alone is not a mirror: a half turn keeps an asymmetric pattern.
	g := NewGrid(n, n, true)
	g.Randomize(1, 0.4)
	g.Symmetrize("r2")
	mirrored := true
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			mirrored = mirrored && g.Cells[y*n+x] == g.Cells[y*n+n-1-x]
		}
	}
	if mirrored {
		t.Error("r2 gave a left/right mirror")
	}
}

// Starting from a full grid, each shape keeps exactly its area: probe a few
// cells inside and outside (60×40, so the centre is (29.5, 19.5), r = 20).
func TestShapes(t *testing.T) {
	const w, h = 60, 40
	for _, tc := range []struct {
		shape      string
		live, dead [][2]int
	}{
		{"all", [][2]int{{0, 0}, {30, 20}}, nil},
		{"dot", [][2]int{{30, 20}}, [][2]int{{30, 23}, {0, 0}}},
		{"disc", [][2]int{{30, 20}}, [][2]int{{0, 0}, {30, 5}}},
		{"ring", [][2]int{{30 + 13, 20}}, [][2]int{{30, 20}, {0, 0}}},
		{"cross", [][2]int{{30, 20}, {30, 0}, {0, 20}}, [][2]int{{0, 0}, {10, 5}}},
		{"frame", [][2]int{{0, 0}, {59, 39}, {30, 1}}, [][2]int{{30, 20}}},
		{"stripes", [][2]int{{0, 5}, {16, 5}}, [][2]int{{8, 5}, {59, 5}}},
		{"target", [][2]int{{30, 20}, {35, 20}}, [][2]int{{33, 20}, {38, 20}}},
		{"checker", [][2]int{{0, 0}, {8, 5}}, [][2]int{{8, 0}, {0, 5}}},
		{"spiral", [][2]int{{37, 20}, {30, 25}}, [][2]int{{35, 20}, {30, 29}}},
		{"rpentomino", [][2]int{{30, 19}, {29, 20}, {30, 21}}, [][2]int{{29, 19}, {31, 20}, {0, 0}}},
	} {
		g := NewGrid(w, h, true)
		for i := range g.Cells {
			g.Cells[i] = 1
		}
		g.KeepShape(tc.shape)
		for _, p := range tc.live {
			if g.Cells[p[1]*w+p[0]] == 0 {
				t.Errorf("%s: (%d,%d) dead, want live", tc.shape, p[0], p[1])
			}
		}
		for _, p := range tc.dead {
			if g.Cells[p[1]*w+p[0]] != 0 {
				t.Errorf("%s: (%d,%d) live, want dead", tc.shape, p[0], p[1])
			}
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
	// Every named gradient gives dead cells its background and live ones
	// a different colour.
	for name, gr := range gradients {
		p := gr.palette(256, true, func(s int) float64 { return float64(s-1) / 254 })
		if p[0] != gr.bg || p[1] == gr.bg {
			t.Errorf("%s: dead %v, first live %v", name, p[0], p[1])
		}
	}
	for _, bad := range []string{"", "fff", "gg0000", "1234567"} {
		if _, err := parseHex(bad); err == nil {
			t.Errorf("parseHex(%q): want error", bad)
		}
	}
}

// neighbours counts the cells within radius of (x, y) for which match is
// true, in a Moore (square) or Von Neumann (diamond) neighbourhood.
func (g *Grid) refNeighbours(x, y, radius int, vonNeumann bool, match func(uint8) bool) int {
	n := 0
	for dy := -radius; dy <= radius; dy++ {
		for dx := -radius; dx <= radius; dx++ {
			if dx == 0 && dy == 0 || vonNeumann && max(dx, -dx)+max(dy, -dy) > radius {
				continue
			}
			nx, ny := x+dx, y+dy
			if g.Wrap {
				nx, ny = (nx+g.W)%g.W, (ny+g.H)%g.H
			} else if nx < 0 || ny < 0 || nx >= g.W || ny >= g.H {
				continue
			}
			if match(g.Cells[ny*g.W+nx]) {
				n++
			}
		}
	}
	return n
}

// refStep is the original, straightforward Step, kept to check the fast one.
func (g *Grid) refStep(r Rule) *Grid {
	live := func(v uint8) bool { return v > 0 }
	if r.States > 0 {
		live = func(v uint8) bool { return v == 1 } // dying cells don't count
	}
	next := NewGrid(g.W, g.H, g.Wrap)
	for y := 0; y < g.H; y++ {
		for x := 0; x < g.W; x++ {
			n, i := g.refNeighbours(x, y, 1, false, live), y*g.W+x
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

// refStepCyclic is the original StepCyclic, kept to check the fast one.
func (g *Grid) refStepCyclic(c Cyclic) *Grid {
	next := NewGrid(g.W, g.H, g.Wrap)
	for y := 0; y < g.H; y++ {
		for x := 0; x < g.W; x++ {
			i := y*g.W + x
			k := g.Cells[i]
			succ := uint8((int(k) + 1) % c.States)
			next.Cells[i] = k
			if g.refNeighbours(x, y, c.Radius, c.VonNeumann, func(v uint8) bool { return v == succ }) >= c.Threshold {
				next.Cells[i] = succ
			}
		}
	}
	return next
}

// The fast steppers must match the straightforward ones exactly, with and
// without wrapped edges, on grids too small for any interior (radius 3 on
// 6 cells) and large enough to have one.
func TestFastStepsMatchReference(t *testing.T) {
	for _, wrap := range []bool{true, false} {
		for _, rule := range []string{"B3/S23", "B3678/S34678", "2/23/8", "/2/3"} {
			r, _ := ParseRule(rule)
			g := NewGrid(37, 23, wrap)
			g.Randomize(3, 0.4)
			for i := 0; i < 30; i++ {
				want, got := g.refStep(r), g.Step(r)
				if !slices.Equal(want.Cells, got.Cells) {
					t.Fatalf("%s wrap=%v: generation %d differs", rule, wrap, i)
				}
				g = got
			}
		}
		for _, c := range []Cyclic{{14, 1, 1, true}, {8, 5, 3, false}, {6, 2, 2, true}, {4, 3, 1, false}} {
			for _, size := range []int{6, 41} {
				g := NewGrid(size, size+3, wrap)
				g.RandomizeStates(5, c.States)
				for i := 0; i < 30; i++ {
					want, got := g.refStepCyclic(c), g.StepCyclic(c)
					if !slices.Equal(want.Cells, got.Cells) {
						t.Fatalf("%+v %dx%d wrap=%v: generation %d differs", c, size, size+3, wrap, i)
					}
					g = got
				}
			}
		}
	}
}

// Perlin noise is in [0, 1], the same for the same seed, smooth from cell to
// cell, and not flat.
func TestNoise(t *testing.T) {
	a, b := NoiseField(3, 60, 40, 20), NoiseField(3, 60, 40, 20)
	if !slices.Equal(a, b) {
		t.Fatal("same seed, different noise")
	}
	lo, hi, step := 1.0, 0.0, 0.0
	for i, v := range a {
		lo, hi = min(lo, v), max(hi, v)
		if i%60 != 59 {
			step += math.Abs(a[i+1] - v)
		}
	}
	if lo < 0 || hi > 1 || hi-lo < 0.3 || step/float64(len(a)) > 0.05 {
		t.Errorf("range %.2f-%.2f, mean step %.3f", lo, hi, step/float64(len(a)))
	}
}

// The noise tiles: across the right/left and bottom/top edges it changes no
// more than between neighbours inside, so wrapped grids and mosaics show no
// seam (before, the seam was many times the inner step).
func TestNoiseTiles(t *testing.T) {
	const w, h = 90, 60
	for _, scale := range []float64{15, 40} {
		f := NoiseField(7, w, h, scale)
		stepX, stepY, seamX, seamY := 0.0, 0.0, 0.0, 0.0
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				if x < w-1 {
					stepX += math.Abs(f[y*w+x+1] - f[y*w+x])
				}
				if y < h-1 {
					stepY += math.Abs(f[(y+1)*w+x] - f[y*w+x])
				}
			}
			seamX += math.Abs(f[y*w] - f[y*w+w-1])
		}
		for x := 0; x < w; x++ {
			seamY += math.Abs(f[x] - f[(h-1)*w+x])
		}
		stepX, stepY = stepX/float64(h*(w-1)), stepY/float64(w*(h-1))
		seamX, seamY = seamX/h, seamY/w
		if seamX > 2*stepX || seamY > 2*stepY {
			t.Errorf("scale %v: steps %.4f/%.4f, seams %.4f/%.4f", scale, stepX, stepY, seamX, seamY)
		}
	}
}

// Perlin starts are islands: no cell is born where the noise is low.
func TestNoiseIslands(t *testing.T) {
	const w, h = 60, 40
	field := NoiseField(3, w, h, 15)
	g := NewGrid(w, h, true)
	g.RandomizeNoise(3, 0.3, 15)
	inside, live := 0, 0
	for i, f := range field {
		if f <= 0.5 && g.Cells[i] != 0 {
			t.Fatalf("cell %d alive outside the islands", i)
		}
		if f > 0.5 {
			inside++
			live += int(g.Cells[i])
		}
	}
	if inside == 0 || float64(live)/float64(inside) < 0.4 {
		t.Errorf("islands too sparse: %d live of %d", live, inside)
	}
}

// -at switches the rule at its generation, keeps the cells across families,
// and rejects options that can't change during a run.
func TestKeyframes(t *testing.T) {
	var o options
	newFlagSet(&o).Parse([]string{"-w=20", "-h=20", "-density=0.5", "-at=3:rule=B/S 5:cyclic=true 5:states=4"})
	g, step, pal, err := o.setup()
	if err != nil {
		t.Fatal(err)
	}
	for gen := 1; gen <= 4; gen++ {
		g = step(g)
		if live := slices.ContainsFunc(g.Cells, func(v uint8) bool { return v != 0 }); live != (gen < 3) {
			t.Errorf("gen %d: live cells %v, want %v (B/S kills all from gen 3)", gen, live, gen < 3)
		}
	}
	g = step(g) // gen 5: cyclic, 4 states, from an empty grid
	if len(pal()) != 4 || slices.ContainsFunc(g.Cells, func(v uint8) bool { return v != 0 }) {
		t.Errorf("gen 5: palette of %d, cells %v", len(pal()), g.Cells)
	}

	o.at = "3:density=0.1"
	if _, _, _, err := o.setup(); err == nil {
		t.Error("-at density: want an error")
	}
}

// Cells keep their meaning when the family changes.
func TestConvert(t *testing.T) {
	bs, gen4, gen3, cyc3 := automaton{}, automaton{states: 4}, automaton{states: 3}, automaton{cyclic: true, states: 3}
	for _, tc := range []struct {
		from, to automaton
		in, want []uint8
	}{
		{bs, gen4, []uint8{0, 1, 7}, []uint8{0, 1, 1}},
		{gen4, gen3, []uint8{0, 1, 2, 3}, []uint8{0, 1, 2, 0}},
		{gen4, bs, []uint8{0, 1, 2}, []uint8{0, 1, 0}},
		{cyc3, bs, []uint8{0, 1, 2}, []uint8{0, 1, 0}},
		{bs, cyc3, []uint8{0, 1, 5}, []uint8{0, 1, 2}},
	} {
		cells := slices.Clone(tc.in)
		convert(cells, tc.from, tc.to)
		if !slices.Equal(cells, tc.want) {
			t.Errorf("%+v → %+v: %v gives %v, want %v", tc.from, tc.to, tc.in, cells, tc.want)
		}
	}
}

// A one-colour gradient (bw) still gives cyclic states different shades.
func TestCyclicBW(t *testing.T) {
	var o options
	newFlagSet(&o).Parse([]string{"-cyclic", "-palette=bw", "-w=20", "-h=20"})
	_, _, palette, err := o.setup()
	if err != nil {
		t.Fatal(err)
	}
	if pal := palette(); len(pal) < 3 || pal[1] == pal[len(pal)-1] {
		t.Fatalf("palette %v, %v", pal, err)
	}
}

// Larger than Life at radius 1 must be exactly Life, whether the middle cell
// is counted (then survival is 3-4) or not (2-3), with or without wrapped
// edges: this checks the summed-area table, padding included.
func TestLtLIsLifeAtRadius1(t *testing.T) {
	for _, rule := range []string{"R1,C0,M0,S2..3,B3..3,NM", "r1,c2,m1,s3..4,b3..3,nm"} {
		r, err := ParseLtL(rule)
		if err != nil {
			t.Fatal(err)
		}
		for _, wrap := range []bool{true, false} {
			life, ltl := NewGrid(37, 23, wrap), NewGrid(37, 23, wrap)
			life.Randomize(4, 0.35)
			copy(ltl.Cells, life.Cells)
			for i := 0; i < 40; i++ {
				life = life.Step(Life)
				next := NewGrid(37, 23, wrap)
				ltl.StepLtLInto(next, r)
				ltl = next
				if !slices.Equal(life.Cells, ltl.Cells) {
					t.Fatalf("%s wrap=%v: generation %d differs from Life", rule, wrap, i+1)
				}
			}
		}
	}
}

func TestParseLtL(t *testing.T) {
	r, err := ParseLtL("R5,C0,M1,S34..58,B34..45,NM")
	if err != nil || r != (LtL{Radius: 5, Middle: true, SMin: 34, SMax: 58, BMin: 34, BMax: 45}) {
		t.Fatalf("got %+v, %v", r, err)
	}
	for _, bad := range []string{"R5", "R0,S1..2,B1..2", "R5,S3,B1..2", "R5,S1..2,B0..2", "R5,C1,S1..2,B1..2",
		"R5,S1..2,B1..2,NN", "R5,S1..2,B1..2,X3", "R5,,S1..2,B1..2", "R5,M2,S1..2,B1..2"} {
		if _, err := ParseLtL(bad); err == nil {
			t.Errorf("ParseLtL(%q): want an error", bad)
		}
	}
}

// Step returns the next generation under rule r.
func (g *Grid) Step(r Rule) *Grid {
	next := NewGrid(g.W, g.H, g.Wrap)
	g.StepInto(next, r)
	return next
}

// StepCyclic returns the next generation under cyclic rule c.
func (g *Grid) StepCyclic(c Cyclic) *Grid {
	next := NewGrid(g.W, g.H, g.Wrap)
	g.StepCyclicInto(next, c)
	return next
}
