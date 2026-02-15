package asset

import (
	"math"
	"testing"

	"flicker/core"
	"golang.org/x/image/font"
)

func TestRasterizeTextI(t *testing.T) {
	f, err := LoadFont("../Oxanium/static/Oxanium-Regular.ttf")
	if err != nil {
		t.Fatalf("LoadFont: %v", err)
	}

	layout := RasterizeText("I", TextOptions{
		Font:  f,
		Size:  48,
		Color: core.Color{R: 255, G: 255, B: 255},
	})

	if layout == nil {
		t.Fatal("RasterizeText returned nil")
	}

	bm := layout.Bitmap
	m := f.Metrics(48)
	expectedH := int(math.Ceil(m.Height))
	if bm.Height != expectedH {
		t.Errorf("height: got %d, want %d", bm.Height, expectedH)
	}

	// Verify that some pixels along the vertical stroke have alpha > 0.
	midX := bm.Width / 2
	found := false
	for y := 0; y < bm.Height; y++ {
		_, a := bm.Get(midX, y)
		if a > 0 {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("no non-zero alpha pixels found along vertical center of 'I'")
	}

	// Width should roughly match the advance of 'I'.
	ppem := fixed26_6(48)
	gi, _ := f.SFNTFont().GlyphIndex(f.Buffer(), 'I')
	adv, _ := f.SFNTFont().GlyphAdvance(f.Buffer(), gi, ppem, font.HintingNone)
	advPx := float64(adv) / 64.0
	if math.Abs(float64(bm.Width)-advPx) > 2 {
		t.Errorf("width: got %d, want ~%f", bm.Width, advPx)
	}

	// Verify glyph layout.
	if len(layout.Glyphs) != 1 {
		t.Errorf("expected 1 glyph, got %d", len(layout.Glyphs))
	}
	if len(layout.Glyphs) > 0 && layout.Glyphs[0].Rune != 'I' {
		t.Errorf("expected glyph 'I', got %q", layout.Glyphs[0].Rune)
	}
}

func TestRasterizeTextMultiChar(t *testing.T) {
	f, err := LoadFont("../Oxanium/static/Oxanium-Regular.ttf")
	if err != nil {
		t.Fatalf("LoadFont: %v", err)
	}

	layoutA := RasterizeText("A", TextOptions{Font: f, Size: 48, Color: core.Color{R: 255}})
	layoutAB := RasterizeText("AB", TextOptions{Font: f, Size: 48, Color: core.Color{R: 255}})

	if layoutA == nil || layoutAB == nil {
		t.Fatal("RasterizeText returned nil")
	}

	if layoutAB.Bitmap.Width <= layoutA.Bitmap.Width {
		t.Errorf(
			"AB width (%d) should be greater than A width (%d)",
			layoutAB.Bitmap.Width,
			layoutA.Bitmap.Width,
		)
	}

	// Verify glyph counts.
	if len(layoutA.Glyphs) != 1 {
		t.Errorf("expected 1 glyph for 'A', got %d", len(layoutA.Glyphs))
	}
	if len(layoutAB.Glyphs) != 2 {
		t.Errorf("expected 2 glyphs for 'AB', got %d", len(layoutAB.Glyphs))
	}
}

func TestRasterizeTextEmpty(t *testing.T) {
	f, err := LoadFont("../Oxanium/static/Oxanium-Regular.ttf")
	if err != nil {
		t.Fatalf("LoadFont: %v", err)
	}

	layout := RasterizeText("", TextOptions{Font: f, Size: 48, Color: core.Color{R: 255}})
	if layout != nil {
		t.Errorf(
			"expected nil for empty string, got layout with bitmap %dx%d",
			layout.Bitmap.Width,
			layout.Bitmap.Height,
		)
	}
}
