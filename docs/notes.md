# Notes

Decisions, dead ends and traps, and what is verified on which OS.
[Français](notes.fr.md)

## Verified, per OS

| | macOS | Linux | Windows |
|---|---|---|---|
| `go vet` and `go test` | CI (`macos-14`) and by hand | CI (`ubuntu-22.04`) | CI (`windows-latest`) |
| CLI renders (PNG, GIF, ZIP) | PNG and GIF by hand, ZIP in tests | not run | not run |
| Web page (`-serve`) | by hand (requests and the page) | not run | not run |
| MCP (`-mcp`) | tests only | tests only | tests only |
| Release binaries | built by CI, not recorded as run | built by CI only | built by CI only |

The tests render real PNGs, GIFs and ZIPs and drive the web handlers and the
MCP server, so passing on all three OSes covers more than compiling. Running the
binaries there by hand is still left to do.

## Grid and steppers

- One `uint8` per cell, whatever the family: the age for B/S and two-state
  Larger than Life (it saturates at 255), the state for Generations and cyclic
  automata, V rounded to 1-255 for Gray-Scott. Rendering, palettes, masks,
  symmetries and `-at` only ever see that byte.
- `swapping` alternates two grids instead of allocating one per generation:
  the grid a step returns is only valid until the call after next. `generate`,
  `activity` and the contact sheet rely on it; keep a copy (`slices.Clone`) to
  hold a generation longer.
- `-at` builds a new automaton per change and `convert` rewrites the cells when
  the two hold them differently (ages, states, cyclic bands).

## Gray-Scott

- Each cell holds two floats (U and V) kept inside the step function's
  closure; `Cells` only shows V. A cell changed from outside since the last
  step (the starting grid, a mask, an `-at` change) restarts from its byte: a
  live one with U = V = 0.5, a dead one empty.
- **It must start from islands.** From plain random cells, at any density, the
  cells blur into an even mix within a few steps and the whole grid fades out
  at once. Measured over the five presets, three seeds and six densities:
  Perlin islands grew patterns every time from density 0.1, plain random
  hardly ever. So a Gray-Scott rule starts from Perlin islands 30 cells wide
  when `-noise` is 0.
- The seed value matters as much: U = 0.5, V = 0.25 (the textbook square seed)
  faded out from islands for worms at every density, and for reef below 0.5;
  U = V = 0.5 worked for all five.
- Neighbours are summed in pairs that mirrors and quarter turns only swap, so
  a symmetric start stays exactly symmetric despite floating point
  (`TestGrayScott` checks it).
- 10 reaction-diffusion steps per generation, about 6× the cost of a rule.
  The web limits weigh Gray-Scott 6×, but not a switch to it through `-at`.
- With a mask, the dead cells around it act as a source of U: patterns grow a
  bright rim along the mask's edge.

## `-stop`

- It compares the whole grid byte for byte (a 64-bit hash per generation). A
  B/S pattern that looks frozen still ages, so it only stops once every live
  cell reaches 255, up to 255 generations later. Gray-Scott only shows V
  rounded, so it stops when it looks still, even if it drifts below that.
- A repeat before the last `-at` change doesn't count: the change may wake it.

## Random and Mutate

- `cyclicLively` was measured: thresholds that keep cyclic automata moving,
  per neighbourhood, radius and number of states.
- RANDOM draws the family once, then retries the rest: retrying the family too
  would favour cyclic automata, which pass the interest test most often.
- Mutate weighs neighbour counts by how often they occur: flipping 7 or 8
  rarely changes anything.
- Mutants looked too much alike with 2 to 4 changes and a 0.3 difference
  threshold (`minLook`): now 2 to 6 changes and 0.5, at the same speed (under
  half a second a click). Gray-Scott steps were far too small (0.001 to 0.003):
  now up to 0.012 on feed and 0.004 on kill, which is the more sensitive one.
  Turning B/S into Generations one change in six made nearly every Life mutant
  Generations, as changes add up: one in twelve.

## Web page and MCP

- Web limits (`server.go`) stop one request from taking the whole machine:
  cells × generations, and the pixels of all GIF frames held in memory.
- `-mask` is a path on the CLI only: the web page uploads the image, so the
  server never reads its own files for a visitor.
- The MCP server is written by hand (tools only). Take the official Go SDK if
  prompts, resources or the HTTP transport are ever needed.

## Traps

- **Restarting the server**: kill only the process listening on the port,
  `kill $(lsof -nP -t -iTCP:8080 -sTCP:LISTEN)`. A bare `lsof -t -i :8080`
  also lists the browser connected to it, and kills it.
- **Windows paths in JSON**: a path pasted into a JSON string breaks on its
  backslashes; encode it with `json.Marshal` (`mcp_test.go`).
- **Release builds use the latest Go**, not go.mod's minimum: macOS refuses
  to run binaries from Go 1.22's linker ("missing LC_UUID").
