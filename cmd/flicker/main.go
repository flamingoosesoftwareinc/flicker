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

	// Box B: blue-ish, renders on top (added second).
	boxB := world.Spawn()
	world.AddTransform(boxB, &core.Transform{
		Position: fmath.Vec3{X: float64(sw - 14), Y: 5},
	})
	world.AddDrawable(boxB, &core.Rect{
		Width:  12,
		Height: 6,
		Rune:   '▓',
		FG:     core.Color{R: 60, G: 60, B: 200},
		BG:     core.Color{R: 0, G: 0, B: 40},
	})
	world.AddLayer(boxB, 1)
	world.AddRoot(boxB)

	world.AddBehavior(boxB, func(t core.Time, e core.Entity, w *core.World) {
		v := fmath.Triangle(t.Total / 8.0)
		w.Transform(e).Position.X = fmath.Remap(0, 1, float64(sw-14), 2, v)
	})

	world.AddMaterial(boxB, func(x, y int, t core.Time, cell core.Cell) core.Cell {
		cell.Alpha = 0.5
		gradient := float64(y) / 5.0
		pulse := (math.Sin(2*math.Pi*t.Total+math.Pi) + 1) / 2 // offset phase
		brightness := gradient*0.5 + pulse*0.5
		cell.FG = core.Color{
			R: uint8(float64(cell.FG.R) * brightness),
			G: uint8(float64(cell.FG.G) * brightness),
			B: uint8(float64(cell.FG.B) * brightness),
		}
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

	start := time.Now()
	last := start

	for {
		// Drain events (non-blocking).
		done := false
		for !done {
			select {
			case ev := <-events:
				if _, ok := ev.(*tcell.EventKey); ok {
					return
				}
			default:
				done = true
			}
		}

		now := time.Now()
		t := core.Time{
			Total: now.Sub(start).Seconds(),
			Delta: now.Sub(last).Seconds(),
		}
		last = now

		core.UpdateBehaviors(world, t)

		canvas.Clear()
		canvas.DrawBorder()
		comp.Composite(world, canvas, t)
		screen.Flush(canvas)
	}
}
