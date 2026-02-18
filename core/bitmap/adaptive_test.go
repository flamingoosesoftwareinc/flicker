package bitmap

import (
	"testing"

	"flicker/core"
)

func TestAdaptiveEmptyCell(t *testing.T) {
	bm := New(6, 9)
	canvas := core.NewCanvas(1, 1)
	ad := &Adaptive{Bitmap: bm}
	ad.Draw(canvas, 0, 0)

	cell := canvas.Get(0, 0)
	if cell.Rune != 0 {
		t.Errorf("empty cell should not write a rune, got %U", cell.Rune)
	}
	if cell.FGAlpha != 0 {
		t.Errorf("empty cell FGAlpha = %v, want 0", cell.FGAlpha)
	}
}

func TestAdaptiveFullBlock(t *testing.T) {
	bm := New(6, 9)
	c := core.Color{R: 100, G: 100, B: 100}
	for y := range 9 {
		for x := range 6 {
			bm.SetDot(x, y, c)
		}
	}

	canvas := core.NewCanvas(1, 1)
	ad := &Adaptive{Bitmap: bm}
	ad.Draw(canvas, 0, 0)

	cell := canvas.Get(0, 0)
	if cell.Rune != '\u2588' {
		t.Errorf("full block rune = %U, want U+2588", cell.Rune)
	}
	if cell.FG != c {
		t.Errorf("FG = %v, want %v", cell.FG, c)
	}
}

func TestAdaptiveUpperHalf(t *testing.T) {
	// Fill the top 5 rows (approximate upper half in the 6×9 grid).
	bm := New(6, 9)
	c := core.Color{R: 200, G: 0, B: 0}
	for y := range 5 {
		for x := range 6 {
			bm.SetDot(x, y, c)
		}
	}

	ad := &Adaptive{Bitmap: bm}
	cell := ad.CellAt(0, 0)
	// Should match upper half block U+2580 or a similar block element.
	if cell.Rune == 0 {
		t.Error("upper half pattern should produce a rune")
	}
	if cell.Rune == '\u2588' {
		t.Error("upper half pattern should not produce full block")
	}
}

func TestAdaptiveLeftHalf(t *testing.T) {
	// Fill the left 3 columns — should match left half block or sextant.
	bm := New(6, 9)
	c := core.Color{R: 0, G: 255, B: 0}
	for y := range 9 {
		for x := range 3 {
			bm.SetDot(x, y, c)
		}
	}

	ad := &Adaptive{Bitmap: bm}
	cell := ad.CellAt(0, 0)
	// Should match left half block U+258C.
	if cell.Rune != '\u258C' {
		t.Errorf("left half rune = %U, want U+258C", cell.Rune)
	}
}

func TestAdaptiveBounds(t *testing.T) {
	bm := New(12, 18)
	ad := &Adaptive{Bitmap: bm}
	w, h := ad.Bounds()
	if w != 2 || h != 2 {
		t.Errorf("Adaptive bounds(12,18) = (%d, %d), want (2, 2)", w, h)
	}
}

func TestAdaptiveBoundsOdd(t *testing.T) {
	bm := New(7, 10)
	ad := &Adaptive{Bitmap: bm}
	w, h := ad.Bounds()
	if w != 2 || h != 2 {
		t.Errorf("Adaptive bounds(7,10) = (%d, %d), want (2, 2)", w, h)
	}
}

func TestAdaptiveNil(t *testing.T) {
	ad := &Adaptive{Bitmap: nil}
	w, h := ad.Bounds()
	if w != 0 || h != 0 {
		t.Errorf("nil bitmap bounds = (%d, %d), want (0, 0)", w, h)
	}
	// Draw should not panic.
	canvas := core.NewCanvas(5, 5)
	ad.Draw(canvas, 0, 0)
}

func TestAdaptiveDrawWithOffset(t *testing.T) {
	bm := New(6, 9)
	for y := range 9 {
		for x := range 6 {
			bm.SetDot(x, y, core.Color{R: 255})
		}
	}

	canvas := core.NewCanvas(5, 5)
	ad := &Adaptive{Bitmap: bm}
	ad.Draw(canvas, 2, 3)

	cell := canvas.Get(2, 3)
	if cell.Rune != '\u2588' {
		t.Errorf("offset draw rune at (2,3) = %U, want U+2588", cell.Rune)
	}
	cell = canvas.Get(0, 0)
	if cell.Rune != 0 {
		t.Errorf("(0,0) should be empty, got %U", cell.Rune)
	}
}

func TestAdaptiveDiagonalPattern(t *testing.T) {
	// Fill a diagonal pattern (lower-left triangle).
	bm := New(6, 9)
	c := core.Color{R: 128, G: 128, B: 128}
	for y := range 9 {
		for x := range 6 {
			// Fill if below the diagonal from (0,0) to (6,9).
			if float64(y)/9.0 > float64(x)/6.0 {
				bm.SetDot(x, y, c)
			}
		}
	}

	ad := &Adaptive{Bitmap: bm}
	cell := ad.CellAt(0, 0)
	if cell.Rune == 0 {
		t.Error("diagonal pattern should produce a rune")
	}
	if cell.Rune == ' ' {
		t.Error("diagonal pattern should not produce space")
	}
}

func TestAdaptiveAlphaThreshold(t *testing.T) {
	// Fill one cell with low-alpha pixels in the right half.
	// Without threshold, sextant should include those pixels.
	// With threshold, they should be ignored.
	bm := New(6, 9)
	solid := core.Color{R: 255}
	// Left 3 columns: fully opaque.
	for y := range 9 {
		for x := range 3 {
			bm.SetDot(x, y, solid)
		}
	}
	// Right 3 columns: low alpha (antialiased fringe).
	for y := range 9 {
		for x := 3; x < 6; x++ {
			idx := y*bm.Width + x
			bm.Pix[idx] = solid
			bm.Alpha[idx] = 0.1
		}
	}

	// Without threshold: both halves count → full block.
	ad := &Adaptive{Bitmap: bm}
	cell := ad.CellAt(0, 0)
	if cell.Rune != '\u2588' {
		t.Errorf("no threshold: expected full block U+2588, got %U", cell.Rune)
	}

	// With threshold 0.3: right half is below threshold → left half block.
	ad = &Adaptive{Bitmap: bm, AlphaThreshold: 0.3}
	cell = ad.CellAt(0, 0)
	if cell.Rune != '\u258C' {
		t.Errorf("threshold 0.3: expected left half U+258C, got %U", cell.Rune)
	}
}

func TestCandidatesNotEmpty(t *testing.T) {
	// 63 sextants (1–63, no space) + 2 half blocks + 10 quadrants = 75
	if len(candidates) < 75 {
		t.Errorf("expected at least 75 candidates, got %d", len(candidates))
	}
}

func TestBestMatchFullBits(t *testing.T) {
	// All 54 bits set should match full block.
	var full uint64
	for i := range 54 {
		full |= 1 << uint(i)
	}
	r, dist := bestMatch(full)
	if r != '\u2588' {
		t.Errorf("full pattern rune = %U, want U+2588", r)
	}
	if dist != 0 {
		t.Errorf("full pattern distance = %d, want 0", dist)
	}
}

func TestBestMatchEmpty(t *testing.T) {
	// With space removed from candidates, bestMatch(0) returns the
	// candidate with the fewest set bits. Callers handle empty patterns
	// before calling bestMatch (sampleCell returns 0 → early return).
	r, dist := bestMatch(0)
	if r == 0 {
		t.Error("empty pattern should still return a rune")
	}
	if dist == 0 {
		t.Error("empty pattern should have non-zero distance (no space candidate)")
	}
}
