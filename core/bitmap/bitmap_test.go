package bitmap

import (
	"testing"

	"flicker/core"
)

// --- Bitmap basics ---

func TestBitmapSetGet(t *testing.T) {
	bm := New(4, 4)
	c := core.Color{R: 100, G: 150, B: 200}
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
	bm := New(2, 2)
	// Set out-of-bounds should not panic.
	bm.Set(-1, 0, core.Color{R: 255}, 1)
	bm.Set(0, -1, core.Color{R: 255}, 1)
	bm.Set(2, 0, core.Color{R: 255}, 1)
	bm.Set(0, 2, core.Color{R: 255}, 1)

	// Get out-of-bounds should return zero.
	c, a := bm.Get(-1, 0)
	if c != (core.Color{}) || a != 0 {
		t.Errorf("OOB Get should return zero, got %v, %v", c, a)
	}
	c, a = bm.Get(2, 0)
	if c != (core.Color{}) || a != 0 {
		t.Errorf("OOB Get should return zero, got %v, %v", c, a)
	}
}

func TestBitmapClear(t *testing.T) {
	bm := New(3, 3)
	bm.SetDot(1, 1, core.Color{R: 255})
	bm.Clear()

	c, a := bm.Get(1, 1)
	if c != (core.Color{}) || a != 0 {
		t.Errorf("after Clear, pixel should be zero, got %v, %v", c, a)
	}
}

func TestBitmapSetDot(t *testing.T) {
	bm := New(2, 2)
	bm.SetDot(0, 0, core.Color{R: 10, G: 20, B: 30})

	c, a := bm.Get(0, 0)
	if c != (core.Color{R: 10, G: 20, B: 30}) {
		t.Errorf("SetDot color = %v, want {10 20 30}", c)
	}
	if a != 1.0 {
		t.Errorf("SetDot alpha = %v, want 1.0", a)
	}
}

// --- Line drawing ---

func TestLineHorizontal(t *testing.T) {
	bm := New(10, 10)
	bm.Line(1, 5, 8, 5, core.Color{R: 255})

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
	bm := New(10, 10)
	bm.Line(3, 2, 3, 7, core.Color{G: 255})

	for y := 2; y <= 7; y++ {
		_, a := bm.Get(3, y)
		if a == 0 {
			t.Errorf("pixel (3,%d) should be set", y)
		}
	}
}

func TestLineDiagonal(t *testing.T) {
	bm := New(10, 10)
	bm.Line(0, 0, 9, 9, core.Color{B: 255})

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
	bm := New(5, 5)
	bm.Line(2, 3, 2, 3, core.Color{R: 100})

	_, a := bm.Get(2, 3)
	if a == 0 {
		t.Error("single-point line should set pixel")
	}
}
