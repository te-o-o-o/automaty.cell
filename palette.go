package main

import (
	"fmt"
	"image/color"
	"sort"
	"strconv"
	"strings"
)

// A gradient colours dead cells with bg, and spreads live states along stops.
type gradient struct {
	bg    color.RGBA
	stops []color.RGBA
}

var black = color.RGBA{0, 0, 0, 255}

// Young or alive states take the first stop, old or dying ones the last.
var gradients = map[string]gradient{
	// age: pale yellow through orange and magenta to deep blue.
	"age": {color.RGBA{11, 11, 20, 255}, []color.RGBA{
		{255, 246, 192, 255}, {255, 138, 61, 255}, {194, 24, 91, 255}, {40, 53, 147, 255}}},
	"bw": {black, []color.RGBA{{255, 255, 255, 255}}},
	"fire": {black, []color.RGBA{
		{255, 255, 220, 255}, {255, 214, 64, 255}, {245, 120, 20, 255}, {190, 30, 10, 255}, {70, 8, 8, 255}}},
	"ocean": {color.RGBA{4, 12, 28, 255}, []color.RGBA{
		{225, 250, 255, 255}, {80, 200, 230, 255}, {20, 110, 190, 255}, {15, 45, 110, 255}}},
	"viridis": {color.RGBA{10, 10, 10, 255}, []color.RGBA{
		{253, 231, 37, 255}, {94, 201, 98, 255}, {33, 145, 140, 255}, {59, 82, 139, 255}, {68, 1, 84, 255}}},
	"mono": {black, []color.RGBA{{255, 255, 255, 255}, {60, 60, 60, 255}}},
	"neon": {color.RGBA{13, 7, 20, 255}, []color.RGBA{
		{255, 255, 255, 255}, {62, 243, 255, 255}, {255, 62, 165, 255}, {106, 27, 154, 255}}},
	"sunset": {color.RGBA{18, 8, 24, 255}, []color.RGBA{
		{255, 243, 176, 255}, {255, 179, 71, 255}, {255, 94, 98, 255}, {142, 45, 226, 255}, {43, 16, 85, 255}}},
	"toxic": {color.RGBA{5, 10, 5, 255}, []color.RGBA{
		{244, 255, 138, 255}, {157, 255, 62, 255}, {47, 191, 113, 255}, {11, 93, 59, 255}}},
	"aurora": {color.RGBA{6, 8, 22, 255}, []color.RGBA{
		{234, 255, 208, 255}, {125, 255, 178, 255}, {45, 226, 230, 255}, {123, 92, 255, 255}, {59, 29, 110, 255}}},
	"candy": {color.RGBA{26, 11, 22, 255}, []color.RGBA{
		{255, 240, 246, 255}, {255, 154, 213, 255}, {179, 136, 255, 255}, {122, 215, 255, 255}}},
	"rainbow": {black, []color.RGBA{
		{255, 59, 59, 255}, {255, 177, 59, 255}, {246, 255, 59, 255}, {59, 255, 107, 255},
		{59, 216, 255, 255}, {123, 59, 255, 255}, {255, 59, 216, 255}}},
	"gold": {color.RGBA{14, 10, 4, 255}, []color.RGBA{
		{255, 248, 220, 255}, {255, 215, 0, 255}, {192, 138, 0, 255}, {92, 61, 0, 255}}},
}

// paletteNames lists the gradients by name, sorted.
func paletteNames() []string {
	var names []string
	for name := range gradients {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// at returns the colour at position t in [0, 1] along the stops.
func (gr gradient) at(t float64) color.RGBA {
	if len(gr.stops) == 1 {
		return gr.stops[0]
	}
	t *= float64(len(gr.stops) - 1)
	i := min(int(t), len(gr.stops)-2)
	f := t - float64(i)
	a, b := gr.stops[i], gr.stops[i+1]
	lerp := func(x, y uint8) uint8 { return uint8(float64(x) + (float64(y)-float64(x))*f + 0.5) }
	return color.RGBA{lerp(a.R, b.R), lerp(a.G, b.G), lerp(a.B, b.B), 255}
}

// palette colours n states: state s gets the gradient colour at pos(s), except
// state 0 which gets bg when withBg is set (dead cells).
func (gr gradient) palette(n int, withBg bool, pos func(s int) float64) color.Palette {
	p := color.Palette{}
	for s := 0; s < n; s++ {
		if s == 0 && withBg {
			p = append(p, gr.bg)
		} else {
			p = append(p, gr.at(pos(s)))
		}
	}
	// Render clamps states to the last colour, so trailing repeats can go:
	// bw stays a 2-colour palette, which keeps PNG and GIF files small.
	for len(p) > 1 && p[len(p)-1] == p[len(p)-2] {
		p = p[:len(p)-1]
	}
	return p
}

// parseHex reads a colour written RRGGBB or #RRGGBB.
func parseHex(s string) (color.RGBA, error) {
	s = strings.TrimPrefix(s, "#")
	v, err := strconv.ParseUint(s, 16, 32)
	if err != nil || len(s) != 6 {
		return color.RGBA{}, fmt.Errorf("colour %q: want RRGGBB", s)
	}
	return color.RGBA{uint8(v >> 16), uint8(v >> 8), uint8(v), 255}, nil
}
