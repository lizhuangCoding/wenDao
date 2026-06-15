package watermark

import (
	"image"
	"image/color"
	"testing"
)

func buildSolidImage(width, height int, bg color.RGBA) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.SetRGBA(x, y, bg)
		}
	}
	return img
}

func unchangedPixelCount(img image.Image, expected color.RGBA) int {
	count := 0
	bounds := img.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := img.At(x, y).RGBA()
			if uint8(r>>8) == expected.R && uint8(g>>8) == expected.G && uint8(b>>8) == expected.B && uint8(a>>8) == expected.A {
				count++
			}
		}
	}
	return count
}

func TestApplyModeNone(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 10, 10))
	result, err := Apply(img, "test", ModeNone)
	if err != nil {
		t.Fatalf("unexpected error for ModeNone: %v", err)
	}
	if result != img {
		t.Fatal("expected same image reference for ModeNone")
	}
}

func TestApplyEmptyText(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 10, 10))
	_, err := Apply(img, "", ModeText)
	if err == nil {
		t.Fatal("expected error for empty text")
	}
}

func TestApplyModeText(t *testing.T) {
	bg := color.RGBA{R: 20, G: 80, B: 140, A: 255}
	img := buildSolidImage(240, 120, bg)
	result, err := Apply(img, "lizhuang", ModeText)
	if err != nil {
		t.Fatalf("unexpected error for ModeText: %v", err)
	}
	totalPixels := result.Bounds().Dx() * result.Bounds().Dy()
	if unchangedPixelCount(result, bg) == totalPixels {
		t.Fatal("expected watermark to change pixels")
	}
	rgba, ok := result.(*image.RGBA)
	if !ok {
		t.Fatal("expected result to be *image.RGBA")
	}
	if rgba.Bounds().Dx() != 240 || rgba.Bounds().Dy() != 120 {
		t.Fatalf("expected dimensions 240x120, got %dx%d", rgba.Bounds().Dx(), rgba.Bounds().Dy())
	}
}

func TestApplyModeTile(t *testing.T) {
	bg := color.RGBA{R: 20, G: 80, B: 140, A: 255}
	img := buildSolidImage(240, 120, bg)
	result, err := Apply(img, "lizhuang", ModeTile)
	if err != nil {
		t.Fatalf("unexpected error for ModeTile: %v", err)
	}
	totalPixels := result.Bounds().Dx() * result.Bounds().Dy()
	if unchangedPixelCount(result, bg) == totalPixels {
		t.Fatal("expected watermark to change pixels")
	}
}

func TestApplySmallImage(t *testing.T) {
	bg := color.RGBA{R: 20, G: 80, B: 140, A: 255}
	img := buildSolidImage(20, 10, bg)
	result, err := Apply(img, "lizhuang", ModeText)
	if err != nil {
		t.Fatalf("unexpected error for small image: %v", err)
	}
	totalPixels := result.Bounds().Dx() * result.Bounds().Dy()
	if unchangedPixelCount(result, bg) != totalPixels {
		t.Fatal("small image should not be watermarked")
	}
}

func TestApplyUnknownMode(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	_, err := Apply(img, "test", Mode(99))
	if err == nil {
		t.Fatal("expected error for unknown mode")
	}
}
