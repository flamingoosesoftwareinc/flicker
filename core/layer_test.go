package core

import "testing"

func TestCompositor_SingleLayer(t *testing.T) {
	world := NewWorld()
	box := world.Spawn()
	world.AddTransform(box, &Transform{})
	world.AddDrawable(
		box,
		&Rect{Width: 3, Height: 2, Rune: 'X', FG: Color{255, 0, 0}, BG: Color{0, 0, 0}},
	)
	world.AddRoot(box)

	dst := NewCanvas(5, 3)
	comp := NewCompositor(5, 3)
	comp.Composite(world, dst, Time{})

	cell := dst.Get(1, 0)
	if cell.Rune != 'X' {
		t.Errorf("expected 'X', got %c", cell.Rune)
	}
	if cell.FG.R != 255 {
		t.Errorf("expected FG.R=255, got %d", cell.FG.R)
	}
}

func TestCompositor_TwoLayers_Opaque(t *testing.T) {
	world := NewWorld()

	// Layer 0: red 'A'
	boxA := world.Spawn()
	world.AddTransform(boxA, &Transform{})
	world.AddDrawable(boxA, &Rect{Width: 3, Height: 2, Rune: 'A', FG: Color{200, 0, 0}})
	world.AddLayer(boxA, 0)
	world.AddRoot(boxA)

	// Layer 1: blue 'B' (opaque, fully overwrites)
	boxB := world.Spawn()
	world.AddTransform(boxB, &Transform{})
	world.AddDrawable(boxB, &Rect{Width: 3, Height: 2, Rune: 'B', FG: Color{0, 0, 200}})
	world.AddLayer(boxB, 1)
	world.AddRoot(boxB)

	dst := NewCanvas(3, 2)
	comp := NewCompositor(3, 2)
	comp.Composite(world, dst, Time{})

	cell := dst.Get(0, 0)
	if cell.Rune != 'B' {
		t.Errorf("layer 1 should overwrite layer 0: expected 'B', got %c", cell.Rune)
	}
	if cell.FG.B != 200 {
		t.Errorf("expected FG.B=200, got %d", cell.FG.B)
	}
}

func TestCompositor_TwoLayers_SemiTransparent(t *testing.T) {
	world := NewWorld()

	// Layer 0: red 'A', opaque
	boxA := world.Spawn()
	world.AddTransform(boxA, &Transform{})
	world.AddDrawable(boxA, &Rect{Width: 3, Height: 2, Rune: 'A', FG: Color{200, 0, 0}})
	world.AddLayer(boxA, 0)
	world.AddRoot(boxA)

	// Layer 1: blue 'B', semi-transparent via material
	boxB := world.Spawn()
	world.AddTransform(boxB, &Transform{})
	world.AddDrawable(boxB, &Rect{Width: 3, Height: 2, Rune: 'B', FG: Color{0, 0, 200}})
	world.AddLayer(boxB, 1)
	world.AddMaterial(boxB, func(x, y int, t Time, cell Cell) Cell {
		cell.Alpha = 0.5
		return cell
	})
	world.AddRoot(boxB)

	dst := NewCanvas(3, 2)
	comp := NewCompositor(3, 2)
	comp.Composite(world, dst, Time{})

	cell := dst.Get(0, 0)
	// FG.R: 200*(1-0.5) + 0*0.5 = 100
	if cell.FG.R != 100 {
		t.Errorf("expected blended FG.R=100, got %d", cell.FG.R)
	}
	// FG.B: 0*(1-0.5) + 200*0.5 = 100
	if cell.FG.B != 100 {
		t.Errorf("expected blended FG.B=100, got %d", cell.FG.B)
	}
	if cell.Rune != 'B' {
		t.Errorf("expected rune 'B', got %c", cell.Rune)
	}
}

func TestCompositor_LayerOrder(t *testing.T) {
	world := NewWorld()

	// Add root on layer 5 first, then layer 2 — compositor should sort.
	boxHigh := world.Spawn()
	world.AddTransform(boxHigh, &Transform{})
	world.AddDrawable(boxHigh, &Rect{Width: 2, Height: 1, Rune: 'H', FG: Color{0, 255, 0}})
	world.AddLayer(boxHigh, 5)
	world.AddRoot(boxHigh)

	boxLow := world.Spawn()
	world.AddTransform(boxLow, &Transform{})
	world.AddDrawable(boxLow, &Rect{Width: 2, Height: 1, Rune: 'L', FG: Color{255, 0, 0}})
	world.AddLayer(boxLow, 2)
	world.AddRoot(boxLow)

	dst := NewCanvas(2, 1)
	comp := NewCompositor(2, 1)
	comp.Composite(world, dst, Time{})

	// Layer 5 composited after layer 2 → 'H' wins.
	cell := dst.Get(0, 0)
	if cell.Rune != 'H' {
		t.Errorf("higher layer should render on top: expected 'H', got %c", cell.Rune)
	}
}

func TestCompositor_PostProcess(t *testing.T) {
	world := NewWorld()

	box := world.Spawn()
	world.AddTransform(box, &Transform{})
	world.AddDrawable(box, &Rect{Width: 2, Height: 1, Rune: 'P', FG: Color{100, 100, 100}})
	world.AddLayer(box, 0)
	world.AddRoot(box)

	dst := NewCanvas(2, 1)
	comp := NewCompositor(2, 1)

	// Post-process: zero out FG green channel.
	comp.SetPostProcess(0, func(c *Canvas, t Time) {
		for y := 0; y < c.Height; y++ {
			for x := 0; x < c.Width; x++ {
				cell := c.Get(x, y)
				cell.FG.G = 0
				c.Set(x, y, cell)
			}
		}
	})

	comp.Composite(world, dst, Time{})

	cell := dst.Get(0, 0)
	if cell.FG.G != 0 {
		t.Errorf("post-process should zero G: expected 0, got %d", cell.FG.G)
	}
	if cell.FG.R != 100 {
		t.Errorf("R should be unchanged: expected 100, got %d", cell.FG.R)
	}
}

func TestCompositor_DefaultLayer(t *testing.T) {
	world := NewWorld()

	// Entity without AddLayer should use default layer 0.
	box := world.Spawn()
	world.AddTransform(box, &Transform{})
	world.AddDrawable(box, &Rect{Width: 2, Height: 1, Rune: 'D'})
	world.AddRoot(box)

	if world.Layer(box) != 0 {
		t.Errorf("unassigned entity should be on layer 0, got %d", world.Layer(box))
	}

	dst := NewCanvas(2, 1)
	comp := NewCompositor(2, 1)
	comp.Composite(world, dst, Time{})

	if dst.Get(0, 0).Rune != 'D' {
		t.Errorf("expected 'D', got %c", dst.Get(0, 0).Rune)
	}
}
