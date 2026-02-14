package core

import "testing"

func TestBlendColor(t *testing.T) {
	// 50% blend of black and white → mid-gray
	c := BlendColor(Color{0, 0, 0}, Color{255, 255, 255}, 0.5, BlendNormal)
	if c.R < 126 || c.R > 128 {
		t.Errorf("expected R≈127, got %d", c.R)
	}

	// alpha=0 → dst unchanged
	c = BlendColor(Color{100, 100, 100}, Color{200, 200, 200}, 0, BlendNormal)
	if c != (Color{100, 100, 100}) {
		t.Errorf("alpha=0 should return dst, got %v", c)
	}

	// alpha=1 → src fully
	c = BlendColor(Color{100, 100, 100}, Color{200, 200, 200}, 1, BlendNormal)
	if c != (Color{200, 200, 200}) {
		t.Errorf("alpha=1 should return src, got %v", c)
	}

	// clamp: alpha > 1 treated as 1
	c = BlendColor(Color{0, 0, 0}, Color{255, 255, 255}, 2.0, BlendNormal)
	if c != (Color{255, 255, 255}) {
		t.Errorf("alpha>1 should clamp to 1, got %v", c)
	}

	// clamp: alpha < 0 treated as 0
	c = BlendColor(Color{100, 100, 100}, Color{255, 255, 255}, -1.0, BlendNormal)
	if c != (Color{100, 100, 100}) {
		t.Errorf("alpha<0 should clamp to 0, got %v", c)
	}
}

func TestBlendCell_SrcInvisible(t *testing.T) {
	dst := Cell{Rune: 'A', FG: Color{100, 0, 0}, Alpha: 1}
	src := Cell{Rune: 'B', FG: Color{0, 100, 0}, Alpha: 0}
	out := BlendCell(dst, src, NormalColorBlend)
	if out != dst {
		t.Errorf("src.Alpha==0 should return dst, got %v", out)
	}
}

func TestBlendCell_SrcOpaqueDstEmpty(t *testing.T) {
	dst := Cell{}
	src := Cell{Rune: 'X', FG: Color{200, 0, 0}, BG: Color{0, 0, 50}, Alpha: 1}
	out := BlendCell(dst, src, NormalColorBlend)
	if out != src {
		t.Errorf("src opaque over empty dst should return src, got %v", out)
	}
}

func TestBlendCell_SemiTransparent(t *testing.T) {
	dst := Cell{Rune: 'A', FG: Color{200, 0, 0}, BG: Color{40, 0, 0}, Alpha: 1}
	src := Cell{Rune: 'B', FG: Color{0, 0, 200}, BG: Color{0, 0, 40}, Alpha: 0.5}
	out := BlendCell(dst, src, NormalColorBlend)

	// Rune: src has a real character, so it wins.
	if out.Rune != 'B' {
		t.Errorf("expected rune 'B', got %c", out.Rune)
	}

	// FG.R should be blended: 200*(1-0.5) + 0*0.5 = 100
	if out.FG.R != 100 {
		t.Errorf("expected FG.R=100, got %d", out.FG.R)
	}
	// FG.B should be blended: 0*(1-0.5) + 200*0.5 = 100
	if out.FG.B != 100 {
		t.Errorf("expected FG.B=100, got %d", out.FG.B)
	}
}

func TestBlendCell_EmptySrcRunePreservesDst(t *testing.T) {
	dst := Cell{Rune: 'Z', FG: Color{200, 0, 0}, Alpha: 1}
	src := Cell{Rune: 0, FG: Color{0, 200, 0}, Alpha: 0.5} // color-only overlay
	out := BlendCell(dst, src, NormalColorBlend)
	if out.Rune != 'Z' {
		t.Errorf("empty src rune should keep dst rune 'Z', got %c", out.Rune)
	}
}

func TestCanvasComposite(t *testing.T) {
	dst := NewCanvas(3, 2)
	src := NewCanvas(3, 2)

	// Fill dst with red 'A'
	for y := 0; y < 2; y++ {
		for x := 0; x < 3; x++ {
			dst.Set(x, y, Cell{Rune: 'A', FG: Color{200, 0, 0}, Alpha: 1})
		}
	}

	// Fill src with blue 'B' at 0.5 alpha
	for y := 0; y < 2; y++ {
		for x := 0; x < 3; x++ {
			src.Set(x, y, Cell{Rune: 'B', FG: Color{0, 0, 200}, Alpha: 0.5})
		}
	}

	dst.Composite(src, NormalColorBlend)

	cell := dst.Get(0, 0)
	if cell.Rune != 'B' {
		t.Errorf("expected rune 'B' after composite, got %c", cell.Rune)
	}
	if cell.FG.R != 100 {
		t.Errorf("expected blended FG.R=100, got %d", cell.FG.R)
	}
	if cell.FG.B != 100 {
		t.Errorf("expected blended FG.B=100, got %d", cell.FG.B)
	}
}

func TestCanvasComposite_DifferentSizes(t *testing.T) {
	dst := NewCanvas(4, 4)
	src := NewCanvas(2, 2)

	src.Set(0, 0, Cell{Rune: 'X', Alpha: 1})
	dst.Composite(src, NormalColorBlend)

	if dst.Get(0, 0).Rune != 'X' {
		t.Errorf("expected 'X' at (0,0)")
	}
	// Cells outside src bounds should be unchanged (zero value).
	if dst.Get(3, 3).Rune != 0 {
		t.Errorf("expected zero rune at (3,3), got %c", dst.Get(3, 3).Rune)
	}
}
