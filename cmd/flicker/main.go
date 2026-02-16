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

	// Load font for text rendering.
	textFont, fontErr := asset.LoadFont("Oxanium/static/Oxanium-Bold.ttf")
	if fontErr != nil {
		fmt.Fprintf(os.Stderr, "error loading font: %v\n", fontErr)
		os.Exit(1)
	}

	// Create scene manager
	sm := core.NewSceneManager(sw, sh)

	// Scene 1: Intro - Animated text slide-in using Timeline
	scene1 := createIntroScene(sw, sh, textFont)
	sm.Add(scene1)

	// Scene 2: Timeline demo - Showcase Timeline features
	scene2 := createTimelineScene(sw, sh, textFont)
	sm.Add(scene2)

	// Scene 3: Particle morph demo
	scene3 := createParticleScene(sw, sh, textFont)
	sm.Add(scene3)

	// Scene 4: Thanks - Animated "THANKS" with Timeline
	scene4 := createThanksScene(sw, sh, textFont)
	sm.Add(scene4)

	// Transition shaders to cycle through
	transitions := []core.TransitionShader{
		core.CrossFade,
		core.RadialWipe,
		core.Pixelate,
		core.WipeLeft,
		core.PushRight,
	}
	transitionIndex := 0

	// Start with first scene
	sm.Start()

	// Pump PollEvent in a goroutine so the tick loop never blocks on input
	events := make(chan tcell.Event, 1)
	go func() {
		for {
			events <- screen.PollEvent()
		}
	}()

	var simTime float64
	last := time.Now()

	for {
		// Drain events (non-blocking)
		quit := false
		nextSlide := false
		for draining := true; draining; {
			select {
			case ev := <-events:
				if kev, ok := ev.(*tcell.EventKey); ok {
					switch {
					case kev.Key() == tcell.KeyEscape:
						quit = true
					case kev.Rune() == ' ':
						nextSlide = true
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
		if nextSlide && !sm.IsTransitioning() {
			// Use next transition shader and cycle
			shader := transitions[transitionIndex]
			transitionIndex = (transitionIndex + 1) % len(transitions)
			sm.Next(shader, 1.5)
		}

		now := time.Now()
		wallDelta := now.Sub(last).Seconds()
		last = now
		simTime += wallDelta

		t := core.Time{
			Total: simTime,
			Delta: wallDelta,
		}

		sm.Update(t)

		canvas.Clear()
		canvas.DrawBorder()
		sm.Render(canvas, t)
		screen.Flush(canvas)
	}
}

// createIntroScene creates an animated "INTRO" text scene using Timeline.
func createIntroScene(sw, sh int, font *asset.Font) *core.BasicScene {
	scene := core.NewBasicScene(sw, sh)
	var timeline *core.Timeline
	var text core.Entity
	var centerX float64

	scene.SetEnter(func(w *core.World, ctx core.SceneContext) {
		// Rasterize "INTRO" text
		textSize := float64(sh) * 0.6
		layout := asset.RasterizeText("INTRO", asset.TextOptions{
			Font:  font,
			Size:  textSize,
			Color: core.Color{R: 100, G: 200, B: 255},
		})

		// Center position (target)
		centerX = float64(sw/2) - float64(layout.Bitmap.Width)/2
		centerY := float64(sh/2) - float64(layout.Bitmap.Height)/2

		// Create text entity (start off-screen to the left)
		text = w.Spawn()
		w.AddTransform(text, &core.Transform{
			Position: fmath.Vec3{X: -float64(layout.Bitmap.Width), Y: centerY},
			Scale:    fmath.Vec3{X: 1, Y: 1, Z: 1},
		})
		w.AddDrawable(text, &bitmap.HalfBlock{Bitmap: layout.Bitmap})
		w.AddRoot(text)

		// Create Timeline (will start in OnReady)
		timeline = core.NewTimeline(w)
	})

	scene.SetReady(func(w *core.World) {
		// Start animation when scene is ready (transition complete)
		track := timeline.AddTrack()
		track.Add(core.NewPropertyTweenClip(
			text,
			"position.x",
			-float64(w.Transform(text).Position.X), // Get current position
			centerX,
			1.5,
		).WithEasing(fmath.EaseOutCubic))

		timeline.Start(core.Time{Total: 0})
	})

	scene.SetExit(func(w *core.World) {
		if timeline != nil {
			timeline.Cleanup()
		}
	})

	return scene
}

// createTimelineScene demonstrates Timeline features with multiple text animations.
func createTimelineScene(sw, sh int, font *asset.Font) *core.BasicScene {
	scene := core.NewBasicScene(sw, sh)
	var timeline *core.Timeline
	var text1, text2 core.Entity
	var targetY1 float64
	var layout1Height float64

	scene.SetEnter(func(w *core.World, ctx core.SceneContext) {
		timeline = core.NewTimeline(w)

		// Word 1: "TIMELINE" - Fade in from top
		word1 := "TIMELINE"
		layout1 := asset.RasterizeText(word1, asset.TextOptions{
			Font:  font,
			Size:  float64(sh) * 0.3,
			Color: core.Color{R: 255, G: 100, B: 100},
		})
		centerX1 := float64(sw/2) - float64(layout1.Bitmap.Width)/2
		targetY1 = float64(sh) * 0.25
		layout1Height = float64(layout1.Bitmap.Height)

		text1 = w.Spawn()
		w.AddTransform(text1, &core.Transform{
			Position: fmath.Vec3{X: centerX1, Y: -layout1Height},
			Scale:    fmath.Vec3{X: 1, Y: 1, Z: 1},
		})
		w.AddDrawable(text1, &bitmap.HalfBlock{Bitmap: layout1.Bitmap})
		w.AddRoot(text1)

		// Word 2: "DEMO" - Scale up from center
		word2 := "DEMO"
		layout2 := asset.RasterizeText(word2, asset.TextOptions{
			Font:  font,
			Size:  float64(sh) * 0.4,
			Color: core.Color{R: 100, G: 255, B: 100},
		})
		centerX2 := float64(sw/2) - float64(layout2.Bitmap.Width)/2
		centerY2 := float64(sh) * 0.55

		text2 = w.Spawn()
		w.AddTransform(text2, &core.Transform{
			Position: fmath.Vec3{X: centerX2, Y: centerY2},
			Scale:    fmath.Vec3{X: 0.1, Y: 0.1, Z: 1},
		})
		w.AddDrawable(text2, &bitmap.HalfBlock{Bitmap: layout2.Bitmap})
		w.AddRoot(text2)
	})

	scene.SetReady(func(w *core.World) {
		// Start animations when scene is ready (transition complete)
		// Track 1: Word 1 slides down with bounce
		track1 := timeline.AddTrack()
		track1.Add(core.NewPropertyTweenClip(
			text1,
			"position.y",
			-layout1Height,
			targetY1,
			1.2,
		).WithEasing(fmath.EaseOutBounce))

		// Track 2: Word 2 scales up after word 1 appears
		track2 := timeline.AddTrack()
		track2.At(0.8, core.NewParallelClip(
			core.NewPropertyTweenClip(text2, "scale.x", 0.1, 1.0, 1.0).
				WithEasing(fmath.EaseOutElastic),
			core.NewPropertyTweenClip(text2, "scale.y", 0.1, 1.0, 1.0).
				WithEasing(fmath.EaseOutElastic),
		))

		timeline.Start(core.Time{Total: 0})
	})

	scene.SetExit(func(w *core.World) {
		if timeline != nil {
			timeline.Cleanup()
		}
	})

	return scene
}

// createParticleScene creates a particle morph scene.
func createParticleScene(sw, sh int, font *asset.Font) *core.BasicScene {
	scene := core.NewBasicScene(sw, sh)

	scene.SetEnter(func(w *core.World, ctx core.SceneContext) {
		textSize := float64(sh) * 0.8

		// Create words for morphing
		words := []string{"GO", "BURST", "END"}
		layouts := make([]*asset.TextLayout, len(words))
		clouds := make([][]fmath.Vec2, len(words))

		for i, word := range words {
			layouts[i] = asset.RasterizeText(word, asset.TextOptions{
				Font:  font,
				Size:  textSize,
				Color: core.Color{R: 255, G: 255, B: 255},
			})
			clouds[i] = particle.BitmapToCloud(layouts[i].Bitmap)
		}

		// Single pixel bitmap for particles
		pixel := bitmap.New(1, 1)
		pixel.SetDot(0, 0, core.Color{R: 255, G: 255, B: 255})

		// Center initial cloud
		offsetX := float64(sw/2) - float64(layouts[0].Bitmap.Width)/2
		offsetY := float64(sh/2) - float64(layouts[0].Bitmap.Height)/2

		initialCloud := make([]fmath.Vec2, len(clouds[0]))
		for i, pos := range clouds[0] {
			initialCloud[i] = fmath.Vec2{X: pos.X + offsetX, Y: pos.Y + offsetY}
		}

		// Create particle material
		material := core.ComposeMaterials(
			particle.BrailleDirectional(),
			particle.RainbowTime(2.0),
		)

		// Create point cloud sequence
		seq := particle.NewPointCloudSequence(
			w,
			initialCloud,
			&bitmap.Braille{Bitmap: pixel},
			material,
			0,
		)

		// Add morph targets
		for i := 1; i < len(words); i++ {
			targetOffsetX := float64(sw/2) - float64(layouts[i].Bitmap.Width)/2
			offsetTargetCloud := make([]fmath.Vec2, len(clouds[i]))
			for j, pos := range clouds[i] {
				offsetTargetCloud[j] = fmath.Vec2{X: pos.X + targetOffsetX, Y: pos.Y + offsetY}
			}

			var phases []particle.TransitionPhase
			if i == 1 {
				// Burst transition
				phases = []particle.TransitionPhase{
					particle.BurstPhase(float64(sh) * 0.4),
					particle.SeekPhase(),
				}
			} else {
				// Smooth keyframe
				phases = []particle.TransitionPhase{
					&particle.KeyframePhase{Easing: particle.EaseInOutQuad},
				}
			}

			seq.AddTarget(particle.MorphTarget{
				Cloud:    offsetTargetCloud,
				Duration: 4.0,
				Strategy: particle.LinearDistribution(),
				Phases:   phases,
			})
		}
	})

	return scene
}

// createThanksScene creates an animated "THANKS" scene with Timeline and rainbow effect.
func createThanksScene(sw, sh int, font *asset.Font) *core.BasicScene {
	scene := core.NewBasicScene(sw, sh)
	var timeline *core.Timeline
	var text core.Entity

	scene.SetEnter(func(w *core.World, ctx core.SceneContext) {
		// Rasterize "THANKS" text
		textSize := float64(sh) * 0.5
		layout := asset.RasterizeText("THANKS", asset.TextOptions{
			Font:  font,
			Size:  textSize,
			Color: core.Color{R: 255, G: 255, B: 255},
		})

		// Center text
		centerX := float64(sw/2) - float64(layout.Bitmap.Width)/2
		centerY := float64(sh/2) - float64(layout.Bitmap.Height)/2

		// Create text entity with rainbow material (start small)
		text = w.Spawn()
		w.AddTransform(text, &core.Transform{
			Position: fmath.Vec3{X: centerX, Y: centerY},
			Scale:    fmath.Vec3{X: 0.3, Y: 0.3, Z: 1},
			Rotation: 0,
		})
		w.AddDrawable(text, &bitmap.HalfBlock{Bitmap: layout.Bitmap})
		w.AddMaterial(text, particle.RainbowTime(3.0))
		w.AddRoot(text)

		// Create Timeline (will start in OnReady)
		timeline = core.NewTimeline(w)
	})

	scene.SetReady(func(w *core.World) {
		// Start animation when scene is ready (transition complete)
		track := timeline.AddTrack()
		track.Add(core.NewParallelClip(
			core.NewPropertyTweenClip(text, "scale.x", 0.3, 1.0, 1.5).
				WithEasing(fmath.EaseOutElastic),
			core.NewPropertyTweenClip(text, "scale.y", 0.3, 1.0, 1.5).
				WithEasing(fmath.EaseOutElastic),
			core.NewPropertyTweenClip(text, "rotation", -0.5, 0.0, 1.5).
				WithEasing(fmath.EaseOutCubic),
		))

		timeline.Start(core.Time{Total: 0})
	})

	scene.SetExit(func(w *core.World) {
		if timeline != nil {
			timeline.Cleanup()
		}
	})

	return scene
}
