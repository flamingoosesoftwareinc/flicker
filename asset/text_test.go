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

	bm := RasterizeText("I", TextOptions{
		Font:  f,
		Size:  48,
		Color: core.Color{R: 255, G: 255, B: 255},
	})

	if bm == nil {
		t.Fatal("RasterizeText returned nil")
	}

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
}

func TestRasterizeTextMultiChar(t *testing.T) {
	f, err := LoadFont("../Oxanium/static/Oxanium-Regular.ttf")
	if err != nil {
		t.Fatalf("LoadFont: %v", err)
	}

	bmA := RasterizeText("A", TextOptions{Font: f, Size: 48, Color: core.Color{R: 255}})
	bmAB := RasterizeText("AB", TextOptions{Font: f, Size: 48, Color: core.Color{R: 255}})

	if bmA == nil || bmAB == nil {
		t.Fatal("RasterizeText returned nil")
	}

	if bmAB.Width <= bmA.Width {
		t.Errorf("AB width (%d) should be greater than A width (%d)", bmAB.Width, bmA.Width)
	}
}

func TestRasterizeTextEmpty(t *testing.T) {
	f, err := LoadFont("../Oxanium/static/Oxanium-Regular.ttf")
	if err != nil {
		t.Fatalf("LoadFont: %v", err)
	}

	bm := RasterizeText("", TextOptions{Font: f, Size: 48, Color: core.Color{R: 255}})
	if bm != nil {
		t.Errorf("expected nil for empty string, got bitmap %dx%d", bm.Width, bm.Height)
	}
}
