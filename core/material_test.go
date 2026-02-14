package core

import "testing"

func TestMaterialApplication(t *testing.T) {
	world := NewWorld()
	e := world.Spawn()
	world.AddTransform(e, &Transform{})
	world.AddDrawable(e, &Rect{
		Width:  3,
		Height: 2,
		Rune:   '#',
		FG:     Color{R: 100, G: 100, B: 100},
		BG:     Color{R: 0, G: 0, B: 0},
	})
	world.AddRoot(e)

	// Material that sets FG red channel to 10*x and green channel to 10*y.
	world.AddMaterial(e, func(f Fragment) Cell {
		f.Cell.FG.R = uint8(10 * f.X)
		f.Cell.FG.G = uint8(10 * f.Y)
		f.Cell.FG.B = 0
		return f.Cell
	})

	canvas := NewCanvas(10, 10)
	Render(world, canvas, Time{Total: 1.0, Delta: 0.016})

	tests := []struct {
		cx, cy int // canvas coords
		wantR  uint8
		wantG  uint8
	}{
		{0, 0, 0, 0},   // local (0,0)
		{1, 0, 10, 0},  // local (1,0)
		{2, 0, 20, 0},  // local (2,0)
		{0, 1, 0, 10},  // local (0,1)
		{2, 1, 20, 10}, // local (2,1)
	}

	for _, tc := range tests {
		cell := canvas.Get(tc.cx, tc.cy)
		if cell.FG.R != tc.wantR || cell.FG.G != tc.wantG {
			t.Errorf("cell(%d,%d) FG=(%d,%d,_), want (%d,%d,_)",
				tc.cx, tc.cy, cell.FG.R, cell.FG.G, tc.wantR, tc.wantG)
		}
		if cell.FG.B != 0 {
			t.Errorf("cell(%d,%d) FG.B=%d, want 0", tc.cx, tc.cy, cell.FG.B)
		}
		if cell.Alpha != 1.0 {
			t.Errorf("cell(%d,%d) Alpha=%f, want 1.0", tc.cx, tc.cy, cell.Alpha)
		}
	}
}

func TestMaterialPreservesAlpha(t *testing.T) {
	world := NewWorld()
	e := world.Spawn()
	world.AddTransform(e, &Transform{})
	world.AddDrawable(e, &Rect{
		Width:  2,
		Height: 2,
		Rune:   'X',
		FG:     Color{R: 255, G: 255, B: 255},
	})
	world.AddRoot(e)

	// Identity material — returns cell unchanged.
	world.AddMaterial(e, func(f Fragment) Cell {
		return f.Cell
	})

	canvas := NewCanvas(5, 5)
	Render(world, canvas, Time{})

	// Cells inside drawable should have Alpha 1.0.
	for y := range 2 {
		for x := range 2 {
			cell := canvas.Get(x, y)
			if cell.Alpha != 1.0 {
				t.Errorf("cell(%d,%d) Alpha=%f, want 1.0", x, y, cell.Alpha)
			}
			if cell.Rune != 'X' {
				t.Errorf("cell(%d,%d) Rune=%q, want 'X'", x, y, cell.Rune)
			}
		}
	}

	// Cell outside drawable should have Alpha 0.
	cell := canvas.Get(3, 3)
	if cell.Alpha != 0 {
		t.Errorf("cell(3,3) Alpha=%f, want 0", cell.Alpha)
	}
}
