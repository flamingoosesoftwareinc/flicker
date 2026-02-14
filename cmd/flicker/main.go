package main

import (
	"fmt"
	"math"
	"os"
	"time"

	"flicker/asset"
	"flicker/core"
	"flicker/core/bitmap"
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
	world.AddDrawable(boxA, &bitmap.Rect{
		Width:   12,
		Height:  6,
		FG:      core.Color{R: 200, G: 60, B: 60},
		BG:      core.Color{R: 200, G: 60, B: 60},
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
	world.AddDrawable(boxB, &bitmap.Rect{
		Width:   12,
		Height:  6,
		FG:      core.Color{R: 60, G: 200, B: 60},
		BG:      core.Color{R: 60, G: 200, B: 60},
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
	world.AddDrawable(boxC, &bitmap.Rect{
		Width:   12,
		Height:  6,
		FG:      core.Color{R: 60, G: 60, B: 200},
		BG:      core.Color{R: 60, G: 60, B: 200},
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
	world.AddDrawable(boxD, &bitmap.Rect{
		Width:   12,
		Height:  6,
		FG:      core.Color{R: 200, G: 200, B: 60},
		BG:      core.Color{R: 200, G: 200, B: 60},
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
	world.AddDrawable(boxE, &bitmap.Rect{
		Width:   12,
		Height:  6,
		FG:      core.Color{R: 60, G: 200, B: 200},
		BG:      core.Color{R: 60, G: 200, B: 200},
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
	brailleBm := bitmap.New(30, 20)
	brailleEnt := world.Spawn()
	world.AddTransform(brailleEnt, &core.Transform{
		Position: fmath.Vec3{X: 2, Y: float64(sh - 7)},
		Scale:    fmath.Vec3{X: 1, Y: 1, Z: 1},
	})
	brailleDraw := &bitmap.Braille{Bitmap: brailleBm}
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
	hbBm := bitmap.New(16, 10)
	hbEnt := world.Spawn()
	world.AddTransform(hbEnt, &core.Transform{
		Position: fmath.Vec3{X: float64(sw - 20), Y: 2},
		Scale:    fmath.Vec3{X: 1, Y: 1, Z: 1},
	})
	hbDraw := &bitmap.HalfBlock{Bitmap: hbBm}
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
	pivotBm := bitmap.New(8, 8)
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
	world.AddDrawable(rotPivot, &bitmap.Braille{Bitmap: pivotBm})
	world.AddLayer(rotPivot, 7)
	world.AddRoot(rotPivot)

	// Child: offset from parent, orbits as parent rotates.
	orbiterBm := bitmap.New(6, 4)
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
	world.AddDrawable(orbiter, &bitmap.Braille{Bitmap: orbiterBm})
	world.Attach(orbiter, rotPivot)

	world.AddBehavior(rotPivot, func(t core.Time, e core.Entity, w *core.World) {
		tr := w.Transform(e)
		tr.Rotation = t.Total * fmath.DegToRad(60)
	})

	// Layer 8: Suzanne OBJ wireframe — spinning 3D model via braille.
	suzanneMesh, err := asset.LoadOBJ("suzanne.obj")
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: %v (skipping OBJ layer)\n", err)
	} else {
		objBm := bitmap.New(60, 60)
		objEnt := world.Spawn()
		world.AddTransform(objEnt, &core.Transform{
			Position: fmath.Vec3{X: float64(sw/2 - 15), Y: 1},
			Scale:    fmath.Vec3{X: 1, Y: 1, Z: 1},
		})
		objDraw := &bitmap.Braille{Bitmap: objBm}
		world.AddDrawable(objEnt, objDraw)
		world.AddLayer(objEnt, 8)
		world.AddRoot(objEnt)

		proj := fmath.Mat4Perspective(math.Pi/3, 1.0, 0.1, 100)
		view := fmath.Mat4Translate(0, 0, -3)

		world.AddBehavior(objEnt, func(t core.Time, e core.Entity, w *core.World) {
			objBm.Clear()
			model := fmath.Mat4RotateY(t.Total * 0.8).Multiply(fmath.Mat4RotateX(0.3))
			mvp := proj.Multiply(view).Multiply(model)
			c := core.Color{R: 180, G: 255, B: 200}
			asset.RasterizeWireframe(suzanneMesh, mvp, objBm, c)
		})
	}

	// Layer 9: Full-block plasma — 1:1 pixel-to-cell animated color field.
	fbBm := bitmap.New(14, 8)
	fbEnt := world.Spawn()
	world.AddTransform(fbEnt, &core.Transform{
		Position: fmath.Vec3{X: 2, Y: 1},
		Scale:    fmath.Vec3{X: 1, Y: 1, Z: 1},
	})
	world.AddDrawable(fbEnt, &bitmap.FullBlock{Bitmap: fbBm})
	world.AddLayer(fbEnt, 9)
	world.AddRoot(fbEnt)

	world.AddBehavior(fbEnt, func(t core.Time, e core.Entity, w *core.World) {
		for py := range 8 {
			for px := range 14 {
				fx := float64(px)
				fy := float64(py)
				v1 := math.Sin(fx*0.3 + t.Total*1.5)
				v2 := math.Sin(fy*0.4 + t.Total*1.2)
				v3 := math.Sin((fx+fy)*0.2 + t.Total*0.8)
				v := (v1 + v2 + v3 + 3) / 6 // normalize to [0,1]
				r := uint8(v * 255)
				g := uint8((1 - v) * 180)
				b := uint8(math.Abs(v-0.5) * 2 * 255)
				fbBm.Set(px, py, core.Color{R: r, G: g, B: b}, 1.0)
			}
		}
		v := fmath.Triangle(t.Total / 12.0)
		w.Transform(e).Position.X = fmath.Remap(0, 1, 2, float64(sw-16), v)
	})

	// Layer 10: BG-only color wash — transparent FG lets underlying content bleed through.
	bgBm := bitmap.New(16, 10)
	bgEnt := world.Spawn()
	world.AddTransform(bgEnt, &core.Transform{
		Position: fmath.Vec3{X: float64(sw/2 - 8), Y: float64(sh/2 - 5)},
		Scale:    fmath.Vec3{X: 1, Y: 1, Z: 1},
	})
	world.AddDrawable(bgEnt, &bitmap.BGBlock{Bitmap: bgBm})
	world.AddLayer(bgEnt, 10)
	world.AddRoot(bgEnt)

	world.AddBehavior(bgEnt, func(t core.Time, e core.Entity, w *core.World) {
		for py := range 10 {
			for px := range 16 {
				fx := float64(px)
				fy := float64(py)
				// Slow rolling color bands.
				r := uint8((math.Sin(fy*0.5+t.Total*0.6) + 1) / 2 * 200)
				g := uint8((math.Sin(fx*0.4+t.Total*0.4) + 1) / 2 * 160)
				b := uint8((math.Cos((fx+fy)*0.3+t.Total*0.5) + 1) / 2 * 220)
				bgBm.Set(px, py, core.Color{R: r, G: g, B: b}, 0.6)
			}
		}
		v := fmath.Triangle(t.Total / 11.0)
		w.Transform(e).Position.Y = fmath.Remap(0, 1, 1, float64(sh-12), v)
	})

	// Layer 11: Text — "FLICKER" with SDF threshold materialization.
	textFont, fontErr := asset.LoadFont("Oxanium/static/Oxanium-Bold.ttf")
	if fontErr != nil {
		fmt.Fprintf(os.Stderr, "warning: %v (skipping text layer)\n", fontErr)
	} else {
		textSize := float64(sh) * 1.4 // scale text to terminal height
		textBm := asset.RasterizeText("FLICKER", asset.TextOptions{
			Font:  textFont,
			Size:  textSize,
			Color: core.Color{R: 220, G: 240, B: 255},
		})
		if textBm != nil {
			textSDF := bitmap.ComputeSDF(textBm, 30)
			textDraw := &bitmap.HalfBlock{Bitmap: textBm}
			textBW, textBH := textDraw.Bounds()

			textEnt := world.Spawn()
			world.AddTransform(textEnt, &core.Transform{
				Position: fmath.Vec3{
					X: float64(sw/2) - float64(textBW)/2,
					Y: float64(sh/2) - float64(textBH)/2,
				},
				Scale: fmath.Vec3{X: 1, Y: 1, Z: 1},
			})
			world.AddDrawable(textEnt, textDraw)
			world.AddLayer(textEnt, 11)
			world.AddRoot(textEnt)

			// SDF threshold materialization: sweep threshold from
			// most-negative (skeleton) to 0 (full shape) over ~3 seconds.
			// After reveal, text remains fully visible.
			revealDuration := 3.0
			// Find the most-negative distance in the SDF (deepest interior).
			mostNeg := 0.0
			for _, d := range textSDF.Dist {
				if d < mostNeg {
					mostNeg = d
				}
			}

			world.AddMaterial(textEnt, func(f core.Fragment) core.Cell {
				// Map local coords to bitmap pixel coords for SDF lookup.
				// HalfBlock: ly is cell row, need pixel rows ly*2 and ly*2+1.
				topD := textSDF.At(f.X, f.Y*2)
				botD := textSDF.At(f.X, f.Y*2+1)

				// Compute threshold: sweeps from mostNeg to 0 over revealDuration.
				progress := f.Time.Total / revealDuration
				if progress > 1 {
					progress = 1
				}
				threshold := mostNeg * (1 - progress)

				// Discard pixels whose SDF distance is below the threshold
				// (i.e., they haven't been "revealed" yet).
				topVisible := topD <= threshold
				botVisible := botD <= threshold

				if !topVisible && !botVisible {
					return core.Cell{} // fully hidden
				}

				cell := f.Cell
				if !topVisible {
					// Only bottom half visible.
					cell.FGAlpha = 0
				}
				if !botVisible {
					// Only top half visible.
					cell.BGAlpha = 0
				}
				return cell
			})
		}
	}

	// Camera: gentle circular pan + zoom pulse.
	cam := world.Spawn()
	world.AddTransform(cam, &core.Transform{
		Position: fmath.Vec3{X: float64(sw) / 2.0, Y: float64(sh) / 2.0},
		Scale:    fmath.Vec3{X: 1, Y: 1, Z: 1},
	})
	world.AddCamera(cam, &core.Camera{Zoom: 1})
	world.SetActiveCamera(cam)

	// Camera rests at screen center so world (0,0) maps to screen (0,0).
	cx, cy := float64(sw)/2.0, float64(sh)/2.0
	world.AddBehavior(cam, func(t core.Time, e core.Entity, w *core.World) {
		tr := w.Transform(e)
		// Gentle circular pan: radius 3, period ~20s, centered on neutral position.
		tr.Position.X = cx + 3*math.Cos(t.Total*0.3)
		tr.Position.Y = cy + 3*math.Sin(t.Total*0.3)
		// Zoom pulse: oscillates between 0.7 and 1.3 (~10s period).
		w.Camera(e).Zoom = 1.0 + 0.3*math.Sin(t.Total*0.6)
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
