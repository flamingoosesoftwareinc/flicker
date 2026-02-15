package core

import (
	"testing"

	"flicker/fmath"
)

// ============================================================================
// Test 1: Basic Callback - Instant callback execution
// ============================================================================

func TestTimelineBasicCallback(t *testing.T) {
	world := NewWorld()
	timeline := NewTimeline(world)

	executed := false
	callback := func(w *World, time Time) {
		executed = true
	}

	track := timeline.AddTrack()
	track.Add(NewCallbackClip(callback))

	// Start timeline
	timeline.Start(Time{Total: 0.0, Delta: 0.0})

	// Update once - callback should execute immediately
	UpdateBehaviors(world, Time{Total: 0.0, Delta: 0.016})

	if !executed {
		t.Error("Callback should have executed immediately")
	}

	if timeline.isRunning {
		t.Error("Timeline should have completed")
	}

	timeline.Cleanup()
}

// ============================================================================
// Test 2: Delayed Callback - Callback at specific time via At()
// ============================================================================

func TestTimelineDelayedCallback(t *testing.T) {
	world := NewWorld()
	timeline := NewTimeline(world)

	executed := false
	callback := func(w *World, time Time) {
		executed = true
	}

	track := timeline.AddTrack()
	track.At(2.0, NewCallbackClip(callback)) // Execute at 2 seconds

	// Start timeline
	timeline.Start(Time{Total: 0.0, Delta: 0.0})

	// Update at t=1.0 - should not execute yet
	UpdateBehaviors(world, Time{Total: 1.0, Delta: 0.016})
	if executed {
		t.Error("Callback should not have executed yet at t=1.0")
	}

	// Update at t=2.1 - should have executed
	UpdateBehaviors(world, Time{Total: 2.1, Delta: 0.016})
	if !executed {
		t.Error("Callback should have executed at t=2.1")
	}

	timeline.Cleanup()
}

// ============================================================================
// Test 3: Parallel Tracks - Multiple tracks firing simultaneously
// ============================================================================

func TestTimelineParallelTracks(t *testing.T) {
	world := NewWorld()
	timeline := NewTimeline(world)

	track1Executed := false
	track2Executed := false

	track1 := timeline.AddTrack()
	track1.At(1.0, NewCallbackClip(func(w *World, time Time) {
		track1Executed = true
	}))

	track2 := timeline.AddTrack()
	track2.At(1.0, NewCallbackClip(func(w *World, time Time) {
		track2Executed = true
	}))

	timeline.Start(Time{Total: 0.0, Delta: 0.0})

	// Update at t=1.1 - both should execute
	UpdateBehaviors(world, Time{Total: 1.1, Delta: 0.016})

	if !track1Executed {
		t.Error("Track 1 callback should have executed")
	}
	if !track2Executed {
		t.Error("Track 2 callback should have executed")
	}

	timeline.Cleanup()
}

// ============================================================================
// Test 4: Sequential Clips - Clips on same track running in sequence
// ============================================================================

func TestTimelineSequentialClips(t *testing.T) {
	world := NewWorld()
	timeline := NewTimeline(world)

	executionOrder := []int{}

	track := timeline.AddTrack()
	track.Add(NewDelayClip(1.0))
	track.Add(NewCallbackClip(func(w *World, time Time) {
		executionOrder = append(executionOrder, 1)
	}))
	track.Add(NewDelayClip(0.5))
	track.Add(NewCallbackClip(func(w *World, time Time) {
		executionOrder = append(executionOrder, 2)
	}))

	timeline.Start(Time{Total: 0.0, Delta: 0.0})

	// Update at t=0.5 - nothing should execute
	UpdateBehaviors(world, Time{Total: 0.5, Delta: 0.016})
	if len(executionOrder) != 0 {
		t.Errorf("Expected no executions at t=0.5, got %d", len(executionOrder))
	}

	// Update at t=1.1 - first callback should execute
	UpdateBehaviors(world, Time{Total: 1.1, Delta: 0.016})
	if len(executionOrder) != 1 || executionOrder[0] != 1 {
		t.Errorf("Expected [1] at t=1.1, got %v", executionOrder)
	}

	// Update at t=1.7 - second callback should execute
	UpdateBehaviors(world, Time{Total: 1.7, Delta: 0.016})
	if len(executionOrder) != 2 || executionOrder[1] != 2 {
		t.Errorf("Expected [1, 2] at t=1.7, got %v", executionOrder)
	}

	timeline.Cleanup()
}

// ============================================================================
// Test 5: TweenClip - Value interpolation over time
// ============================================================================

func TestTimelineTween(t *testing.T) {
	world := NewWorld()
	timeline := NewTimeline(world)

	tweenedValue := 0.0
	setter := func(v float64) {
		tweenedValue = v
	}

	track := timeline.AddTrack()
	track.Add(NewTweenClip(0.0, 100.0, 2.0, setter))

	timeline.Start(Time{Total: 0.0, Delta: 0.0})

	// Update at t=0.0 - should be at start value
	UpdateBehaviors(world, Time{Total: 0.0, Delta: 0.016})
	if tweenedValue != 0.0 {
		t.Errorf("Expected 0.0 at t=0.0, got %f", tweenedValue)
	}

	// Update at t=1.0 - should be halfway (linear easing)
	UpdateBehaviors(world, Time{Total: 1.0, Delta: 0.016})
	if tweenedValue < 49.0 || tweenedValue > 51.0 {
		t.Errorf("Expected ~50.0 at t=1.0, got %f", tweenedValue)
	}

	// Update at t=2.1 - should be at end value
	UpdateBehaviors(world, Time{Total: 2.1, Delta: 0.016})
	if tweenedValue != 100.0 {
		t.Errorf("Expected 100.0 at t=2.1, got %f", tweenedValue)
	}

	timeline.Cleanup()
}

// ============================================================================
// Test 6: PropertyTweenClip - Animating entity transform
// ============================================================================

func TestTimelinePropertyTween(t *testing.T) {
	world := NewWorld()
	timeline := NewTimeline(world)

	// Create entity with transform
	entity := world.Spawn()
	world.AddTransform(entity, &Transform{
		Position: fmath.Vec3{X: 0, Y: 0, Z: 0},
		Scale:    fmath.Vec3{X: 1, Y: 1, Z: 1},
		Rotation: 0,
	})

	// Animate position.x from 0 to 100 over 2 seconds
	track := timeline.AddTrack()
	track.Add(NewPropertyTweenClip(entity, "position.x", 0.0, 100.0, 2.0))

	timeline.Start(Time{Total: 0.0, Delta: 0.0})

	// Update at t=1.0 - should be halfway
	UpdateBehaviors(world, Time{Total: 1.0, Delta: 0.016})
	transform := world.Transform(entity)
	if transform.Position.X < 49.0 || transform.Position.X > 51.0 {
		t.Errorf("Expected position.x ~50.0 at t=1.0, got %f", transform.Position.X)
	}

	// Update at t=2.1 - should be at end value
	UpdateBehaviors(world, Time{Total: 2.1, Delta: 0.016})
	if transform.Position.X != 100.0 {
		t.Errorf("Expected position.x 100.0 at t=2.1, got %f", transform.Position.X)
	}

	timeline.Cleanup()
	world.Despawn(entity)
}

// ============================================================================
// Test 7: ParallelClip - Multiple clips running simultaneously
// ============================================================================

func TestTimelineParallelClip(t *testing.T) {
	world := NewWorld()
	timeline := NewTimeline(world)

	value1 := 0.0
	value2 := 0.0

	parallel := NewParallelClip(
		NewTweenClip(0.0, 100.0, 1.0, func(v float64) { value1 = v }),
		NewTweenClip(0.0, 200.0, 1.0, func(v float64) { value2 = v }),
	)

	track := timeline.AddTrack()
	track.Add(parallel)

	timeline.Start(Time{Total: 0.0, Delta: 0.0})

	// Update at t=0.5 - both should be halfway
	UpdateBehaviors(world, Time{Total: 0.5, Delta: 0.016})
	if value1 < 49.0 || value1 > 51.0 {
		t.Errorf("Expected value1 ~50.0 at t=0.5, got %f", value1)
	}
	if value2 < 99.0 || value2 > 101.0 {
		t.Errorf("Expected value2 ~100.0 at t=0.5, got %f", value2)
	}

	// Update at t=1.1 - both should be at end values
	UpdateBehaviors(world, Time{Total: 1.1, Delta: 0.016})
	if value1 != 100.0 {
		t.Errorf("Expected value1 100.0 at t=1.1, got %f", value1)
	}
	if value2 != 200.0 {
		t.Errorf("Expected value2 200.0 at t=1.1, got %f", value2)
	}

	timeline.Cleanup()
}

// ============================================================================
// Test 8: SequenceClip - Clips running one after another
// ============================================================================

func TestTimelineSequenceClip(t *testing.T) {
	world := NewWorld()
	timeline := NewTimeline(world)

	executionOrder := []int{}

	sequence := NewSequenceClip(
		NewDelayClip(0.5),
		NewCallbackClip(func(w *World, time Time) {
			executionOrder = append(executionOrder, 1)
		}),
		NewDelayClip(0.5),
		NewCallbackClip(func(w *World, time Time) {
			executionOrder = append(executionOrder, 2)
		}),
	)

	track := timeline.AddTrack()
	track.Add(sequence)

	timeline.Start(Time{Total: 0.0, Delta: 0.0})

	// Update at t=0.6 - first callback should execute
	UpdateBehaviors(world, Time{Total: 0.6, Delta: 0.016})
	if len(executionOrder) != 1 || executionOrder[0] != 1 {
		t.Errorf("Expected [1] at t=0.6, got %v", executionOrder)
	}

	// Update at t=1.2 - second callback should execute
	UpdateBehaviors(world, Time{Total: 1.2, Delta: 0.016})
	if len(executionOrder) != 2 || executionOrder[1] != 2 {
		t.Errorf("Expected [1, 2] at t=1.2, got %v", executionOrder)
	}

	timeline.Cleanup()
}

// ============================================================================
// Test 9: Loop - Timeline loops back to start
// ============================================================================

func TestTimelineLoop(t *testing.T) {
	world := NewWorld()
	timeline := NewTimeline(world)
	timeline.SetLoop(true)

	executionCount := 0
	callback := func(w *World, time Time) {
		executionCount++
	}

	track := timeline.AddTrack()
	track.Add(NewDelayClip(0.5))
	track.Add(NewCallbackClip(callback))

	timeline.Start(Time{Total: 0.0, Delta: 0.0})

	// First execution at t=0.6
	UpdateBehaviors(world, Time{Total: 0.6, Delta: 0.016})
	if executionCount != 1 {
		t.Errorf("Expected 1 execution at t=0.6, got %d", executionCount)
	}

	// Should loop - second execution at t=1.2
	UpdateBehaviors(world, Time{Total: 1.2, Delta: 0.016})
	if executionCount != 2 {
		t.Errorf("Expected 2 executions at t=1.2 (looped), got %d", executionCount)
	}

	// Timeline should still be running
	if !timeline.isRunning {
		t.Error("Timeline should still be running in loop mode")
	}

	timeline.Cleanup()
}

// ============================================================================
// Test 10: Pause/Resume - Timeline can be paused and resumed
// ============================================================================

func TestTimelinePauseResume(t *testing.T) {
	world := NewWorld()
	timeline := NewTimeline(world)

	tweenedValue := 0.0
	setter := func(v float64) {
		tweenedValue = v
	}

	track := timeline.AddTrack()
	track.Add(NewTweenClip(0.0, 100.0, 2.0, setter))

	timeline.Start(Time{Total: 0.0, Delta: 0.0})

	// Update to t=1.0 - should be at 50
	UpdateBehaviors(world, Time{Total: 1.0, Delta: 0.016})
	value1 := tweenedValue
	if value1 < 49.0 || value1 > 51.0 {
		t.Errorf("Expected ~50.0 at t=1.0, got %f", tweenedValue)
	}

	// Pause timeline
	timeline.Pause()

	// Update to t=2.0 - value should not change (paused)
	UpdateBehaviors(world, Time{Total: 2.0, Delta: 0.016})
	if tweenedValue != value1 {
		t.Errorf("Value should not change while paused, was %f, now %f", value1, tweenedValue)
	}

	// Resume timeline
	timeline.Resume()

	// Update to t=3.0 - should continue from where it paused (50 -> 100)
	// Elapsed time from timeline start is 3.0, so should be complete
	UpdateBehaviors(world, Time{Total: 3.0, Delta: 0.016})
	if tweenedValue != 100.0 {
		t.Errorf("Expected 100.0 after resume at t=3.0, got %f", tweenedValue)
	}

	timeline.Cleanup()
}

// ============================================================================
// Test 11: PropertyTween with Easing
// ============================================================================

func TestTimelinePropertyTweenWithEasing(t *testing.T) {
	world := NewWorld()
	timeline := NewTimeline(world)

	entity := world.Spawn()
	world.AddTransform(entity, &Transform{
		Position: fmath.Vec3{X: 0, Y: 0, Z: 0},
		Scale:    fmath.Vec3{X: 1, Y: 1, Z: 1},
		Rotation: 0,
	})

	// Animate with EaseInOutQuad
	track := timeline.AddTrack()
	track.Add(NewPropertyTweenClip(entity, "position.x", 0.0, 100.0, 2.0).
		WithEasing(fmath.EaseInOutQuad))

	timeline.Start(Time{Total: 0.0, Delta: 0.0})

	// Update at t=1.0 - with EaseInOutQuad, should still be near 50
	UpdateBehaviors(world, Time{Total: 1.0, Delta: 0.016})
	transform := world.Transform(entity)

	// EaseInOutQuad at t=0.5 should be 0.5 (symmetric)
	if transform.Position.X < 49.0 || transform.Position.X > 51.0 {
		t.Errorf(
			"Expected position.x ~50.0 at t=1.0 with EaseInOutQuad, got %f",
			transform.Position.X,
		)
	}

	timeline.Cleanup()
	world.Despawn(entity)
}

// ============================================================================
// Test 12: Multiple Properties
// ============================================================================

func TestTimelineMultipleProperties(t *testing.T) {
	world := NewWorld()
	timeline := NewTimeline(world)

	entity := world.Spawn()
	world.AddTransform(entity, &Transform{
		Position: fmath.Vec3{X: 0, Y: 0, Z: 0},
		Scale:    fmath.Vec3{X: 1, Y: 1, Z: 1},
		Rotation: 0,
	})

	// Animate multiple properties in parallel
	track := timeline.AddTrack()
	track.Add(NewParallelClip(
		NewPropertyTweenClip(entity, "position.x", 0.0, 100.0, 1.0),
		NewPropertyTweenClip(entity, "position.y", 0.0, 50.0, 1.0),
		NewPropertyTweenClip(entity, "rotation", 0.0, 3.14159, 1.0),
	))

	timeline.Start(Time{Total: 0.0, Delta: 0.0})

	// Update at t=1.1 - all should be at end values
	UpdateBehaviors(world, Time{Total: 1.1, Delta: 0.016})
	transform := world.Transform(entity)

	if transform.Position.X != 100.0 {
		t.Errorf("Expected position.x 100.0, got %f", transform.Position.X)
	}
	if transform.Position.Y != 50.0 {
		t.Errorf("Expected position.y 50.0, got %f", transform.Position.Y)
	}
	if transform.Rotation < 3.14 || transform.Rotation > 3.15 {
		t.Errorf("Expected rotation ~3.14159, got %f", transform.Rotation)
	}

	timeline.Cleanup()
	world.Despawn(entity)
}

// ============================================================================
// Test 13: Stop Timeline
// ============================================================================

func TestTimelineStop(t *testing.T) {
	world := NewWorld()
	timeline := NewTimeline(world)

	tweenedValue := 0.0
	setter := func(v float64) {
		tweenedValue = v
	}

	track := timeline.AddTrack()
	track.Add(NewTweenClip(0.0, 100.0, 2.0, setter))

	timeline.Start(Time{Total: 0.0, Delta: 0.0})

	// Update to t=1.0
	UpdateBehaviors(world, Time{Total: 1.0, Delta: 0.016})
	value1 := tweenedValue

	// Stop timeline
	timeline.Stop()

	// Update to t=2.0 - value should not change (stopped)
	UpdateBehaviors(world, Time{Total: 2.0, Delta: 0.016})
	if tweenedValue != value1 {
		t.Errorf("Value should not change after stop, was %f, now %f", value1, tweenedValue)
	}

	if timeline.isRunning {
		t.Error("Timeline should not be running after stop")
	}

	timeline.Cleanup()
}
