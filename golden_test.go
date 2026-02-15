package flicker_test

import (
	"fmt"
	"math"
	"strings"
	"testing"

	"flicker/asset"
	"flicker/core"
	"flicker/core/bitmap"
	"flicker/fmath"
	"flicker/particle"
	"flicker/physics"
	"flicker/terminal"
	"github.com/sebdah/goldie/v2"
)

func TestBasicExample(t *testing.T) {
	const (
		w = 40
		h = 20
	)

	screen := terminal.NewSimScreen(w, h)

	world := core.NewWorld()
	box := world.Spawn()
	world.AddTransform(box, &core.Transform{
		Position: fmath.Vec3{X: 10, Y: 5},
		Scale:    fmath.Vec3{X: 1, Y: 1, Z: 1},
	})
	world.AddDrawable(box, &bitmap.Rect{
		Width:  20,
		Height: 10,
	})
	world.AddRoot(box)

	canvas := core.NewCanvas(w, h)
	canvas.Clear()
	canvas.DrawBorder()
	core.Render(world, canvas, core.Time{})
	screen.Flush(canvas)

	// Build golden text from captured frames.
	var b strings.Builder
	for i, frame := range screen.Frames() {
		fmt.Fprintf(&b, "--- frame %d ---\n", i)
		b.WriteString(frame)
		b.WriteByte('\n')
	}

	g := goldie.New(t)
	g.Assert(t, "basic_example", []byte(b.String()))
}

func TestAnimatedBehavior(t *testing.T) {
	const (
		w      = 60
		h      = 12
		frames = 5
		dt     = 0.5 // fixed dt per tick
	)

	screen := terminal.NewSimScreen(w, h)
	canvas := core.NewCanvas(w, h)

	world := core.NewWorld()
	box := world.Spawn()
	world.AddTransform(box, &core.Transform{
		Position: fmath.Vec3{X: 5, Y: 1},
		Scale:    fmath.Vec3{X: 1, Y: 1, Z: 1},
	})
	world.AddDrawable(box, &bitmap.Rect{
		Width:  10,
		Height: 5,
	})
	world.AddRoot(box)

	elapsed := 0.0
	world.AddBehavior(box, func(t core.Time, e core.Entity, w *core.World) {
		elapsed += t.Delta
		v := fmath.Triangle(elapsed / 2.0)
		w.Transform(e).Position.X = fmath.Remap(0, 1, 5, 50, v)
	})

	for i := range frames {
		t := core.Time{
			Total: float64(i+1) * dt,
			Delta: dt,
		}
		core.UpdateBehaviors(world, t)

		canvas.Clear()
		canvas.DrawBorder()
		core.Render(world, canvas, t)
		screen.Flush(canvas)
	}

	var b strings.Builder
	for i, frame := range screen.Frames() {
		fmt.Fprintf(&b, "--- frame %d ---\n", i)
		b.WriteString(frame)
		b.WriteByte('\n')
	}

	g := goldie.New(t)
	g.Assert(t, "animated_behavior", []byte(b.String()))
}

func TestOverlappingObjects(t *testing.T) {
	const (
		w      = 60
		h      = 16
		frames = 8
		dt     = 0.4
	)

	screen := terminal.NewSimScreen(w, h)
	canvas := core.NewCanvas(w, h)

	world := core.NewWorld()

	// Box A: red-ish, renders underneath (added first).
	boxA := world.Spawn()
	world.AddTransform(boxA, &core.Transform{
		Position: fmath.Vec3{X: 2, Y: 5},
		Scale:    fmath.Vec3{X: 1, Y: 1, Z: 1},
	})
	world.AddDrawable(boxA, &bitmap.Rect{
		Width:  12,
		Height: 6,
		FG:     core.Color{R: 200, G: 60, B: 60},
		BG:     core.Color{R: 40, G: 0, B: 0},
	})
	world.AddRoot(boxA)

	elapsedA := 0.0
	world.AddBehavior(boxA, func(t core.Time, e core.Entity, w *core.World) {
		elapsedA += t.Delta
		v := fmath.Triangle(elapsedA / 2.0)
		w.Transform(e).Position.X = fmath.Remap(0, 1, 2, 45, v)
	})

	// Box B: blue-ish, renders on top (added second).
	boxB := world.Spawn()
	world.AddTransform(boxB, &core.Transform{
		Position: fmath.Vec3{X: 45, Y: 5},
		Scale:    fmath.Vec3{X: 1, Y: 1, Z: 1},
	})
	world.AddDrawable(boxB, &bitmap.Rect{
		Width:  12,
		Height: 6,
		FG:     core.Color{R: 60, G: 60, B: 200},
		BG:     core.Color{R: 0, G: 0, B: 40},
	})
	world.AddRoot(boxB)

	elapsedB := 0.0
	world.AddBehavior(boxB, func(t core.Time, e core.Entity, w *core.World) {
		elapsedB += t.Delta
		v := fmath.Triangle(elapsedB / 2.0)
		w.Transform(e).Position.X = fmath.Remap(0, 1, 45, 2, v)
	})

	for i := range frames {
		t := core.Time{
			Total: float64(i+1) * dt,
			Delta: dt,
		}
		core.UpdateBehaviors(world, t)

		canvas.Clear()
		canvas.DrawBorder()
		core.Render(world, canvas, t)
		screen.Flush(canvas)
	}

	var b strings.Builder
	for i, frame := range screen.Frames() {
		fmt.Fprintf(&b, "--- frame %d ---\n", i)
		b.WriteString(frame)
		b.WriteByte('\n')
	}

	g := goldie.New(t)
	g.Assert(t, "overlapping_objects", []byte(b.String()))
}

func TestLayerBlending(t *testing.T) {
	const (
		w = 30
		h = 10
	)

	screen := terminal.NewSimScreen(w, h)
	canvas := core.NewCanvas(w, h)

	world := core.NewWorld()

	// Layer 0: red box, stationary, opaque.
	boxA := world.Spawn()
	world.AddTransform(boxA, &core.Transform{
		Position: fmath.Vec3{X: 5, Y: 2},
		Scale:    fmath.Vec3{X: 1, Y: 1, Z: 1},
	})
	world.AddDrawable(boxA, &bitmap.Rect{
		Width:  15,
		Height: 6,
		FG:     core.Color{R: 200, G: 60, B: 60},
		BG:     core.Color{R: 40, G: 0, B: 0},
	})
	world.AddLayer(boxA, 0)
	world.AddRoot(boxA)

	// Layer 1: blue box, overlapping, semi-transparent via material.
	boxB := world.Spawn()
	world.AddTransform(boxB, &core.Transform{
		Position: fmath.Vec3{X: 10, Y: 3},
		Scale:    fmath.Vec3{X: 1, Y: 1, Z: 1},
	})
	world.AddDrawable(boxB, &bitmap.Rect{
		Width:  15,
		Height: 6,
		FG:     core.Color{R: 60, G: 60, B: 200},
		BG:     core.Color{R: 0, G: 0, B: 40},
	})
	world.AddLayer(boxB, 1)
	world.AddMaterial(boxB, func(f core.Fragment) core.Cell {
		f.Cell.FGAlpha = 0.5
		f.Cell.BGAlpha = 0.5
		return f.Cell
	})
	world.AddRoot(boxB)

	comp := core.NewCompositor(w, h)

	canvas.Clear()
	canvas.DrawBorder()
	comp.Composite(world, canvas, core.Time{})
	screen.Flush(canvas)

	var b strings.Builder
	for i, frame := range screen.Frames() {
		fmt.Fprintf(&b, "--- frame %d ---\n", i)
		b.WriteString(frame)
		b.WriteByte('\n')
	}

	g := goldie.New(t)
	g.Assert(t, "layer_blending", []byte(b.String()))
}

func TestTween(t *testing.T) {
	const (
		w      = 40
		h      = 10
		frames = 6
		dt     = 0.5 // 0.5s per frame; tween duration = 2.0s → done at frame 4
	)

	screen := terminal.NewSimScreen(w, h)
	canvas := core.NewCanvas(w, h)

	world := core.NewWorld()
	box := world.Spawn()
	world.AddTransform(box, &core.Transform{
		Position: fmath.Vec3{X: 2, Y: 3},
		Scale:    fmath.Vec3{X: 1, Y: 1, Z: 1},
	})
	world.AddDrawable(box, &bitmap.Rect{
		Width:  8,
		Height: 4,
	})
	world.AddRoot(box)

	// Ping-pong tween: move X from 2 to 30, then back, using EaseInOutQuad.
	forward := true
	tw := &fmath.Tween{From: 2, To: 30, Duration: 2.0, Easing: fmath.EaseInOutQuad}
	world.AddBehavior(box, func(t core.Time, e core.Entity, w *core.World) {
		pos := tw.Update(t.Delta)
		w.Transform(e).Position.X = pos
		if tw.Done() {
			tw.Reset()
			if forward {
				tw.From, tw.To = 30, 2
			} else {
				tw.From, tw.To = 2, 30
			}
			forward = !forward
		}
	})

	for i := range frames {
		t := core.Time{
			Total: float64(i+1) * dt,
			Delta: dt,
		}
		core.UpdateBehaviors(world, t)

		canvas.Clear()
		canvas.DrawBorder()
		core.Render(world, canvas, t)
		screen.Flush(canvas)
	}

	var b strings.Builder
	for i, frame := range screen.Frames() {
		fmt.Fprintf(&b, "--- frame %d ---\n", i)
		b.WriteString(frame)
		b.WriteByte('\n')
	}

	g := goldie.New(t)
	g.Assert(t, "tween", []byte(b.String()))
}

func TestBlendModes(t *testing.T) {
	const (
		w = 40
		h = 12
	)

	screen := terminal.NewSimScreen(w, h)
	canvas := core.NewCanvas(w, h)

	world := core.NewWorld()

	// Layer 0: red box (Normal blend, base layer).
	boxA := world.Spawn()
	world.AddTransform(boxA, &core.Transform{
		Position: fmath.Vec3{X: 5, Y: 2},
		Scale:    fmath.Vec3{X: 1, Y: 1, Z: 1},
	})
	world.AddDrawable(boxA, &bitmap.Rect{
		Width:  20,
		Height: 8,
		FG:     core.Color{R: 200, G: 60, B: 60},
		BG:     core.Color{R: 80, G: 20, B: 20},
	})
	world.AddLayer(boxA, 0)
	world.AddRoot(boxA)

	// Layer 1: green box (Multiply blend), overlaps red.
	boxB := world.Spawn()
	world.AddTransform(boxB, &core.Transform{
		Position: fmath.Vec3{X: 10, Y: 3},
		Scale:    fmath.Vec3{X: 1, Y: 1, Z: 1},
	})
	world.AddDrawable(boxB, &bitmap.Rect{
		Width:  15,
		Height: 6,
		FG:     core.Color{R: 60, G: 200, B: 60},
		BG:     core.Color{R: 20, G: 80, B: 20},
	})
	world.AddLayer(boxB, 1)
	world.AddMaterial(boxB, func(f core.Fragment) core.Cell {
		f.Cell.FGAlpha = 0.8
		f.Cell.BGAlpha = 0.8
		return f.Cell
	})
	world.AddRoot(boxB)

	// Layer 2: blue box (Screen blend), overlaps both.
	boxC := world.Spawn()
	world.AddTransform(boxC, &core.Transform{
		Position: fmath.Vec3{X: 15, Y: 4},
		Scale:    fmath.Vec3{X: 1, Y: 1, Z: 1},
	})
	world.AddDrawable(boxC, &bitmap.Rect{
		Width:  15,
		Height: 6,
		FG:     core.Color{R: 60, G: 60, B: 200},
		BG:     core.Color{R: 20, G: 20, B: 80},
	})
	world.AddLayer(boxC, 2)
	world.AddMaterial(boxC, func(f core.Fragment) core.Cell {
		f.Cell.FGAlpha = 0.8
		f.Cell.BGAlpha = 0.8
		return f.Cell
	})
	world.AddRoot(boxC)

	comp := core.NewCompositor(w, h)
	comp.SetBlend(1, core.MultiplyColorBlend)
	comp.SetBlend(2, core.ScreenColorBlend)

	canvas.Clear()
	canvas.DrawBorder()
	comp.Composite(world, canvas, core.Time{})
	screen.Flush(canvas)

	var b strings.Builder
	for i, frame := range screen.Frames() {
		fmt.Fprintf(&b, "--- frame %d ---\n", i)
		b.WriteString(frame)
		b.WriteByte('\n')
	}

	g := goldie.New(t)
	g.Assert(t, "blend_modes", []byte(b.String()))
}

func TestBitmapRendering(t *testing.T) {
	const (
		w = 40
		h = 16
	)

	screen := terminal.NewSimScreen(w, h)

	world := core.NewWorld()

	// Entity 1: Braille diagonal line.
	brailleBm := bitmap.New(16, 16)
	for i := range 16 {
		brailleBm.SetDot(i, i, core.Color{R: 0, G: 255, B: 100})
	}
	brailleEnt := world.Spawn()
	world.AddTransform(brailleEnt, &core.Transform{
		Position: fmath.Vec3{X: 1, Y: 1},
		Scale:    fmath.Vec3{X: 1, Y: 1, Z: 1},
	})
	world.AddDrawable(brailleEnt, &bitmap.Braille{
		Bitmap: brailleBm,
	})
	world.AddRoot(brailleEnt)

	// Entity 2: Half-block gradient.
	hbBm := bitmap.New(10, 8)
	for y := range 8 {
		for x := range 10 {
			r := uint8(x * 25)
			b := uint8(y * 30)
			hbBm.SetDot(x, y, core.Color{R: r, G: 0, B: b})
		}
	}
	hbEnt := world.Spawn()
	world.AddTransform(hbEnt, &core.Transform{
		Position: fmath.Vec3{X: 20, Y: 2},
		Scale:    fmath.Vec3{X: 1, Y: 1, Z: 1},
	})
	world.AddDrawable(hbEnt, &bitmap.HalfBlock{
		Bitmap: hbBm,
	})
	world.AddRoot(hbEnt)

	canvas := core.NewCanvas(w, h)
	canvas.Clear()
	canvas.DrawBorder()
	core.Render(world, canvas, core.Time{})
	screen.Flush(canvas)

	var bb strings.Builder
	for i, frame := range screen.Frames() {
		fmt.Fprintf(&bb, "--- frame %d ---\n", i)
		bb.WriteString(frame)
		bb.WriteByte('\n')
	}

	g := goldie.New(t)
	g.Assert(t, "bitmap_rendering", []byte(bb.String()))
}

func TestTransformRotation(t *testing.T) {
	const (
		w      = 30
		h      = 12
		frames = 4
		dt     = 0.25
	)

	screen := terminal.NewSimScreen(w, h)
	canvas := core.NewCanvas(w, h)

	world := core.NewWorld()

	// A small box with rotation that changes each frame.
	box := world.Spawn()
	world.AddTransform(box, &core.Transform{
		Position: fmath.Vec3{X: 10, Y: 3},
		Scale:    fmath.Vec3{X: 1, Y: 1, Z: 1},
	})
	world.AddDrawable(box, &bitmap.Rect{
		Width:  8,
		Height: 4,
		FG:     core.Color{R: 200, G: 100, B: 50},
	})
	world.AddRoot(box)

	// Child entity offset from parent — inherits parent's transform.
	child := world.Spawn()
	world.AddTransform(child, &core.Transform{
		Position: fmath.Vec3{X: 10, Y: 0},
		Scale:    fmath.Vec3{X: 1, Y: 1, Z: 1},
	})
	world.AddDrawable(child, &bitmap.Rect{
		Width:  4,
		Height: 2,
		FG:     core.Color{R: 50, G: 200, B: 100},
	})
	world.Attach(child, box)

	world.AddBehavior(box, func(t core.Time, e core.Entity, w *core.World) {
		w.Transform(e).Rotation = t.Total * fmath.DegToRad(90)
	})

	for i := range frames {
		ti := core.Time{
			Total: float64(i+1) * dt,
			Delta: dt,
		}
		core.UpdateBehaviors(world, ti)

		canvas.Clear()
		canvas.DrawBorder()
		core.Render(world, canvas, ti)
		screen.Flush(canvas)
	}

	var b strings.Builder
	for i, frame := range screen.Frames() {
		fmt.Fprintf(&b, "--- frame %d ---\n", i)
		b.WriteString(frame)
		b.WriteByte('\n')
	}

	g := goldie.New(t)
	g.Assert(t, "transform_rotation", []byte(b.String()))
}

func TestCameraStaticPan(t *testing.T) {
	const (
		w = 40
		h = 12
	)

	screen := terminal.NewSimScreen(w, h)
	canvas := core.NewCanvas(w, h)

	world := core.NewWorld()

	// A box at world position (0,0).
	box := world.Spawn()
	world.AddTransform(box, &core.Transform{
		Position: fmath.Vec3{X: 0, Y: 0},
		Scale:    fmath.Vec3{X: 1, Y: 1, Z: 1},
	})
	world.AddDrawable(box, &bitmap.Rect{
		Width:  8,
		Height: 4,
		FG:     core.Color{R: 200, G: 100, B: 50},
	})
	world.AddRoot(box)

	// Camera panned to (10, 3) — box should appear shifted on screen.
	cam := world.Spawn()
	world.AddTransform(cam, &core.Transform{
		Position: fmath.Vec3{X: 10, Y: 3},
		Scale:    fmath.Vec3{X: 1, Y: 1, Z: 1},
	})
	world.AddCamera(cam, &core.Camera{Zoom: 1})
	world.SetActiveCamera(cam)

	canvas.Clear()
	canvas.DrawBorder()
	core.Render(world, canvas, core.Time{})
	screen.Flush(canvas)

	var b strings.Builder
	for i, frame := range screen.Frames() {
		fmt.Fprintf(&b, "--- frame %d ---\n", i)
		b.WriteString(frame)
		b.WriteByte('\n')
	}

	g := goldie.New(t)
	g.Assert(t, "camera_static_pan", []byte(b.String()))
}

func TestCameraAnimatedZoom(t *testing.T) {
	const (
		w      = 40
		h      = 12
		frames = 3
		dt     = 1.0
	)

	screen := terminal.NewSimScreen(w, h)
	canvas := core.NewCanvas(w, h)

	world := core.NewWorld()

	// A box at world origin.
	box := world.Spawn()
	world.AddTransform(box, &core.Transform{
		Position: fmath.Vec3{X: 0, Y: 0},
		Scale:    fmath.Vec3{X: 1, Y: 1, Z: 1},
	})
	world.AddDrawable(box, &bitmap.Rect{
		Width:  6,
		Height: 3,
		FG:     core.Color{R: 100, G: 200, B: 100},
	})
	world.AddRoot(box)

	// Camera at origin, zoom animated via behavior.
	cam := world.Spawn()
	world.AddTransform(cam, &core.Transform{
		Scale: fmath.Vec3{X: 1, Y: 1, Z: 1},
	})
	camComp := &core.Camera{Zoom: 1}
	world.AddCamera(cam, camComp)
	world.SetActiveCamera(cam)

	world.AddBehavior(cam, func(t core.Time, e core.Entity, w *core.World) {
		// Zoom: 1 → 2 → 3 over frames.
		w.Camera(e).Zoom = t.Total + 1.0
	})

	for i := range frames {
		ti := core.Time{
			Total: float64(i+1) * dt,
			Delta: dt,
		}
		core.UpdateBehaviors(world, ti)

		canvas.Clear()
		canvas.DrawBorder()
		core.Render(world, canvas, ti)
		screen.Flush(canvas)
	}

	var b strings.Builder
	for i, frame := range screen.Frames() {
		fmt.Fprintf(&b, "--- frame %d ---\n", i)
		b.WriteString(frame)
		b.WriteByte('\n')
	}

	g := goldie.New(t)
	g.Assert(t, "camera_animated_zoom", []byte(b.String()))
}

func TestTextRendering(t *testing.T) {
	const (
		w = 40
		h = 10
	)

	f, err := asset.LoadFont("Oxanium/static/Oxanium-Regular.ttf")
	if err != nil {
		t.Fatalf("LoadFont: %v", err)
	}

	textLayout := asset.RasterizeText("Hi", asset.TextOptions{
		Font:  f,
		Size:  16,
		Color: core.Color{R: 255, G: 255, B: 255},
	})
	if textLayout == nil {
		t.Fatal("RasterizeText returned nil")
	}

	screen := terminal.NewSimScreen(w, h)
	canvas := core.NewCanvas(w, h)

	world := core.NewWorld()
	textEnt := world.Spawn()
	world.AddTransform(textEnt, &core.Transform{
		Position: fmath.Vec3{X: 2, Y: 2},
		Scale:    fmath.Vec3{X: 1, Y: 1, Z: 1},
	})
	world.AddDrawable(textEnt, &bitmap.HalfBlock{Bitmap: textLayout.Bitmap})
	world.AddRoot(textEnt)

	canvas.Clear()
	canvas.DrawBorder()
	core.Render(world, canvas, core.Time{})
	screen.Flush(canvas)

	var b strings.Builder
	for i, frame := range screen.Frames() {
		fmt.Fprintf(&b, "--- frame %d ---\n", i)
		b.WriteString(frame)
		b.WriteByte('\n')
	}

	g := goldie.New(t)
	g.Assert(t, "text_rendering", []byte(b.String()))
}

func TestOBJWireframe(t *testing.T) {
	const (
		w      = 40
		h      = 20
		frames = 3
		dt     = 0.5
	)

	mesh, err := asset.LoadOBJ("suzanne.obj")
	if err != nil {
		t.Fatalf("LoadOBJ: %v", err)
	}

	screen := terminal.NewSimScreen(w, h)
	canvas := core.NewCanvas(w, h)

	world := core.NewWorld()

	objBm := bitmap.New(60, 60)
	objEnt := world.Spawn()
	world.AddTransform(objEnt, &core.Transform{
		Position: fmath.Vec3{X: 5, Y: 2},
		Scale:    fmath.Vec3{X: 1, Y: 1, Z: 1},
	})
	objDraw := &bitmap.Braille{Bitmap: objBm}
	world.AddDrawable(objEnt, objDraw)
	world.AddRoot(objEnt)

	proj := fmath.Mat4Perspective(math.Pi/3, 1.0, 0.1, 100)
	view := fmath.Mat4Translate(0, 0, -3)

	world.AddBehavior(objEnt, func(t core.Time, e core.Entity, w *core.World) {
		objBm.Clear()
		model := fmath.Mat4RotateY(t.Total * 0.8).Multiply(fmath.Mat4RotateX(0.3))
		mvp := proj.Multiply(view).Multiply(model)
		asset.RasterizeWireframe(mesh, mvp, objBm, core.Color{R: 180, G: 255, B: 200})
	})

	for i := range frames {
		ti := core.Time{
			Total: float64(i+1) * dt,
			Delta: dt,
		}
		core.UpdateBehaviors(world, ti)

		canvas.Clear()
		canvas.DrawBorder()
		core.Render(world, canvas, ti)
		screen.Flush(canvas)
	}

	var b strings.Builder
	for i, frame := range screen.Frames() {
		fmt.Fprintf(&b, "--- frame %d ---\n", i)
		b.WriteString(frame)
		b.WriteByte('\n')
	}

	g := goldie.New(t)
	g.Assert(t, "obj_wireframe", []byte(b.String()))
}

func TestParticleAttractor(t *testing.T) {
	const (
		w      = 40
		h      = 20
		frames = 10
		dt     = 0.1
	)

	screen := terminal.NewSimScreen(w, h)
	canvas := core.NewCanvas(w, h)

	world := core.NewWorld()

	// Import physics package for this test.
	// (The import is at the top of the file, but we'll reference it here.)
	// Create a circular attractor pattern: 5 particles in a circle around center,
	// with an attractor force at center. They should converge over time.

	center := fmath.Vec2{X: 20, Y: 10}
	radius := 8.0
	particleCount := 5

	// Single pixel bitmap for particles.
	pixel := bitmap.New(1, 1)
	pixel.SetDot(0, 0, core.Color{R: 100, G: 200, B: 255})

	for i := range particleCount {
		angle := float64(i) * 2.0 * math.Pi / float64(particleCount)
		x := center.X + radius*math.Cos(angle)
		y := center.Y + radius*math.Sin(angle)

		p := world.Spawn()
		world.AddTransform(p, &core.Transform{
			Position: fmath.Vec3{X: x, Y: y},
			Scale:    fmath.Vec3{X: 1, Y: 1, Z: 1},
		})
		world.AddBody(p, &core.Body{})
		world.AddDrawable(p, &bitmap.Braille{Bitmap: pixel})
		world.AddRoot(p)

		// Add combined physics behavior: integration, attractor, and drag.
		euler := physics.EulerIntegration()
		attractor := physics.Attractor(center, 200.0)
		drag := physics.Drag(0.5)

		world.AddBehavior(p, func(t core.Time, e core.Entity, w *core.World) {
			attractor(t, e, w)
			drag(t, e, w)
			euler(t, e, w)
		})
	}

	for i := range frames {
		ti := core.Time{
			Total: float64(i+1) * dt,
			Delta: dt,
		}
		core.UpdateBehaviors(world, ti)

		canvas.Clear()
		canvas.DrawBorder()
		core.Render(world, canvas, ti)
		screen.Flush(canvas)
	}

	var b strings.Builder
	for i, frame := range screen.Frames() {
		fmt.Fprintf(&b, "--- frame %d ---\n", i)
		b.WriteString(frame)
		b.WriteByte('\n')
	}

	g := goldie.New(t)
	g.Assert(t, "particle_attractor", []byte(b.String()))
}

func TestParticleAppearance(t *testing.T) {
	const (
		w = 60
		h = 20
	)

	screen := terminal.NewSimScreen(w, h)
	world := core.NewWorld()

	// Single pixel bitmap for particles.
	pixel := bitmap.New(1, 1)
	pixel.SetDot(0, 0, core.Color{R: 255, G: 255, B: 255})

	// Create particles with different velocities to test directional appearance.
	positions := []struct {
		x, y   float64
		vx, vy float64
		label  string
	}{
		{10, 5, 10, 0, "East"},   // moving right
		{20, 5, 0, 10, "South"},  // moving down
		{30, 5, -10, 0, "West"},  // moving left
		{40, 5, 0, -10, "North"}, // moving up
		{10, 10, 7, 7, "SE"},     // southeast
		{20, 10, -7, 7, "SW"},    // southwest
		{30, 10, 7, -7, "NE"},    // northeast
		{40, 10, -7, -7, "NW"},   // northwest
		{50, 5, 0, 0, "Idle"},    // stationary (should show default)
		{50, 10, 20, 0, "Fast"},  // fast particle (red)
		{50, 15, 1, 0, "Slow"},   // slow particle (blue)
	}

	for _, p := range positions {
		ent := world.Spawn()
		world.AddTransform(ent, &core.Transform{
			Position: fmath.Vec3{X: p.x, Y: p.y},
			Scale:    fmath.Vec3{X: 1, Y: 1, Z: 1},
		})
		world.AddBody(ent, &core.Body{
			Velocity: fmath.Vec2{X: p.vx, Y: p.vy},
		})
		world.AddDrawable(ent, &bitmap.Braille{Bitmap: pixel})

		// Add particle materials for dynamic appearance.
		world.AddMaterial(ent, core.ComposeMaterials(
			particle.BrailleDirectional(),
			particle.VelocityColor(particle.ColorGradient{
				MinSpeed: 0.0,
				MaxSpeed: 20.0,
				MinColor: core.Color{R: 100, G: 150, B: 255}, // blue = slow
				MaxColor: core.Color{R: 255, G: 100, B: 100}, // red = fast
			}),
		))

		world.AddRoot(ent)
	}

	canvas := core.NewCanvas(w, h)
	canvas.Clear()
	canvas.DrawBorder()
	core.Render(world, canvas, core.Time{})
	screen.Flush(canvas)

	var b strings.Builder
	for i, frame := range screen.Frames() {
		fmt.Fprintf(&b, "--- frame %d ---\n", i)
		b.WriteString(frame)
		b.WriteByte('\n')
	}

	g := goldie.New(t)
	g.Assert(t, "particle_appearance", []byte(b.String()))
}
