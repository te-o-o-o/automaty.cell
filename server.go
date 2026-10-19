//go:build !js

package main

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"html/template"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

//go:embed index.html
var indexHTML string

//go:embed worker.js
var workerJS []byte

var indexTmpl = template.Must(template.New("index").Parse(indexHTML))

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

// page writes the page in a theme (bonbon or arcade) and a language (en or
// fr). static makes it run the WebAssembly engine instead of calling the server.
func page(w io.Writer, theme, lang string, static bool) error {
	type palette struct{ Name, CSS string }
	var palettes []palette
	for _, name := range paletteNames() {
		palettes = append(palettes, palette{name, gradients[name].css()})
	}
	return indexTmpl.Execute(w, map[string]any{
		"Presets": Presets, "Palettes": palettes, "MaxSide": maxSide, "MaxScale": maxScale, "MaxGens": maxGens,
		"Theme": theme, "Lang": lang, "T": texts[lang], "Lively": livelyKeys(), "Static": static,
	})
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
	if err := page(w, theme, lang, false); err != nil {
		log.Print(err)
	}
}

// writeSite writes the static page, one file per language, and its worker
// into dir. The WebAssembly engine and Go's wasm_exec.js go next to them
// (see .github/workflows/pages.yml); the theme is set in the browser.
func writeSite(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	for file, lang := range map[string]string{"index.html": "en", "fr.html": "fr"} {
		var buf bytes.Buffer
		if err := page(&buf, "bonbon", lang, true); err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(dir, file), buf.Bytes(), 0o644); err != nil {
			return err
		}
	}
	return os.WriteFile(filepath.Join(dir, "worker.js"), workerJS, 0o644)
}

// handleRender replies with the image the query asks for (see parseQuery).
func handleRender(w http.ResponseWriter, r *http.Request) {
	o, err := parseQuery(r.URL.Query())
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
	w.Header().Set("Content-Type", o.contentType())
	if o.zip {
		w.Header().Set("Content-Disposition", `attachment; filename="automaty.cell.zip"`)
	}
	w.Write(buf.Bytes())
}

// handleSurprise replies with random options that make an interesting
// animated automaton (see surprise), as JSON {flag: value}.
func handleSurprise(w http.ResponseWriter, r *http.Request) {
	o, err := parseQuery(r.URL.Query())
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
	json.NewEncoder(w).Encode(surpriseReply(s))
}

// handleMutate replies with a lively mutant of the page's rule that looks
// different (see mutateLively), as JSON {"rule": …}.
func handleMutate(w http.ResponseWriter, r *http.Request) {
	o, err := parseQuery(r.URL.Query())
	if err == nil {
		err = checkMutate(o)
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
