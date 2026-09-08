# ascii-converter

Convert an image into ASCII art. Output as plain text, ANSI-colored text
(for bash/zsh terminals), colored HTML, a standalone bash script, or a
rasterized PNG. Supports PNG, JPEG, and GIF input.

Rewrote this from python to go just for fun. Needed it for my new wallpaper.

## Build

```bash
go build -o ascii-converter .
```

## Usage

```bash
ascii-converter <image> [flags]
```

### Output modes (`-mode`)

- `plain` — uncolored ASCII text
- `ansi` — ASCII with ANSI escape codes, printed straight to stdout for
  bash/zsh terminals (default)
- `html` — ASCII with inline `<span style="color:...">` for the web
- `bash` — a standalone, runnable/sourceable `.sh` script that prints the
  ANSI art, for embedding into other scripts (see below)
- `png` — the art rasterized to an actual image, e.g. for use as a
  desktop background (see below)

```bash
ascii-converter photo.jpg -mode plain
ascii-converter photo.jpg -mode ansi
ascii-converter photo.jpg -mode html -o photo.html
ascii-converter photo.jpg -mode bash -o banner.sh
ascii-converter photo.jpg -mode png -o photo_ascii.png
```

ANSI output (used by both `ansi` and `bash` modes) defaults to 24-bit
truecolor; pass `-ansi-color-mode 256` for terminals that only support the
256-color palette.

### Size (`-width`, `-height`, `-scale`)

Give one of `-width`/`-height` and the other is derived from the image's
aspect ratio (with a correction factor for terminal glyphs being taller
than wide, tunable via `-aspect`, default `0.55`). Give both to set an
exact grid size. Source images are almost always far higher-resolution
than any character grid needs, so there's no separate "source scale" —
`-width`/`-height` (or the 100-column default) already do that downscale.

`-scale` is a quick multiplier on top of whatever size you'd otherwise
get, for "same as usual, just smaller/bigger" without doing the math:

```bash
ascii-converter photo.jpg -width 120
ascii-converter photo.jpg -width 80 -height 40
ascii-converter photo.jpg -scale 0.5   # half the default 100-column size
ascii-converter photo.jpg -width 200 -scale 0.5   # -> 100 columns
```

### Character ramp (`-ramp`)

Controls which characters represent luminance, from sparse to dense.
Presets: `simple` (default), `detailed`, `blocks`, `minimal`. Or pass any
literal string of 2+ characters ordered dark/sparse -> light/dense.

```bash
ascii-converter photo.jpg -ramp blocks
ascii-converter photo.jpg -ramp " .oO@"
```

### Color palette (`-palette`)

Recolors the output independently of the character ramp:

- `original` (default) — actual pixel colors
- `grayscale`
- `matrix`, `amber`, `ice`, `fire`, `sepia`, `mono` — stylized gradients
  mapped by pixel luminance
- `custom:#RRGGBB,#RRGGBB,...` — your own gradient (2+ stops)

```bash
ascii-converter photo.jpg -mode ansi -palette matrix
ascii-converter photo.jpg -mode html -palette "custom:#1a0033,#ff2079,#ffffff"
```

Ignored in `plain` mode, since plain text carries no color.

### Filters (`-filters`)

Comma-separated pipeline applied in order: `grayscale`, `invert`, `sepia`,
`edge`, `brightness=<factor>`, `contrast=<factor>`, `blur=<radius>`,
`threshold=<cutoff 0-1>`, `posterize=<levels>`, `noise=<amount 0-1>`,
`gamma=<value>`.

Everything except `noise` runs on the full-resolution source image before
it's downscaled to the character grid (so `threshold`/`posterize` edges
come out naturally antialiased into the ramp). `noise` runs *after*
downscaling, one speckle per output character, since per-source-pixel
noise would otherwise be averaged away by the resize.

```bash
ascii-converter photo.jpg -filters "grayscale,contrast=1.3,blur=1"
ascii-converter photo.jpg -filters "sepia,brightness=1.1"
ascii-converter photo.jpg -filters "edge"

# Hard-edged silhouette (crisp white fill, black background)
ascii-converter photo.jpg -mode plain -filters "grayscale,threshold=0.5"

# Posterized color bands
ascii-converter photo.jpg -mode html -filters "posterize=3"

# Scattered colorful speckles, like a glitch/dither texture
ascii-converter photo.jpg -mode ansi -filters "noise=0.1"

# Lift shadow detail on a dark image (default gamma 1.8)
ascii-converter photo.jpg -mode png -filters "gamma=1.8"
```

Dark pixels already pick sparse/thin ramp characters (that's the whole
point of the ramp), so a dark *color* on top of a sparse *glyph*
compounds into shadows disappearing almost entirely against a dark
background — especially noticeable in `png` mode. `gamma` (values > 1)
lifts shadows and midtones without blowing out highlights the way a flat
`brightness` multiply would; reach for it whenever a result looks "too
dark" or is missing detail in darker regions.

### HTML-specific flags

- `-html-fragment` — emit just the `<pre>` block instead of a full document
- `-html-bg` — background color (default `#000000`)

### Bash-specific flags (`-mode bash`)

Wraps the ANSI art in a quoted heredoc inside a shell function, so escape
codes pass through literally. `-o` output files are made executable
automatically.

- Run it directly and it prints immediately: `./banner.sh`
- `source` it from another script and it just defines the function, ready
  to call whenever you want (e.g. as a startup banner):

  ```bash
  source banner.sh
  echo "Deploying..."
  show_ascii_art
  ```

Flags:

- `-bash-func` — the function name to wrap the art in (default
  `show_ascii_art`)
- `-bash-fragment` — emit just the `cat <<'EOF' ... EOF` snippet, to paste
  directly into an existing script instead of getting a full runnable file

```bash
ascii-converter photo.jpg -mode bash -palette matrix -o banner.sh
./banner.sh
ascii-converter photo.jpg -mode bash -bash-func print_logo -bash-fragment
```

### PNG-specific flags (`-mode png`)

Rasterizes the character grid to an actual image using an embedded
monospace font — one real, anti-aliased glyph per cell, colored like
ANSI/HTML output. Useful for saving the art as a shareable image or a
desktop background.

- `-png-font-size` — glyph size in points, ~pixels at 72 DPI (default
  `16`). This is what actually controls the output image's resolution —
  combine with `-width`/`-height`/`-scale` (which control the character
  grid) to hit a target pixel size, e.g. a 200-column grid at 20pt lands
  around 2000px wide.
- `-png-bg` — background color (default `#000000`)
- `-screen` — fit the PNG to an exact monitor resolution instead of
  sizing it off the character grid (see below)
- `-screen-fit` — `cover` (default) or `contain`, how `-screen` fits the
  image (see below)

The embedded font only covers ASCII, so the `blocks` ramp preset (which
uses Unicode shading characters `░▒▓█`) won't render distinct shading in
PNG mode — use `simple`, `detailed`, `minimal`, or a custom ASCII ramp
instead.

```bash
# Compact PNG
ascii-converter photo.jpg -mode png -width 80 -palette matrix -o art.png

# Larger, wallpaper-sized PNG
ascii-converter photo.jpg -mode png -width 240 -png-font-size 24 -palette fire -o wallpaper.png
```

#### Wallpapers (`-screen`, `-screen-fit`)

Source photos are almost always a completely different shape than a
monitor, so just cranking up `-width` leaves you with the wrong aspect
ratio. `-screen` instead targets an exact resolution directly: it works
out the character grid and font metrics needed to fill it exactly. The
output PNG is pinned to that exact pixel size either way. It overrides
`-width`/`-height`/`-scale`/`-aspect`.

Accepts a preset (`720p`, `1080p`, `1440p`, `4k`, `5k`, `ultrawide`) or an
explicit `WIDTHxHEIGHT`:

```bash
ascii-converter photo.jpg -mode png -screen 1080p -palette matrix -o wallpaper.png
ascii-converter photo.jpg -mode png -screen 4k -palette fire -o wallpaper.png
ascii-converter photo.jpg -mode png -screen 3440x1440 -palette ice -o wallpaper.png
```

`-screen-fit` controls how the image is fit into that resolution when its
aspect ratio doesn't match the screen's:

- `cover` (default) — center-crops the image to the screen's aspect ratio
  first (like CSS `background-size: cover`), so it fills the frame
  edge-to-edge with no stretching, but the parts that don't fit are cut off.
- `contain` — fits the entire image with no cropping, padding the
  leftover space (left/right or top/bottom, whichever doesn't match) with
  `-png-bg` — nothing gets cut off, at the cost of bars on the sides.

```bash
ascii-converter photo.jpg -mode png -screen ultrawide -screen-fit contain -o wallpaper.png
```

### Discover presets

```bash
ascii-converter -list-ramps
ascii-converter -list-palettes
ascii-converter -list-filters
```

## Examples

```bash
# Quick look in the terminal
ascii-converter photo.jpg

# Matrix-style HTML for a webpage, wide
ascii-converter photo.jpg -mode html -width 160 -palette matrix -o out.html

# Moody, blurred, sepia-toned plain art
ascii-converter photo.jpg -mode plain -filters "sepia,blur=2" -ramp detailed

# A banner script other scripts can source and call
ascii-converter photo.jpg -mode bash -width 80 -palette matrix -o banner.sh

# A proper desktop wallpaper, cropped and sized to fit exactly
ascii-converter huge_photo.jpg -mode png -screen 1440p -palette ice -o wallpaper.png
```
