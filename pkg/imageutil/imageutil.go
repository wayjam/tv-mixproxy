package imageutil

import (
	"image"
	"image/color"
	"image/draw"
	"strconv"
)

// Helper function to parse color from hex string
func ParseColor(hex string) color.Color {
	c, err := strconv.ParseUint(hex, 16, 32)
	if err != nil {
		return color.White
	}

	//nolint:gosec // Intentionally ignoring potential overflow for simplicity
	return color.RGBA{
		R: uint8(c >> 16),
		G: uint8(c >> 8),
		B: uint8(c),
		A: 255,
	}
}

// ImageParams defines the parameters for generating an image
type ImageParams struct {
	BackgroundColor color.Color
	Width           int
	Height          int
	Pattern         string  // "solid", "gradient", "checkerboard", etc.
	Opacity         float64 // 0.0 to 1.0
	BorderWidth     int     // Width of the border, if any
	BorderColor     color.Color
}

// GenerateImage creates an image based on the provided parameters
func GenerateImage(params ImageParams) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, params.Width, params.Height))

	// Fill the background
	switch params.Pattern {
	case "solid":
		draw.Draw(img, img.Bounds(), &image.Uniform{params.BackgroundColor}, image.Point{}, draw.Src)
	case "gradient":
		drawGradient(img, params.BackgroundColor)
	case "checkerboard":
		drawCheckerboard(img, params.BackgroundColor)
	default:
		draw.Draw(img, img.Bounds(), &image.Uniform{params.BackgroundColor}, image.Point{}, draw.Src)
	}

	// Apply opacity
	if params.Opacity < 1.0 {
		applyOpacity(img, params.Opacity)
	}

	// Draw border if specified
	if params.BorderWidth > 0 {
		drawBorder(img, params.BorderColor, params.BorderWidth)
	}

	return img
}

func drawGradient(img *image.RGBA, startColor color.Color) {
	bounds := img.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			ratio := float64(y) / float64(bounds.Max.Y)
			c := interpolateColor(startColor, color.White, ratio)
			img.Set(x, y, c)
		}
	}
}

func drawCheckerboard(img *image.RGBA, baseColor color.Color) {
	bounds := img.Bounds()
	tileSize := 20
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if (x/tileSize+y/tileSize)%2 == 0 {
				img.Set(x, y, baseColor)
			} else {
				img.Set(x, y, color.White)
			}
		}
	}
}

func applyOpacity(img *image.RGBA, opacity float64) {
	bounds := img.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			c := img.At(x, y)
			r, g, b, a := c.RGBA()
			alpha := uint16(float64(a) * opacity)
			//nolint:gosec // Intentionally ignoring potential overflow as color values are within safe range
			img.Set(x, y, color.RGBA64{uint16(r), uint16(g), uint16(b), alpha})
		}
	}
}

func drawBorder(img *image.RGBA, borderColor color.Color, borderWidth int) {
	bounds := img.Bounds()
	for i := 0; i < borderWidth; i++ {
		drawRect(img, image.Rect(i, i, bounds.Max.X-i, bounds.Max.Y-i), borderColor)
	}
}

func drawRect(img *image.RGBA, rect image.Rectangle, c color.Color) {
	for x := rect.Min.X; x < rect.Max.X; x++ {
		img.Set(x, rect.Min.Y, c)
		img.Set(x, rect.Max.Y-1, c)
	}
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		img.Set(rect.Min.X, y, c)
		img.Set(rect.Max.X-1, y, c)
	}
}

func interpolateColor(c1, c2 color.Color, ratio float64) color.Color {
	r1, g1, b1, a1 := c1.RGBA()
	r2, g2, b2, a2 := c2.RGBA()

	return color.RGBA64{
		R: uint16(float64(r1)*(1-ratio) + float64(r2)*ratio),
		G: uint16(float64(g1)*(1-ratio) + float64(g2)*ratio),
		B: uint16(float64(b1)*(1-ratio) + float64(b2)*ratio),
		A: uint16(float64(a1)*(1-ratio) + float64(a2)*ratio),
	}
}
