package particle

import "flicker/core"

// Emit returns a behavior that spawns entities at a given rate (entities per second).
// spawnFunc defines what to spawn (particles, text, shapes, etc.).
// Attached to an emitter entity (doesn't need physics itself).
// Maintains accumulated time in closure.
func Emit(rate float64, spawnFunc func(*core.World) core.Entity) core.BehaviorFunc {
	accumulated := 0.0
	interval := 1.0 / rate

	return func(t core.Time, e core.Entity, w *core.World) {
		accumulated += t.Delta

		// Spawn entities based on accumulated time.
		for accumulated >= interval {
			spawnFunc(w)
			accumulated -= interval
		}
	}
}
