package main

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"html/template"
	"io"
	"log"
	"math/rand"
	"net/http"
	"time"
)

//go:embed index.html
var indexHTML string

var indexTmpl = template.Must(template.New("index").Parse(indexHTML))

// Limits for the web server only: a public page must not let one request
// take the whole machine. The CLI has none.
const (
	maxSide   = 500         // cells per side
	maxScale  = 8           // pixels per cell
	maxGens   = 2000        // generations
	maxWork   = 111_000_000 // cells × generations: about 1 s at worst, whatever the rule
	maxPixels = 80_000_000  // GIF: pixels over all frames, held in memory (about 80 MB)
)

// busy lets at most 2 renders run at once; other requests wait their turn.
var busy = make(chan struct{}, 2)

func serve(addr string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", indexHandler)
	mux.HandleFunc("GET /render", handleRender)
	mux.HandleFunc("POST /render", handleRender) // with a mask image
	mux.HandleFunc("GET /surprise", handleSurprise)
	mux.HandleFunc("GET /mutate", handleMutate)

	log.Printf("automaty.cell: listening on %s", addr)
	srv := &http.Server{Addr: addr, Handler: mux, ReadHeaderTimeout: 10 * time.Second, WriteTimeout: time.Minute}
	return srv.ListenAndServe()
}

// indexHandler serves the page in the theme named by ?theme= (bonbon, the
// default, or arcade) and the language named by ?lang= (en, the default, or fr).
func indexHandler(w http.ResponseWriter, r *http.Request) {
	theme := r.URL.Query().Get("theme")
	if theme != "arcade" {
		theme = "bonbon"
	}
	lang := r.URL.Query().Get("lang")
	if lang != "fr" {
		lang = "en"
	}
	type palette struct{ Name, CSS string }
	var palettes []palette
	for _, name := range paletteNames() {
		palettes = append(palettes, palette{name, gradients[name].css()})
	}
	err := indexTmpl.Execute(w, map[string]any{
		"Presets": Presets, "Palettes": palettes, "MaxSide": maxSide, "MaxScale": maxScale, "MaxGens": maxGens,
		"Theme": theme, "Lang": lang, "T": texts[lang], "Lively": livelyKeys(),
	})
	if err != nil {
		log.Print(err)
	}
}

// handleRender reads the same options as the CLI from the query string
// (?rule=B3/S23&w=100 is -rule=B3/S23 -w=100), plus format=png|gif, and
// replies with the image.
func handleRender(w http.ResponseWriter, r *http.Request) {
	o, err := parseQuery(r)
	if err == nil && r.Method == http.MethodPost { // a mask image, as multipart "mask"
		r.Body = http.MaxBytesReader(w, r.Body, 10<<20)
		var f io.ReadCloser
		if f, _, err = r.FormFile("mask"); err == nil {
			o.maskData, err = io.ReadAll(f)
			f.Close()
		}
	}
	if err == nil {
		err = checkLimits(o)
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	select {
	case busy <- struct{}{}:
		defer func() { <-busy }()
	case <-r.Context().Done():
		return
	}
	var buf bytes.Buffer
	if err := o.generate(&buf); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	switch {
	case o.gif:
		w.Header().Set("Content-Type", "image/gif")
	case o.zip:
		w.Header().Set("Content-Type", "application/zip")
		w.Header().Set("Content-Disposition", `attachment; filename="automaty.cell.zip"`)
	default:
		w.Header().Set("Content-Type", "image/png")
	}
	w.Write(buf.Bytes())
}

func parseQuery(r *http.Request) (*options, error) {
	q := r.URL.Query()
	var o options
	switch q.Get("format") {
	case "", "png":
	case "gif":
		o.gif = true
	case "zip":
		o.zip = true
	default:
		return nil, fmt.Errorf("unknown format %q (want png, gif or zip)", q.Get("format"))
	}
	q.Del("format")

	var args []string
	for k, vs := range q {
		// mask is a path: the server must never read its own files for a visitor.
		if k == "o" || k == "serve" || k == "list-rules" || k == "mcp" || k == "mask" {
			return nil, fmt.Errorf("option %q is CLI only", k)
		}
		for _, v := range vs {
			args = append(args, "-"+k+"="+v)
		}
	}
	fs := newFlagSet(&o)
	fs.SetOutput(io.Discard)
	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	return &o, nil
}

func checkLimits(o *options) error {
	switch {
	case o.w > maxSide || o.h > maxSide:
		return fmt.Errorf("grid is at most %d×%d cells here (use the CLI for more)", maxSide, maxSide)
	case o.scale > maxScale:
		return fmt.Errorf("scale is at most %d here", maxScale)
	case o.gens > maxGens:
		return fmt.Errorf("at most %d generations here", maxGens)
	case o.w*o.h*o.gens > maxWork:
		return errors.New("too much work for the web page: lower the grid size or generations (or use the CLI)")
	case (o.gif || o.zip) && o.w*o.h*o.scale*o.scale*o.gens > maxPixels:
		return errors.New("GIF too large for the web page: lower the grid size, scale or generations (or use the CLI)")
	}
	return nil
}

// handleSurprise replies with random options that make an interesting
// animated automaton (see surprise), as JSON {flag: value}.
func handleSurprise(w http.ResponseWriter, r *http.Request) {
	o, err := parseQuery(r)
	if err == nil {
		err = checkLimits(o)
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	select {
	case busy <- struct{}{}:
		defer func() { <-busy }()
	case <-r.Context().Done():
		return
	}
	s := surprise(rand.New(rand.NewSource(time.Now().UnixNano())), *o)
	w.Header().Set("Content-Type", "application/json")
	// Every flag, as the page's fields hold them, but those the page has no
	// field for, and the mask it keeps.
	reply := map[string]string{"format": "gif"}
	cur := s
	fs := newFlagSet(&cur) // resets cur to the defaults…
	cur = s                // …so put the settings back
	fs.VisitAll(func(f *flag.Flag) { reply[f.Name] = f.Value.String() })
	for _, k := range []string{"o", "serve", "list-rules", "mcp", "mask"} {
		delete(reply, k)
	}
	if s.colors != "" { // the page switches to a custom gradient
		delete(reply, "palette")
	} else {
		delete(reply, "colors")
	}
	json.NewEncoder(w).Encode(reply)
}

// handleMutate replies with a lively mutant of the page's rule that looks
// different (see mutateLively), as JSON {"rule": …}.
func handleMutate(w http.ResponseWriter, r *http.Request) {
	o, err := parseQuery(r)
	if err == nil && o.cyclic {
		err = errors.New("mutate works on rules, not the cyclic mode")
	}
	if err == nil {
		_, err = o.automaton()
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	select {
	case busy <- struct{}{}:
		defer func() { <-busy }()
	case <-r.Context().Done():
		return
	}
	rule := mutateLively(rand.New(rand.NewSource(time.Now().UnixNano())), *o)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"rule": rule})
}
