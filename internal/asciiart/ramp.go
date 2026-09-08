package asciiart

import "fmt"

// A ramp is a sequence of characters ordered from lowest visual density
// (e.g. a blank space) to highest visual density (e.g. '@'). Pixel
// luminance is mapped onto this sequence, so bright pixels render with
// dense glyphs and dark pixels render with sparse ones.
var rampPresets = map[string][]rune{
	"simple":   []rune(" .:-=+*#%@"),
	"detailed": []rune(" .'`^\",:;Il!i><~+_-?][}{1)(|\\/tfjrxnuvczXYUJCLQ0OZmwqpdbkhao*#MW&8%B@$"),
	"blocks":   []rune(" ░▒▓█"),
	"minimal":  []rune(" .*#"),
}

// RampNames returns the sorted list of built-in ramp preset names.
func RampNames() []string {
	return []string{"simple", "detailed", "blocks", "minimal"}
}

// ResolveRamp turns a ramp spec into an ordered rune slice. The spec is
// either the name of a built-in preset, or a literal string of at least
// two characters to use as a custom ramp (ordered dark/sparse -> light/dense).
func ResolveRamp(spec string) ([]rune, error) {
	if preset, ok := rampPresets[spec]; ok {
		return preset, nil
	}
	runes := []rune(spec)
	if len(runes) < 2 {
		return nil, fmt.Errorf("ramp %q: not a known preset (%v) and too short to use literally (need >= 2 characters)", spec, RampNames())
	}
	return runes, nil
}
