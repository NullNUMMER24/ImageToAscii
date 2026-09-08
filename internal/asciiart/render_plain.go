package asciiart

import (
	"io"
	"strings"
)

// RenderPlain writes the grid as uncolored text, one line per row.
func RenderPlain(w io.Writer, g *Grid) error {
	var sb strings.Builder
	for y := 0; y < g.Height; y++ {
		for x := 0; x < g.Width; x++ {
			sb.WriteRune(g.At(x, y).Char)
		}
		sb.WriteByte('\n')
	}
	_, err := io.WriteString(w, sb.String())
	return err
}
