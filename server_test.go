package main

import (
	"bytes"
	"encoding/json"
	"math/rand"
	"net/http"
	"net/http/httptest"
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

// Surprises must be interesting, keep the grid, and only use symmetry 8 on
// square grids.
func TestSurprise(t *testing.T) {
	var base options
	newFlagSet(&base).Parse([]string{"-w=120", "-h=80"})
	rng := rand.New(rand.NewSource(1))
	for i := 0; i < 10; i++ {
		o := surprise(rng, base)
		if !interesting(o) || o.w != 120 || o.h != 80 || o.symmetry == 8 {
			t.Fatalf("surprise %d: %+v", i, o)
		}
	}

	rec := httptest.NewRecorder()
	handleSurprise(rec, httptest.NewRequest("GET", "/surprise?w=50&h=50", nil))
	var got map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil || rec.Code != http.StatusOK || got["seed"] == "" {
		t.Fatalf("status %d, %v, %v", rec.Code, got, err)
	}
}
