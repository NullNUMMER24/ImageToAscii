package asciiart

import (
	"fmt"
	"image/color"
	"io"
	"strings"
)

// AnsiColorMode selects how RGB colors are encoded as ANSI escapes.
type AnsiColorMode int

const (
	// AnsiTrueColor emits 24-bit "\x1b[38;2;R;G;Bm" escapes. Supported by
	// most modern terminal emulators.
	AnsiTrueColor AnsiColorMode = iota
	// Ansi256 quantizes to the 256-color xterm palette, for terminals
	// without true-color support.
	Ansi256
)

const ansiReset = "\x1b[0m"

// RenderANSI writes the grid as ANSI-colored text. Consecutive cells on a
// row that share a color reuse the same escape sequence to keep output
// compact.
func RenderANSI(w io.Writer, g *Grid, mode AnsiColorMode) error {
	var sb strings.Builder
	for y := 0; y < g.Height; y++ {
		var last color.RGBA
		haveLast := false
		for x := 0; x < g.Width; x++ {
			cell := g.At(x, y)
			if !haveLast || cell.Color != last {
				sb.WriteString(ansiColorEscape(cell.Color, mode))
				last = cell.Color
				haveLast = true
			}
			sb.WriteRune(cell.Char)
		}
		sb.WriteString(ansiReset)
		sb.WriteByte('\n')
	}
	_, err := io.WriteString(w, sb.String())
	return err
}

func ansiColorEscape(c color.RGBA, mode AnsiColorMode) string {
	if mode == Ansi256 {
		return fmt.Sprintf("\x1b[38;5;%dm", rgbTo256(c.R, c.G, c.B))
	}
	return fmt.Sprintf("\x1b[38;2;%d;%d;%dm", c.R, c.G, c.B)
}

// rgbTo256 approximates an RGB color as an xterm 256-color palette index,
// choosing the closer of the 6x6x6 color cube and the 24-step grayscale ramp.
func rgbTo256(r, g, b uint8) int {
	toCube := func(v uint8) int {
		if v < 48 {
			return 0
		}
		if v < 115 {
			return 1
		}
		return (int(v) - 35) / 40
	}
	cubeLevels := [6]int{0, 95, 135, 175, 215, 255}
	cr, cg, cb := toCube(r), toCube(g), toCube(b)
	cubeIdx := 16 + 36*cr + 6*cg + cb
	cubeErr := sq(int(r)-cubeLevels[cr]) + sq(int(g)-cubeLevels[cg]) + sq(int(b)-cubeLevels[cb])

	gray := (int(r) + int(g) + int(b)) / 3
	grayIdx := (gray - 8) / 10
	if grayIdx < 0 {
		grayIdx = 0
	}
	if grayIdx > 23 {
		grayIdx = 23
	}
	grayLevel := 8 + grayIdx*10
	grayErr := sq(int(r)-grayLevel) + sq(int(g)-grayLevel) + sq(int(b)-grayLevel)

	if grayErr < cubeErr {
		return 232 + grayIdx
	}
	return cubeIdx
}

func sq(v int) int { return v * v }
