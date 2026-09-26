package main

import (
	"fmt"
	"math"
	"math/rand"
	"slices"
	"strconv"
	"strings"
)

// cyclicLively lists, per neighbourhood and state count, the thresholds for
// which a cyclic automaton keeps moving in waves or spirals: above them it
// freezes, below them it boils into noise. Measured on 100×100 grids over 150
// generations, with two seeds, for 3 to 16 states.
var cyclicLively = []struct {
	neighborhood string
	radius       int
	thresholds   map[int][]int // states → thresholds
}{
	{"moore", 1, map[int][]int{3: {3}, 4: {2, 3}, 5: {2, 3}, 6: {2}, 7: {2}, 8: {1, 2}, 9: {1, 2},
		10: {1}, 11: {1}, 12: {1}, 13: {1}, 14: {1}, 15: {1}, 16: {1}}},
	{"moore", 2, map[int][]int{3: {6, 7, 8, 9, 10}, 4: {5, 6, 7, 8, 9}, 5: {4, 5, 6, 7, 8}, 6: {3, 4, 5, 6, 7},
		7: {3, 4, 5, 6}, 8: {3, 4, 5}, 9: {2, 3, 4}, 10: {2, 3, 4}, 11: {2, 3, 4}, 12: {2},
		13: {2, 3}, 14: {2, 3}, 15: {2, 3}, 16: {2, 3}}},
	{"moore", 3, map[int][]int{3: {12}, 4: {9, 10, 11, 12}, 5: {7, 8, 9, 10, 11, 12}, 6: {6, 7, 9, 10, 11, 12},
		7: {5, 6, 7, 9, 10, 11}, 8: {4, 5, 6, 7, 8, 9, 10}, 9: {4, 5, 6, 7, 8, 9}, 10: {4, 6, 7, 8},
		11: {4, 6, 7}, 12: {3, 4, 6}, 13: {3, 6}, 14: {3, 4, 5, 6}, 15: {3, 5}, 16: {3, 5}}},
	{"vonneumann", 1, map[int][]int{14: {1}, 15: {1}, 16: {1}}},
	{"vonneumann", 2, map[int][]int{3: {4}, 4: {3, 4}, 5: {2, 3, 4}, 6: {2, 3}, 7: {2, 3}, 8: {2, 3},
		9: {2}, 10: {2}, 11: {2}, 12: {2}, 13: {2}, 14: {2}, 15: {1}, 16: {1}}},
	{"vonneumann", 3, map[int][]int{3: {6, 7, 8, 9}, 4: {5, 6, 7, 8, 9}, 5: {4, 5, 6, 7, 8}, 6: {3, 4, 5, 6, 7},
		7: {3, 4, 5, 6}, 8: {3, 4, 5}, 9: {3, 4, 5}, 10: {2, 3, 4}, 11: {2, 4}, 12: {2, 3},
		13: {2, 3}, 14: {2, 3}, 15: {2, 3}}},
}

// livelySettings flattens cyclicLively, in a fixed order (so a seed always
// gives the same pick): random picks each setting with the same odds.
var livelySettings = func() (all []Cyclic) {
	for _, e := range cyclicLively {
		for states := 3; states <= 16; states++ {
			for _, t := range e.thresholds[states] {
				all = append(all, Cyclic{States: states, Threshold: t, Radius: e.radius, VonNeumann: e.neighborhood == "vonneumann"})
			}
		}
	}
	return all
}()

// livelyKeys lists the lively settings as "neighborhood,radius,states,threshold",
// for the web page's warning.
func livelyKeys() []string {
	var keys []string
	for _, c := range livelySettings {
		keys = append(keys, fmt.Sprintf("%s,%d,%d,%d", neighborhoodName(c.VonNeumann), c.Radius, c.States, c.Threshold))
	}
	return keys
}

func neighborhoodName(vonNeumann bool) string {
	if vonNeumann {
		return "vonneumann"
	}
	return "moore"
}

// surprise returns random settings on top of base, retrying until the
// automaton looks interesting. It varies the family (a lively cyclic setting,
// a preset, a mutated preset or a random rule), density, colours (a palette
// or a random gradient), seed, symmetry, start shape, Perlin blobs, and the
// animation (a GIF): generations, speed, edges and ping-pong. The grid size and cell size stay those of
// base; generations are cut if needed to stay within the web page's limits.
func surprise(rng *rand.Rand, base options) options {
	var colourful []string // every gradient but black and white
	for _, name := range paletteNames() {
		if name != "bw" {
			colourful = append(colourful, name)
		}
	}
	// The family is drawn once: retrying it too would favour the families
	// that pass the interest test most often (cyclic ones nearly always do).
	family := rng.Float64()
	for try := 0; try < 300; try++ {
		o := base
		o.seed = rng.Int63n(1_000_000)
		o.palette = colourful[rng.Intn(len(colourful))]
		o.colors = ""
		if rng.Float64() < 0.35 {
			o.colors = randomColors(rng)
		}
		o.symmetry = []string{"1", "1", "2", "4", "8", "r2", "r4"}[rng.Intn(7)]
		if (o.symmetry == "8" || o.symmetry == "r4") && o.w != o.h {
			o.symmetry = "r2"
		}
		o.shape = "all"
		o.at = "" // changes made for the old settings would not fit the new ones
		if rng.Intn(3) == 0 {
			o.shape = shapes[1+rng.Intn(len(shapes)-1)]
		}
		o.pingpong = rng.Intn(3) == 0
		o.gif = true
		o.gens = 60 + rng.Intn(91)
		o.delay = []int{3, 5, 8, 12}[rng.Intn(4)]
		o.wrap = rng.Float64() < 0.75

		o.cyclic = false
		o.density = 0.05 + float64(rng.Intn(66))/100
		o.noise = 0
		if rng.Intn(3) == 0 {
			o.noise = float64(15 + rng.Intn(56))
		}
		switch {
		case family < 0.25:
			c := livelySettings[rng.Intn(len(livelySettings))]
			o.cyclic, o.neighborhood = true, neighborhoodName(c.VonNeumann)
			o.states, o.threshold, o.radius = c.States, c.Threshold, c.Radius
		case family < 0.45:
			o.rule = Presets[rng.Intn(len(Presets))].Rule
		case family < 0.65:
			o.rule = mutate(rng, Presets[rng.Intn(len(Presets))].Rule)
		default:
			o.rule = randomRule(rng)
		}
		for o.gens > 20 && checkLimits(&o) != nil {
			o.gens -= 10
		}
		if interesting(o) {
			return o
		}
	}
	return base // ponytail: 300 duds in a row never happened in tests; add a better fallback if it does
}

// minLook is how different a mutant must look from its rule: the largest
// relative gap between their live, change and structure measures (see
// activity), from 0 (the same) to 1. Raise it for wilder mutants.
const minLook = 0.3

// mutateLively returns a mutant of o's rule, 2 to 4 mutations away (see
// mutate), that stays interesting with o's other settings and looks
// different from the rule it came from: many mutants behave just like it.
// Failing that, it returns the most different lively mutant tried. When the
// rule itself fails the test on the small grid, as Larger than Life rules that
// freeze into blobs do, a mutant that neither dies, fills the grid nor turns
// to noise will do.
func mutateLively(rng *rand.Rand, o options) string {
	rule := strings.ToUpper(resolvePreset(o.rule))
	pl, pc, ps := activity(o)
	strict := lively(false, pl, pc, ps)
	rel := func(a, b float64) float64 {
		if a+b == 0 {
			return 0
		}
		return math.Abs(a-b) / (a + b)
	}
	best, bestLook := "", -1.0
	for try := 0; try < 50; try++ {
		m := rule
		for range 2 + rng.Intn(3) {
			m = mutate(rng, m)
		}
		om := o
		om.rule = m
		if _, err := om.automaton(); err != nil || m == rule {
			continue
		}
		l, c, s := activity(om)
		if strict && !lively(false, l, c, s) || !strict && (l < 0.02 || l > 0.98 || s < 0.1) {
			continue
		}
		look := max(rel(l, pl), rel(c, pc), rel(s, ps))
		if look >= minLook {
			return m
		}
		if look > bestLook {
			best, bestLook = m, look
		}
	}
	if best == "" {
		return mutate(rng, rule)
	}
	return best
}

// digitWeights makes mutate favour the neighbour counts that come up most at
// usual densities: flipping 7 or 8 rarely changes anything.
var digitWeights = [9]int{1, 3, 3, 3, 3, 3, 2, 1, 1}

// mutate flips one neighbour count in a B/S or Generations rule (never birth
// on 0), or for Generations sometimes changes the number of states. Larger
// than Life rules go to mutateLtL.
func mutate(rng *rand.Rand, rule string) string {
	if IsLtL(rule) {
		return mutateLtL(rng, rule)
	}
	parts := strings.Split(rule, "/")
	toggle := func(digits string, from int) string {
		total := 0
		for _, w := range digitWeights[from:] {
			total += w
		}
		n, x := from, rng.Intn(total)
		for x >= digitWeights[n] {
			x -= digitWeights[n]
			n++
		}
		d := string(rune('0' + n))
		if strings.Contains(digits, d) {
			return strings.Replace(digits, d, "", 1)
		}
		b := []byte(digits + d)
		slices.Sort(b)
		return string(b)
	}
	if len(parts) == 3 { // S/B/C
		switch rng.Intn(3) {
		case 0:
			parts[0] = toggle(parts[0], 0)
		case 1:
			parts[1] = toggle(parts[1], 1)
		default:
			c, _ := strconv.Atoi(parts[2])
			parts[2] = strconv.Itoa(max(3, c+[]int{-2, -1, 1, 2}[rng.Intn(4)]))
		}
		return strings.Join(parts, "/")
	}
	if len(parts) != 2 { // not a rule mutate knows: leave it
		return rule
	}
	i, from := rng.Intn(2), 0 // B…/S…
	if parts[i][0] == 'B' {
		from = 1
	}
	parts[i] = parts[i][:1] + toggle(parts[i][1:], from)
	return strings.Join(parts, "/")
}

// mutateLtL changes one thing in a Larger than Life rule: the radius (its
// ranges scaled along, so it looks alike at another size), one end of the
// survival or birth range (by up to a twelfth of the neighbourhood), or
// whether the middle cell counts.
func mutateLtL(rng *rand.Rand, rule string) string {
	r, err := ParseLtL(rule)
	if err != nil {
		return rule
	}
	side := func(radius int) int { return (2*radius + 1) * (2*radius + 1) }
	n := side(r.Radius)
	nudge := func(v int) int { return v + (1+rng.Intn(max(1, n/12)))*(1-2*rng.Intn(2)) }
	switch rng.Intn(6) {
	case 0:
		radius := min(20, max(1, r.Radius+1-2*rng.Intn(2)))
		k := float64(side(radius)) / float64(n)
		scale := func(v int) int { return int(math.Round(float64(v) * k)) }
		r.Radius, r.SMin, r.SMax, r.BMin, r.BMax = radius, scale(r.SMin), scale(r.SMax), scale(r.BMin), scale(r.BMax)
		n = side(radius)
	case 1:
		r.SMin = nudge(r.SMin)
	case 2:
		r.SMax = nudge(r.SMax)
	case 3:
		r.BMin = nudge(r.BMin)
	case 4:
		r.BMax = nudge(r.BMax)
	default:
		r.Middle = !r.Middle
	}
	r.SMin = min(max(r.SMin, 0), n)
	r.SMax = min(max(r.SMax, r.SMin), n)
	r.BMin = min(max(r.BMin, 1), n)
	r.BMax = min(max(r.BMax, r.BMin), n)
	middle := 0
	if r.Middle {
		middle = 1
	}
	return fmt.Sprintf("R%d,C%d,M%d,S%d..%d,B%d..%d,NM", r.Radius, r.States, middle, r.SMin, r.SMax, r.BMin, r.BMax)
}

// randomColors returns 2 to 5 colours as RRGGBB,…: hues a random step apart,
// from light (young or alive cells) to dark (old or dying ones), so they read
// well on a dark background.
func randomColors(rng *rand.Rand) string {
	n := 2 + rng.Intn(4)
	hue := rng.Float64() * 360
	step := (20 + rng.Float64()*100) * float64(1-2*rng.Intn(2))
	sat := 0.65 + rng.Float64()*0.35
	var hex []string
	for i := 0; i < n; i++ {
		light := 0.85 - 0.55*float64(i)/float64(n-1)
		r, g, b := hsl(math.Mod(hue+step*float64(i)+720, 360), sat, light)
		hex = append(hex, fmt.Sprintf("%02x%02x%02x", r, g, b))
	}
	return strings.Join(hex, ",")
}

// hsl converts a hue (degrees), saturation and lightness (0-1) to RGB.
func hsl(h, s, l float64) (r, g, b uint8) {
	c := (1 - math.Abs(2*l-1)) * s
	x := c * (1 - math.Abs(math.Mod(h/60, 2)-1))
	var rf, gf, bf float64
	switch {
	case h < 60:
		rf, gf = c, x
	case h < 120:
		rf, gf = x, c
	case h < 180:
		gf, bf = c, x
	case h < 240:
		gf, bf = x, c
	case h < 300:
		rf, bf = x, c
	default:
		rf, bf = c, x
	}
	m := l - c/2
	to := func(v float64) uint8 { return uint8(math.Round((v + m) * 255)) }
	return to(rf), to(gf), to(bf)
}

// randomRule returns a B/S rule, or a Generations rule one time in three.
// Birth on 0 neighbours is left out: the whole empty grid would flash.
func randomRule(rng *rand.Rand) string {
	digits := func(from int, p float64) string {
		var b strings.Builder
		for n := from; n <= 8; n++ {
			if rng.Float64() < p {
				b.WriteByte(byte('0' + n))
			}
		}
		return b.String()
	}
	birth, survive := digits(1, 0.3), digits(0, 0.4)
	if rng.Intn(3) == 0 {
		return survive + "/" + birth + "/" + strconv.Itoa(3+rng.Intn(8))
	}
	return "B" + birth + "/S" + survive
}

// interesting runs o on a small grid and checks that it neither dies out,
// freezes, boils, nor stays noise: some cells live, a moderate share keeps
// changing, and neighbours look more alike than random cells would.
func interesting(o options) bool {
	live, change, structure := activity(o)
	return lively(o.cyclic, live, change, structure)
}

// lively is interesting's test on activity's measures.
func lively(cyclic bool, live, change, structure float64) bool {
	if cyclic {
		// Spirals and waves change everywhere at once, like noise does: only
		// structure sets them apart.
		return change > 0.02 && structure > 0.2
	}
	return live > 0.02 && live < 0.8 && change > 0.0005 && change < 0.3 && structure > 0.01
}

// activity runs o on a 64×64 grid and returns, at the end: the share of live
// (non-zero) cells, the share of cells changing per generation over the last
// 10, and structure, the share of neighbour pairs in the same state (alive or
// dead for rules) minus what randomly placed cells with the same counts give.
func activity(o options) (live, change, structure float64) {
	const side, gens = 64, 150
	o.w, o.h, o.gens = side, side, gens
	g, step, _, err := o.setup()
	if err != nil {
		return 0, 0, 0
	}
	key := func(v uint8) uint8 { return v }
	if !o.cyclic {
		key = func(v uint8) uint8 { return min(v, 1) }
	}

	var changed int
	for i := 0; i < gens; i++ {
		next := step(g)
		if i >= gens-10 {
			for j, v := range next.Cells {
				if key(v) != key(g.Cells[j]) {
					changed++
				}
			}
		}
		g = next
	}

	var counts [256]int
	var same int
	for y := 0; y < side; y++ {
		for x := 0; x < side; x++ {
			k := key(g.Cells[y*side+x])
			counts[k]++
			if k == key(g.Cells[y*side+(x+1)%side]) {
				same++
			}
			if k == key(g.Cells[(y+1)%side*side+x]) {
				same++
			}
		}
	}
	n := float64(side * side)
	random := 0.0 // chance that two random cells share a state
	for _, c := range counts {
		random += float64(c) / n * float64(c) / n
	}
	return 1 - float64(counts[0])/n, float64(changed) / (10 * n), float64(same)/(2*n) - random
}
