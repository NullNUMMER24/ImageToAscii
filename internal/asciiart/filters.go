package asciiart

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"math/rand"
	"strconv"
	"strings"
)

// A Filter mutates an RGBA image in place.
type Filter func(img *image.RGBA)

// postStageFilters names filters that must run on the already-downscaled
// character grid rather than the full-resolution source image. "noise"
// scatters single-pixel speckles; applied before resizing they'd almost
// always be averaged away by the area-average downscale in Resize.
var postStageFilters = map[string]bool{
	"noise": true,
}

// ParseFilters parses a comma-separated filter pipeline spec, e.g.
// "grayscale,brightness=1.2,contrast=1.1,blur=1.5,sepia,edge,invert,noise=0.1"
// and returns the filters in the order they should be applied, split into
// pre-resize (full-resolution) and post-resize (character-grid) stages.
func ParseFilters(spec string) (pre []Filter, post []Filter, err error) {
	if strings.TrimSpace(spec) == "" {
		return nil, nil, nil
	}
	for _, part := range strings.Split(spec, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		name, arg, hasArg := strings.Cut(part, "=")
		f, err := buildFilter(name, arg, hasArg)
		if err != nil {
			return nil, nil, err
		}
		if postStageFilters[name] {
			post = append(post, f)
		} else {
			pre = append(pre, f)
		}
	}
	return pre, post, nil
}

// FilterNames lists the built-in filter keywords accepted by ParseFilters.
func FilterNames() []string {
	return []string{
		"grayscale", "invert", "sepia", "edge",
		"brightness=<factor>", "contrast=<factor>", "blur=<radius>",
		"threshold=<cutoff 0-1>", "posterize=<levels>", "noise=<amount 0-1>",
		"gamma=<value>",
	}
}

func buildFilter(name, arg string, hasArg bool) (Filter, error) {
	switch name {
	case "grayscale":
		return Grayscale, nil
	case "invert":
		return Invert, nil
	case "sepia":
		return Sepia, nil
	case "edge":
		return EdgeDetect, nil
	case "brightness":
		factor, err := parseFactor(name, arg, hasArg, 1.2)
		if err != nil {
			return nil, err
		}
		return func(img *image.RGBA) { Brightness(img, factor) }, nil
	case "contrast":
		factor, err := parseFactor(name, arg, hasArg, 1.2)
		if err != nil {
			return nil, err
		}
		return func(img *image.RGBA) { Contrast(img, factor) }, nil
	case "blur":
		radius, err := parseFactor(name, arg, hasArg, 1)
		if err != nil {
			return nil, err
		}
		return func(img *image.RGBA) { BoxBlur(img, int(radius+0.5)) }, nil
	case "threshold":
		cutoff, err := parseFactor(name, arg, hasArg, 0.5)
		if err != nil {
			return nil, err
		}
		return func(img *image.RGBA) { Threshold(img, cutoff) }, nil
	case "posterize":
		levels, err := parseFactor(name, arg, hasArg, 4)
		if err != nil {
			return nil, err
		}
		return func(img *image.RGBA) { Posterize(img, int(levels+0.5)) }, nil
	case "noise":
		amount, err := parseFactor(name, arg, hasArg, 0.08)
		if err != nil {
			return nil, err
		}
		return func(img *image.RGBA) { Noise(img, amount) }, nil
	case "gamma":
		g, err := parseFactor(name, arg, hasArg, 1.8)
		if err != nil {
			return nil, err
		}
		return func(img *image.RGBA) { Gamma(img, g) }, nil
	default:
		return nil, fmt.Errorf("unknown filter %q (known: %v)", name, FilterNames())
	}
}

func parseFactor(name, arg string, hasArg bool, def float64) (float64, error) {
	if !hasArg {
		return def, nil
	}
	v, err := strconv.ParseFloat(arg, 64)
	if err != nil {
		return 0, fmt.Errorf("filter %q: invalid numeric argument %q: %w", name, arg, err)
	}
	return v, nil
}

func eachPixel(img *image.RGBA, fn func(r, g, b, a uint8) (uint8, uint8, uint8, uint8)) {
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			c := img.RGBAAt(x, y)
			nr, ng, nb, na := fn(c.R, c.G, c.B, c.A)
			img.SetRGBA(x, y, color.RGBA{nr, ng, nb, na})
		}
	}
}

// Grayscale collapses every pixel to its perceptual luminance.
func Grayscale(img *image.RGBA) {
	eachPixel(img, func(r, g, b, a uint8) (uint8, uint8, uint8, uint8) {
		l := clampByte(Luminance(r, g, b) * 255)
		return l, l, l, a
	})
}

// Invert flips each color channel.
func Invert(img *image.RGBA) {
	eachPixel(img, func(r, g, b, a uint8) (uint8, uint8, uint8, uint8) {
		return 255 - r, 255 - g, 255 - b, a
	})
}

// Sepia applies a classic sepia color matrix.
func Sepia(img *image.RGBA) {
	eachPixel(img, func(r, g, b, a uint8) (uint8, uint8, uint8, uint8) {
		fr, fg, fb := float64(r), float64(g), float64(b)
		nr := clampByte(fr*0.393 + fg*0.769 + fb*0.189)
		ng := clampByte(fr*0.349 + fg*0.686 + fb*0.168)
		nb := clampByte(fr*0.272 + fg*0.534 + fb*0.131)
		return nr, ng, nb, a
	})
}

// Brightness multiplies each color channel by factor.
func Brightness(img *image.RGBA, factor float64) {
	eachPixel(img, func(r, g, b, a uint8) (uint8, uint8, uint8, uint8) {
		return clampByte(float64(r) * factor), clampByte(float64(g) * factor), clampByte(float64(b) * factor), a
	})
}

// Contrast scales each channel around the mid-gray point by factor.
func Contrast(img *image.RGBA, factor float64) {
	eachPixel(img, func(r, g, b, a uint8) (uint8, uint8, uint8, uint8) {
		adjust := func(c uint8) uint8 {
			return clampByte((float64(c)-127.5)*factor + 127.5)
		}
		return adjust(r), adjust(g), adjust(b), a
	})
}

// Gamma applies gamma correction: output = 255*(input/255)^(1/gamma).
// gamma > 1 lifts shadows/midtones without blowing out highlights (unlike
// a flat Brightness multiply); gamma < 1 crushes them further. Useful when
// dark image regions are rendering as near-invisible on a dark
// background, since low luminance already picks sparse/thin ramp
// characters — a dark color on top compounds that into near-total loss
// of shadow detail.
func Gamma(img *image.RGBA, gamma float64) {
	if gamma <= 0 {
		gamma = 1
	}
	invGamma := 1 / gamma
	adjust := func(c uint8) uint8 {
		return clampByte(math.Pow(float64(c)/255, invGamma) * 255)
	}
	eachPixel(img, func(r, g, b, a uint8) (uint8, uint8, uint8, uint8) {
		return adjust(r), adjust(g), adjust(b), a
	})
}

// BoxBlur applies a separable box blur of the given pixel radius.
func BoxBlur(img *image.RGBA, radius int) {
	if radius <= 0 {
		return
	}
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	if w == 0 || h == 0 {
		return
	}

	tmp := make([]color.RGBA, w*h)
	out := make([]color.RGBA, w*h)
	at := func(buf []color.RGBA, x, y int) color.RGBA { return buf[y*w+x] }
	set := func(buf []color.RGBA, x, y int, c color.RGBA) { buf[y*w+x] = c }

	src := make([]color.RGBA, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			src[y*w+x] = img.RGBAAt(bounds.Min.X+x, bounds.Min.Y+y)
		}
	}

	blurLine := func(get func(i int) color.RGBA, n int, set func(i int, c color.RGBA)) {
		for i := 0; i < n; i++ {
			var sr, sg, sb, sa, count int
			for k := i - radius; k <= i+radius; k++ {
				if k < 0 || k >= n {
					continue
				}
				c := get(k)
				sr += int(c.R)
				sg += int(c.G)
				sb += int(c.B)
				sa += int(c.A)
				count++
			}
			set(i, color.RGBA{
				R: uint8(sr / count),
				G: uint8(sg / count),
				B: uint8(sb / count),
				A: uint8(sa / count),
			})
		}
	}

	for y := 0; y < h; y++ {
		blurLine(
			func(x int) color.RGBA { return src[y*w+x] },
			w,
			func(x int, c color.RGBA) { tmp[y*w+x] = c },
		)
	}
	for x := 0; x < w; x++ {
		blurLine(
			func(y int) color.RGBA { return at(tmp, x, y) },
			h,
			func(y int, c color.RGBA) { set(out, x, y, c) },
		)
	}

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetRGBA(bounds.Min.X+x, bounds.Min.Y+y, out[y*w+x])
		}
	}
}

// EdgeDetect applies a Sobel operator and replaces the image with a
// grayscale edge-magnitude map.
func EdgeDetect(img *image.RGBA) {
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	if w == 0 || h == 0 {
		return
	}

	gray := make([]float64, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := img.RGBAAt(bounds.Min.X+x, bounds.Min.Y+y)
			gray[y*w+x] = Luminance(c.R, c.G, c.B) * 255
		}
	}

	at := func(x, y int) float64 {
		if x < 0 {
			x = 0
		}
		if x >= w {
			x = w - 1
		}
		if y < 0 {
			y = 0
		}
		if y >= h {
			y = h - 1
		}
		return gray[y*w+x]
	}

	out := make([]uint8, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			gx := -at(x-1, y-1) - 2*at(x-1, y) - at(x-1, y+1) +
				at(x+1, y-1) + 2*at(x+1, y) + at(x+1, y+1)
			gy := -at(x-1, y-1) - 2*at(x, y-1) - at(x+1, y-1) +
				at(x-1, y+1) + 2*at(x, y+1) + at(x+1, y+1)
			mag := math.Sqrt(gx*gx + gy*gy)
			out[y*w+x] = clampByte(mag)
		}
	}

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			v := out[y*w+x]
			img.SetRGBA(bounds.Min.X+x, bounds.Min.Y+y, color.RGBA{v, v, v, 0xff})
		}
	}
}

// Threshold collapses every pixel to pure black or white based on whether
// its luminance meets cutoff (0-1). Produces a hard-edged silhouette.
func Threshold(img *image.RGBA, cutoff float64) {
	eachPixel(img, func(r, g, b, a uint8) (uint8, uint8, uint8, uint8) {
		if Luminance(r, g, b) >= cutoff {
			return 255, 255, 255, a
		}
		return 0, 0, 0, a
	})
}

// Posterize quantizes each color channel independently into the given
// number of discrete levels (minimum 2), flattening smooth gradients into
// sharply separated color bands while keeping hue.
func Posterize(img *image.RGBA, levels int) {
	if levels < 2 {
		levels = 2
	}
	step := 255.0 / float64(levels-1)
	quantize := func(c uint8) uint8 {
		return clampByte(math.Round(float64(c)/step) * step)
	}
	eachPixel(img, func(r, g, b, a uint8) (uint8, uint8, uint8, uint8) {
		return quantize(r), quantize(g), quantize(b), a
	})
}

// Noise randomly replaces pixels with vivid, randomly-hued speckles.
// amount (0-1) is the per-pixel probability of being speckled. Intended to
// run on the already-downscaled character grid (see postStageFilters) so
// each speckle survives as one visible, distinctly colored cell.
func Noise(img *image.RGBA, amount float64) {
	if amount <= 0 {
		return
	}
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if rand.Float64() < amount {
				img.SetRGBA(x, y, randomVividColor())
			}
		}
	}
}

func randomVividColor() color.RGBA {
	h := rand.Float64() * 360
	s := 0.75 + rand.Float64()*0.25
	v := 0.75 + rand.Float64()*0.25
	r, g, bl := hsvToRGB(h, s, v)
	return color.RGBA{r, g, bl, 0xff}
}

// hsvToRGB converts h in [0,360), s and v in [0,1] to 8-bit RGB.
func hsvToRGB(h, s, v float64) (uint8, uint8, uint8) {
	c := v * s
	x := c * (1 - math.Abs(math.Mod(h/60, 2)-1))
	m := v - c

	var r, g, b float64
	switch {
	case h < 60:
		r, g, b = c, x, 0
	case h < 120:
		r, g, b = x, c, 0
	case h < 180:
		r, g, b = 0, c, x
	case h < 240:
		r, g, b = 0, x, c
	case h < 300:
		r, g, b = x, 0, c
	default:
		r, g, b = c, 0, x
	}
	return clampByte((r + m) * 255), clampByte((g + m) * 255), clampByte((b + m) * 255)
}
