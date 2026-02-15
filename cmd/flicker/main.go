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
	"flicker/physics"
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

	// Text rendering setup
	textSize := float64(sh) * 0.8
	textOpts := asset.TextOptions{
		Font:  textFont,
		Size:  textSize,
		Color: core.Color{R: 255, G: 255, B: 255},
	}

	// Create 10 different text morphs to stress-test the system
	words := []string{
		"GO",
		"WAVE",
		"SPIN",
		"MORPH",
		"BURST",
		"FLOW",
		"DANCE",
		"ZOOM",
		"TWIST",
		"FLY",
	}

	// Rasterize all text layouts
	layouts := make([]*asset.TextLayout, len(words))
	clouds := make([][]fmath.Vec2, len(words))

	for i, word := range words {
		layout := asset.RasterizeText(word, textOpts)
		if layout == nil {
			fmt.Fprintf(os.Stderr, "error: failed to rasterize text: %s\n", word)
			os.Exit(1)
		}
		layouts[i] = layout
		clouds[i] = particle.BitmapToCloud(layout.Bitmap)
	}

	// Single pixel bitmap for particles.
	pixel := bitmap.New(1, 1)
	pixel.SetDot(0, 0, core.Color{R: 255, G: 255, B: 255})

	// Calculate center offset for initial cloud.
	offsetX := float64(sw/2) - float64(layouts[0].Bitmap.Width)/2
	offsetY := float64(sh/2) - float64(layouts[0].Bitmap.Height)/2

	// Offset initial cloud
	initialCloud := make([]fmath.Vec2, len(clouds[0]))
	for i, pos := range clouds[0] {
		initialCloud[i] = fmath.Vec2{
			X: pos.X + offsetX,
			Y: pos.Y + offsetY,
		}
	}

	// Create particle material: directional appearance + time-based rainbow
	material := core.ComposeMaterials(
		particle.BrailleDirectional(),
		particle.RainbowTime(2.0),
	)

	// Create point cloud sequence
	seq := particle.NewPointCloudSequence(
		world,
		initialCloud,
		&bitmap.Braille{Bitmap: pixel},
		material,
		0, // layer
	)

	// Add turbulence to initial particles
	turbConfig := &particle.TurbulenceConfig{Scale: 0.05, Strength: 30.0}
	for _, p := range seq.Particles() {
		tb := world.AddBehavior(p, core.NewBehavior(physics.Turbulence(turbConfig.Scale, turbConfig.Strength))).(*core.FuncBehavior)
		tb.SetEnabled(true)
	}

	// Distribution strategies to cycle through
	strategies := []particle.DistributionStrategy{
		particle.LinearDistribution(),
		particle.RoundRobinDistribution(),
		nil, // placeholder for ClosestPoint (needs runtime state)
	}

	// Add morph targets with varying settings
	for i := 1; i < len(words); i++ {
		targetCloud := clouds[i]
		targetLayout := layouts[i]

		// Center target cloud
		targetOffsetX := float64(sw/2) - float64(targetLayout.Bitmap.Width)/2
		offsetTargetCloud := make([]fmath.Vec2, len(targetCloud))
		for j, pos := range targetCloud {
			offsetTargetCloud[j] = fmath.Vec2{
				X: pos.X + targetOffsetX,
				Y: pos.Y + offsetY,
			}
		}

		// Cycle through strategies
		strategyIdx := i % 3
		var strategy particle.DistributionStrategy
		if strategyIdx == 2 {
			// ClosestPoint needs current particle state - will be computed at morph time
			// For now, use a closure that captures seq
			strategy = particle.LinearDistribution() // Fallback
		} else {
			strategy = strategies[strategyIdx]
		}

		// Toggle turbulence every other morph
		var turb *particle.TurbulenceConfig
		if i%2 == 0 {
			turb = turbConfig
		}

		// Use burst transition for every third morph
		transitionType := particle.TransitionDirect
		var burstDist, burstDur float64
		if i%3 == 0 {
			transitionType = particle.TransitionBurst
			burstDist = float64(sh) * 0.3 // Burst 30% of screen height
			burstDur = 0.4                // 40% of duration for burst phase
		}

		seq.AddTarget(particle.MorphTarget{
			Cloud:          offsetTargetCloud,
			Duration:       6.0,
			Strategy:       strategy,
			Turbulence:     turb,
			TransitionType: transitionType,
			BurstDistance:  burstDist,
			BurstDuration:  burstDur,
		})
	}

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
