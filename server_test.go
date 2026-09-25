package main

import (
	"bytes"
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
