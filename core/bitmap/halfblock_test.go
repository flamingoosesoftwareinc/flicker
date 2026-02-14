package bitmap

import (
	"testing"

	"flicker/core"
)

func TestHalfBlockBothOn(t *testing.T) {
	bm := New(1, 2)
	bm.SetDot(0, 0, core.Color{R: 200, G: 0, B: 0}) // top
	bm.SetDot(0, 1, core.Color{R: 0, G: 0, B: 200}) // bottom

	canvas := core.NewCanvas(1, 1)
	hb := &HalfBlock{Bitmap: bm}
	hb.Draw(canvas, 0, 0)

	cell := canvas.Get(0, 0)
	if cell.Rune != '▀' {
		t.Errorf("both-on rune = %c, want ▀", cell.Rune)
	}
	if cell.FG != (core.Color{R: 200}) {
		t.Errorf("FG = %v, want top color {200 0 0}", cell.FG)
	}
	if cell.BG != (core.Color{B: 200}) {
		t.Errorf("BG = %v, want bottom color {0 0 200}", cell.BG)
	}
	if cell.FGAlpha != 1.0 {
		t.Errorf("FGAlpha = %v, want 1.0", cell.FGAlpha)
	}
	if cell.BGAlpha != 1.0 {
		t.Errorf("BGAlpha = %v, want 1.0", cell.BGAlpha)
	}
}

func TestHalfBlockTopOnly(t *testing.T) {
	bm := New(1, 2)
	bm.SetDot(0, 0, core.Color{R: 150})

	canvas := core.NewCanvas(1, 1)
	hb := &HalfBlock{Bitmap: bm}
	hb.Draw(canvas, 0, 0)

	cell := canvas.Get(0, 0)
	if cell.Rune != '▀' {
		t.Errorf("top-only rune = %c, want ▀", cell.Rune)
	}
	if cell.FG != (core.Color{R: 150}) {
		t.Errorf("FG = %v, want {150 0 0}", cell.FG)
	}
	if cell.BG != (core.Color{}) {
		t.Errorf("BG should be zero, got %v", cell.BG)
	}
}

func TestHalfBlockBottomOnly(t *testing.T) {
	bm := New(1, 2)
	bm.SetDot(0, 1, core.Color{G: 180})

	canvas := core.NewCanvas(1, 1)
	hb := &HalfBlock{Bitmap: bm}
	hb.Draw(canvas, 0, 0)

	cell := canvas.Get(0, 0)
	if cell.Rune != '▄' {
		t.Errorf("bottom-only rune = %c, want ▄", cell.Rune)
	}
	if cell.FG != (core.Color{G: 180}) {
		t.Errorf("FG = %v, want {0 180 0}", cell.FG)
	}
}

func TestHalfBlockBothOff(t *testing.T) {
	bm := New(1, 2)
	canvas := core.NewCanvas(1, 1)
	hb := &HalfBlock{Bitmap: bm}
	hb.Draw(canvas, 0, 0)

	cell := canvas.Get(0, 0)
	if cell.Rune != 0 || cell.FGAlpha != 0 || cell.BGAlpha != 0 {
		t.Errorf(
			"both-off should be transparent, got rune=%U FGAlpha=%v BGAlpha=%v",
			cell.Rune,
			cell.FGAlpha,
			cell.BGAlpha,
		)
	}
}

func TestHalfBlockBounds(t *testing.T) {
	bm := New(10, 8)
	hb := &HalfBlock{Bitmap: bm}
	w, h := hb.Bounds()
	if w != 10 || h != 4 {
		t.Errorf("HalfBlock bounds = (%d, %d), want (10, 4)", w, h)
	}
}

func TestHalfBlockBoundsOdd(t *testing.T) {
	bm := New(3, 5)
	hb := &HalfBlock{Bitmap: bm}
	w, h := hb.Bounds()
	if w != 3 || h != 3 {
		t.Errorf("HalfBlock bounds(3,5) = (%d, %d), want (3, 3)", w, h)
	}
}

func TestHalfBlockDraw(t *testing.T) {
	bm := New(2, 2)
	bm.SetDot(0, 0, core.Color{R: 100})
	bm.SetDot(0, 1, core.Color{G: 100})
	bm.SetDot(1, 0, core.Color{B: 100})

	canvas := core.NewCanvas(4, 4)
	hb := &HalfBlock{Bitmap: bm}
	hb.Draw(canvas, 1, 1)

	// col=0: both on → ▀, FG=top, BG=bottom
	cell := canvas.Get(1, 1)
	if cell.Rune != '▀' {
		t.Errorf("col0 rune = %c, want ▀", cell.Rune)
	}
	if cell.FG != (core.Color{R: 100}) {
		t.Errorf("col0 FG = %v, want {100 0 0}", cell.FG)
	}
	if cell.BG != (core.Color{G: 100}) {
		t.Errorf("col0 BG = %v, want {0 100 0}", cell.BG)
	}

	// col=1: top only → ▀, FG=top
	cell = canvas.Get(2, 1)
	if cell.Rune != '▀' {
		t.Errorf("col1 rune = %c, want ▀", cell.Rune)
	}
	if cell.FG != (core.Color{B: 100}) {
		t.Errorf("col1 FG = %v, want {0 0 100}", cell.FG)
	}
}
