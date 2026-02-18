package core

// SceneAutoAdvance configures automatic scene advancement.
type SceneAutoAdvance struct {
	Duration         float64          // How long the scene shows before advancing (0 = manual)
	TransitionShader TransitionShader // Shader to use when auto-advancing
	TransitionTime   float64          // Duration of the transition animation
}

// SceneManager manages a linear sequence of scenes with transitions.
type SceneManager struct {
	scenes      []Scene
	autoAdvance []SceneAutoAdvance // parallel to scenes; nil entry = manual
	current     int                // Index of current scene (-1 = none)
	active      bool               // Whether a scene is active
	transition  *Transition
	sceneTime   float64 // Time elapsed in current scene (reset on scene change)

	width  int
	height int
}

// NewSceneManager creates a scene manager for the given canvas dimensions.
func NewSceneManager(width, height int) *SceneManager {
	return &SceneManager{
		scenes:  make([]Scene, 0),
		current: -1,
		width:   width,
		height:  height,
	}
}

// Add appends a scene to the sequence.
func (sm *SceneManager) Add(scene Scene) {
	sm.scenes = append(sm.scenes, scene)
	sm.autoAdvance = append(sm.autoAdvance, SceneAutoAdvance{})
}

// SetAutoAdvance configures auto-advance for a scene by index.
func (sm *SceneManager) SetAutoAdvance(index int, aa SceneAutoAdvance) {
	if index >= 0 && index < len(sm.autoAdvance) {
		sm.autoAdvance[index] = aa
	}
}

// Count returns the number of scenes.
func (sm *SceneManager) Count() int {
	return len(sm.scenes)
}

// Current returns the index of the current scene, or -1 if none.
func (sm *SceneManager) Current() int {
	return sm.current
}

// IsTransitioning returns true if a transition is in progress.
func (sm *SceneManager) IsTransitioning() bool {
	return sm.transition != nil
}

// Start activates the first scene.
func (sm *SceneManager) Start() {
	if len(sm.scenes) == 0 {
		return
	}
	sm.GoTo(0)
}

// Next transitions to the next scene. Wraps to first scene after last.
func (sm *SceneManager) Next(shader TransitionShader, duration float64) {
	if len(sm.scenes) == 0 {
		return
	}
	nextIndex := (sm.current + 1) % len(sm.scenes)
	sm.transitionTo(nextIndex, shader, duration)
}

// Previous transitions to the previous scene. Does nothing if already at first scene.
func (sm *SceneManager) Previous(shader TransitionShader, duration float64) {
	if sm.current > 0 {
		sm.transitionTo(sm.current-1, shader, duration)
	}
}

// GoTo jumps to a specific scene index with no transition.
func (sm *SceneManager) GoTo(index int) {
	if index < 0 || index >= len(sm.scenes) {
		return
	}

	// Exit current scene if active
	if sm.active && sm.current >= 0 {
		sm.scenes[sm.current].OnExit()
	}

	// Enter new scene
	sm.current = index
	sm.active = true
	sm.sceneTime = 0
	ctx := SceneContext{Width: sm.width, Height: sm.height}
	sm.scenes[sm.current].OnEnter(ctx)

	// Scene is immediately ready (no transition)
	sm.scenes[sm.current].OnReady()
}

// transitionTo starts a transition to the target scene.
func (sm *SceneManager) transitionTo(
	targetIndex int,
	shader TransitionShader,
	duration float64,
) {
	if targetIndex < 0 || targetIndex >= len(sm.scenes) {
		return
	}
	if sm.transition != nil {
		return // Already transitioning
	}
	if !sm.active {
		return // No current scene
	}

	// Enter target scene (both scenes active during transition)
	ctx := SceneContext{Width: sm.width, Height: sm.height}
	sm.scenes[targetIndex].OnEnter(ctx)

	// Create transition
	from := sm.scenes[sm.current]
	to := sm.scenes[targetIndex]
	sm.transition = NewTransition(from, to, duration, shader)
}

// Update updates the current scene and any active transition.
func (sm *SceneManager) Update(t Time) {
	if sm.transition != nil {
		// Update transition
		done := sm.transition.Update(t.Delta)

		// Update both scenes during transition
		sm.transition.From.OnUpdate(t)
		sm.transition.To.OnUpdate(t)

		if done {
			// Transition complete - exit old scene, activate new scene
			sm.transition.From.OnExit()

			// Find new scene's index
			for i, s := range sm.scenes {
				if s == sm.transition.To {
					sm.current = i
					break
				}
			}

			sm.sceneTime = 0

			// Notify new scene that transition is complete
			sm.transition.To.OnReady()

			sm.transition = nil
		}
	} else if sm.active && sm.current >= 0 {
		// Update current scene
		sm.scenes[sm.current].OnUpdate(t)

		// Check auto-advance
		sm.sceneTime += t.Delta
		aa := sm.autoAdvance[sm.current]
		if aa.Duration > 0 && sm.sceneTime >= aa.Duration {
			shader := aa.TransitionShader
			if shader == nil {
				shader = CrossFade
			}
			transTime := aa.TransitionTime
			if transTime <= 0 {
				transTime = 1.0
			}
			nextIndex := (sm.current + 1) % len(sm.scenes)
			sm.transitionTo(nextIndex, shader, transTime)
		}
	}
}

// Render renders the current scene or active transition to dst.
func (sm *SceneManager) Render(dst *Canvas, t Time) {
	if sm.transition != nil {
		// Render transition
		sm.transition.Render(dst, t)
	} else if sm.active && sm.current >= 0 {
		// Render current scene
		sm.scenes[sm.current].Render(dst, t)
	}
}
