package core

// SceneContext provides resources available during scene initialization.
type SceneContext struct {
	Width  int
	Height int
}

// Scene represents a discrete slide/screen with lifecycle management.
type Scene interface {
	OnEnter(ctx SceneContext) // Setup (called when scene becomes active)
	OnUpdate(t Time)          // Per-frame update
	OnExit()                  // Cleanup (called when scene becomes inactive)
	Render(canvas *Canvas, t Time)
	World() *World // Access to scene's entity world
}

// BasicScene is a concrete Scene implementation that owns its own World + Compositor.
type BasicScene struct {
	world      *World
	compositor *Compositor
	width      int
	height     int

	// Optional lifecycle hooks
	enterFn  func(world *World, ctx SceneContext)
	updateFn func(world *World, t Time)
	exitFn   func(world *World)
}

// NewBasicScene creates a scene with isolated world and compositor.
func NewBasicScene(width, height int) *BasicScene {
	return &BasicScene{
		world:      NewWorld(),
		compositor: NewCompositor(width, height),
		width:      width,
		height:     height,
	}
}

// World returns the scene's entity world for direct manipulation.
func (s *BasicScene) World() *World {
	return s.world
}

// Compositor returns the scene's compositor for layer configuration.
func (s *BasicScene) Compositor() *Compositor {
	return s.compositor
}

// SetEnter registers a hook called when the scene becomes active.
func (s *BasicScene) SetEnter(fn func(world *World, ctx SceneContext)) {
	s.enterFn = fn
}

// SetUpdate registers a hook called every frame while the scene is active.
func (s *BasicScene) SetUpdate(fn func(world *World, t Time)) {
	s.updateFn = fn
}

// SetExit registers a hook called when the scene becomes inactive.
func (s *BasicScene) SetExit(fn func(world *World)) {
	s.exitFn = fn
}

// OnEnter implements Scene.
func (s *BasicScene) OnEnter(ctx SceneContext) {
	if s.enterFn != nil {
		s.enterFn(s.world, ctx)
	}
}

// OnUpdate implements Scene.
func (s *BasicScene) OnUpdate(t Time) {
	UpdateBehaviors(s.world, t)
	if s.updateFn != nil {
		s.updateFn(s.world, t)
	}
}

// OnExit implements Scene.
func (s *BasicScene) OnExit() {
	if s.exitFn != nil {
		s.exitFn(s.world)
	}

	// Clean up ALL entities when exiting scene (not just roots)
	// This ensures the scene starts completely fresh when re-entered,
	// including non-root entities like behavior containers
	entities := s.world.Entities()
	for _, e := range entities {
		s.world.Despawn(e)
	}
}

// Render implements Scene.
func (s *BasicScene) Render(canvas *Canvas, t Time) {
	s.compositor.Composite(s.world, canvas, t)
}
