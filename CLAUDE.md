# automaty.cell

Cellular automata → PNG, GIF and ZIP image sequences, from the CLI, a web page
(`-serve`, or static with the engine in WebAssembly: `-site`, GitHub Pages) or
an MCP server (`-mcp`). Go, stdlib only, one `main` package.
Decisions and traps in detail: `docs/notes.md`.

## Layout

| File | Role | Rule |
|---|---|---|
| `cli.go` | `main` (native only) | |
| `main.go` | `options`, flags, `setup` (start grid + step + palette), `generate`, `-at` keyframes, `-stop` | One `options` struct and one `newFlagSet` feed the CLI, the web query and the MCP args: add an option there only. |
| `grid.go` | `Grid`, rules (B/S, Generations, Larger than Life, cyclic, Gray-Scott), starts (random, Perlin, symmetries, shapes) | Cells are one `uint8` each; anything richer (Gray-Scott's floats) lives in the step closure. |
| `render.go` | Paletted rendering, masks | Index 0 is the background. |
| `palette.go` | Named gradients, `-colors` | |
| `surprise.go` | RANDOM, Mutate, the interest test | Every new rule family needs a `mutate` branch. |
| `web.go` | Limits, query parsing, replies: shared by `-serve` and the wasm build | Any costlier family must weigh in `checkLimits`. |
| `server.go` | Web handlers, `-site` | Native only (`!js`), like `mcp.go`, `cli.go` and their tests. |
| `wasm.go`, `worker.js` | The engine in the browser: `automaty.render/surprise/mutate` in a Web Worker | `js && wasm` only. |
| `index.html`, `i18n.go` | Web page (embedded), English and French texts | Every text in both languages (`TestTexts`). |
| `mcp.go` | MCP over stdio, by hand; embeds `OPTIONS.md` as the tool's doc | |
| `OPTIONS.md` | Every option in one sentence | Update it with every flag. |

## Commands

```
go vet ./... && go test ./...        # before every commit
go build -o automaty.cell .
./automaty.cell -serve :8080         # web page
kill $(lsof -nP -t -iTCP:8080 -sTCP:LISTEN)   # stop it, never a bare lsof -i :8080
GOOS=js GOARCH=wasm go vet .         # the wasm build, tests included
# static page, as pages.yml builds it:
go run . -site site && GOOS=js GOARCH=wasm go build -trimpath -ldflags=-s -o site/automaty.wasm . \
  && cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" site/ && python3 -m http.server -d site 8090
```

Release: bump the README badge (both READMEs), date the CHANGELOG section,
`docs: bump README to vX.Y.Z`, then `git tag vX.Y.Z && git push origin vX.Y.Z`.
`release.yml` refuses a tag that doesn't match the README badge.

## Traps

- A step's returned grid is only valid until the call after next (`swapping`):
  clone it to keep it.
- Gray-Scott only grows from islands: it always starts from Perlin noise.
- `-stop` compares bytes: B/S ages delay it up to 255 generations.
- Don't paste file paths into JSON strings in tests: Windows backslashes.
- Docs come in pairs: `README.md` / `README.fr.md`, `docs/notes.md` / `docs/notes.fr.md`.
