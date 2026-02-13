package main

import (
	"fmt"
	"os"
	"time"

	"flicker/core"
	"flicker/fmath"
	"flicker/terminal"
	"github.com/gdamore/tcell/v2"
)

const targetFPS = 60

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
	})
	world.AddRoot(box)

	elapsed := 0.0
	world.AddBehavior(box, func(dt float64, e core.Entity, w *core.World) {
		elapsed += dt
		v := fmath.Triangle(elapsed / 2.0)
		w.Transform(e).Position.X = fmath.Remap(0, 1, 5, 50, v)
	})

	// Pump PollEvent in a goroutine so the tick loop never blocks on input.
	events := make(chan tcell.Event, 1)
	go func() {
		for {
			events <- screen.PollEvent()
		}
	}()

	frameBudget := time.Second / targetFPS
	last := time.Now()

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
		dt := now.Sub(last).Seconds()
		last = now

		core.UpdateBehaviors(world, dt)

		canvas.Clear()
		canvas.DrawBorder()
		core.Render(world, canvas)
		screen.Flush(canvas)

		// Sleep remainder of frame budget.
		elapsed := time.Since(now)
		if sleep := frameBudget - elapsed; sleep > 0 {
			time.Sleep(sleep)
		}
	}
}
