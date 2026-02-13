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

	w, h := screen.Size()
	canvas := core.NewCanvas(w, h)
	world := core.NewWorld()

	box := world.Spawn()
	world.AddTransform(box, &core.Transform{
		Position: fmath.Vec3{X: 10, Y: 5},
	})
	world.AddDrawable(box, &core.Rect{
		Width:  20,
		Height: 10,
		Rune:   '█',
		FG:     core.Color{R: 100, G: 149, B: 237}, // cornflower blue
		BG:     core.Color{R: 0, G: 0, B: 40},
	})
	world.AddRoot(box)

	world.AddBehavior(box, func(t core.Time, e core.Entity, w *core.World) {
		v := fmath.Triangle(t.Total / 2.0)
		w.Transform(e).Position.X = fmath.Remap(0, 1, 5, 50, v)
	})

	world.AddMaterial(box, func(x, y int, t core.Time, cell core.Cell) core.Cell {
		// Vertical gradient that pulses with time.
		_, bh := 20, 10
		gradient := float64(y) / float64(bh-1)
		pulse := (math.Sin(2*math.Pi*t.Total) + 1) / 2 // [0, 1]
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
		core.Render(world, canvas, t)
		screen.Flush(canvas)
	}
}
