package core

// Example usage patterns for the Timeline system.
// These are not executable tests but serve as documentation.

/*
// ============================================================================
// Example 1: Simple Event Sequence
// ============================================================================

func ExampleSimpleSequence() {
	world := NewWorld()
	timeline := NewTimeline(world)

	track := timeline.AddTrack()
	track.At(0.0, NewCallbackClip(spawnTitle))
	track.At(2.0, NewCallbackClip(spawnSubtitle))
	track.At(4.0, NewCallbackClip(fadeOut))

	timeline.Start(Time{Total: 0.0})

	// In scene OnUpdate:
	// UpdateBehaviors(world, t) - automatically updates timeline

	// In scene OnExit:
	// timeline.Cleanup()
}

// ============================================================================
// Example 2: Parallel Animations
// ============================================================================

func ExampleParallelAnimations() {
	world := NewWorld()
	timeline := NewTimeline(world)

	// Text track
	textTrack := timeline.AddTrack()
	textTrack.At(0.0, NewCallbackClip(spawnText))

	// Particle track (runs in parallel)
	particleTrack := timeline.AddTrack()
	particleTrack.At(1.0, NewCallbackClip(burstEffect))

	// Background animation
	bgAlpha := 0.0
	bgTrack := timeline.AddTrack()
	bgTrack.Add(NewTweenClip(0, 1, 5.0, func(v float64) {
		bgAlpha = v
	}))

	timeline.Start(Time{Total: 0.0})
}

// ============================================================================
// Example 3: Property Animation
// ============================================================================

func ExamplePropertyAnimation() {
	world := NewWorld()
	timeline := NewTimeline(world)

	entity := world.Spawn()
	world.AddTransform(entity, &Transform{
		Position: Vec3{X: 0, Y: 0, Z: 0},
		Scale:    Vec3{X: 1, Y: 1, Z: 1},
	})

	track := timeline.AddTrack()
	track.Add(NewPropertyTweenClip(entity, "position.x", 0, 100, 2.0).
		WithEasing(fmath.EaseInOutQuad))

	timeline.Start(Time{Total: 0.0})
}

// ============================================================================
// Example 4: Composition (Parallel + Sequential)
// ============================================================================

func ExampleComposition() {
	world := NewWorld()
	timeline := NewTimeline(world)

	entity1 := world.Spawn()
	entity2 := world.Spawn()
	// ... setup entities ...

	track := timeline.AddTrack()

	// At 3 seconds, animate multiple properties in parallel
	track.At(3.0, NewParallelClip(
		NewPropertyTweenClip(entity1, "position.x", 0, 100, 1.0),
		NewPropertyTweenClip(entity2, "position.x", 100, 0, 1.0),
	))

	// Then run a sequence of animations
	track.Add(NewSequenceClip(
		NewDelayClip(0.5),
		NewCallbackClip(showMessage),
		NewDelayClip(2.0),
		NewCallbackClip(hideMessage),
	))

	timeline.Start(Time{Total: 0.0})
}

// ============================================================================
// Example 5: BasicScene Integration
// ============================================================================

func ExampleBasicSceneIntegration() {
	scene := NewBasicScene(160, 48)
	var timeline *Timeline

	scene.SetEnter(func(w *World, ctx SceneContext) {
		// Create timeline
		timeline = NewTimeline(w)

		// Add timed events
		track := timeline.AddTrack()
		track.At(2.0, NewCallbackClip(func(w *World, t Time) {
			// Spawn text entity at 2s
			e := w.Spawn()
			// ... setup entity ...
		}))

		track.At(5.0, NewCallbackClip(func(w *World, t Time) {
			// Trigger particle burst at 5s
		}))

		// Start timeline at scene entry
		timeline.Start(Time{Total: 0})
	})

	scene.SetExit(func(w *World) {
		if timeline != nil {
			timeline.Cleanup() // Despawn container entity
		}
	})

	// UpdateBehaviors(world, t) is called automatically in BasicScene.OnUpdate
}

// ============================================================================
// Example 6: Loop Mode
// ============================================================================

func ExampleLoopMode() {
	world := NewWorld()
	timeline := NewTimeline(world)
	timeline.SetLoop(true) // Enable looping

	track := timeline.AddTrack()
	track.Add(NewDelayClip(1.0))
	track.Add(NewCallbackClip(pulse))
	track.Add(NewDelayClip(1.0))

	timeline.Start(Time{Total: 0.0})
	// Timeline will restart after completing all tracks
}

// ============================================================================
// Example 7: Pause/Resume
// ============================================================================

func ExamplePauseResume() {
	world := NewWorld()
	timeline := NewTimeline(world)

	// ... setup timeline ...

	timeline.Start(Time{Total: 0.0})

	// Later...
	timeline.Pause() // Pause animation

	// Even later...
	timeline.Resume() // Continue from where it paused

	// Or stop completely...
	timeline.Stop() // Stops without completing clips
}
*/
