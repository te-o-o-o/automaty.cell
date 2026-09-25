package main

import (
	"bytes"
	"encoding/json"
	"math/rand"
	"net/http"
	"net/http/httptest"
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

// Surprises must be interesting GIFs that keep the grid and cell size, use
// symmetry 8 only on square grids, and stay within the web page's limits.
func TestSurprise(t *testing.T) {
	var base options
	newFlagSet(&base).Parse([]string{"-w=120", "-h=80"})
	rng := rand.New(rand.NewSource(1))
	for i := 0; i < 10; i++ {
		o := surprise(rng, base)
		if !interesting(o) || !o.gif || o.w != 120 || o.h != 80 || o.symmetry == 8 || checkLimits(&o) != nil {
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
