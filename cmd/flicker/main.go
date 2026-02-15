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

	// Text rendering setup
	textSize := float64(sh) * 0.8
	textOpts := asset.TextOptions{
		Font:  textFont,
		Size:  textSize,
		Color: core.Color{R: 255, G: 255, B: 255},
	}

	// Create different words to demonstrate various transition types
	words := []string{
		"GO",      // Start
		"BURST",   // Burst transition
		"CURVE",   // Curved arc transition
		"EASE",    // Keyframe with easing
		"SEEK",    // Direct seek (behavior)
		"SWIRL",   // Multi-phase: turbulence + seek
		"ARC",     // Another curve
		"SMOOTH",  // Smooth easing
		"EXPLODE", // Burst again
		"END",     // Final
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

	// Single pixel bitmap for particles
	pixel := bitmap.New(1, 1)
	pixel.SetDot(0, 0, core.Color{R: 255, G: 255, B: 255})

	// Calculate center offset for initial cloud
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

	// Add targets with different phase combinations
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

		// Create strategy dynamically (ClosestPoint needs current particle state)
		var strategy particle.DistributionStrategy
		switch i % 3 {
		case 0:
			strategy = particle.LinearDistribution()
		case 1:
			strategy = particle.RoundRobinDistribution()
		case 2:
			// Create ClosestPoint with current particles each time
			strategy = particle.ClosestPointDistribution(seq.Particles(), offsetTargetCloud, world)
		}

		// Create different phase combinations for each transition
		var phases []particle.TransitionPhase
		duration := 6.0

		switch i % 5 {
		case 0:
			// Direct seek (single behavior phase)
			phases = []particle.TransitionPhase{
				particle.SeekPhase(),
			}

		case 1:
			// Burst + seek (two behavior phases)
			burstDist := float64(sh) * 0.4
			phases = []particle.TransitionPhase{
				particle.BurstPhase(burstDist), // 50% of duration
				particle.SeekPhase(),           // 50% of duration
			}

		case 2:
			// Keyframe with smooth easing
			phases = []particle.TransitionPhase{
				&particle.KeyframePhase{Easing: particle.EaseInOutQuad},
			}

		case 3:
			// Curved arc motion
			arcHeight := float64(sh) * 0.3
			phases = []particle.TransitionPhase{
				&particle.CurvePhase{ArcHeight: arcHeight},
			}

		case 4:
			// Multi-phase: turbulence, then smooth keyframe seek
			phases = []particle.TransitionPhase{
				particle.TurbulencePhase(0.05, 30.0),                   // 33% turbulence
				&particle.KeyframePhase{Easing: particle.EaseOutCubic}, // 67% smooth ease
			}
		}

		seq.AddTarget(particle.MorphTarget{
			Cloud:    offsetTargetCloud,
			Duration: duration,
			Strategy: strategy,
			Phases:   phases,
		})
	}

	// Camera
	cam := world.Spawn()
	world.AddTransform(cam, &core.Transform{
		Position: fmath.Vec3{X: float64(sw) / 2.0, Y: float64(sh) / 2.0},
		Scale:    fmath.Vec3{X: 1, Y: 1, Z: 1},
	})
	world.AddCamera(cam, &core.Camera{Zoom: 1})
	world.SetActiveCamera(cam)

	// Pump PollEvent in a goroutine so the tick loop never blocks on input
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
		// Drain events (non-blocking)
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
