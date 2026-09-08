// Command ascii-converter turns an image into ASCII art, rendered as
// plain text, ANSI-colored terminal output, or colored HTML.
package main

import (
	"flag"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"strings"

	"ascii-converter/internal/asciiart"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	fs := flag.NewFlagSet("ascii-converter", flag.ContinueOnError)
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), `ascii-converter converts an image to ASCII art.

Usage:
  ascii-converter -i <image> [flags]
  ascii-converter <image> [flags]

Flags:
`)
		fs.PrintDefaults()
		fmt.Fprintf(fs.Output(), `
Examples:
  ascii-converter photo.jpg
  ascii-converter -i photo.jpg -mode html -o photo.html -palette matrix
  ascii-converter -i photo.jpg -mode ansi -width 120 -filters "grayscale,contrast=1.3,blur=1"
  ascii-converter -i photo.jpg -palette "custom:#001100,#00ff44" -ramp blocks
  ascii-converter -i photo.jpg -mode bash -o banner.sh && ./banner.sh
  ascii-converter -i photo.jpg -mode png -scale 0.25 -o small.png
  ascii-converter -i photo.jpg -mode png -width 300 -png-font-size 24 -o wallpaper.png
  ascii-converter -i photo.jpg -mode png -screen 1920x1080 -palette matrix -o wallpaper.png
  ascii-converter -i photo.jpg -mode png -screen ultrawide -screen-fit contain -o wallpaper.png
  ascii-converter -i photo.jpg -filters "gamma=1.8" -mode png -screen 4k -o wallpaper.png
`)
	}

	var (
		input          string
		output         string
		mode           string
		width          int
		height         int
		scale          float64
		aspect         float64
		rampSpec       string
		paletteSpec    string
		filterSpec     string
		ansiColorMode  string
		htmlFragment   bool
		htmlBackground string
		bashFragment   bool
		bashFuncName   string
		pngFontSize    float64
		pngBackground  string
		pngScreen      string
		pngScreenFit   string
		listRamps      bool
		listPalettes   bool
		listFilters    bool
	)

	fs.StringVar(&input, "input", "", "input image path (png, jpeg, gif)")
	fs.StringVar(&input, "i", "", "shorthand for -input")
	fs.StringVar(&output, "output", "", "output file path (default: stdout)")
	fs.StringVar(&output, "o", "", "shorthand for -output")
	fs.StringVar(&mode, "mode", "ansi", "output mode: plain | ansi | html | bash | png")
	fs.IntVar(&width, "width", 0, "output width in characters (0 = auto)")
	fs.IntVar(&height, "height", 0, "output height in characters (0 = auto)")
	fs.Float64Var(&scale, "scale", 1, "multiplies the resulting character-grid size, e.g. 0.5 to halve it (applies on top of -width/-height or the auto default)")
	fs.Float64Var(&aspect, "aspect", 0.55, "character aspect-ratio correction factor")
	fs.StringVar(&rampSpec, "ramp", "simple", "character ramp preset ("+joinNames(asciiart.RampNames())+") or a literal dark->light string")
	fs.StringVar(&paletteSpec, "palette", "original", "color palette preset ("+joinNames(asciiart.PaletteNames())+") or custom:#hex,#hex,...")
	fs.StringVar(&filterSpec, "filters", "", "comma-separated filter pipeline, e.g. grayscale,brightness=1.2,blur=2")
	fs.StringVar(&ansiColorMode, "ansi-color-mode", "truecolor", "ANSI color encoding: truecolor | 256")
	fs.BoolVar(&htmlFragment, "html-fragment", false, "emit only the <pre> block instead of a full HTML document")
	fs.StringVar(&htmlBackground, "html-bg", "#000000", "HTML background color")
	fs.BoolVar(&bashFragment, "bash-fragment", false, "emit only the \"cat <<EOF\" snippet instead of a full runnable script")
	fs.StringVar(&bashFuncName, "bash-func", "show_ascii_art", "shell function name the art is wrapped in")
	fs.Float64Var(&pngFontSize, "png-font-size", 16, "PNG glyph size in points (~pixels at 72 DPI); bigger = higher-resolution image")
	fs.StringVar(&pngBackground, "png-bg", "#000000", "PNG background color")
	fs.StringVar(&pngScreen, "screen", "", "fit the PNG to a monitor resolution (preset "+joinNames(asciiart.ScreenPresetNames())+", or WIDTHxHEIGHT); overrides -width/-height/-scale/-aspect")
	fs.StringVar(&pngScreenFit, "screen-fit", "cover", "how -screen fits the image: cover (crop to fill, default) | contain (fit whole image, pad with -png-bg)")
	fs.BoolVar(&listRamps, "list-ramps", false, "list built-in ramp presets and exit")
	fs.BoolVar(&listPalettes, "list-palettes", false, "list built-in palette presets and exit")
	fs.BoolVar(&listFilters, "list-filters", false, "list built-in filter keywords and exit")

	// Go's flag package stops parsing at the first non-flag token, so a
	// filename given before other flags (the natural "ascii-converter
	// photo.jpg -mode html" form shown in -h and the README) would
	// silently swallow every flag after it. Pull the filename out first
	// so flags can appear anywhere on the command line.
	boolFlags := map[string]bool{
		"html-fragment": true,
		"bash-fragment": true,
		"list-ramps":    true,
		"list-palettes": true,
		"list-filters":  true,
	}
	flagArgs, positional := splitPositional(args, boolFlags)

	if err := fs.Parse(flagArgs); err != nil {
		return err
	}

	if listRamps {
		fmt.Println(joinNames(asciiart.RampNames()))
		return nil
	}
	if listPalettes {
		fmt.Println(joinNames(asciiart.PaletteNames()))
		return nil
	}
	if listFilters {
		fmt.Println(joinNames(asciiart.FilterNames()))
		return nil
	}

	if input == "" && len(positional) > 0 {
		input = positional[0]
	}
	if input == "" {
		fs.Usage()
		return fmt.Errorf("no input image given")
	}

	ramp, err := asciiart.ResolveRamp(rampSpec)
	if err != nil {
		return err
	}
	palette, err := asciiart.ResolvePalette(paletteSpec)
	if err != nil {
		return err
	}
	preFilters, postFilters, err := asciiart.ParseFilters(filterSpec)
	if err != nil {
		return err
	}

	var colorMode asciiart.AnsiColorMode
	switch ansiColorMode {
	case "truecolor":
		colorMode = asciiart.AnsiTrueColor
	case "256":
		colorMode = asciiart.Ansi256
	default:
		return fmt.Errorf("invalid -ansi-color-mode %q: want truecolor or 256", ansiColorMode)
	}

	img, err := loadImage(input)
	if err != nil {
		return err
	}

	convCols, convRows, convScale := width, height, scale
	procImg := img
	var pngCanvasW, pngCanvasH int
	if pngScreen != "" {
		if mode != "png" {
			return fmt.Errorf("-screen requires -mode png")
		}
		screenW, screenH, err := asciiart.ParseScreenSize(pngScreen)
		if err != nil {
			return err
		}
		cellWidth, cellHeight, err := asciiart.PNGCellSize(pngFontSize)
		if err != nil {
			return err
		}
		imgB := img.Bounds()
		cols, rows, crop, err := asciiart.FitDimensions(imgB.Dx(), imgB.Dy(), screenW, screenH, cellWidth, cellHeight, pngScreenFit)
		if err != nil {
			return err
		}
		convCols, convRows, convScale = cols, rows, 1
		if crop {
			// "cover": crop to the screen's aspect ratio first so the
			// character grid fills the frame completely instead of
			// preserving the source image's own aspect.
			procImg = asciiart.CropToAspect(img, screenW, screenH)
		}
		// "contain" grids are deliberately smaller than the canvas in one
		// dimension to preserve the image's aspect ratio; RenderPNG
		// centers them and fills the rest with -png-bg (letterbox/pillarbox).
		pngCanvasW, pngCanvasH = screenW, screenH
	}

	grid := asciiart.Convert(procImg, asciiart.Options{
		Columns:     convCols,
		Rows:        convRows,
		Scale:       convScale,
		CharAspect:  aspect,
		Ramp:        ramp,
		Palette:     palette,
		Filters:     preFilters,
		PostFilters: postFilters,
	})

	out := os.Stdout
	if output != "" {
		f, err := os.Create(output)
		if err != nil {
			return fmt.Errorf("creating output file: %w", err)
		}
		defer f.Close()
		out = f

		if mode == "bash" && !bashFragment {
			if err := f.Chmod(0o755); err != nil {
				return fmt.Errorf("marking output file executable: %w", err)
			}
		}
	}

	switch mode {
	case "plain":
		return asciiart.RenderPlain(out, grid)
	case "ansi":
		return asciiart.RenderANSI(out, grid, colorMode)
	case "html":
		return asciiart.RenderHTML(out, grid, asciiart.HTMLOptions{
			Fragment:   htmlFragment,
			Background: htmlBackground,
			Title:      input,
		})
	case "bash":
		return asciiart.RenderBash(out, grid, colorMode, asciiart.BashOptions{
			Fragment: bashFragment,
			FuncName: bashFuncName,
			Title:    input,
		})
	case "png":
		bg, err := asciiart.ParseHexColor(pngBackground)
		if err != nil {
			return fmt.Errorf("-png-bg: %w", err)
		}
		return asciiart.RenderPNG(out, grid, asciiart.PNGOptions{
			FontSize:     pngFontSize,
			Background:   bg,
			CanvasWidth:  pngCanvasW,
			CanvasHeight: pngCanvasH,
		})
	default:
		return fmt.Errorf("invalid -mode %q: want plain, ansi, html, bash, or png", mode)
	}
}

func loadImage(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening image: %w", err)
	}
	defer f.Close()

	img, format, err := image.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("decoding image (supported: png, jpeg, gif): %w", err)
	}
	_ = format
	return img, nil
}

// splitPositional separates args into flag tokens (suitable for
// flag.FlagSet.Parse) and positional tokens (e.g. a bare filename),
// regardless of where the positional tokens fall in the argument list.
// boolFlags names flags that take no value, so their following token
// isn't mistaken for a flag's argument.
func splitPositional(args []string, boolFlags map[string]bool) (flagArgs, positional []string) {
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "-" || !strings.HasPrefix(a, "-") {
			positional = append(positional, a)
			continue
		}
		flagArgs = append(flagArgs, a)
		name := strings.TrimLeft(a, "-")
		if strings.Contains(name, "=") || boolFlags[name] {
			continue
		}
		if i+1 < len(args) {
			i++
			flagArgs = append(flagArgs, args[i])
		}
	}
	return flagArgs, positional
}

func joinNames(names []string) string {
	out := ""
	for i, n := range names {
		if i > 0 {
			out += ", "
		}
		out += n
	}
	return out
}
