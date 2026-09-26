# Options

Every option in one sentence. On the web page each has a field with the same
name; the command line below the image shows the ones you changed.

## The automaton

- `-rule` — the rule to run: B/S like `B3/S23` (Game of Life), Generations like `345/2/4`, Larger than Life like `R5,C0,M1,S34..58,B34..45,NM`, or a preset name.
- `-list-rules` — prints the preset names and their rules, then exits.
- `-cyclic` — runs a cyclic automaton instead, where each cell moves to the next state when enough neighbours are already there.
- `-states` — cyclic: how many states cells go round through (2 to 256).
- `-threshold` — cyclic: how many neighbours in the next state it takes for a cell to move on.
- `-neighborhood` — cyclic: which neighbours count, `moore` (a square) or `vonneumann` (a diamond).
- `-radius` — cyclic: how far the neighbourhood reaches.

## The start

- `-seed` — the random seed: the same seed always gives the same image.
- `-density` — the share of cells alive at the start (B/S, Generations and Larger than Life).
- `-noise` — starts in Perlin noise islands about this many cells wide instead of plain random (`0` turns it off).
- `-symmetry` — makes the start symmetric: mirrors `2`, `4`, `8` or rotations `r2`, `r4` (`8` and `r4` need a square grid).
- `-shape` — keeps the start inside a shape (`dot`, `disc`, `ring`, `cross`, `frame`, `stripes`, `target`, `checker`, `spiral`), dead everywhere else, or starts from a single methuselah (`rpentomino`, `acorn`, `diehard`).
- `-mask` — an image whose light (or opaque) areas are the only place cells may live, from start to end.

## Changes during the run

- `-at` — changes options from a generation on while the cells carry on, like `'80:rule=bosco 150:palette=candy'`: `rule`, `cyclic`, `states`, `threshold`, `radius`, `neighborhood`, `palette`, `colors` and `wrap`.

## The grid

- `-w`, `-h` — the grid's width and height, in cells.
- `-scale` — how many pixels wide each cell is drawn.
- `-wrap` — joins opposite edges so patterns wrap around; `-wrap=false` makes cells beyond the edges dead.

## The output

- `-gens` — how many generations to run, the start included.
- `-o` — the output file: `.png` for the last generation, `.gif` for an animation, `.zip` for one PNG per generation.
- `-delay` — how long each GIF frame shows, in hundredths of a second.
- `-pingpong` — plays the animation forward then backward, for a loop without a jump.

## The colours

- `-palette` — the colour gradient, from young cells to old ones (`-h` lists all of them).
- `-colors` — a custom gradient of 2 to 8 colours, like `1a0033,ff3ea5,ffcc00`, instead of a palette.

## The web page

- `-serve` — serves the web page on an address such as `:8080` instead of writing a file.
- `-mcp` — serves the Model Context Protocol on stdin/stdout, so an agent can render what you describe.
- **? RANDOM** — picks random settings that give a lively, animated result.
- **Mutate** — changes the rule a little (2 to 4 digits, or for Larger than Life its radius, ranges or middle cell), keeping only lively mutants that look different.
- **Copy command** — copies the command line that makes the same image.
- **Copy link** — copies a link to the page with the current settings.
- **Download** — saves the image or animation shown.
- **Frames (ZIP)** — downloads every generation as a PNG, for video mapping tools.
- **Theme** and **Language** — switch between the Candy and Arcade looks, and between English and French.
