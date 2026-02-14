package main

import (
	"fmt"
	"math"
	"os"
	"time"

	"flicker/core"
	"flicker/fmath"
	"flicker/terminal"
	"github.com/gdamore/tcell/v2"
)

func main() {
	screen, err := terminal.NewTcellScreen()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	defer screen.Fini()

	sw, sh := screen.Size()
	canvas := core.NewCanvas(sw, sh)
	world := core.NewWorld()

	// Layer 0: Red box — Normal blend (base layer), slow seesaw left→right.
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
	world.AddLayer(boxA, 0)
	world.AddRoot(boxA)

	world.AddBehavior(boxA, func(t core.Time, e core.Entity, w *core.World) {
		v := fmath.Triangle(t.Total / 16.0)
		w.Transform(e).Position.X = fmath.Remap(0, 1, 2, float64(sw-14), v)
	})

	world.AddMaterial(boxA, func(x, y int, t core.Time, cell core.Cell) core.Cell {
		gradient := float64(y) / 5.0
		pulse := (math.Sin(2*math.Pi*t.Total) + 1) / 2
		brightness := gradient*0.5 + pulse*0.5
		cell.FG = core.Color{
			R: uint8(float64(cell.FG.R) * brightness),
			G: uint8(float64(cell.FG.G) * brightness),
			B: uint8(float64(cell.FG.B) * brightness),
		}
		return cell
	})

	// Layer 1: Green box — Multiply blend, seesaw right→left (faster).
	boxB := world.Spawn()
	world.AddTransform(boxB, &core.Transform{
		Position: fmath.Vec3{X: float64(sw - 14), Y: 5},
	})
	world.AddDrawable(boxB, &core.Rect{
		Width:  12,
		Height: 6,
		Rune:   '▒',
		FG:     core.Color{R: 60, G: 200, B: 60},
		BG:     core.Color{R: 0, G: 40, B: 0},
	})
	world.AddLayer(boxB, 1)
	world.AddRoot(boxB)

	world.AddBehavior(boxB, func(t core.Time, e core.Entity, w *core.World) {
		v := fmath.Triangle(t.Total / 8.0)
		w.Transform(e).Position.X = fmath.Remap(0, 1, float64(sw-14), 2, v)
	})

	world.AddMaterial(boxB, func(x, y int, t core.Time, cell core.Cell) core.Cell {
		cell.Alpha = 0.7
		return cell
	})

	// Layer 2: Blue box — Screen blend, vertical bounce.
	boxC := world.Spawn()
	world.AddTransform(boxC, &core.Transform{
		Position: fmath.Vec3{X: float64(sw/2 - 6), Y: 2},
	})
	world.AddDrawable(boxC, &core.Rect{
		Width:  12,
		Height: 6,
		Rune:   '▓',
		FG:     core.Color{R: 60, G: 60, B: 200},
		BG:     core.Color{R: 0, G: 0, B: 40},
	})
	world.AddLayer(boxC, 2)
	world.AddRoot(boxC)

	world.AddBehavior(boxC, func(t core.Time, e core.Entity, w *core.World) {
		v := fmath.Triangle(t.Total / 6.0)
		w.Transform(e).Position.Y = fmath.Remap(0, 1, 1, float64(sh-8), v)
	})

	world.AddMaterial(boxC, func(x, y int, t core.Time, cell core.Cell) core.Cell {
		cell.Alpha = 0.7
		return cell
	})

	// Layer 3: Yellow box — Overlay blend, diagonal drift.
	boxD := world.Spawn()
	world.AddTransform(boxD, &core.Transform{
		Position: fmath.Vec3{X: 2, Y: 2},
	})
	world.AddDrawable(boxD, &core.Rect{
		Width:  12,
		Height: 6,
		Rune:   '█',
		FG:     core.Color{R: 200, G: 200, B: 60},
		BG:     core.Color{R: 40, G: 40, B: 0},
	})
	world.AddLayer(boxD, 3)
	world.AddRoot(boxD)

	world.AddBehavior(boxD, func(t core.Time, e core.Entity, w *core.World) {
		vx := fmath.Triangle(t.Total / 10.0)
		vy := fmath.Triangle(t.Total / 12.0)
		w.Transform(e).Position.X = fmath.Remap(0, 1, 2, float64(sw-14), vx)
		w.Transform(e).Position.Y = fmath.Remap(0, 1, 1, float64(sh-8), vy)
	})

	world.AddMaterial(boxD, func(x, y int, t core.Time, cell core.Cell) core.Cell {
		cell.Alpha = 0.7
		return cell
	})

	// Layer 4: Cyan box — Difference blend, opposite horizontal.
	boxE := world.Spawn()
	world.AddTransform(boxE, &core.Transform{
		Position: fmath.Vec3{X: float64(sw/2 - 6), Y: float64(sh/2 - 3)},
	})
	world.AddDrawable(boxE, &core.Rect{
		Width:  12,
		Height: 6,
		Rune:   '◆',
		FG:     core.Color{R: 60, G: 200, B: 200},
		BG:     core.Color{R: 0, G: 40, B: 40},
	})
	world.AddLayer(boxE, 4)
	world.AddRoot(boxE)

	world.AddBehavior(boxE, func(t core.Time, e core.Entity, w *core.World) {
		v := fmath.Triangle(t.Total / 7.0)
		w.Transform(e).Position.X = fmath.Remap(0, 1, float64(sw-14), 2, v)
	})

	world.AddMaterial(boxE, func(x, y int, t core.Time, cell core.Cell) core.Cell {
		cell.Alpha = 0.7
		return cell
	})

	// Pump PollEvent in a goroutine so the tick loop never blocks on input.
	events := make(chan tcell.Event, 1)
	go func() {
		for {
			events <- screen.PollEvent()
		}
	}()

	comp := core.NewCompositor(sw, sh)
	comp.SetBlend(1, core.MultiplyColorBlend)
	comp.SetBlend(2, core.ScreenColorBlend)
	comp.SetBlend(3, core.OverlayColorBlend)
	comp.SetBlend(4, core.DifferenceColorBlend)

	const stepSize = 1.0 / 60.0

	var simTime float64
	paused := false
	last := time.Now()

	for {
		// Drain events (non-blocking).
		step := false
		quit := false
		for draining := true; draining; {
			select {
			case ev := <-events:
				if kev, ok := ev.(*tcell.EventKey); ok {
					switch {
					case kev.Key() == tcell.KeyEscape:
						quit = true
					case kev.Key() == tcell.KeyRight:
						step = true
					case kev.Rune() == ' ':
						paused = !paused
					case kev.Rune() == '.':
						step = true
					case kev.Rune() == 'q':
						quit = true
					}
				}
			default:
				draining = false
			}
		}
		if quit {
			return
		}

		now := time.Now()
		wallDelta := now.Sub(last).Seconds()
		last = now

		var simDelta float64
		switch {
		case step:
			simDelta = stepSize
			paused = true
		case !paused:
			simDelta = wallDelta
		}
		simTime += simDelta

		t := core.Time{
			Total: simTime,
			Delta: simDelta,
		}

		core.UpdateBehaviors(world, t)

		canvas.Clear()
		canvas.DrawBorder()
		comp.Composite(world, canvas, t)
		screen.Flush(canvas)
	}
}
