package main

import (
	"fmt"
	"os"
	"time"

	"flicker/asset"
	"flicker/core"
	"flicker/core/bitmap"
	"flicker/fmath"
	"flicker/particle"
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

	// Load font for text rendering.
	textFont, fontErr := asset.LoadFont("Oxanium/static/Oxanium-Bold.ttf")
	if fontErr != nil {
		fmt.Fprintf(os.Stderr, "error loading font: %v\n", fontErr)
		os.Exit(1)
	}

	// Make text MUCH larger to see particles traveling in different directions.
	textSize := float64(sh) * 1
	textOpts := asset.TextOptions{
		Font:  textFont,
		Size:  textSize,
		Color: core.Color{R: 255, G: 255, B: 255},
	}

	// Rasterize two text layouts: "GO" and "FLYING" (wider for more directional variety)
	layoutA := asset.RasterizeText("GO", textOpts)
	layoutB := asset.RasterizeText("FLYING", textOpts)

	if layoutA == nil || layoutB == nil {
		fmt.Fprintf(os.Stderr, "error: failed to rasterize text\n")
		os.Exit(1)
	}

	// Convert bitmaps to point clouds.
	cloudA := particle.BitmapToCloud(layoutA.Bitmap)
	cloudB := particle.BitmapToCloud(layoutB.Bitmap)

	if len(cloudA) == 0 || len(cloudB) == 0 {
		fmt.Fprintf(os.Stderr, "error: empty point clouds\n")
		os.Exit(1)
	}

	// Single pixel bitmap for particles.
	pixel := bitmap.New(1, 1)
	pixel.SetDot(0, 0, core.Color{R: 255, G: 255, B: 255})

	// Calculate center offset to center the text.
	offsetX := float64(sw/2) - float64(layoutA.Bitmap.Width)/2
	offsetY := float64(sh/2) - float64(layoutA.Bitmap.Height)/2

	// Spawn particles at cloud A positions.
	particles := make([]core.Entity, len(cloudA))
	for i, pos := range cloudA {
		p := world.Spawn()
		particles[i] = p
		world.AddTransform(p, &core.Transform{
			Position: fmath.Vec3{X: pos.X + offsetX, Y: pos.Y + offsetY},
			Scale:    fmath.Vec3{X: 1, Y: 1, Z: 1},
		})
		world.AddBody(p, &core.Body{})
		world.AddDrawable(p, &bitmap.Braille{Bitmap: pixel})

		// Add dynamic particle materials: directional appearance + velocity-based color
		world.AddMaterial(p, core.ComposeMaterials(
			particle.BrailleDirectional(),
			particle.VelocityColor(particle.ColorGradient{
				MinSpeed: 0.0,
				MaxSpeed: 20.0,
				MinColor: core.Color{R: 100, G: 150, B: 255}, // blue = slow
				MaxColor: core.Color{R: 255, G: 100, B: 100}, // red = fast
			}),
		))

		world.AddRoot(p)
	}

	// After 2 seconds, distribute targets from cloud B.
	targetDistributed := false
	startTime := 0.0

	// Add a behavior to the world to handle the morph trigger.
	morphTrigger := world.Spawn()
	world.AddBehavior(morphTrigger, func(t core.Time, e core.Entity, w *core.World) {
		if startTime == 0 {
			startTime = t.Total
		}

		// At 2 seconds, distribute targets.
		if !targetDistributed && t.Total-startTime >= 2.0 {
			targetDistributed = true
			// Offset cloud B to center it as well.
			offsetCloudB := make([]fmath.Vec2, len(cloudB))
			for i, pos := range cloudB {
				offsetCloudB[i] = fmath.Vec2{
					X: pos.X + offsetX,
					Y: pos.Y + offsetY,
				}
			}
			particle.DistributeTargets(particles, offsetCloudB, 10.0, w)
		}
	})

	// Camera.
	cam := world.Spawn()
	world.AddTransform(cam, &core.Transform{
		Position: fmath.Vec3{X: float64(sw) / 2.0, Y: float64(sh) / 2.0},
		Scale:    fmath.Vec3{X: 1, Y: 1, Z: 1},
	})
	world.AddCamera(cam, &core.Camera{Zoom: 1})
	world.SetActiveCamera(cam)

	// Pump PollEvent in a goroutine so the tick loop never blocks on input.
	events := make(chan tcell.Event, 1)
	go func() {
		for {
			events <- screen.PollEvent()
		}
	}()

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
		core.Render(world, canvas, t)
		screen.Flush(canvas)
	}
}
