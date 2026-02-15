package particle

import "flicker/core"

// AgeAndDespawn returns a behavior that increments Age.Age by dt each frame.
// If Lifetime > 0 and Age >= Lifetime, despawns the entity via world.Despawn(e).
// Requires Age component.
func AgeAndDespawn() core.BehaviorFunc {
	return func(t core.Time, e core.Entity, w *core.World) {
		age := w.Age(e)
		if age == nil {
			return
		}

		// Increment age.
		age.Age += t.Delta

		// Despawn if lifetime exceeded.
		if age.Lifetime > 0 && age.Age >= age.Lifetime {
			w.Despawn(e)
		}
	}
}
