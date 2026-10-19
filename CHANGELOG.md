# Changelog

## Unreleased

### Added

- The web page runs in your browser, no install: [te-o-o-o.github.io/automaty.cell](https://te-o-o-o.github.io/automaty.cell/). The engine is compiled to WebAssembly and renders in a background worker; shared links work there as they do with `-serve`.
- `-site` writes that static page into a directory.

### Changed

- **Mutate** is a big button next to **? RANDOM**.
- **Mutate** goes further: more changes per mutant, bigger Gray-Scott steps, and B/S rules sometimes turn into Generations.
- Release notes explain how to run the unsigned binaries on macOS, Windows and Linux.

## 0.5.0 (2026-10-17)

### Added

- `-stop` ends an animation as soon as a generation repeats an earlier one (frozen or looping), never before the last `-at` change. On the web page it is a checkbox, and **? RANDOM** always ticks it.

## 0.4.1 (2026-10-17)

### Changed

- README: a Gray-Scott example, mitosis turning to reef halfway through.

## 0.4.0 (2026-10-17)

### Added

- Gray-Scott reaction-diffusion rules, written `F0.055,K0.062` (feed and kill rates), with five presets: `mitosis`, `reef`, `fingerprint`, `worms` and `solitons`. They always start from Perlin islands.
- **Mutate** nudges the feed or kill rate of a Gray-Scott rule.
- Top/bottom mirror, one or both diagonals, glide reflection and 2×2, 3×3 or 4×4 tiles for `-symmetry`, on the web page too.

## 0.3.1 (2026-09-26)

### Added

- Binaries for macOS, Linux and Windows on each release.
- `go install github.com/te-o-o-o/automaty.cell@latest`.

## 0.3.0 (2026-09-26)

### Fixed

- The folded epilepsy warning stays in place instead of jumping to the centre.

## 0.2.0 (2026-09-26)

### Added

- `-mcp` serves the Model Context Protocol, so an agent can render automata: its `render` tool takes the command-line options and shows the agent a contact sheet of the run with a verdict (died out, frozen, fills the grid…).

## 0.1.0 (2026-09-26)

### Added

- B/S rules (Game of Life and co), Generations, Larger than Life and cyclic automata, with named presets (`-list-rules`).
- Starts: random or Perlin noise islands, mirror and rotation symmetries, shapes (disc, ring, spiral…), methuselahs (R-pentomino, Acorn, Diehard) and image masks.
- `-at` changes the rule, palette or edges during a run while the cells carry on.
- Output as a PNG of the last generation, an animated GIF (with back-and-forth loops) or a ZIP of one PNG per generation for video mapping tools.
- Gradient palettes, or your own colours.
- A web page (`-serve`) with a field per option, **? RANDOM** for lively settings, **Mutate** for neighbouring rules, and links that keep the settings.
