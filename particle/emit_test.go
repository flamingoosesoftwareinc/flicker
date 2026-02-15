package particle

import (
	"testing"

	"flicker/core"
)

func TestEmit(t *testing.T) {
	w := core.NewWorld()
	emitter := w.Spawn()

	spawnCount := 0
	spawnFunc := func(w *core.World) core.Entity {
		spawnCount++
		return w.Spawn()
	}

	rate := 10.0 // 10 entities per second
	behaviorFn := Emit(rate, spawnFunc)

	// Run for 1 second with 10 steps of 0.1 seconds each.
	for i := 0; i < 10; i++ {
		behaviorFn(core.Time{Delta: 0.1}, emitter, w)
	}

	// Should spawn ~10 entities.
	if spawnCount != 10 {
		t.Errorf("Expected 10 spawns, got %d", spawnCount)
	}
}

func TestEmitFractional(t *testing.T) {
	w := core.NewWorld()
	emitter := w.Spawn()

	spawnCount := 0
	spawnFunc := func(w *core.World) core.Entity {
		spawnCount++
		return w.Spawn()
	}

	rate := 5.0 // 5 entities per second
	behaviorFn := Emit(rate, spawnFunc)

	// Run for 0.5 seconds with 5 steps of 0.1 seconds each.
	for i := 0; i < 5; i++ {
		behaviorFn(core.Time{Delta: 0.1}, emitter, w)
	}

	// interval = 1/5 = 0.2
	// After 0.5 seconds, should spawn 2 entities (at 0.2s and 0.4s).
	if spawnCount != 2 {
		t.Errorf("Expected 2 spawns, got %d", spawnCount)
	}
}

func TestEmitBurst(t *testing.T) {
	w := core.NewWorld()
	emitter := w.Spawn()

	spawnCount := 0
	spawnFunc := func(w *core.World) core.Entity {
		spawnCount++
		return w.Spawn()
	}

	rate := 10.0 // 10 entities per second
	behaviorFn := Emit(rate, spawnFunc)

	// Large time step (2 seconds) should spawn multiple entities in one frame.
	behaviorFn(core.Time{Delta: 2.0}, emitter, w)

	// Should spawn 20 entities (or 19 due to floating-point precision).
	if spawnCount < 19 || spawnCount > 20 {
		t.Errorf("Expected 19-20 spawns from burst, got %d", spawnCount)
	}
}
