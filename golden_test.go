package flicker_test

import (
	"fmt"
	"strings"
	"testing"

	"flicker/core"
	"flicker/fmath"
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
	})
	world.AddDrawable(box, &core.Rect{
		Width:  20,
		Height: 10,
		Rune:   '█',
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
	})
	world.AddDrawable(box, &core.Rect{
		Width:  10,
		Height: 5,
		Rune:   '#',
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
	})
	world.AddDrawable(boxA, &core.Rect{
		Width:  12,
		Height: 6,
		Rune:   '░',
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
	})
	world.AddDrawable(boxB, &core.Rect{
		Width:  12,
		Height: 6,
		Rune:   '▓',
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
