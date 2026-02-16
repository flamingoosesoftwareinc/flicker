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

	// Load font for text rendering.
	textFont, fontErr := asset.LoadFont("Oxanium/static/Oxanium-Bold.ttf")
	if fontErr != nil {
		fmt.Fprintf(os.Stderr, "error loading font: %v\n", fontErr)
		os.Exit(1)
	}

	// Create scene manager
	sm := core.NewSceneManager(sw, sh)

	// Add trail demo scenes
	sm.Add(createNoTrailScene(sw, sh, textFont))
	sm.Add(createGhostTrailScene(sw, sh, textFont))
	sm.Add(createBlurTrailScene(sw, sh, textFont))
	sm.Add(createFloatyTrailScene(sw, sh, textFont))
	sm.Add(createGravityTrailScene(sw, sh, textFont))
	sm.Add(createDustTrailScene(sw, sh, textFont))
	sm.Add(createFireTrailScene(sw, sh, textFont))

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
			sm.Next(core.CrossFade, 1.0)
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

// createNoTrailScene - Control scene with no trails
func createNoTrailScene(sw, sh int, font *asset.Font) *core.BasicScene {
	scene := core.NewBasicScene(sw, sh)

	scene.SetEnter(func(w *core.World, ctx core.SceneContext) {
		// Create title
		titleLayout := asset.RasterizeText("NO TRAILS", asset.TextOptions{
			Font:  font,
			Size:  float64(sh) * 0.2,
			Color: core.Color{R: 255, G: 255, B: 255},
		})
		titleEntity := w.Spawn()
		w.AddTransform(titleEntity, &core.Transform{
			Position: fmath.Vec3{
				X: float64(sw/2) - float64(titleLayout.Bitmap.Width)/2,
				Y: float64(sh) * 0.15,
			},
			Scale: fmath.Vec3{X: 1, Y: 1, Z: 1},
		})
		w.AddDrawable(titleEntity, &bitmap.HalfBlock{Bitmap: titleLayout.Bitmap})
		w.AddRoot(titleEntity)

		// Create moving text
		textLayout := asset.RasterizeText("MOVING", asset.TextOptions{
			Font:  font,
			Size:  float64(sh) * 0.3,
			Color: core.Color{R: 100, G: 200, B: 255},
		})
		movingText := w.Spawn()
		startX := float64(sw) * 0.1
		centerY := float64(sh/2) - float64(textLayout.Bitmap.Height)/2
		w.AddTransform(movingText, &core.Transform{
			Position: fmath.Vec3{X: startX, Y: centerY},
			Scale:    fmath.Vec3{X: 1, Y: 1, Z: 1},
		})
		w.AddDrawable(movingText, &bitmap.HalfBlock{Bitmap: textLayout.Bitmap})
		w.AddRoot(movingText)

		// Add oscillating behavior
		tween := &fmath.TweenVec3{
			From:     fmath.Vec3{X: startX, Y: centerY, Z: 0},
			To:       fmath.Vec3{X: float64(sw) * 0.6, Y: centerY, Z: 0},
			Duration: 2.0,
			Easing:   fmath.EaseInOutQuad,
		}

		w.AddBehavior(movingText, core.NewBehavior(func(t core.Time, e core.Entity, w *core.World) {
			tween.Update(t.Delta)
			if tween.Done() {
				tween.Reset()
			}
			trans := w.Transform(e)
			trans.Position = tween.Update(0)
		}))
	})

	return scene
}

// createGhostTrailScene - Simple alpha fade trail
func createGhostTrailScene(sw, sh int, font *asset.Font) *core.BasicScene {
	scene := core.NewBasicScene(sw, sh)

	scene.SetEnter(func(w *core.World, ctx core.SceneContext) {
		// Set ghost trail on layer 0
		scene.Compositor().SetPreProcess(0, core.GhostTrail(0.95))

		// Create title
		titleLayout := asset.RasterizeText("GHOST TRAIL", asset.TextOptions{
			Font:  font,
			Size:  float64(sh) * 0.2,
			Color: core.Color{R: 255, G: 255, B: 255},
		})
		titleEntity := w.Spawn()
		w.AddTransform(titleEntity, &core.Transform{
			Position: fmath.Vec3{
				X: float64(sw/2) - float64(titleLayout.Bitmap.Width)/2,
				Y: float64(sh) * 0.15,
			},
			Scale: fmath.Vec3{X: 1, Y: 1, Z: 1},
		})
		w.AddDrawable(titleEntity, &bitmap.HalfBlock{Bitmap: titleLayout.Bitmap})
		w.AddRoot(titleEntity)

		// Create moving text
		textLayout := asset.RasterizeText("FADE", asset.TextOptions{
			Font:  font,
			Size:  float64(sh) * 0.4,
			Color: core.Color{R: 100, G: 255, B: 100},
		})
		movingText := w.Spawn()
		startX := float64(sw) * 0.1
		centerY := float64(sh/2) - float64(textLayout.Bitmap.Height)/2
		w.AddTransform(movingText, &core.Transform{
			Position: fmath.Vec3{X: startX, Y: centerY},
			Scale:    fmath.Vec3{X: 1, Y: 1, Z: 1},
		})
		w.AddDrawable(movingText, &bitmap.HalfBlock{Bitmap: textLayout.Bitmap})
		w.AddRoot(movingText)

		// Add oscillating behavior
		tween := &fmath.TweenVec3{
			From:     fmath.Vec3{X: startX, Y: centerY, Z: 0},
			To:       fmath.Vec3{X: float64(sw) * 0.6, Y: centerY, Z: 0},
			Duration: 2.5,
			Easing:   fmath.EaseInOutQuad,
		}

		w.AddBehavior(movingText, core.NewBehavior(func(t core.Time, e core.Entity, w *core.World) {
			tween.Update(t.Delta)
			if tween.Done() {
				tween.Reset()
			}
			trans := w.Transform(e)
			trans.Position = tween.Update(0)
		}))
	})

	return scene
}

// createBlurTrailScene - Blur/diffusion trail effect
func createBlurTrailScene(sw, sh int, font *asset.Font) *core.BasicScene {
	scene := core.NewBasicScene(sw, sh)

	scene.SetEnter(func(w *core.World, ctx core.SceneContext) {
		// Set blur trail on layer 0
		scene.Compositor().SetPreProcess(0, core.BlurTrail(0.94, 0.3))

		// Create title
		titleLayout := asset.RasterizeText("BLUR TRAIL", asset.TextOptions{
			Font:  font,
			Size:  float64(sh) * 0.2,
			Color: core.Color{R: 255, G: 255, B: 255},
		})
		titleEntity := w.Spawn()
		w.AddTransform(titleEntity, &core.Transform{
			Position: fmath.Vec3{
				X: float64(sw/2) - float64(titleLayout.Bitmap.Width)/2,
				Y: float64(sh) * 0.15,
			},
			Scale: fmath.Vec3{X: 1, Y: 1, Z: 1},
		})
		w.AddDrawable(titleEntity, &bitmap.HalfBlock{Bitmap: titleLayout.Bitmap})
		w.AddRoot(titleEntity)

		// Create moving circle
		pixel := bitmap.New(1, 1)
		pixel.SetDot(0, 0, core.Color{R: 255, G: 100, B: 255})

		movingText := w.Spawn()
		centerX := float64(sw / 2)
		centerY := float64(sh / 2)
		w.AddTransform(movingText, &core.Transform{
			Position: fmath.Vec3{X: centerX, Y: centerY},
			Scale:    fmath.Vec3{X: 15, Y: 15, Z: 1},
		})
		w.AddDrawable(movingText, &bitmap.Braille{Bitmap: pixel})
		w.AddRoot(movingText)

		// Circular motion
		var angle float64
		w.AddBehavior(movingText, core.NewBehavior(func(t core.Time, e core.Entity, w *core.World) {
			angle += t.Delta * 1.5
			radius := float64(sw) * 0.25
			trans := w.Transform(e)
			trans.Position.X = centerX + math.Cos(angle)*radius
			trans.Position.Y = centerY + math.Sin(angle)*radius*0.5
		}))
	})

	return scene
}

// createFloatyTrailScene - Noise-distorted trail
func createFloatyTrailScene(sw, sh int, font *asset.Font) *core.BasicScene {
	scene := core.NewBasicScene(sw, sh)

	scene.SetEnter(func(w *core.World, ctx core.SceneContext) {
		// Set floaty trail on layer 0
		scene.Compositor().SetPreProcess(0, core.FloatyTrail(0.96, 3.0))

		// Create title
		titleLayout := asset.RasterizeText("FLOATY TRAIL", asset.TextOptions{
			Font:  font,
			Size:  float64(sh) * 0.2,
			Color: core.Color{R: 255, G: 255, B: 255},
		})
		titleEntity := w.Spawn()
		w.AddTransform(titleEntity, &core.Transform{
			Position: fmath.Vec3{
				X: float64(sw/2) - float64(titleLayout.Bitmap.Width)/2,
				Y: float64(sh) * 0.15,
			},
			Scale: fmath.Vec3{X: 1, Y: 1, Z: 1},
		})
		w.AddDrawable(titleEntity, &bitmap.HalfBlock{Bitmap: titleLayout.Bitmap})
		w.AddRoot(titleEntity)

		// Create moving text
		textLayout := asset.RasterizeText("DRIFT", asset.TextOptions{
			Font:  font,
			Size:  float64(sh) * 0.4,
			Color: core.Color{R: 255, G: 200, B: 100},
		})
		movingText := w.Spawn()
		startX := float64(sw) * 0.1
		centerY := float64(sh/2) - float64(textLayout.Bitmap.Height)/2
		w.AddTransform(movingText, &core.Transform{
			Position: fmath.Vec3{X: startX, Y: centerY},
			Scale:    fmath.Vec3{X: 1, Y: 1, Z: 1},
		})
		w.AddDrawable(movingText, &bitmap.HalfBlock{Bitmap: textLayout.Bitmap})
		w.AddRoot(movingText)

		// Add oscillating behavior
		tween := &fmath.TweenVec3{
			From:     fmath.Vec3{X: startX, Y: centerY, Z: 0},
			To:       fmath.Vec3{X: float64(sw) * 0.6, Y: centerY, Z: 0},
			Duration: 3.0,
			Easing:   fmath.EaseInOutCubic,
		}

		w.AddBehavior(movingText, core.NewBehavior(func(t core.Time, e core.Entity, w *core.World) {
			tween.Update(t.Delta)
			if tween.Done() {
				tween.Reset()
			}
			trans := w.Transform(e)
			trans.Position = tween.Update(0)
		}))
	})

	return scene
}

// createGravityTrailScene - Downward-drifting trail
func createGravityTrailScene(sw, sh int, font *asset.Font) *core.BasicScene {
	scene := core.NewBasicScene(sw, sh)

	scene.SetEnter(func(w *core.World, ctx core.SceneContext) {
		// Set gravity trail on layer 0
		scene.Compositor().SetPreProcess(0, core.GravityTrail(0.96, 5.0))

		// Create title
		titleLayout := asset.RasterizeText("GRAVITY TRAIL", asset.TextOptions{
			Font:  font,
			Size:  float64(sh) * 0.2,
			Color: core.Color{R: 255, G: 255, B: 255},
		})
		titleEntity := w.Spawn()
		w.AddTransform(titleEntity, &core.Transform{
			Position: fmath.Vec3{
				X: float64(sw/2) - float64(titleLayout.Bitmap.Width)/2,
				Y: float64(sh) * 0.15,
			},
			Scale: fmath.Vec3{X: 1, Y: 1, Z: 1},
		})
		w.AddDrawable(titleEntity, &bitmap.HalfBlock{Bitmap: titleLayout.Bitmap})
		w.AddRoot(titleEntity)

		// Create moving text
		textLayout := asset.RasterizeText("FALL", asset.TextOptions{
			Font:  font,
			Size:  float64(sh) * 0.4,
			Color: core.Color{R: 255, G: 100, B: 100},
		})
		movingText := w.Spawn()
		startX := float64(sw) * 0.1
		startY := float64(sh) * 0.35
		w.AddTransform(movingText, &core.Transform{
			Position: fmath.Vec3{X: startX, Y: startY},
			Scale:    fmath.Vec3{X: 1, Y: 1, Z: 1},
		})
		w.AddDrawable(movingText, &bitmap.HalfBlock{Bitmap: textLayout.Bitmap})
		w.AddRoot(movingText)

		// Horizontal motion only - gravity trail creates vertical effect
		tween := &fmath.TweenVec3{
			From:     fmath.Vec3{X: startX, Y: startY, Z: 0},
			To:       fmath.Vec3{X: float64(sw) * 0.6, Y: startY, Z: 0},
			Duration: 2.5,
			Easing:   fmath.EaseInOutQuad,
		}

		w.AddBehavior(movingText, core.NewBehavior(func(t core.Time, e core.Entity, w *core.World) {
			tween.Update(t.Delta)
			if tween.Done() {
				tween.Reset()
			}
			trans := w.Transform(e)
			trans.Position = tween.Update(0)
		}))
	})

	return scene
}

// createDustTrailScene - Dissolving dust trail
func createDustTrailScene(sw, sh int, font *asset.Font) *core.BasicScene {
	scene := core.NewBasicScene(sw, sh)

	scene.SetEnter(func(w *core.World, ctx core.SceneContext) {
		// Set dust trail on layer 0
		dustColor := core.Color{R: 120, G: 120, B: 120}
		scene.Compositor().SetPreProcess(0, core.DustTrail(0.93, 0.6, dustColor))

		// Create title
		titleLayout := asset.RasterizeText("DUST TRAIL", asset.TextOptions{
			Font:  font,
			Size:  float64(sh) * 0.2,
			Color: core.Color{R: 255, G: 255, B: 255},
		})
		titleEntity := w.Spawn()
		w.AddTransform(titleEntity, &core.Transform{
			Position: fmath.Vec3{
				X: float64(sw/2) - float64(titleLayout.Bitmap.Width)/2,
				Y: float64(sh) * 0.15,
			},
			Scale: fmath.Vec3{X: 1, Y: 1, Z: 1},
		})
		w.AddDrawable(titleEntity, &bitmap.HalfBlock{Bitmap: titleLayout.Bitmap})
		w.AddRoot(titleEntity)

		// Create moving text
		textLayout := asset.RasterizeText("DUST", asset.TextOptions{
			Font:  font,
			Size:  float64(sh) * 0.4,
			Color: core.Color{R: 200, G: 150, B: 255},
		})
		movingText := w.Spawn()
		startX := float64(sw) * 0.1
		centerY := float64(sh/2) - float64(textLayout.Bitmap.Height)/2
		w.AddTransform(movingText, &core.Transform{
			Position: fmath.Vec3{X: startX, Y: centerY},
			Scale:    fmath.Vec3{X: 1, Y: 1, Z: 1},
		})
		w.AddDrawable(movingText, &bitmap.HalfBlock{Bitmap: textLayout.Bitmap})
		w.AddRoot(movingText)

		// Add oscillating behavior
		tween := &fmath.TweenVec3{
			From:     fmath.Vec3{X: startX, Y: centerY, Z: 0},
			To:       fmath.Vec3{X: float64(sw) * 0.6, Y: centerY, Z: 0},
			Duration: 2.0,
			Easing:   fmath.EaseInOutQuad,
		}

		w.AddBehavior(movingText, core.NewBehavior(func(t core.Time, e core.Entity, w *core.World) {
			tween.Update(t.Delta)
			if tween.Done() {
				tween.Reset()
			}
			trans := w.Transform(e)
			trans.Position = tween.Update(0)
		}))
	})

	return scene
}

// createFireTrailScene - Fire color-shift trail
func createFireTrailScene(sw, sh int, font *asset.Font) *core.BasicScene {
	scene := core.NewBasicScene(sw, sh)

	scene.SetEnter(func(w *core.World, ctx core.SceneContext) {
		// Set fire trail on layer 0
		scene.Compositor().SetPreProcess(0, core.FireTrail(0.94, 2.0))

		// Create title
		titleLayout := asset.RasterizeText("FIRE TRAIL", asset.TextOptions{
			Font:  font,
			Size:  float64(sh) * 0.2,
			Color: core.Color{R: 255, G: 255, B: 255},
		})
		titleEntity := w.Spawn()
		w.AddTransform(titleEntity, &core.Transform{
			Position: fmath.Vec3{
				X: float64(sw/2) - float64(titleLayout.Bitmap.Width)/2,
				Y: float64(sh) * 0.15,
			},
			Scale: fmath.Vec3{X: 1, Y: 1, Z: 1},
		})
		w.AddDrawable(titleEntity, &bitmap.HalfBlock{Bitmap: titleLayout.Bitmap})
		w.AddRoot(titleEntity)

		// Create moving text
		textLayout := asset.RasterizeText("BURN", asset.TextOptions{
			Font:  font,
			Size:  float64(sh) * 0.4,
			Color: core.Color{R: 255, G: 255, B: 100},
		})
		movingText := w.Spawn()
		startX := float64(sw) * 0.1
		centerY := float64(sh/2) - float64(textLayout.Bitmap.Height)/2
		w.AddTransform(movingText, &core.Transform{
			Position: fmath.Vec3{X: startX, Y: centerY},
			Scale:    fmath.Vec3{X: 1, Y: 1, Z: 1},
		})
		w.AddDrawable(movingText, &bitmap.HalfBlock{Bitmap: textLayout.Bitmap})
		w.AddRoot(movingText)

		// Add oscillating behavior
		tween := &fmath.TweenVec3{
			From:     fmath.Vec3{X: startX, Y: centerY, Z: 0},
			To:       fmath.Vec3{X: float64(sw) * 0.6, Y: centerY, Z: 0},
			Duration: 2.5,
			Easing:   fmath.EaseInOutQuad,
		}

		w.AddBehavior(movingText, core.NewBehavior(func(t core.Time, e core.Entity, w *core.World) {
			tween.Update(t.Delta)
			if tween.Done() {
				tween.Reset()
			}
			trans := w.Transform(e)
			trans.Position = tween.Update(0)
		}))
	})

	return scene
}
