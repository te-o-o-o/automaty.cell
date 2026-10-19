//go:build !js

package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	var o options
	fs := newFlagSet(&o)
	if err := fs.Parse(os.Args[1:]); err != nil {
		os.Exit(2) // the flag package already printed the error and usage
	}

	switch {
	case o.listRules:
		for _, p := range Presets {
			fmt.Printf("%-18s %s\n", p.Name, p.Rule)
		}
	case o.serve != "":
		fail(1, serve(o.serve))
	case o.site != "":
		if err := writeSite(o.site); err != nil {
			fail(1, err)
		}
	case o.mcp:
		if err := mcp(os.Stdin, os.Stdout); err != nil {
			fail(1, err)
		}
	default:
		o.gif = strings.EqualFold(filepath.Ext(o.out), ".gif")
		o.zip = strings.EqualFold(filepath.Ext(o.out), ".zip")
		var buf bytes.Buffer
		if err := o.generate(&buf); err != nil {
			fail(2, err)
		}
		if err := os.WriteFile(o.out, buf.Bytes(), 0o644); err != nil {
			fail(1, err)
		}
	}
}

func fail(code int, err error) {
	fmt.Fprintln(os.Stderr, "automaty.cell:", err)
	os.Exit(code)
}
