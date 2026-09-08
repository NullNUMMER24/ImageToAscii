package asciiart

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

// PNGOptions controls PNG rendering.
type PNGOptions struct {
	// FontSize is the glyph size in points (~pixels at 72 DPI). Defaults
	// to 16 if <= 0. Bigger values produce a higher-resolution image,
	// useful when the output is meant to be used as a desktop background.
	FontSize float64
	// Background fills the canvas behind the glyphs. Defaults to opaque
	// black if the zero value.
	Background color.RGBA
	// CanvasWidth/CanvasHeight, if both > 0, fix the output image to
	// exactly this pixel size (e.g. a monitor resolution) and center the
	// character grid within it. Otherwise the canvas is sized to exactly
	// fit the grid (columns*cellWidth x rows*cellHeight).
	CanvasWidth, CanvasHeight int
}

func loadMonoFace(fontSize float64) (font.Face, error) {
	if fontSize <= 0 {
		fontSize = 16
	}
	ttf, err := opentype.Parse(gomono.TTF)
	if err != nil {
		return nil, fmt.Errorf("parsing embedded font: %w", err)
	}
	face, err := opentype.NewFace(ttf, &opentype.FaceOptions{
		Size:    fontSize,
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		return nil, fmt.Errorf("creating font face: %w", err)
	}
	return face, nil
}

// PNGCellSize returns the pixel dimensions of one monospace character cell
// at the given font size, using the same embedded font as RenderPNG.
// Useful for computing a character grid that will exactly fill a target
// pixel resolution (see -mode png -screen).
func PNGCellSize(fontSize float64) (cellWidth, cellHeight int, err error) {
	face, err := loadMonoFace(fontSize)
	if err != nil {
		return 0, 0, err
	}
	defer face.Close()
	cw, ch := faceCellSize(face)
	return cw, ch, nil
}

func faceCellSize(face font.Face) (cellWidth, cellHeight int) {
	metrics := face.Metrics()
	cellHeight = metrics.Height.Ceil()
	cellWidth = cellHeight / 2
	if advance, ok := face.GlyphAdvance('M'); ok {
		cellWidth = advance.Ceil()
	}
	return cellWidth, cellHeight
}

// RenderPNG rasterizes the grid to a PNG image using an embedded
// monospace font, one glyph per cell, colored per-cell like ANSI/HTML
// output. Note: the embedded font only covers ASCII, so the "blocks"
// ramp preset (which uses Unicode shading characters) won't render
// correctly here — use "simple", "detailed", "minimal", or a custom
// ASCII ramp instead.
func RenderPNG(w io.Writer, g *Grid, opts PNGOptions) error {
	if (opts.Background == color.RGBA{}) {
		opts.Background = color.RGBA{0, 0, 0, 255}
	}

	face, err := loadMonoFace(opts.FontSize)
	if err != nil {
		return err
	}
	defer face.Close()

	cellWidth, cellHeight := faceCellSize(face)

	gridW := cellWidth * g.Width
	gridH := cellHeight * g.Height
	if gridW < 1 {
		gridW = 1
	}
	if gridH < 1 {
		gridH = 1
	}

	imgW, imgH := gridW, gridH
	offsetX, offsetY := 0, 0
	if opts.CanvasWidth > 0 && opts.CanvasHeight > 0 {
		imgW, imgH = opts.CanvasWidth, opts.CanvasHeight
		offsetX = (imgW - gridW) / 2
		offsetY = (imgH - gridH) / 2
	}

	img := image.NewRGBA(image.Rect(0, 0, imgW, imgH))
	draw.Draw(img, img.Bounds(), image.NewUniform(opts.Background), image.Point{}, draw.Src)

	baseline := face.Metrics().Ascent
	for y := 0; y < g.Height; y++ {
		for x := 0; x < g.Width; x++ {
			cell := g.At(x, y)
			if cell.Char == ' ' {
				continue
			}
			d := &font.Drawer{
				Dst:  img,
				Src:  image.NewUniform(cell.Color),
				Face: face,
				Dot: fixed.Point26_6{
					X: fixed.I(offsetX + x*cellWidth),
					Y: fixed.I(offsetY+y*cellHeight) + baseline,
				},
			}
			d.DrawString(string(cell.Char))
		}
	}

	if err := png.Encode(w, img); err != nil {
		return fmt.Errorf("encoding png: %w", err)
	}
	return nil
}
