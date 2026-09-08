package asciiart

import (
	"image"
	"image/color"
)

// Luminance returns perceptual brightness in [0, 1] using Rec. 709
// coefficients.
func Luminance(r, g, b uint8) float64 {
	return (0.2126*float64(r) + 0.7152*float64(g) + 0.0722*float64(b)) / 255
}

// Resize scales img to exactly width x height using area averaging when
// downscaling (each destination pixel is the average of the source
// rectangle it covers) and nearest-neighbor when upscaling.
func Resize(img image.Image, width, height int) *image.RGBA {
	src := toRGBA(img)
	srcBounds := src.Bounds()
	sw, sh := srcBounds.Dx(), srcBounds.Dy()

	dst := image.NewRGBA(image.Rect(0, 0, width, height))
	if width == 0 || height == 0 || sw == 0 || sh == 0 {
		return dst
	}

	xScale := float64(sw) / float64(width)
	yScale := float64(sh) / float64(height)

	for dy := 0; dy < height; dy++ {
		srcY0 := int(float64(dy) * yScale)
		srcY1 := int(float64(dy+1) * yScale)
		if srcY1 <= srcY0 {
			srcY1 = srcY0 + 1
		}
		if srcY1 > sh {
			srcY1 = sh
		}

		for dx := 0; dx < width; dx++ {
			srcX0 := int(float64(dx) * xScale)
			srcX1 := int(float64(dx+1) * xScale)
			if srcX1 <= srcX0 {
				srcX1 = srcX0 + 1
			}
			if srcX1 > sw {
				srcX1 = sw
			}

			var sr, sg, sb, sa, count int
			for sy := srcY0; sy < srcY1; sy++ {
				for sx := srcX0; sx < srcX1; sx++ {
					c := src.RGBAAt(srcBounds.Min.X+sx, srcBounds.Min.Y+sy)
					sr += int(c.R)
					sg += int(c.G)
					sb += int(c.B)
					sa += int(c.A)
					count++
				}
			}
			if count == 0 {
				count = 1
			}
			dst.SetRGBA(dx, dy, color.RGBA{
				R: uint8(sr / count),
				G: uint8(sg / count),
				B: uint8(sb / count),
				A: uint8(sa / count),
			})
		}
	}
	return dst
}

// CropToAspect center-crops img to match the target width:height aspect
// ratio, trimming the wider or taller dimension symmetrically (the same
// "cover" behavior as CSS background-size: cover) so a subsequent resize
// to that aspect fills the frame completely with no stretching or
// letterboxing.
func CropToAspect(img image.Image, targetW, targetH int) image.Image {
	if targetW <= 0 || targetH <= 0 {
		return img
	}
	src := toRGBA(img)
	b := src.Bounds()
	srcW, srcH := b.Dx(), b.Dy()
	if srcW == 0 || srcH == 0 {
		return img
	}

	targetAspect := float64(targetW) / float64(targetH)
	srcAspect := float64(srcW) / float64(srcH)

	cropW, cropH := srcW, srcH
	switch {
	case srcAspect > targetAspect:
		cropW = int(float64(srcH) * targetAspect)
	case srcAspect < targetAspect:
		cropH = int(float64(srcW) / targetAspect)
	default:
		return img
	}
	if cropW < 1 {
		cropW = 1
	}
	if cropH < 1 {
		cropH = 1
	}

	offsetX := b.Min.X + (srcW-cropW)/2
	offsetY := b.Min.Y + (srcH-cropH)/2
	rect := image.Rect(offsetX, offsetY, offsetX+cropW, offsetY+cropH)
	return src.SubImage(rect)
}

func toRGBA(img image.Image) *image.RGBA {
	if rgba, ok := img.(*image.RGBA); ok {
		return rgba
	}
	b := img.Bounds()
	out := image.NewRGBA(b)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			out.Set(x, y, img.At(x, y))
		}
	}
	return out
}

// TargetDimensions computes the character-grid width/height for an image
// of size (imgW, imgH) given desired columns/rows (0 = auto), a character
// aspect-ratio correction factor (terminal glyphs are taller than they
// are wide, so rows are scaled down by this factor to avoid vertical
// stretching), and an overall scale multiplier applied to the result
// (e.g. 0.5 to halve whatever size would otherwise have been used).
func TargetDimensions(imgW, imgH, wantCols, wantRows int, charAspect, scale float64) (cols, rows int) {
	switch {
	case wantCols > 0 && wantRows > 0:
		cols, rows = wantCols, wantRows
	case wantCols > 0:
		cols = wantCols
		rows = int(float64(wantCols) * float64(imgH) / float64(imgW) * charAspect)
	case wantRows > 0:
		rows = wantRows
		cols = int(float64(wantRows) * float64(imgW) / float64(imgH) / charAspect)
	default:
		cols = 100
		rows = int(float64(cols) * float64(imgH) / float64(imgW) * charAspect)
	}

	if scale <= 0 {
		scale = 1
	}
	cols = int(float64(cols)*scale + 0.5)
	rows = int(float64(rows)*scale + 0.5)
	if cols < 1 {
		cols = 1
	}
	if rows < 1 {
		rows = 1
	}
	return cols, rows
}
