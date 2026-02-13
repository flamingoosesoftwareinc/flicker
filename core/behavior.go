package core

// Behavior is a per-entity update function. It receives the time delta since
// the last tick and can read/write the entity's components through the World.
type Behavior func(dt float64, e Entity, w *World)

// UpdateBehaviors runs every entity's Behavior component once per tick.
func UpdateBehaviors(world *World, dt float64) {
	for e, b := range world.behaviors {
		b(dt, e, world)
	}
}
