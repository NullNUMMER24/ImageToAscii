package asciiart

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

var screenPresets = map[string][2]int{
	"720p":      {1280, 720},
	"1080p":     {1920, 1080},
	"1440p":     {2560, 1440},
	"4k":        {3840, 2160},
	"5k":        {5120, 2880},
	"ultrawide": {3440, 1440},
}

// ScreenPresetNames returns the built-in -screen preset names.
func ScreenPresetNames() []string {
	return []string{"720p", "1080p", "1440p", "4k", "5k", "ultrawide"}
}

// ParseScreenSize parses a screen-size spec: either a preset name (see
// ScreenPresetNames) or an explicit "WIDTHxHEIGHT" pixel resolution.
func ParseScreenSize(spec string) (w, h int, err error) {
	key := strings.ToLower(strings.TrimSpace(spec))
	if dims, ok := screenPresets[key]; ok {
		return dims[0], dims[1], nil
	}

	parts := strings.SplitN(key, "x", 2)
	if len(parts) == 2 {
		pw, errW := strconv.Atoi(strings.TrimSpace(parts[0]))
		ph, errH := strconv.Atoi(strings.TrimSpace(parts[1]))
		if errW == nil && errH == nil && pw > 0 && ph > 0 {
			return pw, ph, nil
		}
	}

	return 0, 0, fmt.Errorf("screen size %q: not a known preset (%v) or WIDTHxHEIGHT (e.g. 1920x1080)", spec, ScreenPresetNames())
}

// ScreenFitNames returns the valid -screen-fit values.
func ScreenFitNames() []string {
	return []string{"cover", "contain"}
}

// FitDimensions computes the character-grid size needed to place an image
// of imgW x imgH pixels into a screenW x screenH pixel canvas, given the
// font's cellWidth x cellHeight, per fit mode:
//
//	"cover"   (default) - fills the whole canvas; the caller should
//	                       CropToAspect the source image first so no
//	                       stretching occurs, cropping any overflow.
//	"contain"           - fits the entire image without cropping,
//	                       leaving canvas background visible on the
//	                       sides that don't match (letterbox/pillarbox).
//
// crop reports whether the caller should crop the source image first.
func FitDimensions(imgW, imgH, screenW, screenH, cellWidth, cellHeight int, fit string) (cols, rows int, crop bool, err error) {
	if imgW <= 0 || imgH <= 0 {
		return 0, 0, false, fmt.Errorf("invalid image dimensions %dx%d", imgW, imgH)
	}
	if cellWidth <= 0 || cellHeight <= 0 {
		return 0, 0, false, fmt.Errorf("invalid cell dimensions %dx%d", cellWidth, cellHeight)
	}

	switch fit {
	case "", "cover":
		cols = round(float64(screenW) / float64(cellWidth))
		rows = round(float64(screenH) / float64(cellHeight))
		crop = true
	case "contain":
		scale := math.Min(float64(screenW)/float64(imgW), float64(screenH)/float64(imgH))
		cols = round(float64(imgW) * scale / float64(cellWidth))
		rows = round(float64(imgH) * scale / float64(cellHeight))
		crop = false
	default:
		return 0, 0, false, fmt.Errorf("invalid screen fit %q: want %v", fit, ScreenFitNames())
	}

	if cols < 1 {
		cols = 1
	}
	if rows < 1 {
		rows = 1
	}
	return cols, rows, crop, nil
}

func round(v float64) int {
	return int(v + 0.5)
}
