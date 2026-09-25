package main

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"
)

//go:embed index.html
var indexHTML string

var indexTmpl = template.Must(template.New("index").Parse(indexHTML))

// Limits for the web server only: a public page must not let one request
// take the whole machine. The CLI has none.
const (
	maxSide   = 500           // cells per side
	maxScale  = 8             // pixels per cell
	maxGens   = 2000          // generations
	maxWork   = 1_000_000_000 // cells × generations × neighbourhood size
	maxPixels = 200_000_000   // GIF: pixels over all frames, held in memory
)

// busy lets at most 2 renders run at once; other requests wait their turn.
var busy = make(chan struct{}, 2)

func serve(addr string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", indexHandler)
	mux.HandleFunc("GET /render", handleRender)
	mux.HandleFunc("GET /surprise", handleSurprise)

	log.Printf("cellgen: listening on %s", addr)
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
	jsText := map[string]string{} // the words the page's script writes itself
	for k, v := range texts[lang] {
		if strings.HasPrefix(k, "js") {
			jsText[k] = v
		}
	}
	type palette struct{ Name, CSS string }
	var palettes []palette
	for _, name := range paletteNames() {
		palettes = append(palettes, palette{name, gradients[name].css()})
	}
	err := indexTmpl.Execute(w, map[string]any{
		"Presets": Presets, "Palettes": palettes, "MaxSide": maxSide, "MaxScale": maxScale, "MaxGens": maxGens,
		"Theme": theme, "Lang": lang, "T": texts[lang], "JSText": jsText,
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
	w.Header().Set("Content-Type", "image/png")
	if o.gif {
		w.Header().Set("Content-Type", "image/gif")
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
	default:
		return nil, fmt.Errorf("unknown format %q (want png or gif)", q.Get("format"))
	}
	q.Del("format")

	var args []string
	for k, vs := range q {
		if k == "o" || k == "serve" || k == "list-rules" {
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
	n := 3 // neighbourhood side
	if o.cyclic {
		n = 2*o.radius + 1
	}
	switch {
	case o.w > maxSide || o.h > maxSide:
		return fmt.Errorf("grid is at most %d×%d cells here (use the CLI for more)", maxSide, maxSide)
	case o.scale > maxScale:
		return fmt.Errorf("scale is at most %d here", maxScale)
	case o.gens > maxGens:
		return fmt.Errorf("at most %d generations here", maxGens)
	case o.radius > maxSide: // before the product below, which could overflow
		return fmt.Errorf("radius is at most %d here", maxSide)
	case o.w*o.h*o.gens*n*n > maxWork:
		return errors.New("too much work for the web page: lower the grid size, generations or radius (or use the CLI)")
	case o.gif && o.w*o.h*o.scale*o.scale*o.gens > maxPixels:
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
	json.NewEncoder(w).Encode(map[string]string{
		"cyclic":       strconv.FormatBool(s.cyclic),
		"rule":         s.rule,
		"density":      strconv.FormatFloat(s.density, 'f', -1, 64),
		"states":       strconv.Itoa(s.states),
		"threshold":    strconv.Itoa(s.threshold),
		"radius":       strconv.Itoa(s.radius),
		"neighborhood": s.neighborhood,
		"palette":      s.palette,
		"seed":         strconv.FormatInt(s.seed, 10),
		"symmetry":     strconv.Itoa(s.symmetry),
		"format":       "gif",
		"gens":         strconv.Itoa(s.gens),
		"delay":        strconv.Itoa(s.delay),
		"wrap":         strconv.FormatBool(s.wrap),
	})
}
