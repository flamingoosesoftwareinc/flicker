package bitmap

import (
	"testing"

	"flicker/core"
)

func TestBGBlockDraw(t *testing.T) {
	bm := New(2, 2)
	bm.Set(0, 0, core.Color{R: 200}, 1.0)
	bm.Set(1, 0, core.Color{G: 150}, 0.8)
	bm.Set(0, 1, core.Color{B: 100}, 0.5)
	// (1,1) left empty

	canvas := core.NewCanvas(4, 4)
	bg := &BGBlock{Bitmap: bm}
	bg.Draw(canvas, 1, 1)

	cell := canvas.Get(1, 1)
	if cell.Rune != ' ' {
		t.Errorf("(0,0) rune = %c, want ' '", cell.Rune)
	}
	if cell.BG != (core.Color{R: 200}) {
		t.Errorf("(0,0) BG = %v, want {200 0 0}", cell.BG)
	}
	if cell.BGAlpha != 1.0 {
		t.Errorf("(0,0) BGAlpha = %v, want 1.0", cell.BGAlpha)
	}
	if cell.FGAlpha != 0 {
		t.Errorf("(0,0) FGAlpha = %v, want 0", cell.FGAlpha)
	}

	cell = canvas.Get(2, 1)
	if cell.Rune != ' ' || cell.BG != (core.Color{G: 150}) || cell.BGAlpha != 0.8 {
		t.Errorf("(1,0) = %v, want ' ' {0 150 0} 0.8", cell)
	}

	cell = canvas.Get(1, 2)
	if cell.Rune != ' ' || cell.BG != (core.Color{B: 100}) || cell.BGAlpha != 0.5 {
		t.Errorf("(0,1) = %v, want ' ' {0 0 100} 0.5", cell)
	}

	// Empty pixel should not produce a cell.
	cell = canvas.Get(2, 2)
	if cell.Rune != 0 || cell.BGAlpha != 0 {
		t.Errorf(
			"empty pixel should be transparent, got rune=%U BGAlpha=%v",
			cell.Rune,
			cell.BGAlpha,
		)
	}
}

func TestBGBlockCellAt(t *testing.T) {
	bm := New(3, 3)
	bm.Set(1, 2, core.Color{R: 50, G: 100, B: 150}, 0.9)

	bg := &BGBlock{Bitmap: bm}
	cell := bg.CellAt(1, 2)
	if cell.Rune != ' ' {
		t.Errorf("rune = %c, want ' '", cell.Rune)
	}
	if cell.BG != (core.Color{R: 50, G: 100, B: 150}) {
		t.Errorf("BG = %v, want {50 100 150}", cell.BG)
	}
	if cell.BGAlpha != 0.9 {
		t.Errorf("BGAlpha = %v, want 0.9", cell.BGAlpha)
	}
	if cell.FGAlpha != 0 {
		t.Errorf("FGAlpha = %v, want 0", cell.FGAlpha)
	}

	// Empty pixel returns zero cell.
	cell = bg.CellAt(0, 0)
	if cell.Rune != 0 || cell.BGAlpha != 0 {
		t.Errorf("empty pixel should return zero cell, got %v", cell)
	}
}

func TestBGBlockBounds(t *testing.T) {
	bm := New(10, 8)
	bg := &BGBlock{Bitmap: bm}
	w, h := bg.Bounds()
	if w != 10 || h != 8 {
		t.Errorf("BGBlock bounds = (%d, %d), want (10, 8)", w, h)
	}
}

func TestBGBlockBoundsOdd(t *testing.T) {
	bm := New(3, 5)
	bg := &BGBlock{Bitmap: bm}
	w, h := bg.Bounds()
	if w != 3 || h != 5 {
		t.Errorf("BGBlock bounds(3,5) = (%d, %d), want (3, 5)", w, h)
	}
}

func TestBGBlockDrawViaDrawable(t *testing.T) {
	bm := New(2, 2)
	bm.SetDot(0, 0, core.Color{R: 255})
	bm.SetDot(1, 1, core.Color{G: 255})

	canvas := core.NewCanvas(4, 4)
	bg := &BGBlock{Bitmap: bm}
	bg.Draw(canvas, 1, 1)

	cell := canvas.Get(1, 1)
	if cell.Rune != ' ' || cell.BG != (core.Color{R: 255}) {
		t.Errorf("(1,1) = %v, want ' ' BG={255 0 0}", cell)
	}
	if cell.FGAlpha != 0 {
		t.Errorf("(1,1) FGAlpha = %v, want 0", cell.FGAlpha)
	}

	cell = canvas.Get(2, 2)
	if cell.Rune != ' ' || cell.BG != (core.Color{G: 255}) {
		t.Errorf("(2,2) = %v, want ' ' BG={0 255 0}", cell)
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
