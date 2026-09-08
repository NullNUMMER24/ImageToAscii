// Package asciiart converts images into ASCII-art character grids and
// renders them as plain text, ANSI-colored terminal output, or colored HTML.
package asciiart

import (
	"image"
	"image/color"
)

// Cell is one character of the output grid.
type Cell struct {
	Char  rune
	Color color.RGBA
}

// Grid is a row-major character grid produced from an image.
type Grid struct {
	Width, Height int
	Cells         []Cell
}

// At returns the cell at (x, y).
func (g *Grid) At(x, y int) Cell {
	return g.Cells[y*g.Width+x]
}

// Options controls how an image is converted to a Grid.
type Options struct {
	// Columns/Rows are the target character-grid size. 0 means "derive
	// from the other dimension and the image aspect ratio".
	Columns, Rows int
	// CharAspect corrects for terminal glyphs being taller than they are
	// wide. ~0.55 keeps square subjects looking square. Defaults to 0.55
	// if <= 0.
	CharAspect float64
	// Scale multiplies the resulting character-grid size (whether it came
	// from Columns/Rows or the aspect-derived default), e.g. 0.5 to halve
	// it. Defaults to 1 if <= 0.
	Scale float64
	// Ramp is the ordered (sparse -> dense) set of characters used to
	// represent luminance. Defaults to the "simple" preset if empty.
	Ramp []rune
	// Palette maps a cell's luminance/original color to a render color.
	// Defaults to "original" (pixel color unchanged) if nil.
	Palette PaletteFunc
	// Filters are applied, in order, to the full-resolution image before
	// it is downscaled to the character grid.
	Filters []Filter
	// PostFilters are applied, in order, to the already-downscaled
	// character-grid image (e.g. per-cell speckle noise, which would be
	// averaged away if applied before downscaling).
	PostFilters []Filter
}

func (o Options) withDefaults() Options {
	if o.CharAspect <= 0 {
		o.CharAspect = 0.55
	}
	if o.Scale <= 0 {
		o.Scale = 1
	}
	if len(o.Ramp) == 0 {
		o.Ramp = rampPresets["simple"]
	}
	if o.Palette == nil {
		o.Palette = func(_ float64, original color.RGBA) color.RGBA { return original }
	}
	return o
}

// Convert applies the configured filters, resizes to the target character
// grid, and maps each resulting cell to a character + color.
func Convert(img image.Image, opts Options) *Grid {
	opts = opts.withDefaults()

	full := toRGBA(img)
	for _, f := range opts.Filters {
		f(full)
	}

	b := full.Bounds()
	cols, rows := TargetDimensions(b.Dx(), b.Dy(), opts.Columns, opts.Rows, opts.CharAspect, opts.Scale)

	small := Resize(full, cols, rows)
	for _, f := range opts.PostFilters {
		f(small)
	}

	grid := &Grid{Width: cols, Height: rows, Cells: make([]Cell, cols*rows)}
	rampLen := len(opts.Ramp)
	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			px := small.RGBAAt(x, y)
			lum := Luminance(px.R, px.G, px.B)
			idx := int(lum * float64(rampLen-1))
			if idx < 0 {
				idx = 0
			}
			if idx >= rampLen {
				idx = rampLen - 1
			}
			grid.Cells[y*cols+x] = Cell{
				Char:  opts.Ramp[idx],
				Color: opts.Palette(lum, px),
			}
		}
	}
	return grid
}
