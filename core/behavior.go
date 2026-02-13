package core

// Behavior is a per-entity update function. It receives the engine Time
// and can read/write the entity's components through the World.
type Behavior func(t Time, e Entity, w *World)

// UpdateBehaviors runs every entity's Behavior component once per tick.
func UpdateBehaviors(world *World, t Time) {
	for e, b := range world.behaviors {
		b(t, e, world)
	}
}
