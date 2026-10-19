package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"net/url"
)

// What the web page may ask, served by -serve or run in the browser by the
// WebAssembly build (wasm.go): both go through these, so they share limits.

// Limits for the web page only: a public page must not let one request
// take the whole machine. The CLI has none.
const (
	maxSide   = 500         // cells per side
	maxScale  = 8           // pixels per cell
	maxGens   = 2000        // generations
	maxWork   = 111_000_000 // cells × generations: about 1 s at worst, whatever the rule
	maxPixels = 80_000_000  // GIF: pixels over all frames, held in memory (about 80 MB)
)

// cliOnly are the flags the page may not set. mask is a path: the server
// must never read its own files for a visitor.
var cliOnly = []string{"o", "serve", "list-rules", "mcp", "mask", "site"}

// parseQuery reads the same options as the CLI from a query string
// (?rule=B3/S23&w=100 is -rule=B3/S23 -w=100), plus format=png|gif|zip.
func parseQuery(q url.Values) (*options, error) {
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
		for _, c := range cliOnly {
			if k == c {
				return nil, fmt.Errorf("option %q is CLI only", k)
			}
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
	work := o.w * o.h * o.gens
	if !o.cyclic && IsGrayScott(resolvePreset(o.rule)) {
		// ponytail: -at switching to Gray-Scott mid-run isn't weighed; weigh each keyframe if it ever matters
		work *= 6 // 10 reaction-diffusion steps per generation: about 6× a rule
	}
	switch {
	case o.w > maxSide || o.h > maxSide:
		return fmt.Errorf("grid is at most %d×%d cells here (use the CLI for more)", maxSide, maxSide)
	case o.scale > maxScale:
		return fmt.Errorf("scale is at most %d here", maxScale)
	case o.gens > maxGens:
		return fmt.Errorf("at most %d generations here", maxGens)
	case work > maxWork:
		return errors.New("too much work for the web page: lower the grid size or generations (or use the CLI)")
	case (o.gif || o.zip) && o.w*o.h*o.scale*o.scale*o.gens > maxPixels:
		return errors.New("GIF too large for the web page: lower the grid size, scale or generations (or use the CLI)")
	}
	return nil
}

func (o *options) contentType() string {
	switch {
	case o.gif:
		return "image/gif"
	case o.zip:
		return "application/zip"
	}
	return "image/png"
}

// surpriseReply lists every flag of s as the page's fields hold them, but
// those the page has no field for, and the mask it keeps.
func surpriseReply(s options) map[string]string {
	reply := map[string]string{"format": "gif"}
	cur := s
	fs := newFlagSet(&cur) // resets cur to the defaults…
	cur = s                // …so put the settings back
	fs.VisitAll(func(f *flag.Flag) { reply[f.Name] = f.Value.String() })
	for _, k := range cliOnly {
		delete(reply, k)
	}
	if s.colors != "" { // the page switches to a custom gradient
		delete(reply, "palette")
	} else {
		delete(reply, "colors")
	}
	return reply
}

// checkMutate tells whether o has a rule mutateLively can work on.
func checkMutate(o *options) error {
	if o.cyclic {
		return errors.New("mutate works on rules, not the cyclic mode")
	}
	_, err := o.automaton()
	return err
}
