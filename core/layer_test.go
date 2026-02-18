package core_test

import (
	"testing"

	"flicker/core"
	"flicker/core/bitmap"
	"flicker/fmath"
)

func TestCompositor_SingleLayer(t *testing.T) {
	world := core.NewWorld()
	box := world.Spawn()
	world.AddTransform(box, &core.Transform{Scale: fmath.Vec3{X: 1, Y: 1, Z: 1}})
	world.AddDrawable(
		box,
		&bitmap.Rect{Width: 3, Height: 2, FG: core.Color{R: 255}, BG: core.Color{}},
	)
	world.AddRoot(box)

	dst := core.NewCanvas(5, 3)
	comp := core.NewCompositor(5, 3)
	comp.Composite(world, dst, core.Time{})

	cell := dst.Get(1, 0)
	if cell.Rune != '▀' {
		t.Errorf("expected '▀', got %c", cell.Rune)
	}
	if cell.FG.R != 255 {
		t.Errorf("expected FG.R=255, got %d", cell.FG.R)
	}
}

func TestCompositor_TwoLayers_Opaque(t *testing.T) {
	world := core.NewWorld()

	// Layer 0: red 'A'
	boxA := world.Spawn()
	world.AddTransform(boxA, &core.Transform{Scale: fmath.Vec3{X: 1, Y: 1, Z: 1}})
	world.AddDrawable(boxA, &bitmap.Rect{Width: 3, Height: 2, FG: core.Color{R: 200}})
	world.AddLayer(boxA, 0)
	world.AddRoot(boxA)

	// Layer 1: blue (opaque, fully overwrites)
	boxB := world.Spawn()
	world.AddTransform(boxB, &core.Transform{Scale: fmath.Vec3{X: 1, Y: 1, Z: 1}})
	world.AddDrawable(boxB, &bitmap.Rect{Width: 3, Height: 2, FG: core.Color{B: 200}})
	world.AddLayer(boxB, 1)
	world.AddRoot(boxB)

	dst := core.NewCanvas(3, 2)
	comp := core.NewCompositor(3, 2)
	comp.Composite(world, dst, core.Time{})

	cell := dst.Get(0, 0)
	if cell.Rune != '▀' {
		t.Errorf("layer 1 should overwrite layer 0: expected '▀', got %c", cell.Rune)
	}
	if cell.FG.B != 200 {
		t.Errorf("expected FG.B=200, got %d", cell.FG.B)
	}
}

func TestCompositor_TwoLayers_SemiTransparent(t *testing.T) {
	world := core.NewWorld()

	// Layer 0: red, opaque
	boxA := world.Spawn()
	world.AddTransform(boxA, &core.Transform{Scale: fmath.Vec3{X: 1, Y: 1, Z: 1}})
	world.AddDrawable(boxA, &bitmap.Rect{Width: 3, Height: 2, FG: core.Color{R: 200}})
	world.AddLayer(boxA, 0)
	world.AddRoot(boxA)

	// Layer 1: blue, semi-transparent via material
	boxB := world.Spawn()
	world.AddTransform(boxB, &core.Transform{Scale: fmath.Vec3{X: 1, Y: 1, Z: 1}})
	world.AddDrawable(boxB, &bitmap.Rect{Width: 3, Height: 2, FG: core.Color{B: 200}})
	world.AddLayer(boxB, 1)
	world.AddMaterial(boxB, func(f core.Fragment) core.Cell {
		f.Cell.FGAlpha = 0.5
		f.Cell.BGAlpha = 0.5
		return f.Cell
	})
	world.AddRoot(boxB)

	dst := core.NewCanvas(3, 2)
	comp := core.NewCompositor(3, 2)
	comp.Composite(world, dst, core.Time{})

	cell := dst.Get(0, 0)
	// FG.R: 200*(1-0.5) + 0*0.5 = 100
	if cell.FG.R != 100 {
		t.Errorf("expected blended FG.R=100, got %d", cell.FG.R)
	}
	// FG.B: 0*(1-0.5) + 200*0.5 = 100
	if cell.FG.B != 100 {
		t.Errorf("expected blended FG.B=100, got %d", cell.FG.B)
	}
	if cell.Rune != '▀' {
		t.Errorf("expected rune '▀', got %c", cell.Rune)
	}
}

func TestCompositor_LayerOrder(t *testing.T) {
	world := core.NewWorld()

	// Add root on layer 5 first, then layer 2 — compositor should sort.
	boxHigh := world.Spawn()
	world.AddTransform(boxHigh, &core.Transform{Scale: fmath.Vec3{X: 1, Y: 1, Z: 1}})
	world.AddDrawable(boxHigh, &bitmap.Rect{Width: 2, Height: 1, FG: core.Color{G: 255}})
	world.AddLayer(boxHigh, 5)
	world.AddRoot(boxHigh)

	boxLow := world.Spawn()
	world.AddTransform(boxLow, &core.Transform{Scale: fmath.Vec3{X: 1, Y: 1, Z: 1}})
	world.AddDrawable(boxLow, &bitmap.Rect{Width: 2, Height: 1, FG: core.Color{R: 255}})
	world.AddLayer(boxLow, 2)
	world.AddRoot(boxLow)

	dst := core.NewCanvas(2, 1)
	comp := core.NewCompositor(2, 1)
	comp.Composite(world, dst, core.Time{})

	// Layer 5 composited after layer 2 → 'H' wins.
	cell := dst.Get(0, 0)
	if cell.Rune != '▀' {
		t.Errorf("higher layer should render on top: expected '▀', got %c", cell.Rune)
	}
}

func TestCompositor_PostProcess(t *testing.T) {
	world := core.NewWorld()

	box := world.Spawn()
	world.AddTransform(box, &core.Transform{Scale: fmath.Vec3{X: 1, Y: 1, Z: 1}})
	world.AddDrawable(
		box,
		&bitmap.Rect{Width: 2, Height: 1, FG: core.Color{R: 100, G: 100, B: 100}},
	)
	world.AddLayer(box, 0)
	world.AddRoot(box)

	dst := core.NewCanvas(2, 1)
	comp := core.NewCompositor(2, 1)

	// Post-process: zero out FG green channel.
	comp.SetPostProcess(0, func(f core.Fragment) core.Cell {
		f.Cell.FG.G = 0
		return f.Cell
	})

	comp.Composite(world, dst, core.Time{})

	cell := dst.Get(0, 0)
	if cell.FG.G != 0 {
		t.Errorf("post-process should zero G: expected 0, got %d", cell.FG.G)
	}
	if cell.FG.R != 100 {
		t.Errorf("R should be unchanged: expected 100, got %d", cell.FG.R)
	}
}

func TestCompositor_PerLayerCamera(t *testing.T) {
	world := core.NewWorld()

	// Camera panned far off-screen so entities at (0,0) won't appear
	// on a small canvas when rendered through this camera.
	cam := world.Spawn()
	world.AddTransform(cam, &core.Transform{
		Position: fmath.Vec3{X: 1000, Y: 1000},
		Scale:    fmath.Vec3{X: 1, Y: 1, Z: 1},
	})
	world.AddCamera(cam, &core.Camera{Zoom: 1})

	// Layer 0: screen-space (identity). Entity at (0,0) should render
	// at screen (0,0) because identity view doesn't shift anything.
	screenBox := world.Spawn()
	world.AddTransform(screenBox, &core.Transform{Scale: fmath.Vec3{X: 1, Y: 1, Z: 1}})
	world.AddDrawable(screenBox, &bitmap.Rect{Width: 2, Height: 1, FG: core.Color{R: 255}})
	world.AddLayer(screenBox, 0)
	world.AddRoot(screenBox)

	// Layer 1: world camera. Same entity position (0,0), but viewed
	// through a camera at (1000,1000) — shifts it off-screen entirely.
	worldBox := world.Spawn()
	world.AddTransform(worldBox, &core.Transform{Scale: fmath.Vec3{X: 1, Y: 1, Z: 1}})
	world.AddDrawable(worldBox, &bitmap.Rect{Width: 2, Height: 1, FG: core.Color{B: 255}})
	world.AddLayer(worldBox, 1)
	world.AddRoot(worldBox)

	world.SetLayerCamera(0, 0) // screen-space
	world.SetLayerCamera(1, cam)

	w, h := 10, 5
	dst := core.NewCanvas(w, h)
	comp := core.NewCompositor(w, h)
	comp.Composite(world, dst, core.Time{})

	// Screen-space entity should render at (0,0).
	cell := dst.Get(0, 0)
	if cell.FG.R != 255 {
		t.Errorf("screen-space entity: expected FG.R=255 at (0,0), got %d", cell.FG.R)
	}

	// World-camera entity should be off-screen — no blue anywhere on canvas.
	for y := range h {
		for x := range w {
			c := dst.Get(x, y)
			if c.FG.B == 255 {
				t.Errorf("world-camera entity should be off-screen, found blue at (%d,%d)", x, y)
			}
		}
	}
}

func TestCompositor_DefaultLayer(t *testing.T) {
	world := core.NewWorld()

	// Entity without AddLayer should use default layer 0.
	box := world.Spawn()
	world.AddTransform(box, &core.Transform{Scale: fmath.Vec3{X: 1, Y: 1, Z: 1}})
	world.AddDrawable(box, &bitmap.Rect{Width: 2, Height: 1})
	world.AddRoot(box)

	if world.Layer(box) != 0 {
		t.Errorf("unassigned entity should be on layer 0, got %d", world.Layer(box))
	}

	dst := core.NewCanvas(2, 1)
	comp := core.NewCompositor(2, 1)
	comp.Composite(world, dst, core.Time{})

	if dst.Get(0, 0).Rune != '▀' {
		t.Errorf("expected '▀', got %c", dst.Get(0, 0).Rune)
	}
}
