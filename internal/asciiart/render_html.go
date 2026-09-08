package asciiart

import (
	"fmt"
	"html"
	"image/color"
	"io"
	"strings"
)

// HTMLOptions controls HTML rendering.
type HTMLOptions struct {
	// Fragment emits only the <pre>...</pre> block instead of a full document.
	Fragment bool
	// Background is the page/pre background color, e.g. "#000000".
	Background string
	// Title is used for the document <title> when Fragment is false.
	Title string
}

// RenderHTML writes the grid as HTML, using one <span> per run of cells
// that share a color to keep markup compact.
func RenderHTML(w io.Writer, g *Grid, opts HTMLOptions) error {
	if opts.Background == "" {
		opts.Background = "#000000"
	}
	if opts.Title == "" {
		opts.Title = "ASCII Art"
	}

	var body strings.Builder
	body.WriteString(fmt.Sprintf(`<pre style="background:%s;color:#ffffff;font-family:'Courier New',monospace;line-height:1;white-space:pre;">`, html.EscapeString(opts.Background)))
	body.WriteByte('\n')

	for y := 0; y < g.Height; y++ {
		var run strings.Builder
		var runColor color.RGBA
		haveRun := false
		flush := func() {
			if !haveRun || run.Len() == 0 {
				return
			}
			fmt.Fprintf(&body, `<span style="color:#%02x%02x%02x">%s</span>`, runColor.R, runColor.G, runColor.B, run.String())
			run.Reset()
		}
		for x := 0; x < g.Width; x++ {
			cell := g.At(x, y)
			if !haveRun || cell.Color != runColor {
				flush()
				runColor = cell.Color
				haveRun = true
			}
			run.WriteString(html.EscapeString(string(cell.Char)))
		}
		flush()
		body.WriteByte('\n')
	}
	body.WriteString("</pre>\n")

	var out string
	if opts.Fragment {
		out = body.String()
	} else {
		out = fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<title>%s</title>
<style>body{background:%s;margin:0;padding:1rem;}</style>
</head>
<body>
%s</body>
</html>
`, html.EscapeString(opts.Title), html.EscapeString(opts.Background), body.String())
	}

	_, err := io.WriteString(w, out)
	return err
}
