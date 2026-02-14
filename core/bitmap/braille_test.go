package bitmap

import (
	"testing"

	"flicker/core"
)

func TestBrailleSingleDot(t *testing.T) {
	// A single dot at (0,0) → braille dot1 (0x01) → U+2801.
	bm := New(2, 4)
	bm.SetDot(0, 0, core.Color{R: 255})

	canvas := core.NewCanvas(1, 1)
	br := &Braille{Bitmap: bm}
	br.Draw(canvas, 0, 0)

	cell := canvas.Get(0, 0)
	if cell.Rune != '\u2801' {
		t.Errorf("single dot (0,0) rune = %U, want U+2801", cell.Rune)
	}
	if cell.FG != (core.Color{R: 255}) {
		t.Errorf("FG = %v, want {255 0 0}", cell.FG)
	}
	if cell.FGAlpha != 1.0 {
		t.Errorf("FGAlpha = %v, want 1.0", cell.FGAlpha)
	}
}

func TestBrailleFullBlock(t *testing.T) {
	// All 8 dots lit → U+28FF.
	bm := New(2, 4)
	c := core.Color{R: 100, G: 100, B: 100}
	for y := range 4 {
		for x := range 2 {
			bm.SetDot(x, y, c)
		}
	}

	canvas := core.NewCanvas(1, 1)
	br := &Braille{Bitmap: bm}
	br.Draw(canvas, 0, 0)

	cell := canvas.Get(0, 0)
	if cell.Rune != '\u28FF' {
		t.Errorf("full block rune = %U, want U+28FF", cell.Rune)
	}
}

func TestBrailleTwoColorAverage(t *testing.T) {
	// Two dots with different colors: average should be computed.
	bm := New(2, 4)
	bm.SetDot(0, 0, core.Color{R: 200, G: 0, B: 0})
	bm.SetDot(1, 0, core.Color{R: 0, G: 0, B: 100})

	canvas := core.NewCanvas(1, 1)
	br := &Braille{Bitmap: bm}
	br.Draw(canvas, 0, 0)

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
	bm := New(2, 4)
	canvas := core.NewCanvas(1, 1)
	br := &Braille{Bitmap: bm}
	br.Draw(canvas, 0, 0)

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
		bm := New(2, 4)
		bm.SetDot(tc.x, tc.y, core.Color{R: 255})

		canvas := core.NewCanvas(1, 1)
		br := &Braille{Bitmap: bm}
		br.Draw(canvas, 0, 0)

		want := rune(0x2800 | int(tc.bit))
		got := canvas.Get(0, 0).Rune
		if got != want {
			t.Errorf("dot(%d,%d): rune = %U, want %U", tc.x, tc.y, got, want)
		}
	}
}

func TestBrailleBounds(t *testing.T) {
	bm := New(10, 8)
	br := &Braille{Bitmap: bm}
	w, h := br.Bounds()
	if w != 5 || h != 2 {
		t.Errorf("Braille bounds = (%d, %d), want (5, 2)", w, h)
	}
}

func TestBrailleBoundsOdd(t *testing.T) {
	bm := New(3, 5)
	br := &Braille{Bitmap: bm}
	w, h := br.Bounds()
	if w != 2 || h != 2 {
		t.Errorf("Braille bounds(3,5) = (%d, %d), want (2, 2)", w, h)
	}
}

func TestBrailleNil(t *testing.T) {
	br := &Braille{Bitmap: nil}
	w, h := br.Bounds()
	if w != 0 || h != 0 {
		t.Errorf("nil bitmap bounds = (%d, %d), want (0, 0)", w, h)
	}
	// Draw should not panic.
	canvas := core.NewCanvas(5, 5)
	br.Draw(canvas, 0, 0)
}

func TestBrailleDrawWithOffset(t *testing.T) {
	bm := New(2, 4)
	bm.SetDot(0, 0, core.Color{R: 255})

	canvas := core.NewCanvas(5, 5)
	br := &Braille{Bitmap: bm}
	br.Draw(canvas, 2, 3)

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
