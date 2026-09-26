package main

import (
	"bufio"
	"bytes"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"
)

//go:embed OPTIONS.md
var optionsDoc string

// cliDoc is OPTIONS.md without its web page part, which agents don't use.
var cliDoc, _, _ = strings.Cut(optionsDoc, "## The web page")

// recipes show the render tool's args at work: agents learn best from examples.
const recipes = `## Recipes

- Mirrored organic blobs: -rule=globe -symmetry=4 -density=0.5 -w=300 -h=300 -scale=1 -gens=120 -delay=6 -palette=candy
- Cyclic swirls in custom colours: -cyclic -states=4 -threshold=3 -neighborhood=moore -w=150 -h=150 -gens=120 -colors=ff3b3b,ff3ee0,f7f33c,3ed4f7
- A story in three acts, chaos melting into blobs then rippling out: -cyclic -states=8 -threshold=2 -neighborhood=moore -w=300 -h=300 -scale=1 -gens=150 -palette=cyber -at=50:rule=majority 90:cyclic=true 90:threshold=3 90:states=5
- Two runs in a row, the second carrying on from the first's cells: run A's options with -gens set to both lengths added up, and -at=<A's gens>:rule=<B's rule> <A's gens>:palette=<B's palette>… (only the options -at accepts can change)
- Growth from a single seed: -rule=life -shape=acorn -w=200 -h=200 -gens=300, or -shape=dot with any rule
- Lively rules need a matching density: Majority, Globe and Day & Night want about 0.5, Life about 0.3, Seeds a few percent.
`

// mcp serves the Model Context Protocol over stdio (one JSON-RPC message per
// line), so an agent can render automata for the user with the render tool.
// ponytail: tools only, by hand; take the official Go SDK if prompts,
// resources or the HTTP transport are ever needed.
func mcp(in io.Reader, out io.Writer) error {
	enc := json.NewEncoder(out)
	sc := bufio.NewScanner(in)
	sc.Buffer(make([]byte, 1<<20), 16<<20)
	for sc.Scan() {
		var req struct {
			ID     json.RawMessage `json:"id"`
			Method string          `json:"method"`
			Params json.RawMessage `json:"params"`
		}
		if err := json.Unmarshal(sc.Bytes(), &req); err != nil {
			enc.Encode(map[string]any{"jsonrpc": "2.0", "id": nil, "error": map[string]any{"code": -32700, "message": err.Error()}})
			continue
		}
		if req.ID == nil {
			continue // a notification: nothing to answer
		}
		var result any
		switch req.Method {
		case "initialize":
			var p struct {
				ProtocolVersion string `json:"protocolVersion"`
			}
			json.Unmarshal(req.Params, &p)
			result = map[string]any{
				"protocolVersion": p.ProtocolVersion, // the client's: tools haven't changed across versions
				"capabilities":    map[string]any{"tools": map[string]any{}},
				"serverInfo":      map[string]any{"name": "automaty.cell", "version": "1"},
			}
		case "ping":
			result = map[string]any{}
		case "tools/list":
			result = map[string]any{"tools": []any{renderTool()}}
		case "tools/call":
			var p struct {
				Name      string `json:"name"`
				Arguments struct {
					Args []string `json:"args"`
				} `json:"arguments"`
			}
			json.Unmarshal(req.Params, &p)
			if p.Name != "render" {
				enc.Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID, "error": map[string]any{"code": -32602, "message": "unknown tool " + p.Name}})
				continue
			}
			result = callRender(p.Arguments.Args)
		default:
			enc.Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID, "error": map[string]any{"code": -32601, "message": "unknown method " + req.Method}})
			continue
		}
		if err := enc.Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": result}); err != nil {
			return err
		}
	}
	return sc.Err()
}

func renderTool() map[string]any {
	var presets []string
	for _, p := range Presets {
		presets = append(presets, p.Name+" = "+p.Rule)
	}
	return map[string]any{
		"name": "render",
		"description": "Renders a cellular automaton to a file and returns its path, a contact sheet of 6 generations " +
			"spread over the run (left to right, top to bottom) and how alive the last one is. " +
			"Takes the automaty.cell command-line options, one per item, as -name=value (e.g. [\"-rule=bosco\", \"-gens=120\", \"-palette=candy\", \"-o=/Users/me/bosco.gif\"]). " +
			"-o picks the file and its format (.png, .gif or .zip); give an absolute path, or it lands in the server's working directory. " +
			"Grids are at most 500×500 cells and GIFs about 80 million pixels in all. Look at the sheet and the verdict, and adjust: " +
			"most random rules die out, freeze or boil.\n\n" +
			cliDoc + recipes + "\nPreset rules: " + strings.Join(presets, "; ") + "\nPalettes: " + strings.Join(paletteNames(), ", "),
		"inputSchema": map[string]any{
			"type":       "object",
			"properties": map[string]any{"args": map[string]any{"type": "array", "items": map[string]any{"type": "string"}}},
			"required":   []string{"args"},
		},
	}
}

// callRender runs the render tool: errors go back to the agent as a tool
// result, so it can fix its options.
func callRender(args []string) map[string]any {
	path, sheet, verdict, err := renderFile(args)
	if err != nil {
		return map[string]any{"isError": true, "content": []any{map[string]any{"type": "text", "text": err.Error()}}}
	}
	return map[string]any{"content": []any{
		map[string]any{"type": "text", "text": "wrote " + path + "\n" + verdict},
		map[string]any{"type": "image", "mimeType": "image/png", "data": base64.StdEncoding.EncodeToString(sheet)},
	}}
}

// renderFile writes what args describe, as the CLI would, and returns the
// file's absolute path, a contact sheet of the run and a verdict on it.
func renderFile(args []string) (path string, sheet []byte, verdict string, err error) {
	var o options
	fs := newFlagSet(&o)
	fs.SetOutput(io.Discard)
	if err := fs.Parse(args); err != nil {
		return "", nil, "", err
	}
	if fs.NArg() > 0 {
		return "", nil, "", fmt.Errorf("unexpected %q: want -name=value items", fs.Arg(0))
	}
	if o.serve != "" || o.listRules || o.mcp {
		return "", nil, "", errors.New("-serve, -list-rules and -mcp are CLI only")
	}
	if o.out == "out.png" { // the default: don't write every render over the last one
		o.out = fmt.Sprintf("automaty-%d.png", time.Now().UnixNano())
	}
	o.gif = strings.EqualFold(filepath.Ext(o.out), ".gif")
	o.zip = strings.EqualFold(filepath.Ext(o.out), ".zip")
	if err := checkLimits(&o); err != nil {
		return "", nil, "", err
	}

	var buf bytes.Buffer
	if err := o.generate(&buf); err != nil {
		return "", nil, "", err
	}
	if path, err = filepath.Abs(o.out); err != nil {
		return "", nil, "", err
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		return "", nil, "", err
	}
	sheet, verdict, err = contactSheet(o)
	return path, sheet, verdict, err
}

// contactSheet runs o again and returns a PNG of 6 generations spread over the
// run, 3 by 2, each at most 200 pixels wide, and a verdict measured on the
// last two generations of the real grid.
func contactSheet(o options) ([]byte, string, error) {
	g, step, pal, err := o.setup()
	if err != nil {
		return nil, "", err
	}
	const cols, rows, gap = 3, 2, 4
	f := min(1, 200/float64(max(g.W, g.H)))
	tw, th := max(1, int(float64(g.W)*f)), max(1, int(float64(g.H)*f))
	img := image.NewRGBA(image.Rect(0, 0, cols*tw+(cols-1)*gap, rows*th+(rows-1)*gap))
	draw.Draw(img, img.Bounds(), image.NewUniform(color.Gray{0x80}), image.Point{}, draw.Src)

	var prev []uint8
	var gens []string
	for gen := 0; gen < o.gens; gen++ {
		for tile := 0; tile < cols*rows; tile++ {
			if tile*(o.gens-1)/(cols*rows-1) != gen {
				continue
			}
			ox, oy := tile%cols*(tw+gap), tile/cols*(th+gap)
			p := pal()
			for y := 0; y < th; y++ {
				for x := 0; x < tw; x++ {
					v := g.Cells[y*g.H/th*g.W+x*g.W/tw]
					img.Set(ox+x, oy+y, p[min(int(v), len(p)-1)])
				}
			}
			gens = append(gens, strconv.Itoa(gen))
		}
		if gen == o.gens-2 {
			prev = slices.Clone(g.Cells)
		}
		if gen < o.gens-1 {
			g = step(g)
		}
	}

	// A rule's cell changes when it is born or dies (not as it ages); a
	// cyclic one, whenever its state does.
	live, changed := 0, 0
	for i, v := range g.Cells {
		if v != 0 {
			live++
		}
		if prev != nil && (o.cyclic && v != prev[i] || !o.cyclic && (v != 0) != (prev[i] != 0)) {
			changed++
		}
	}
	n := float64(len(g.Cells))
	verdict := fmt.Sprintf("sheet: generations %s. Last generation: ", strings.Join(gens, ", "))
	if !o.cyclic { // every cyclic state is alive
		verdict += fmt.Sprintf("%.0f%% live, ", 100*float64(live)/n)
	}
	verdict += fmt.Sprintf("%.1f%% of cells changed in the last step", 100*float64(changed)/n)
	switch {
	case !o.cyclic && live == 0:
		verdict += ": died out."
	case prev != nil && changed == 0:
		verdict += ": frozen."
	case !o.cyclic && float64(live)/n > 0.95:
		verdict += ": fills the grid."
	default:
		verdict += "."
	}

	var buf bytes.Buffer
	err = png.Encode(&buf, img)
	return buf.Bytes(), verdict, err
}
