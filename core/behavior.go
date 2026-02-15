package core

// BehaviorFunc is the function signature for behavior logic.
type BehaviorFunc func(Time, Entity, *World)

// Behavior is an interface for per-entity update logic.
// Custom behaviors can implement this with their own state.
type Behavior interface {
	Enabled() bool
	Update(Time, Entity, *World)
}

// FuncBehavior wraps a closure-based behavior with enable/disable support.
// Use NewBehavior to wrap existing function-based behaviors.
type FuncBehavior struct {
	enabled bool
	fn      BehaviorFunc
}

func (f *FuncBehavior) Enabled() bool {
	return f.enabled
}

func (f *FuncBehavior) SetEnabled(enabled bool) {
	f.enabled = enabled
}

func (f *FuncBehavior) Update(t Time, e Entity, w *World) {
	f.fn(t, e, w)
}

// NewBehavior wraps a function as a Behavior interface.
// This allows existing closure-based behaviors to work with the new system.
func NewBehavior(fn BehaviorFunc) *FuncBehavior {
	return &FuncBehavior{enabled: true, fn: fn}
}

// UpdateBehaviors runs all enabled behaviors for each entity once per tick.
func UpdateBehaviors(world *World, t Time) {
	for e, behaviors := range world.behaviors {
		for _, b := range behaviors {
			if b.Enabled() {
				b.Update(t, e, world)
			}
		}
	}
}
