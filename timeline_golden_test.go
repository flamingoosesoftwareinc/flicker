package flicker_test

import (
	"fmt"
	"strings"
	"testing"

	"flicker/core"
	"flicker/core/bitmap"
	"flicker/fmath"
	"flicker/terminal"
	"github.com/sebdah/goldie/v2"
)

// TestTimelinePropertyAnimation captures Timeline property animations over multiple frames.
func TestTimelinePropertyAnimation(t *testing.T) {
	const (
		w      = 40
		h      = 12
		frames = 6
		dt     = 0.5 // 0.5s per frame
	)

	screen := terminal.NewSimScreen(w, h)
	canvas := core.NewCanvas(w, h)

	world := core.NewWorld()
	timeline := core.NewTimeline(world)

	// Create a box that will slide from left to right
	box := world.Spawn()
	world.AddTransform(box, &core.Transform{
		Position: fmath.Vec3{X: 2, Y: 4},
		Scale:    fmath.Vec3{X: 1, Y: 1, Z: 1},
	})
	world.AddDrawable(box, &bitmap.Rect{
		Width:  6,
		Height: 3,
		FG:     core.Color{R: 100, G: 200, B: 255},
	})
	world.AddRoot(box)

	// Animate position.x from 2 to 30 over 2 seconds
	track := timeline.AddTrack()
	track.Add(core.NewPropertyTweenClip(box, "position.x", 2.0, 30.0, 2.0).
		WithEasing(fmath.EaseInOutQuad))

	timeline.Start()

	// Capture frames over time
	for i := range frames {
		ti := core.Time{
			Total: float64(i) * dt,
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
		fmt.Fprintf(&b, "--- frame %d (t=%.1fs) ---\n", i, float64(i)*dt)
		b.WriteString(frame)
		b.WriteByte('\n')
	}

	g := goldie.New(t)
	g.Assert(t, "timeline_property_animation", []byte(b.String()))
}

// TestTimelineSequencedCallbacks captures Timeline with delayed callbacks.
func TestTimelineSequencedCallbacks(t *testing.T) {
	const (
		w      = 50
		h      = 10
		frames = 8
		dt     = 0.5
	)

	screen := terminal.NewSimScreen(w, h)
	canvas := core.NewCanvas(w, h)

	world := core.NewWorld()
	timeline := core.NewTimeline(world)

	// Track which boxes are visible
	var box1, box2, box3 core.Entity

	track := timeline.AddTrack()

	// Spawn box 1 at t=0.5
	track.At(0.5, core.NewCallbackClip(func(w *core.World, _ core.Time) {
		box1 = w.Spawn()
		w.AddTransform(box1, &core.Transform{
			Position: fmath.Vec3{X: 5, Y: 3},
			Scale:    fmath.Vec3{X: 1, Y: 1, Z: 1},
		})
		w.AddDrawable(box1, &bitmap.Rect{
			Width:  8,
			Height: 3,
			FG:     core.Color{R: 255, G: 100, B: 100},
		})
		w.AddRoot(box1)
	}))

	// Spawn box 2 at t=1.5
	track.At(1.5, core.NewCallbackClip(func(w *core.World, _ core.Time) {
		box2 = w.Spawn()
		w.AddTransform(box2, &core.Transform{
			Position: fmath.Vec3{X: 18, Y: 3},
			Scale:    fmath.Vec3{X: 1, Y: 1, Z: 1},
		})
		w.AddDrawable(box2, &bitmap.Rect{
			Width:  8,
			Height: 3,
			FG:     core.Color{R: 100, G: 255, B: 100},
		})
		w.AddRoot(box2)
	}))

	// Spawn box 3 at t=2.5
	track.At(2.5, core.NewCallbackClip(func(w *core.World, _ core.Time) {
		box3 = w.Spawn()
		w.AddTransform(box3, &core.Transform{
			Position: fmath.Vec3{X: 31, Y: 3},
			Scale:    fmath.Vec3{X: 1, Y: 1, Z: 1},
		})
		w.AddDrawable(box3, &bitmap.Rect{
			Width:  8,
			Height: 3,
			FG:     core.Color{R: 100, G: 100, B: 255},
		})
		w.AddRoot(box3)
	}))

	timeline.Start()

	for i := range frames {
		ti := core.Time{
			Total: float64(i) * dt,
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
		fmt.Fprintf(&b, "--- frame %d (t=%.1fs) ---\n", i, float64(i)*dt)
		b.WriteString(frame)
		b.WriteByte('\n')
	}

	g := goldie.New(t)
	g.Assert(t, "timeline_sequenced_callbacks", []byte(b.String()))
}

// TestTimelineParallelTracks captures Timeline with parallel track execution.
func TestTimelineParallelTracks(t *testing.T) {
	const (
		w      = 40
		h      = 12
		frames = 5
		dt     = 0.4
	)

	screen := terminal.NewSimScreen(w, h)
	canvas := core.NewCanvas(w, h)

	world := core.NewWorld()
	timeline := core.NewTimeline(world)

	// Box 1: Moves horizontally
	box1 := world.Spawn()
	world.AddTransform(box1, &core.Transform{
		Position: fmath.Vec3{X: 2, Y: 2},
		Scale:    fmath.Vec3{X: 1, Y: 1, Z: 1},
	})
	world.AddDrawable(box1, &bitmap.Rect{
		Width:  5,
		Height: 2,
		FG:     core.Color{R: 255, G: 100, B: 100},
	})
	world.AddRoot(box1)

	// Box 2: Moves vertically
	box2 := world.Spawn()
	world.AddTransform(box2, &core.Transform{
		Position: fmath.Vec3{X: 15, Y: 2},
		Scale:    fmath.Vec3{X: 1, Y: 1, Z: 1},
	})
	world.AddDrawable(box2, &bitmap.Rect{
		Width:  5,
		Height: 2,
		FG:     core.Color{R: 100, G: 255, B: 100},
	})
	world.AddRoot(box2)

	// Track 1: Move box 1 horizontally
	track1 := timeline.AddTrack()
	track1.Add(core.NewPropertyTweenClip(box1, "position.x", 2.0, 25.0, 1.6))

	// Track 2: Move box 2 vertically
	track2 := timeline.AddTrack()
	track2.Add(core.NewPropertyTweenClip(box2, "position.y", 2.0, 8.0, 1.6))

	timeline.Start()

	for i := range frames {
		ti := core.Time{
			Total: float64(i) * dt,
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
		fmt.Fprintf(&b, "--- frame %d (t=%.1fs) ---\n", i, float64(i)*dt)
		b.WriteString(frame)
		b.WriteByte('\n')
	}

	g := goldie.New(t)
	g.Assert(t, "timeline_parallel_tracks", []byte(b.String()))
}
