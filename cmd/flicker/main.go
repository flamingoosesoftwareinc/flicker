package main

import (
	"fmt"
	"os"

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
		Position: fmath.Vec2{X: 10, Y: 5},
	})
	world.AddGeometry(box, &core.Geometry{
		Kind:   core.GeoRect,
		Width:  20,
		Height: 10,
		Rune:   '█',
	})
	world.AddRoot(box)

	canvas.Clear()
	canvas.DrawBorder()
	core.Render(world, canvas)
	screen.Flush(canvas)

	for {
		ev := screen.PollEvent()
		if _, ok := ev.(*tcell.EventKey); ok {
			return
		}
	}
}
