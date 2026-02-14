package bitmap

import (
	"testing"

	"flicker/core"
)

func TestFullBlockDraw(t *testing.T) {
	bm := New(2, 2)
	bm.Set(0, 0, core.Color{R: 200}, 1.0)
	bm.Set(1, 0, core.Color{G: 150}, 0.8)
	bm.Set(0, 1, core.Color{B: 100}, 0.5)
	// (1,1) left empty

	canvas := core.NewCanvas(4, 4)
	fb := &FullBlock{Bitmap: bm}
	fb.Draw(canvas, 1, 1)

	cell := canvas.Get(1, 1)
	if cell.Rune != '█' {
		t.Errorf("(0,0) rune = %c, want █", cell.Rune)
	}
	if cell.FG != (core.Color{R: 200}) {
		t.Errorf("(0,0) FG = %v, want {200 0 0}", cell.FG)
	}
	if cell.FGAlpha != 1.0 {
		t.Errorf("(0,0) FGAlpha = %v, want 1.0", cell.FGAlpha)
	}

	cell = canvas.Get(2, 1)
	if cell.Rune != '█' || cell.FG != (core.Color{G: 150}) || cell.FGAlpha != 0.8 {
		t.Errorf("(1,0) = %v, want █ {0 150 0} 0.8", cell)
	}

	cell = canvas.Get(1, 2)
	if cell.Rune != '█' || cell.FG != (core.Color{B: 100}) || cell.FGAlpha != 0.5 {
		t.Errorf("(0,1) = %v, want █ {0 0 100} 0.5", cell)
	}

	// Empty pixel should not produce a cell.
	cell = canvas.Get(2, 2)
	if cell.Rune != 0 || cell.FGAlpha != 0 {
		t.Errorf(
			"empty pixel should be transparent, got rune=%U FGAlpha=%v",
			cell.Rune,
			cell.FGAlpha,
		)
	}
}

func TestFullBlockCellAt(t *testing.T) {
	bm := New(3, 3)
	bm.Set(1, 2, core.Color{R: 50, G: 100, B: 150}, 0.9)

	fb := &FullBlock{Bitmap: bm}
	cell := fb.CellAt(1, 2)
	if cell.Rune != '█' {
		t.Errorf("rune = %c, want █", cell.Rune)
	}
	if cell.FG != (core.Color{R: 50, G: 100, B: 150}) {
		t.Errorf("FG = %v, want {50 100 150}", cell.FG)
	}
	if cell.FGAlpha != 0.9 {
		t.Errorf("FGAlpha = %v, want 0.9", cell.FGAlpha)
	}

	// Empty pixel returns zero cell.
	cell = fb.CellAt(0, 0)
	if cell.Rune != 0 || cell.FGAlpha != 0 {
		t.Errorf("empty pixel should return zero cell, got %v", cell)
	}
}

func TestFullBlockBounds(t *testing.T) {
	bm := New(10, 8)
	fb := &FullBlock{Bitmap: bm}
	w, h := fb.Bounds()
	if w != 10 || h != 8 {
		t.Errorf("FullBlock bounds = (%d, %d), want (10, 8)", w, h)
	}
}

func TestFullBlockBoundsOdd(t *testing.T) {
	bm := New(3, 5)
	fb := &FullBlock{Bitmap: bm}
	w, h := fb.Bounds()
	if w != 3 || h != 5 {
		t.Errorf("FullBlock bounds(3,5) = (%d, %d), want (3, 5)", w, h)
	}
}

func TestFullBlockDrawViaDrawable(t *testing.T) {
	bm := New(2, 2)
	bm.SetDot(0, 0, core.Color{R: 255})
	bm.SetDot(1, 1, core.Color{G: 255})

	canvas := core.NewCanvas(4, 4)
	fb := &FullBlock{Bitmap: bm}
	fb.Draw(canvas, 1, 1)

	cell := canvas.Get(1, 1)
	if cell.Rune != '█' || cell.FG != (core.Color{R: 255}) {
		t.Errorf("(1,1) = %v, want █ {255 0 0}", cell)
	}

	cell = canvas.Get(2, 2)
	if cell.Rune != '█' || cell.FG != (core.Color{G: 255}) {
		t.Errorf("(2,2) = %v, want █ {0 255 0}", cell)
	}

	// Empty pixels should not produce cells.
	cell = canvas.Get(2, 1)
	if cell.Rune != 0 {
		t.Errorf("(2,1) should be empty, got %U", cell.Rune)
	}
	cell = canvas.Get(1, 2)
	if cell.Rune != 0 {
		t.Errorf("(1,2) should be empty, got %U", cell.Rune)
	}
}
