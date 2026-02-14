package core

import "testing"

// --- Bitmap basics ---

func TestBitmapSetGet(t *testing.T) {
	bm := NewBitmap(4, 4)
	c := Color{R: 100, G: 150, B: 200}
	bm.Set(1, 2, c, 0.75)

	got, a := bm.Get(1, 2)
	if got != c {
		t.Errorf("Get color = %v, want %v", got, c)
	}
	if a != 0.75 {
		t.Errorf("Get alpha = %v, want 0.75", a)
	}
}

func TestBitmapOutOfBounds(t *testing.T) {
	bm := NewBitmap(2, 2)
	// Set out-of-bounds should not panic.
	bm.Set(-1, 0, Color{R: 255}, 1)
	bm.Set(0, -1, Color{R: 255}, 1)
	bm.Set(2, 0, Color{R: 255}, 1)
	bm.Set(0, 2, Color{R: 255}, 1)

	// Get out-of-bounds should return zero.
	c, a := bm.Get(-1, 0)
	if c != (Color{}) || a != 0 {
		t.Errorf("OOB Get should return zero, got %v, %v", c, a)
	}
	c, a = bm.Get(2, 0)
	if c != (Color{}) || a != 0 {
		t.Errorf("OOB Get should return zero, got %v, %v", c, a)
	}
}

func TestBitmapClear(t *testing.T) {
	bm := NewBitmap(3, 3)
	bm.SetDot(1, 1, Color{R: 255})
	bm.Clear()

	c, a := bm.Get(1, 1)
	if c != (Color{}) || a != 0 {
		t.Errorf("after Clear, pixel should be zero, got %v, %v", c, a)
	}
}

func TestBitmapSetDot(t *testing.T) {
	bm := NewBitmap(2, 2)
	bm.SetDot(0, 0, Color{R: 10, G: 20, B: 30})

	c, a := bm.Get(0, 0)
	if c != (Color{R: 10, G: 20, B: 30}) {
		t.Errorf("SetDot color = %v, want {10 20 30}", c)
	}
	if a != 1.0 {
		t.Errorf("SetDot alpha = %v, want 1.0", a)
	}
}

// --- Braille encoding ---

func TestBrailleSingleDot(t *testing.T) {
	// A single dot at (0,0) → braille dot1 (0x01) → U+2801.
	bm := NewBitmap(2, 4)
	bm.SetDot(0, 0, Color{R: 255})

	canvas := NewCanvas(1, 1)
	bm.DrawBraille(canvas, 0, 0)

	cell := canvas.Get(0, 0)
	if cell.Rune != '\u2801' {
		t.Errorf("single dot (0,0) rune = %U, want U+2801", cell.Rune)
	}
	if cell.FG != (Color{R: 255}) {
		t.Errorf("FG = %v, want {255 0 0}", cell.FG)
	}
	if cell.Alpha != 1.0 {
		t.Errorf("Alpha = %v, want 1.0", cell.Alpha)
	}
}

func TestBrailleFullBlock(t *testing.T) {
	// All 8 dots lit → U+28FF.
	bm := NewBitmap(2, 4)
	c := Color{R: 100, G: 100, B: 100}
	for y := range 4 {
		for x := range 2 {
			bm.SetDot(x, y, c)
		}
	}

	canvas := NewCanvas(1, 1)
	bm.DrawBraille(canvas, 0, 0)

	cell := canvas.Get(0, 0)
	if cell.Rune != '\u28FF' {
		t.Errorf("full block rune = %U, want U+28FF", cell.Rune)
	}
}

func TestBrailleTwoColorAverage(t *testing.T) {
	// Two dots with different colors: average should be computed.
	bm := NewBitmap(2, 4)
	bm.SetDot(0, 0, Color{R: 200, G: 0, B: 0})
	bm.SetDot(1, 0, Color{R: 0, G: 0, B: 100})

	canvas := NewCanvas(1, 1)
	bm.DrawBraille(canvas, 0, 0)

	cell := canvas.Get(0, 0)
	// Average of {200,0,0} and {0,0,100} = {100,0,50}
	if cell.FG.R != 100 {
		t.Errorf("FG.R = %d, want 100", cell.FG.R)
	}
	if cell.FG.B != 50 {
		t.Errorf("FG.B = %d, want 50", cell.FG.B)
	}
}

func TestBrailleEmptyBlockSkipped(t *testing.T) {
	bm := NewBitmap(2, 4)
	canvas := NewCanvas(1, 1)
	bm.DrawBraille(canvas, 0, 0)

	cell := canvas.Get(0, 0)
	if cell.Rune != 0 {
		t.Errorf("empty block should not write a cell, got rune %U", cell.Rune)
	}
	if cell.Alpha != 0 {
		t.Errorf("empty block alpha = %v, want 0", cell.Alpha)
	}
}

func TestBrailleDotPositions(t *testing.T) {
	// Verify each dot position maps to the correct bit.
	type tc struct {
		x, y int
		bit  byte
	}
	cases := []tc{
		{0, 0, 0x01},
		{0, 1, 0x02},
		{0, 2, 0x04},
		{0, 3, 0x40},
		{1, 0, 0x08},
		{1, 1, 0x10},
		{1, 2, 0x20},
		{1, 3, 0x80},
	}
	for _, tc := range cases {
		bm := NewBitmap(2, 4)
		bm.SetDot(tc.x, tc.y, Color{R: 255})

		canvas := NewCanvas(1, 1)
		bm.DrawBraille(canvas, 0, 0)

		want := rune(0x2800 | int(tc.bit))
		got := canvas.Get(0, 0).Rune
		if got != want {
			t.Errorf("dot(%d,%d): rune = %U, want %U", tc.x, tc.y, got, want)
		}
	}
}

// --- Half-block encoding ---

func TestHalfBlockBothOn(t *testing.T) {
	bm := NewBitmap(1, 2)
	bm.SetDot(0, 0, Color{R: 200, G: 0, B: 0}) // top
	bm.SetDot(0, 1, Color{R: 0, G: 0, B: 200}) // bottom

	canvas := NewCanvas(1, 1)
	bm.DrawHalfBlock(canvas, 0, 0)

	cell := canvas.Get(0, 0)
	if cell.Rune != '▀' {
		t.Errorf("both-on rune = %c, want ▀", cell.Rune)
	}
	if cell.FG != (Color{R: 200}) {
		t.Errorf("FG = %v, want top color {200 0 0}", cell.FG)
	}
	if cell.BG != (Color{B: 200}) {
		t.Errorf("BG = %v, want bottom color {0 0 200}", cell.BG)
	}
	if cell.Alpha != 1.0 {
		t.Errorf("Alpha = %v, want 1.0", cell.Alpha)
	}
}

func TestHalfBlockTopOnly(t *testing.T) {
	bm := NewBitmap(1, 2)
	bm.SetDot(0, 0, Color{R: 150})

	canvas := NewCanvas(1, 1)
	bm.DrawHalfBlock(canvas, 0, 0)

	cell := canvas.Get(0, 0)
	if cell.Rune != '▀' {
		t.Errorf("top-only rune = %c, want ▀", cell.Rune)
	}
	if cell.FG != (Color{R: 150}) {
		t.Errorf("FG = %v, want {150 0 0}", cell.FG)
	}
	if cell.BG != (Color{}) {
		t.Errorf("BG should be zero, got %v", cell.BG)
	}
}

func TestHalfBlockBottomOnly(t *testing.T) {
	bm := NewBitmap(1, 2)
	bm.SetDot(0, 1, Color{G: 180})

	canvas := NewCanvas(1, 1)
	bm.DrawHalfBlock(canvas, 0, 0)

	cell := canvas.Get(0, 0)
	if cell.Rune != '▄' {
		t.Errorf("bottom-only rune = %c, want ▄", cell.Rune)
	}
	if cell.FG != (Color{G: 180}) {
		t.Errorf("FG = %v, want {0 180 0}", cell.FG)
	}
}

func TestHalfBlockBothOff(t *testing.T) {
	bm := NewBitmap(1, 2)
	canvas := NewCanvas(1, 1)
	bm.DrawHalfBlock(canvas, 0, 0)

	cell := canvas.Get(0, 0)
	if cell.Rune != 0 || cell.Alpha != 0 {
		t.Errorf("both-off should be transparent, got rune=%U alpha=%v", cell.Rune, cell.Alpha)
	}
}

// --- BitmapDrawable ---

func TestBitmapDrawableBounds(t *testing.T) {
	bm := NewBitmap(10, 8)

	bd := &BitmapDrawable{Bitmap: bm, Mode: EncodeBraille}
	w, h := bd.Bounds()
	if w != 5 || h != 2 {
		t.Errorf("Braille bounds = (%d, %d), want (5, 2)", w, h)
	}

	bd.Mode = EncodeHalfBlock
	w, h = bd.Bounds()
	if w != 10 || h != 4 {
		t.Errorf("HalfBlock bounds = (%d, %d), want (10, 4)", w, h)
	}
}

func TestBitmapDrawableBoundsOdd(t *testing.T) {
	bm := NewBitmap(3, 5)

	bd := &BitmapDrawable{Bitmap: bm, Mode: EncodeBraille}
	w, h := bd.Bounds()
	if w != 2 || h != 2 {
		t.Errorf("Braille bounds(3,5) = (%d, %d), want (2, 2)", w, h)
	}

	bd.Mode = EncodeHalfBlock
	w, h = bd.Bounds()
	if w != 3 || h != 3 {
		t.Errorf("HalfBlock bounds(3,5) = (%d, %d), want (3, 3)", w, h)
	}
}

func TestBitmapDrawableNil(t *testing.T) {
	bd := &BitmapDrawable{Bitmap: nil, Mode: EncodeBraille}
	w, h := bd.Bounds()
	if w != 0 || h != 0 {
		t.Errorf("nil bitmap bounds = (%d, %d), want (0, 0)", w, h)
	}
	// Draw should not panic.
	canvas := NewCanvas(5, 5)
	bd.Draw(canvas, 0, 0)
}

func TestBitmapDrawableDrawWithOffset(t *testing.T) {
	bm := NewBitmap(2, 4)
	bm.SetDot(0, 0, Color{R: 255})

	canvas := NewCanvas(5, 5)
	bd := &BitmapDrawable{Bitmap: bm, Mode: EncodeBraille}
	bd.Draw(canvas, 2, 3)

	// Braille dot should appear at canvas position (2, 3).
	cell := canvas.Get(2, 3)
	if cell.Rune != '\u2801' {
		t.Errorf("offset draw rune at (2,3) = %U, want U+2801", cell.Rune)
	}
	// Original position should be empty.
	cell = canvas.Get(0, 0)
	if cell.Rune != 0 {
		t.Errorf("(0,0) should be empty, got %U", cell.Rune)
	}
}

func TestBitmapDrawableHalfBlockDraw(t *testing.T) {
	bm := NewBitmap(2, 2)
	bm.SetDot(0, 0, Color{R: 100})
	bm.SetDot(0, 1, Color{G: 100})
	bm.SetDot(1, 0, Color{B: 100})

	canvas := NewCanvas(4, 4)
	bd := &BitmapDrawable{Bitmap: bm, Mode: EncodeHalfBlock}
	bd.Draw(canvas, 1, 1)

	// col=0: both on → ▀, FG=top, BG=bottom
	cell := canvas.Get(1, 1)
	if cell.Rune != '▀' {
		t.Errorf("col0 rune = %c, want ▀", cell.Rune)
	}
	if cell.FG != (Color{R: 100}) {
		t.Errorf("col0 FG = %v, want {100 0 0}", cell.FG)
	}
	if cell.BG != (Color{G: 100}) {
		t.Errorf("col0 BG = %v, want {0 100 0}", cell.BG)
	}

	// col=1: top only → ▀, FG=top
	cell = canvas.Get(2, 1)
	if cell.Rune != '▀' {
		t.Errorf("col1 rune = %c, want ▀", cell.Rune)
	}
	if cell.FG != (Color{B: 100}) {
		t.Errorf("col1 FG = %v, want {0 0 100}", cell.FG)
	}
}
