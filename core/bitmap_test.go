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
	if cell.FGAlpha != 1.0 {
		t.Errorf("FGAlpha = %v, want 1.0", cell.FGAlpha)
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
	if cell.FGAlpha != 0 {
		t.Errorf("empty block FGAlpha = %v, want 0", cell.FGAlpha)
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
	if cell.FGAlpha != 1.0 {
		t.Errorf("FGAlpha = %v, want 1.0", cell.FGAlpha)
	}
	if cell.BGAlpha != 1.0 {
		t.Errorf("BGAlpha = %v, want 1.0", cell.BGAlpha)
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
	if cell.Rune != 0 || cell.FGAlpha != 0 || cell.BGAlpha != 0 {
		t.Errorf(
			"both-off should be transparent, got rune=%U FGAlpha=%v BGAlpha=%v",
			cell.Rune,
			cell.FGAlpha,
			cell.BGAlpha,
		)
	}
}

// --- Line drawing ---

func TestLineHorizontal(t *testing.T) {
	bm := NewBitmap(10, 10)
	bm.Line(1, 5, 8, 5, Color{R: 255})

	for x := 1; x <= 8; x++ {
		_, a := bm.Get(x, 5)
		if a == 0 {
			t.Errorf("pixel (%d,5) should be set", x)
		}
	}
	// Outside the line.
	_, a := bm.Get(0, 5)
	if a != 0 {
		t.Error("pixel (0,5) should not be set")
	}
}

func TestLineVertical(t *testing.T) {
	bm := NewBitmap(10, 10)
	bm.Line(3, 2, 3, 7, Color{G: 255})

	for y := 2; y <= 7; y++ {
		_, a := bm.Get(3, y)
		if a == 0 {
			t.Errorf("pixel (3,%d) should be set", y)
		}
	}
}

func TestLineDiagonal(t *testing.T) {
	bm := NewBitmap(10, 10)
	bm.Line(0, 0, 9, 9, Color{B: 255})

	// Diagonal should hit (0,0) and (9,9) at minimum.
	_, a := bm.Get(0, 0)
	if a == 0 {
		t.Error("start pixel should be set")
	}
	_, a = bm.Get(9, 9)
	if a == 0 {
		t.Error("end pixel should be set")
	}
}

func TestLineSinglePoint(t *testing.T) {
	bm := NewBitmap(5, 5)
	bm.Line(2, 3, 2, 3, Color{R: 100})

	_, a := bm.Get(2, 3)
	if a == 0 {
		t.Error("single-point line should set pixel")
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

	bd.Mode = EncodeFullBlock
	w, h = bd.Bounds()
	if w != 10 || h != 8 {
		t.Errorf("FullBlock bounds = (%d, %d), want (10, 8)", w, h)
	}

	bd.Mode = EncodeBGBlock
	w, h = bd.Bounds()
	if w != 10 || h != 8 {
		t.Errorf("BGBlock bounds = (%d, %d), want (10, 8)", w, h)
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

	bd.Mode = EncodeFullBlock
	w, h = bd.Bounds()
	if w != 3 || h != 5 {
		t.Errorf("FullBlock bounds(3,5) = (%d, %d), want (3, 5)", w, h)
	}

	bd.Mode = EncodeBGBlock
	w, h = bd.Bounds()
	if w != 3 || h != 5 {
		t.Errorf("BGBlock bounds(3,5) = (%d, %d), want (3, 5)", w, h)
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

// --- Full-block encoding ---

func TestFullBlockDraw(t *testing.T) {
	bm := NewBitmap(2, 2)
	bm.Set(0, 0, Color{R: 200}, 1.0)
	bm.Set(1, 0, Color{G: 150}, 0.8)
	bm.Set(0, 1, Color{B: 100}, 0.5)
	// (1,1) left empty

	canvas := NewCanvas(4, 4)
	bm.DrawFullBlock(canvas, 1, 1)

	cell := canvas.Get(1, 1)
	if cell.Rune != '█' {
		t.Errorf("(0,0) rune = %c, want █", cell.Rune)
	}
	if cell.FG != (Color{R: 200}) {
		t.Errorf("(0,0) FG = %v, want {200 0 0}", cell.FG)
	}
	if cell.FGAlpha != 1.0 {
		t.Errorf("(0,0) FGAlpha = %v, want 1.0", cell.FGAlpha)
	}

	cell = canvas.Get(2, 1)
	if cell.Rune != '█' || cell.FG != (Color{G: 150}) || cell.FGAlpha != 0.8 {
		t.Errorf("(1,0) = %v, want █ {0 150 0} 0.8", cell)
	}

	cell = canvas.Get(1, 2)
	if cell.Rune != '█' || cell.FG != (Color{B: 100}) || cell.FGAlpha != 0.5 {
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
	bm := NewBitmap(3, 3)
	bm.Set(1, 2, Color{R: 50, G: 100, B: 150}, 0.9)

	cell := bm.FullBlockCellAt(1, 2)
	if cell.Rune != '█' {
		t.Errorf("rune = %c, want █", cell.Rune)
	}
	if cell.FG != (Color{R: 50, G: 100, B: 150}) {
		t.Errorf("FG = %v, want {50 100 150}", cell.FG)
	}
	if cell.FGAlpha != 0.9 {
		t.Errorf("FGAlpha = %v, want 0.9", cell.FGAlpha)
	}

	// Empty pixel returns zero cell.
	cell = bm.FullBlockCellAt(0, 0)
	if cell.Rune != 0 || cell.FGAlpha != 0 {
		t.Errorf("empty pixel should return zero cell, got %v", cell)
	}
}

func TestBitmapDrawableFullBlockBounds(t *testing.T) {
	bm := NewBitmap(10, 8)
	bd := &BitmapDrawable{Bitmap: bm, Mode: EncodeFullBlock}
	w, h := bd.Bounds()
	if w != 10 || h != 8 {
		t.Errorf("FullBlock bounds = (%d, %d), want (10, 8)", w, h)
	}
}

// --- BG-block encoding ---

func TestBGBlockDraw(t *testing.T) {
	bm := NewBitmap(2, 2)
	bm.Set(0, 0, Color{R: 200}, 1.0)
	bm.Set(1, 0, Color{G: 150}, 0.8)
	bm.Set(0, 1, Color{B: 100}, 0.5)
	// (1,1) left empty

	canvas := NewCanvas(4, 4)
	bm.DrawBGBlock(canvas, 1, 1)

	cell := canvas.Get(1, 1)
	if cell.Rune != ' ' {
		t.Errorf("(0,0) rune = %c, want ' '", cell.Rune)
	}
	if cell.BG != (Color{R: 200}) {
		t.Errorf("(0,0) BG = %v, want {200 0 0}", cell.BG)
	}
	if cell.BGAlpha != 1.0 {
		t.Errorf("(0,0) BGAlpha = %v, want 1.0", cell.BGAlpha)
	}
	if cell.FGAlpha != 0 {
		t.Errorf("(0,0) FGAlpha = %v, want 0", cell.FGAlpha)
	}

	cell = canvas.Get(2, 1)
	if cell.Rune != ' ' || cell.BG != (Color{G: 150}) || cell.BGAlpha != 0.8 {
		t.Errorf("(1,0) = %v, want ' ' {0 150 0} 0.8", cell)
	}

	cell = canvas.Get(1, 2)
	if cell.Rune != ' ' || cell.BG != (Color{B: 100}) || cell.BGAlpha != 0.5 {
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
	bm := NewBitmap(3, 3)
	bm.Set(1, 2, Color{R: 50, G: 100, B: 150}, 0.9)

	cell := bm.BGBlockCellAt(1, 2)
	if cell.Rune != ' ' {
		t.Errorf("rune = %c, want ' '", cell.Rune)
	}
	if cell.BG != (Color{R: 50, G: 100, B: 150}) {
		t.Errorf("BG = %v, want {50 100 150}", cell.BG)
	}
	if cell.BGAlpha != 0.9 {
		t.Errorf("BGAlpha = %v, want 0.9", cell.BGAlpha)
	}
	if cell.FGAlpha != 0 {
		t.Errorf("FGAlpha = %v, want 0", cell.FGAlpha)
	}

	// Empty pixel returns zero cell.
	cell = bm.BGBlockCellAt(0, 0)
	if cell.Rune != 0 || cell.BGAlpha != 0 {
		t.Errorf("empty pixel should return zero cell, got %v", cell)
	}
}

func TestBitmapDrawableBGBlockBounds(t *testing.T) {
	bm := NewBitmap(10, 8)
	bd := &BitmapDrawable{Bitmap: bm, Mode: EncodeBGBlock}
	w, h := bd.Bounds()
	if w != 10 || h != 8 {
		t.Errorf("BGBlock bounds = (%d, %d), want (10, 8)", w, h)
	}
}

func TestBitmapDrawableBGBlockDraw(t *testing.T) {
	bm := NewBitmap(2, 2)
	bm.SetDot(0, 0, Color{R: 255})
	bm.SetDot(1, 1, Color{G: 255})

	canvas := NewCanvas(4, 4)
	bd := &BitmapDrawable{Bitmap: bm, Mode: EncodeBGBlock}
	bd.Draw(canvas, 1, 1)

	cell := canvas.Get(1, 1)
	if cell.Rune != ' ' || cell.BG != (Color{R: 255}) {
		t.Errorf("(1,1) = %v, want ' ' BG={255 0 0}", cell)
	}
	if cell.FGAlpha != 0 {
		t.Errorf("(1,1) FGAlpha = %v, want 0", cell.FGAlpha)
	}

	cell = canvas.Get(2, 2)
	if cell.Rune != ' ' || cell.BG != (Color{G: 255}) {
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

func TestBitmapDrawableFullBlockDraw(t *testing.T) {
	bm := NewBitmap(2, 2)
	bm.SetDot(0, 0, Color{R: 255})
	bm.SetDot(1, 1, Color{G: 255})

	canvas := NewCanvas(4, 4)
	bd := &BitmapDrawable{Bitmap: bm, Mode: EncodeFullBlock}
	bd.Draw(canvas, 1, 1)

	cell := canvas.Get(1, 1)
	if cell.Rune != '█' || cell.FG != (Color{R: 255}) {
		t.Errorf("(1,1) = %v, want █ {255 0 0}", cell)
	}

	cell = canvas.Get(2, 2)
	if cell.Rune != '█' || cell.FG != (Color{G: 255}) {
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
