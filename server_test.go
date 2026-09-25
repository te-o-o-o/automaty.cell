package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image/gif"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
)

// The web page must render exactly what the CLI renders for the same options,
// and refuse bad or oversized requests.
func TestHandleRender(t *testing.T) {
	get := func(query string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		handleRender(rec, httptest.NewRequest("GET", "/render?"+query, nil))
		return rec
	}

	if rec := get("colors=1a0033,ff3ea5,ffcc00&w=30&h=30"); rec.Code != http.StatusOK {
		t.Errorf("3-colour gradient: status %d, %s", rec.Code, rec.Body)
	}

	for _, format := range []string{"png", "gif"} {
		o := options{}
		newFlagSet(&o).Parse([]string{"-rule=highlife", "-w=30", "-h=20", "-gens=10", "-seed=3"})
		o.gif = format == "gif"
		var want bytes.Buffer
		if err := o.generate(&want); err != nil {
			t.Fatal(err)
		}
		rec := get("rule=highlife&w=30&h=20&gens=10&seed=3&format=" + format)
		if rec.Code != http.StatusOK || !bytes.Equal(rec.Body.Bytes(), want.Bytes()) {
			t.Errorf("%s: status %d, body differs from CLI: %v", format, rec.Code, !bytes.Equal(rec.Body.Bytes(), want.Bytes()))
		}
		if ct := rec.Header().Get("Content-Type"); ct != "image/"+format {
			t.Errorf("%s: Content-Type %q", format, ct)
		}
	}

	for _, bad := range []string{
		"rule=B9/S23",
		"colors=ff0000",
		"colors=ff0000,zz0000",
		"w=5000",
		"cyclic=true&radius=1000000000000",
		"format=gif&w=400&h=400&scale=8&gens=2000",
		"o=/tmp/x.png",
		"nope=1",
		"format=bmp",
		"w=0",
	} {
		if rec := get(bad); rec.Code != http.StatusBadRequest {
			t.Errorf("%s: status %d, want 400", bad, rec.Code)
		}
	}
}

// The work limit counts cells × generations whatever the rule: a big-radius
// cyclic render is as welcome as Life, and GIFs are capped by memory.
func TestCheckLimits(t *testing.T) {
	for _, tc := range []struct {
		args []string
		gif  bool
		ok   bool
	}{
		{[]string{"-cyclic", "-radius=3", "-neighborhood=moore", "-states=8", "-threshold=5", "-gens=150"}, false, true},
		{[]string{"-w=500", "-h=500", "-gens=444"}, false, true},
		{[]string{"-w=500", "-h=500", "-gens=445"}, false, false},
		{nil, true, true}, // the default GIF
		{[]string{"-gens=140"}, true, false},
	} {
		var o options
		newFlagSet(&o).Parse(tc.args)
		o.gif = tc.gif
		if err := checkLimits(&o); (err == nil) != tc.ok {
			t.Errorf("%v gif=%v: %v, want ok=%v", tc.args, tc.gif, err, tc.ok)
		}
	}
}

// Surprises must be interesting GIFs that keep the grid and cell size, use
// symmetry 8 or r4 only on square grids, and stay within the web page's limits.
func TestSurprise(t *testing.T) {
	var base options
	newFlagSet(&base).Parse([]string{"-w=120", "-h=80"})
	rng := rand.New(rand.NewSource(1))
	for i := 0; i < 10; i++ {
		o := surprise(rng, base)
		if !interesting(o) || !o.gif || o.w != 120 || o.h != 80 || o.symmetry == "8" || o.symmetry == "r4" || checkLimits(&o) != nil {
			t.Fatalf("surprise %d: %+v, limits: %v", i, o, checkLimits(&o))
		}
	}

	rec := httptest.NewRecorder()
	handleSurprise(rec, httptest.NewRequest("GET", "/surprise?w=50&h=50", nil))
	var got map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil || rec.Code != http.StatusOK || got["seed"] == "" {
		t.Fatalf("status %d, %v, %v", rec.Code, got, err)
	}
}

// The page renders in bonbon and English by default (also for unknown
// values), and in arcade or French on request.
func TestIndexThemes(t *testing.T) {
	for _, tc := range []struct {
		query string
		want  []string
	}{
		{"", []string{`data-theme="bonbon"`, `lang="en"`, "? RANDOM", "Automaton"}},
		{"?theme=sobre&lang=de", []string{`data-theme="bonbon"`, `lang="en"`}},
		{"?theme=arcade", []string{`data-theme="arcade"`, "? RANDOM"}},
		{"?lang=fr", []string{`lang="fr"`, "? RANDOM", "Automate", "calcul…"}},
	} {
		rec := httptest.NewRecorder()
		indexHandler(rec, httptest.NewRequest("GET", "/"+tc.query, nil))
		for _, want := range tc.want {
			if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), want) {
				t.Errorf("GET /%s: status %d, want %q in body", tc.query, rec.Code, want)
			}
		}
	}
}

// Every word must exist in every language.
func TestTexts(t *testing.T) {
	for key := range texts["en"] {
		for lang, words := range texts {
			if words[key] == "" {
				t.Errorf("%s: missing %q", lang, key)
			}
		}
	}
	if len(texts["en"]) != len(texts["fr"]) {
		t.Errorf("en has %d words, fr %d", len(texts["en"]), len(texts["fr"]))
	}
}

// The cyclic defaults are lively, mutated presets stay valid rules, and
// random gradients have 2 to 5 valid colours.
func TestSurpriseHelpers(t *testing.T) {
	var o options
	newFlagSet(&o).Parse(nil)
	key := fmt.Sprintf("%s,%d,%d,%d", o.neighborhood, o.radius, o.states, o.threshold)
	if !slices.Contains(livelyKeys(), key) {
		t.Errorf("cyclic defaults %s are not lively", key)
	}
	rng := rand.New(rand.NewSource(1))
	for i := 0; i < 200; i++ {
		rule := mutate(rng, Presets[i%len(Presets)].Rule)
		if _, err := ParseRule(rule); err != nil {
			t.Errorf("mutate(%s) = %s: %v", Presets[i%len(Presets)].Rule, rule, err)
		}
		colors := strings.Split(randomColors(rng), ",")
		for _, c := range colors {
			if _, err := parseHex(c); err != nil || len(colors) < 2 || len(colors) > 5 {
				t.Errorf("randomColors: %v", colors)
			}
		}
	}
}

// Ping-pong plays the frames forward then backward without repeating the
// ends: 5 generations give 5 + 3 frames.
func TestPingPong(t *testing.T) {
	var o options
	newFlagSet(&o).Parse([]string{"-w=10", "-h=10", "-gens=5", "-pingpong"})
	o.gif = true
	var buf bytes.Buffer
	if err := o.generate(&buf); err != nil {
		t.Fatal(err)
	}
	g, err := gif.DecodeAll(&buf)
	if err != nil || len(g.Image) != 8 {
		t.Fatalf("frames: %d, %v", len(g.Image), err)
	}
}
