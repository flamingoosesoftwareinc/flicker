package flicker_test

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"testing"

	"flicker/core"
	"flicker/fmath"
	"flicker/terminal"
)

var update = flag.Bool("update", false, "update golden files")

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
	got := b.String()

	goldenPath := "testdata/basic_example.golden"

	if *update {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(goldenPath, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		t.Log("updated golden file")
		return
	}

	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("golden file not found (run with -update to create): %v", err)
	}

	if got != string(want) {
		t.Errorf("output does not match golden file %s\n\nGot:\n%s\nWant:\n%s", goldenPath, got, string(want))
	}
}
