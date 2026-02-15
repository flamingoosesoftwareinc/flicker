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

	// Scene 1: Intro - Static "INTRO" text
	scene1 := createIntroScene(sw, sh, textFont)
	sm.Add(scene1)

	// Scene 2: Particle morph demo
	scene2 := createParticleScene(sw, sh, textFont)
	sm.Add(scene2)

	// Scene 3: Thanks - "THANKS" with rainbow effect
	scene3 := createThanksScene(sw, sh, textFont)
	sm.Add(scene3)

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

// createIntroScene creates a static "INTRO" text scene.
func createIntroScene(sw, sh int, font *asset.Font) *core.BasicScene {
	scene := core.NewBasicScene(sw, sh)

	scene.SetEnter(func(w *core.World, ctx core.SceneContext) {
		// Rasterize "INTRO" text
		textSize := float64(sh) * 0.6
		layout := asset.RasterizeText("INTRO", asset.TextOptions{
			Font:  font,
			Size:  textSize,
			Color: core.Color{R: 100, G: 200, B: 255},
		})

		// Center text
		offsetX := float64(sw/2) - float64(layout.Bitmap.Width)/2
		offsetY := float64(sh/2) - float64(layout.Bitmap.Height)/2

		// Create text entity
		text := w.Spawn()
		w.AddTransform(text, &core.Transform{
			Position: fmath.Vec3{X: offsetX, Y: offsetY},
			Scale:    fmath.Vec3{X: 1, Y: 1, Z: 1},
		})
		w.AddDrawable(text, &bitmap.HalfBlock{Bitmap: layout.Bitmap})
		w.AddRoot(text)
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

// createThanksScene creates a "THANKS" scene with rainbow effect.
func createThanksScene(sw, sh int, font *asset.Font) *core.BasicScene {
	scene := core.NewBasicScene(sw, sh)

	scene.SetEnter(func(w *core.World, ctx core.SceneContext) {
		// Rasterize "THANKS" text
		textSize := float64(sh) * 0.5
		layout := asset.RasterizeText("THANKS", asset.TextOptions{
			Font:  font,
			Size:  textSize,
			Color: core.Color{R: 255, G: 255, B: 255},
		})

		// Center text
		offsetX := float64(sw/2) - float64(layout.Bitmap.Width)/2
		offsetY := float64(sh/2) - float64(layout.Bitmap.Height)/2

		// Create text entity with rainbow material
		text := w.Spawn()
		w.AddTransform(text, &core.Transform{
			Position: fmath.Vec3{X: offsetX, Y: offsetY},
			Scale:    fmath.Vec3{X: 1, Y: 1, Z: 1},
		})
		w.AddDrawable(text, &bitmap.HalfBlock{Bitmap: layout.Bitmap})
		w.AddMaterial(text, particle.RainbowTime(3.0))
		w.AddRoot(text)
	})

	return scene
}
