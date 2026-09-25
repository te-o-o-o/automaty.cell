package main

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"time"
)

//go:embed index.html
var indexHTML string

var indexTmpl = template.Must(template.New("index").Parse(indexHTML))

// Limits for the web server only: a public page must not let one request
// take the whole machine. The CLI has none.
const (
	maxSide   = 400           // cells per side
	maxScale  = 8             // pixels per cell
	maxGens   = 2000          // generations
	maxWork   = 1_000_000_000 // cells × generations × neighbourhood size
	maxPixels = 200_000_000   // GIF: pixels over all frames, held in memory
)

// busy lets at most 2 renders run at once; other requests wait their turn.
var busy = make(chan struct{}, 2)

func serve(addr string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		err := indexTmpl.Execute(w, map[string]any{
			"Presets": Presets, "MaxSide": maxSide, "MaxScale": maxScale, "MaxGens": maxGens,
		})
		if err != nil {
			log.Print(err)
		}
	})
	mux.HandleFunc("GET /render", handleRender)

	log.Printf("cellgen: listening on %s", addr)
	srv := &http.Server{Addr: addr, Handler: mux, ReadHeaderTimeout: 10 * time.Second, WriteTimeout: time.Minute}
	return srv.ListenAndServe()
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
