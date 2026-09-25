package main

import (
	"math/rand"
	"strconv"
	"strings"
)

var paletteNames = []string{"age", "fire", "ocean", "viridis", "mono"}

// surprise returns random creative settings (rule, palette, seed, symmetry…)
// on top of base, retrying until the automaton looks interesting. Grid size,
// scale, generations and format are kept from base.
func surprise(rng *rand.Rand, base options) options {
	for try := 0; try < 300; try++ {
		o := base
		o.seed = rng.Int63n(1_000_000)
		o.palette = paletteNames[rng.Intn(len(paletteNames))]
		o.palFrom, o.palTo = "", ""
		o.symmetry = []int{1, 1, 2, 4, 8}[rng.Intn(5)]
		if o.symmetry == 8 && o.w != o.h {
			o.symmetry = 4
		}
		o.cyclic = rng.Float64() < 0.25
		if o.cyclic {
			o.states = 3 + rng.Intn(14)
			o.threshold = 1 + rng.Intn(5)
			o.radius = 1 + rng.Intn(3)
			o.neighborhood = []string{"moore", "vonneumann"}[rng.Intn(2)]
		} else {
			o.rule = randomRule(rng)
			o.density = float64(20+rng.Intn(41)) / 100
		}
		if interesting(o) {
			return o
		}
	}
	return base // ponytail: 300 duds in a row never happened in tests; add a better fallback if it does
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
	if o.cyclic {
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
