package watermark

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"strings"
)

// Mode controls how the watermark is applied to the image.
type Mode int

const (
	// ModeNone returns the original image with no watermark.
	ModeNone Mode = iota
	// ModeText draws the watermark text in the bottom-right corner.
	ModeText
	// ModeTile draws the watermark text centered on the image.
	ModeTile
)

// Apply renders a visible text watermark onto img and returns the watermarked
// image. The mode controls watermark positioning.
func Apply(img image.Image, text string, mode Mode) (image.Image, error) {
	if mode == ModeNone {
		return img, nil
	}
	if text == "" {
		return nil, errors.New("watermark: text must not be empty")
	}

	rgba := image.NewRGBA(image.Rect(0, 0, img.Bounds().Dx(), img.Bounds().Dy()))
	draw.Draw(rgba, rgba.Bounds(), img, img.Bounds().Min, draw.Src)

	switch mode {
	case ModeText:
		drawTextWatermark(rgba, text, false)
	case ModeTile:
		drawTextWatermark(rgba, text, true)
	default:
		return nil, fmt.Errorf("watermark: unknown mode %d", mode)
	}

	return rgba, nil
}

func drawTextWatermark(img *image.RGBA, text string, center bool) {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	if width < 24 || height < 16 {
		return
	}

	scale := width / 160
	if height/80 < scale {
		scale = height / 80
	}
	if scale < 2 {
		scale = 2
	}
	if scale > 6 {
		scale = 6
	}

	charWidth := 5 * scale
	gap := scale
	textWidth := len(text)*charWidth + (len(text)-1)*gap
	textHeight := 7 * scale
	margin := 6 * scale
	x := width - textWidth - margin
	y := height - textHeight - margin
	if center {
		x = (width - textWidth) / 2
		y = (height - textHeight) / 2
	}
	if x < margin {
		x = margin
	}
	if y < margin {
		y = margin
	}

	drawBitmapText(img, text, x+scale, y+scale, scale, color.RGBA{A: 120})
	drawBitmapText(img, text, x, y, scale, color.RGBA{R: 255, G: 255, B: 255, A: 190})
}

func drawBitmapText(img *image.RGBA, text string, x, y, scale int, c color.RGBA) {
	cursor := x
	for _, ch := range strings.ToLower(text) {
		glyph, ok := watermarkGlyphs[ch]
		if !ok {
			cursor += 6 * scale
			continue
		}
		for row, pattern := range glyph {
			for col, pixel := range pattern {
				if pixel != '1' {
					continue
				}
				fillRectAlpha(img, cursor+col*scale, y+row*scale, scale, scale, c)
			}
		}
		cursor += 6 * scale
	}
}

func fillRectAlpha(img *image.RGBA, x, y, width, height int, c color.RGBA) {
	bounds := img.Bounds()
	for py := y; py < y+height; py++ {
		if py < bounds.Min.Y || py >= bounds.Max.Y {
			continue
		}
		for px := x; px < x+width; px++ {
			if px < bounds.Min.X || px >= bounds.Max.X {
				continue
			}
			blendPixel(img, px, py, c)
		}
	}
}

func blendPixel(img *image.RGBA, x, y int, overlay color.RGBA) {
	base := img.RGBAAt(x, y)
	alpha := uint32(overlay.A)
	inv := 255 - alpha
	img.SetRGBA(x, y, color.RGBA{
		R: uint8((uint32(overlay.R)*alpha + uint32(base.R)*inv) / 255),
		G: uint8((uint32(overlay.G)*alpha + uint32(base.G)*inv) / 255),
		B: uint8((uint32(overlay.B)*alpha + uint32(base.B)*inv) / 255),
		A: base.A,
	})
}

// watermarkGlyphs is a 7x5 bitmap font containing glyphs for the characters
// needed to render the watermark text. Each glyph is stored as 7 rows of
// 5-column strings where '1' indicates a filled pixel.
var watermarkGlyphs = map[rune][]string{
	'a': {
		"01110",
		"10001",
		"10001",
		"11111",
		"10001",
		"10001",
		"10001",
	},
	'g': {
		"01111",
		"10000",
		"10000",
		"10111",
		"10001",
		"10001",
		"01110",
	},
	'h': {
		"10001",
		"10001",
		"10001",
		"11111",
		"10001",
		"10001",
		"10001",
	},
	'i': {
		"11111",
		"00100",
		"00100",
		"00100",
		"00100",
		"00100",
		"11111",
	},
	'l': {
		"10000",
		"10000",
		"10000",
		"10000",
		"10000",
		"10000",
		"11111",
	},
	'n': {
		"10001",
		"11001",
		"10101",
		"10011",
		"10001",
		"10001",
		"10001",
	},
	'u': {
		"10001",
		"10001",
		"10001",
		"10001",
		"10001",
		"10001",
		"01110",
	},
	'z': {
		"11111",
		"00001",
		"00010",
		"00100",
		"01000",
		"10000",
		"11111",
	},
}
