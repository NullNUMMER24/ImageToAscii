package asciiart

import (
	"fmt"
	"image/color"
	"strconv"
	"strings"
)

// PaletteFunc maps a pixel's luminance (0..1) and original color to the
// color that should actually be rendered for that cell.
type PaletteFunc func(luminance float64, original color.RGBA) color.RGBA

// gradient palettes are defined as a small set of stops (dark -> light);
// the render color is a linear interpolation between the two nearest
// stops for the pixel's luminance.
var gradientPresets = map[string][]color.RGBA{
	"matrix": {
		{0x00, 0x08, 0x00, 0xff},
		{0x00, 0x8f, 0x11, 0xff},
		{0x00, 0xff, 0x41, 0xff},
	},
	"amber": {
		{0x1a, 0x0a, 0x00, 0xff},
		{0xaa, 0x5c, 0x00, 0xff},
		{0xff, 0xb0, 0x00, 0xff},
	},
	"ice": {
		{0x00, 0x08, 0x1a, 0xff},
		{0x0a, 0x6e, 0xb4, 0xff},
		{0xd6, 0xf5, 0xff, 0xff},
	},
	"fire": {
		{0x0a, 0x00, 0x00, 0xff},
		{0xaf, 0x1c, 0x00, 0xff},
		{0xff, 0x8a, 0x00, 0xff},
		{0xff, 0xf3, 0xb0, 0xff},
	},
	"sepia": {
		{0x1b, 0x12, 0x0a, 0xff},
		{0x8a, 0x5a, 0x33, 0xff},
		{0xe4, 0xc9, 0x9c, 0xff},
	},
	"mono": {
		{0x00, 0x00, 0x00, 0xff},
		{0xff, 0xff, 0xff, 0xff},
	},
}

// PaletteNames returns the built-in palette names, excluding "custom".
func PaletteNames() []string {
	return []string{"original", "grayscale", "matrix", "amber", "ice", "fire", "sepia", "mono"}
}

// ResolvePalette turns a palette spec into a PaletteFunc. Recognized specs:
//
//	"original"          - use the pixel's actual color unchanged
//	"grayscale"         - collapse to gray based on luminance
//	<preset name>        - one of PaletteNames()'s gradient presets
//	"custom:#RRGGBB,..."  - interpolate across a user-supplied gradient (>=2 stops)
func ResolvePalette(spec string) (PaletteFunc, error) {
	switch spec {
	case "original":
		return func(_ float64, original color.RGBA) color.RGBA { return original }, nil
	case "grayscale":
		return func(luminance float64, _ color.RGBA) color.RGBA {
			g := clampByte(luminance * 255)
			return color.RGBA{g, g, g, 0xff}
		}, nil
	}

	if strings.HasPrefix(spec, "custom:") {
		stops, err := parseHexStops(strings.TrimPrefix(spec, "custom:"))
		if err != nil {
			return nil, fmt.Errorf("palette %q: %w", spec, err)
		}
		return gradientFunc(stops), nil
	}

	if stops, ok := gradientPresets[spec]; ok {
		return gradientFunc(stops), nil
	}

	return nil, fmt.Errorf("palette %q: not a known preset (%v) or custom:#hex,#hex,...", spec, PaletteNames())
}

func gradientFunc(stops []color.RGBA) PaletteFunc {
	return func(luminance float64, _ color.RGBA) color.RGBA {
		return sampleGradient(stops, luminance)
	}
}

func sampleGradient(stops []color.RGBA, t float64) color.RGBA {
	if t <= 0 {
		return stops[0]
	}
	if t >= 1 {
		return stops[len(stops)-1]
	}
	segments := len(stops) - 1
	scaled := t * float64(segments)
	idx := int(scaled)
	if idx >= segments {
		idx = segments - 1
	}
	localT := scaled - float64(idx)
	a, b := stops[idx], stops[idx+1]
	return color.RGBA{
		R: lerpByte(a.R, b.R, localT),
		G: lerpByte(a.G, b.G, localT),
		B: lerpByte(a.B, b.B, localT),
		A: 0xff,
	}
}

func lerpByte(a, b uint8, t float64) uint8 {
	return clampByte(float64(a) + (float64(b)-float64(a))*t)
}

func clampByte(v float64) uint8 {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return uint8(v + 0.5)
}

func parseHexStops(list string) ([]color.RGBA, error) {
	parts := strings.Split(list, ",")
	stops := make([]color.RGBA, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		c, err := parseHexColor(p)
		if err != nil {
			return nil, err
		}
		stops = append(stops, c)
	}
	if len(stops) < 2 {
		return nil, fmt.Errorf("need >= 2 colors, got %d", len(stops))
	}
	return stops, nil
}

// ParseHexColor parses a "#RRGGBB" (or "RRGGBB") string into an opaque color.
func ParseHexColor(s string) (color.RGBA, error) {
	return parseHexColor(s)
}

func parseHexColor(s string) (color.RGBA, error) {
	s = strings.TrimPrefix(s, "#")
	if len(s) != 6 {
		return color.RGBA{}, fmt.Errorf("invalid hex color %q: want #RRGGBB", s)
	}
	v, err := strconv.ParseUint(s, 16, 32)
	if err != nil {
		return color.RGBA{}, fmt.Errorf("invalid hex color %q: %w", s, err)
	}
	return color.RGBA{
		R: uint8(v >> 16),
		G: uint8(v >> 8),
		B: uint8(v),
		A: 0xff,
	}, nil
}
