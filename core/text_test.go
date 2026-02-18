package core

import (
	"testing"

	"flicker/fmath"
)

func TestTextBounds(t *testing.T) {
	tests := []struct {
		name  string
		text  string
		wantW int
		wantH int
	}{
		{"single line", "Hello", 5, 1},
		{"multiline", "Hi\nWorld", 5, 2},
		{"empty", "", 0, 1}, // strings.Split("", "\n") => [""]
		{"box", "┌──┐\n│  │\n└──┘", 4, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			txt := NewText(tt.text, Color{}, 1.0)
			w, h := txt.Bounds()
			if w != tt.wantW || h != tt.wantH {
				t.Errorf("Bounds() = (%d, %d), want (%d, %d)", w, h, tt.wantW, tt.wantH)
			}
		})
	}
}

func TestTextDraw(t *testing.T) {
	txt := NewText("AB\nCD", Color{R: 255}, 1.0)
	canvas := NewCanvas(10, 10)
	txt.Draw(canvas, 2, 3)

	// Check that characters appear at correct positions
	got := canvas.Get(2, 3)
	if got.Rune != 'A' {
		t.Errorf("expected 'A' at (2,3), got %q", got.Rune)
	}
	got = canvas.Get(3, 3)
	if got.Rune != 'B' {
		t.Errorf("expected 'B' at (3,3), got %q", got.Rune)
	}
	got = canvas.Get(2, 4)
	if got.Rune != 'C' {
		t.Errorf("expected 'C' at (2,4), got %q", got.Rune)
	}
	got = canvas.Get(3, 4)
	if got.Rune != 'D' {
		t.Errorf("expected 'D' at (3,4), got %q", got.Rune)
	}

	// Check FG color propagation
	if got.FG.R != 255 {
		t.Errorf("expected FG.R=255, got %d", got.FG.R)
	}
}

func TestTextDrawSkipsSpaces(t *testing.T) {
	txt := NewText("A B", Color{}, 1.0)
	canvas := NewCanvas(10, 10)
	txt.Draw(canvas, 0, 0)

	got := canvas.Get(1, 0)
	if got.Rune != 0 {
		t.Errorf("expected zero rune at space position, got %q", got.Rune)
	}
}

func TestTextCellAt(t *testing.T) {
	txt := NewText("AB\nCD", Color{R: 100, G: 200, B: 50}, 0.8)

	cell := txt.CellAt(0, 0)
	if cell.Rune != 'A' {
		t.Errorf("CellAt(0,0) rune = %q, want 'A'", cell.Rune)
	}
	if cell.FGAlpha != 0.8 {
		t.Errorf("CellAt(0,0) FGAlpha = %f, want 0.8", cell.FGAlpha)
	}

	// Space returns zero cell
	txt2 := NewText("A B", Color{}, 1.0)
	cell = txt2.CellAt(1, 0)
	if cell.Rune != 0 {
		t.Errorf("CellAt at space should return zero cell, got rune %q", cell.Rune)
	}

	// Out of bounds
	cell = txt.CellAt(-1, 0)
	if cell.Rune != 0 {
		t.Errorf("CellAt(-1,0) should return zero cell")
	}
	cell = txt.CellAt(0, 5)
	if cell.Rune != 0 {
		t.Errorf("CellAt(0,5) should return zero cell")
	}
}

func TestTextSetText(t *testing.T) {
	txt := NewText("Hi", Color{}, 1.0)
	w, h := txt.Bounds()
	if w != 2 || h != 1 {
		t.Errorf("initial Bounds() = (%d, %d), want (2, 1)", w, h)
	}

	txt.SetText("Hello\nWorld!")
	w, h = txt.Bounds()
	if w != 6 || h != 2 {
		t.Errorf("after SetText Bounds() = (%d, %d), want (6, 2)", w, h)
	}
}

func TestTextRenderer(t *testing.T) {
	txt := NewText("AB", Color{R: 255}, 1.0)
	render := txt.Renderer()

	// Identity matrix → characters at their natural positions
	world := fmath.Mat3Identity()
	var cells []struct{ dx, dy, sx, sy int }
	render(world, func(dx, dy, sx, sy int, cell Cell) {
		cells = append(cells, struct{ dx, dy, sx, sy int }{dx, dy, sx, sy})
	})

	if len(cells) != 2 {
		t.Fatalf("expected 2 emitted cells, got %d", len(cells))
	}

	// With identity matrix, screen positions should match local positions
	if cells[0].dx != 0 || cells[0].dy != 0 {
		t.Errorf("cell[0] local = (%d,%d), want (0,0)", cells[0].dx, cells[0].dy)
	}
	if cells[1].dx != 1 || cells[1].dy != 0 {
		t.Errorf("cell[1] local = (%d,%d), want (1,0)", cells[1].dx, cells[1].dy)
	}
}

func TestTextRendererTranslated(t *testing.T) {
	txt := NewText("X", Color{}, 1.0)
	render := txt.Renderer()

	// Translate by (5, 3)
	world := fmath.Mat3Identity()
	world[2] = 5 // tx
	world[5] = 3 // ty

	var cells []struct{ dx, dy, sx, sy int }
	render(world, func(dx, dy, sx, sy int, cell Cell) {
		cells = append(cells, struct{ dx, dy, sx, sy int }{dx, dy, sx, sy})
	})

	if len(cells) != 1 {
		t.Fatalf("expected 1 emitted cell, got %d", len(cells))
	}

	if cells[0].sx != 5 || cells[0].sy != 3 {
		t.Errorf("translated cell screen pos = (%d,%d), want (5,3)", cells[0].sx, cells[0].sy)
	}
}
