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
		Position: fmath.Vec2{X: 10, Y: 5},
	})
	world.AddGeometry(box, &core.Geometry{
		Kind:   core.GeoRect,
		Width:  20,
		Height: 10,
		Rune:   '█',
	})
	world.AddRoot(box)

	canvas := core.NewCanvas(w, h)
	canvas.Clear()
	canvas.DrawBorder()
	core.Render(world, canvas)
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
