# automaty.cell

Procedural images (PNG), animations (GIF) and image sequences (ZIP) from 2D
cellular automata, as a command-line tool and a web page.
Go 1.22+, standard library only.

```
go build -o automaty.cell .
```

## Examples

https://github.com/user-attachments/assets/2b9a8740-b0fd-4d23-a28a-fbf450609832


Game of Life, 300 generations as an animated GIF:

```
./automaty.cell -rule B3/S23 -w 200 -h 200 -scale 4 -gens 300 -seed 42 -o life.gif
```

A still of HighLife's 200th generation:

```
./automaty.cell -rule highlife -density 0.35 -gens 200 -o highlife.png
```

Day & Night, slower animation (10/100 s per frame):

```
./automaty.cell -rule daynight -density 0.5 -w 120 -h 120 -scale 3 -gens 150 -delay 10 -o daynight.gif
```

A maze, with dead edges instead of a wrapping grid:

```
./automaty.cell -rule maze -density 0.1 -w 120 -h 120 -gens 300 -wrap=false -o maze.png
```

Seeds exploding from a few cells, in black and white:

```
./automaty.cell -rule seeds -density 0.02 -gens 60 -palette bw -o seeds.gif
```

Any rule in B/S notation (as in Golly) works, not just the presets:

```
./automaty.cell -rule B35678/S5678 -density 0.5 -gens 100 -o custom.png
```

Generations rules (`S/B/C` notation: survival / birth / number of states):

```
./automaty.cell -rule 345/2/4 -gens 200 -palette fire -o starwars.gif
./automaty.cell -rule brian -gens 150 -o brian.png
```

Cyclic automata: a cell in state k moves to k+1 (modulo `-states`) when at
least `-threshold` neighbours are already in state k+1:

```
./automaty.cell -cyclic -states 8 -threshold 5 -radius 3 -gens 400 -scale 3 -palette viridis -o spirals.png
./automaty.cell -cyclic -states 14 -threshold 1 -neighborhood vonneumann -gens 500 -scale 3 -palette ocean -o cca.png
./automaty.cell -cyclic -states 3 -threshold 3 -gens 300 -scale 3 -o 313.png
./automaty.cell -cyclic -states 6 -threshold 2 -radius 2 -neighborhood vonneumann -gens 400 -scale 3 -o maze-spirals.png
```

Symmetric starts (mandalas and kaleidoscopes): the random start is mirrored,
and since the rules treat every direction alike, the pattern stays symmetric:

```
./automaty.cell -rule belzhab -symmetry 8 -gens 150 -seed 5 -palette fire -o mandala.png
./automaty.cell -rule starwars -symmetry 8 -gens 120 -seed 9 -palette viridis -o butterfly.png
```

Rotations (pinwheels), start shapes, and a back-and-forth loop for video
mapping (with the default wrapping edges, images also tile seamlessly):

```
./automaty.cell -rule belzhab -symmetry r4 -seed 5 -gens 150 -palette cyber -o pinwheel.png
./automaty.cell -rule belzhab -shape ring -seed 5 -gens 90 -palette candy -o ring.png
./automaty.cell -cyclic -shape disc -gens 120 -pingpong -o loop.gif
```

Mapping onto a real shape: a mask (a white silhouette on black, or a cut-out
logo) confines the automaton, and a ZIP of PNGs loads in video mapping tools:

```
./automaty.cell -rule belzhab -mask facade.png -density 0.5 -gens 150 -palette cyber -o facade.png
./automaty.cell -cyclic -mask facade.png -gens 40 -pingpong -o facade.zip
```

A custom gradient:

```
./automaty.cell -rule belzhab -gens 150 -colors 1a0033,ff3ea5,ffcc00,3ef3ff -o belzhab.png
```

List the named rules:

```
./automaty.cell -list-rules
```

## Web page

```
./automaty.cell -serve :8080
```

Then open http://localhost:8080: every option has its field, the image
re-renders on every change, and the equivalent command line shows below it.

- **? RANDOM** picks a rule (B/S, Generations or cyclic), colours, a seed, a
  symmetry, a start shape and an animation (GIF: speed, length, edges,
  back-and-forth), leaving out automata that die, freeze or stay noise (each
  candidate is tried on a small grid first).
- **Theme and language**: pickers at the top right. Theme `bonbon` (default)
  or `arcade`, language English (default) or French. Also in the URL:
  `?theme=arcade&lang=fr`.
- **Copy link**: the page's URL holds its settings (and theme and language):
  send it and the other person sees the same creation. A mask, being a local
  file, is not part of it.
- **Frames (ZIP)**: every generation as a PNG, for Resolume, MadMapper,
  TouchDesigner…

The page runs the same engine as the CLI (the same image, pixel for pixel),
with limits to stay usable online: grids up to 500×500, scale 8, 2000
generations, 111 million cells × generations (about 1 s of work at worst),
GIFs of 80 million pixels (about 80 MB), masks of 10 MB, 2 renders at a time.

## Options

| Flag | Default | Meaning |
|---|---|---|
| `-rule` | `B3/S23` | B/S rule (`B36/S23`, `b2/s`…), Generations S/B/C rule (`345/2/4`, `/2/3`…) or preset name |
| `-list-rules` | | Print the presets and exit |
| `-w`, `-h` | `380` | Grid size in cells |
| `-scale` | `2` | Pixels per cell |
| `-gens` | `100` | Number of generations, the start included |
| `-seed` | `1` | Random seed: same seed, same result |
| `-symmetry` | `1` | Symmetric start: `1` (none), mirrors `2`, `4`, `8`, rotations `r2` (half turn), `r4` (quarter turns); `8` and `r4` need a square grid |
| `-shape` | `all` | Start area, dead elsewhere: `all`, `disc`, `ring`, `cross`, `frame`, `stripes` |
| `-density` | `0.3` | Initial share of live cells (unused in cyclic mode: states are uniform) |
| `-wrap` | `true` | Wrapping (toroidal) edges; `-wrap=false` for dead edges |
| `-palette` | `age` | Gradient (`-h` lists them all): `age`, `aurora`, `berry`, `bw`, `candy`, `cherry`, `cyber`, `dusk`, `fire`, `forest`, `gold`, `lagoon`, `lavender`, `lime`, `mint`, `mono`, `neon`, `ocean`, `peach`, `rainbow`, `sunset`, `toxic`, `viridis` |
| `-colors` | | Custom gradient of 2 to 8 colours: `1a0033,ff3ea5,ffcc00` (replaces `-palette`) |
| `-cyclic` | `false` | Cyclic automaton instead of `-rule` |
| `-states` | `14` | Cyclic: number of states (2-256) |
| `-threshold` | `1` | Cyclic: neighbours in the next state needed to advance |
| `-neighborhood` | `vonneumann` | Cyclic: `moore` (square) or `vonneumann` (diamond) |
| `-radius` | `1` | Cyclic: neighbourhood radius |
| `-delay` | `5` | GIF frame delay, in 1/100 s |
| `-pingpong` | `false` | GIF or ZIP played forward then backward: a loop without a jump |
| `-mask` | | PNG or JPEG image: life stays inside its light areas (or opaque ones, if it has transparency), from start to end |
| `-o` | `out.png` | Output file: `.png` (last generation), `.gif` (animation) or `.zip` (one PNG per generation, for Resolume, MadMapper, TouchDesigner…) |
| `-serve` | | Serve the web page on this address (`:8080`) instead of writing a file |

Each state is placed along the gradient:
- B/S: by the cell's age (log scale), from the first colour (just born) to the
  last (alive for long); dead cells take the background colour.
- Generations: alive is the first colour, then the dying states up to the
  last; dead cells take the background.
- Cyclic: states spread evenly over the whole gradient.

## Tests

```
go test ./...
```

## License

MIT, see [LICENSE](LICENSE).
