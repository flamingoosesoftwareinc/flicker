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
	canvas.Background = core.Cell{
		BG:      core.Color{R: 155, G: 155, B: 155},
		FGAlpha: 1,
		BGAlpha: 1,
	} // Opaque grey — blend modes need a real destination color.
	world := core.NewWorld()

	// Layer 0: Red box — Normal blend (base layer), slow seesaw left→right.
	boxA := world.Spawn()
	world.AddTransform(boxA, &core.Transform{
		Position: fmath.Vec3{X: 2, Y: 5},
		Scale:    fmath.Vec3{X: 1, Y: 1, Z: 1},
	})
	world.AddDrawable(boxA, &core.Rect{
		Width:   12,
		Height:  6,
		Rune:    '░',
		FG:      core.Color{R: 180, G: 120, B: 60},
		BG:      core.Color{R: 180, G: 120, B: 60},
		FGAlpha: 0.7,
		BGAlpha: 0.7,
	})
	world.AddLayer(boxA, 0)
	world.AddRoot(boxA)

	// Tween-based ping-pong with easing.
	tweenA := &fmath.Tween{
		From:     2,
		To:       float64(sw - 14),
		Duration: 8.0,
		Easing:   fmath.EaseInOutCubic,
	}
	forwardA := true
	world.AddBehavior(boxA, func(t core.Time, e core.Entity, w *core.World) {
		w.Transform(e).Position.X = tweenA.Update(t.Delta)
		if tweenA.Done() {
			tweenA.Reset()
			if forwardA {
				tweenA.From, tweenA.To = float64(sw-14), 2
			} else {
				tweenA.From, tweenA.To = 2, float64(sw-14)
			}
			forwardA = !forwardA
		}
	})

	world.AddMaterial(boxA, func(f core.Fragment) core.Cell {
		gradient := float64(f.Y) / 5.0
		pulse := (math.Sin(2*math.Pi*f.Time.Total) + 1) / 2
		brightness := gradient*0.5 + pulse*0.5
		f.Cell.FG = core.Color{
			R: uint8(float64(f.Cell.FG.R) * brightness),
			G: uint8(float64(f.Cell.FG.G) * brightness),
			B: uint8(float64(f.Cell.FG.B) * brightness),
		}
		return f.Cell
	})

	// Layer 1: Green box — Multiply blend, seesaw right→left (faster).
	boxB := world.Spawn()
	world.AddTransform(boxB, &core.Transform{
		Position: fmath.Vec3{X: float64(sw - 14), Y: 5},
		Scale:    fmath.Vec3{X: 1, Y: 1, Z: 1},
	})
	world.AddDrawable(boxB, &core.Rect{
		Width:   12,
		Height:  6,
		Rune:    '▒',
		FG:      core.Color{R: 180, G: 120, B: 60},
		BG:      core.Color{R: 180, G: 120, B: 60},
		FGAlpha: 0.7,
		BGAlpha: 0.7,
	})
	world.AddLayer(boxB, 1)
	world.AddRoot(boxB)

	world.AddBehavior(boxB, func(t core.Time, e core.Entity, w *core.World) {
		v := fmath.Triangle(t.Total / 8.0)
		w.Transform(e).Position.X = fmath.Remap(0, 1, float64(sw-14), 2, v)
	})

	// Layer 2: Blue box — Screen blend, vertical bounce.
	boxC := world.Spawn()
	world.AddTransform(boxC, &core.Transform{
		Position: fmath.Vec3{X: float64(sw/2 - 6), Y: 2},
		Scale:    fmath.Vec3{X: 1, Y: 1, Z: 1},
	})
	world.AddDrawable(boxC, &core.Rect{
		Width:   12,
		Height:  6,
		Rune:    '▓',
		FG:      core.Color{R: 180, G: 120, B: 60},
		BG:      core.Color{R: 180, G: 120, B: 60},
		FGAlpha: 0.7,
		BGAlpha: 0.7,
	})
	world.AddLayer(boxC, 2)
	world.AddRoot(boxC)

	world.AddBehavior(boxC, func(t core.Time, e core.Entity, w *core.World) {
		v := fmath.Triangle(t.Total / 6.0)
		w.Transform(e).Position.Y = fmath.Remap(0, 1, 1, float64(sh-8), v)
	})

	// Layer 3: Yellow box — Overlay blend, diagonal drift.
	boxD := world.Spawn()
	world.AddTransform(boxD, &core.Transform{
		Position: fmath.Vec3{X: 2, Y: 2},
		Scale:    fmath.Vec3{X: 1, Y: 1, Z: 1},
	})
	world.AddDrawable(boxD, &core.Rect{
		Width:   12,
		Height:  6,
		Rune:    '█',
		FG:      core.Color{R: 180, G: 120, B: 60},
		BG:      core.Color{R: 180, G: 120, B: 60},
		FGAlpha: 0.7,
		BGAlpha: 0.7,
	})
	world.AddLayer(boxD, 3)
	world.AddRoot(boxD)

	// TweenVec3-based diagonal ping-pong with easing.
	tweenD := &fmath.TweenVec3{
		From:     fmath.Vec3{X: 2, Y: 1},
		To:       fmath.Vec3{X: float64(sw - 14), Y: float64(sh - 8)},
		Duration: 5.0,
		Easing:   fmath.EaseInOutQuad,
	}
	forwardD := true
	world.AddBehavior(boxD, func(t core.Time, e core.Entity, w *core.World) {
		pos := tweenD.Update(t.Delta)
		w.Transform(e).Position.X = pos.X
		w.Transform(e).Position.Y = pos.Y
		if tweenD.Done() {
			tweenD.Reset()
			if forwardD {
				tweenD.From = fmath.Vec3{X: float64(sw - 14), Y: float64(sh - 8)}
				tweenD.To = fmath.Vec3{X: 2, Y: 1}
			} else {
				tweenD.From = fmath.Vec3{X: 2, Y: 1}
				tweenD.To = fmath.Vec3{X: float64(sw - 14), Y: float64(sh - 8)}
			}
			forwardD = !forwardD
		}
	})

	// Layer 4: Cyan box — Difference blend, opposite horizontal.
	boxE := world.Spawn()
	world.AddTransform(boxE, &core.Transform{
		Position: fmath.Vec3{X: float64(sw/2 - 6), Y: float64(sh/2 - 3)},
		Scale:    fmath.Vec3{X: 1, Y: 1, Z: 1},
	})
	world.AddDrawable(boxE, &core.Rect{
		Width:   12,
		Height:  6,
		Rune:    '◆',
		FG:      core.Color{R: 180, G: 120, B: 60},
		BG:      core.Color{R: 180, G: 120, B: 60},
		FGAlpha: 0.7,
		BGAlpha: 0.7,
	})
	world.AddLayer(boxE, 4)
	world.AddRoot(boxE)

	world.AddBehavior(boxE, func(t core.Time, e core.Entity, w *core.World) {
		v := fmath.Triangle(t.Total / 7.0)
		w.Transform(e).Position.X = fmath.Remap(0, 1, float64(sw-14), 2, v)
	})

	// Layer 5: Braille sine wave — sub-cell resolution wireframe.
	brailleBm := core.NewBitmap(30, 20)
	brailleEnt := world.Spawn()
	world.AddTransform(brailleEnt, &core.Transform{
		Position: fmath.Vec3{X: 2, Y: float64(sh - 7)},
		Scale:    fmath.Vec3{X: 1, Y: 1, Z: 1},
	})
	brailleDraw := &core.BitmapDrawable{Bitmap: brailleBm, Mode: core.EncodeBraille}
	world.AddDrawable(brailleEnt, brailleDraw)
	world.AddLayer(brailleEnt, 5)
	world.AddRoot(brailleEnt)

	world.AddBehavior(brailleEnt, func(t core.Time, e core.Entity, w *core.World) {
		brailleBm.Clear()
		phase := t.Total * 3.0
		for px := range 30 {
			// Sine wave mapped to bitmap Y range.
			fy := (math.Sin(float64(px)*0.4+phase) + 1) / 2 * 19
			py := int(fy)
			c := core.Color{R: 0, G: 220, B: uint8(120 + int(math.Sin(phase)*80))}
			brailleBm.SetDot(px, py, c)
			// Thicken: also plot the pixel above/below if in range.
			if py > 0 {
				brailleBm.SetDot(px, py-1, c)
			}
			if py < 19 {
				brailleBm.SetDot(px, py+1, c)
			}
		}
		v := fmath.Triangle(t.Total / 10.0)
		w.Transform(e).Position.X = fmath.Remap(0, 1, 2, float64(sw-17), v)
	})

	// Layer 6: Half-block color gradient — two colors per cell.
	hbBm := core.NewBitmap(16, 10)
	hbEnt := world.Spawn()
	world.AddTransform(hbEnt, &core.Transform{
		Position: fmath.Vec3{X: float64(sw - 20), Y: 2},
		Scale:    fmath.Vec3{X: 1, Y: 1, Z: 1},
	})
	hbDraw := &core.BitmapDrawable{Bitmap: hbBm, Mode: core.EncodeHalfBlock}
	world.AddDrawable(hbEnt, hbDraw)
	world.AddLayer(hbEnt, 6)
	world.AddRoot(hbEnt)

	world.AddBehavior(hbEnt, func(t core.Time, e core.Entity, w *core.World) {
		for py := range 10 {
			for px := range 16 {
				g := uint8(fmath.Clamp(float64(py)*28, 0, 255))
				rv := fmath.Triangle(float64(px)*0.06 + t.Total*0.3)
				bv := fmath.Triangle(float64(py)*0.1 + t.Total*0.2)
				r := uint8(rv * 255)
				b := uint8(bv * 255)
				hbBm.Set(px, py, core.Color{R: r, G: g, B: b}, 1.0)
			}
		}
		v := fmath.Triangle(t.Total / 9.0)
		w.Transform(e).Position.Y = fmath.Remap(0, 1, 1, float64(sh-7), v)
	})

	// Layer 7: Orbiting pair — parent rotates, child orbits around it.
	// Braille bitmaps give sub-cell rotation resolution.
	pivotBm := core.NewBitmap(8, 8)
	for py := range 8 {
		for px := range 8 {
			pivotBm.SetDot(px, py, core.Color{R: 255, G: 180, B: 0})
		}
	}
	rotPivot := world.Spawn()
	world.AddTransform(rotPivot, &core.Transform{
		Position: fmath.Vec3{X: float64(sw/2 - 3), Y: float64(sh/2 - 2)},
		Scale:    fmath.Vec3{X: 1, Y: 1, Z: 1},
	})
	world.AddDrawable(rotPivot, &core.BitmapDrawable{Bitmap: pivotBm, Mode: core.EncodeBraille})
	world.AddLayer(rotPivot, 7)
	world.AddRoot(rotPivot)

	// Child: offset from parent, orbits as parent rotates.
	orbiterBm := core.NewBitmap(6, 4)
	for py := range 4 {
		for px := range 6 {
			orbiterBm.SetDot(px, py, core.Color{R: 255, G: 100, B: 200})
		}
	}
	orbiter := world.Spawn()
	world.AddTransform(orbiter, &core.Transform{
		Position: fmath.Vec3{X: 10, Y: 0},
		Scale:    fmath.Vec3{X: 1, Y: 1, Z: 1},
	})
	world.AddDrawable(orbiter, &core.BitmapDrawable{Bitmap: orbiterBm, Mode: core.EncodeBraille})
	world.Attach(orbiter, rotPivot)

	world.AddBehavior(rotPivot, func(t core.Time, e core.Entity, w *core.World) {
		tr := w.Transform(e)
		tr.Rotation = t.Total * fmath.DegToRad(60)
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
