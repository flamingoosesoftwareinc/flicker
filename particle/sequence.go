package particle

import (
	"flicker/core"
	"flicker/core/bitmap"
	"flicker/fmath"
	"flicker/physics"
)

// TurbulenceConfig defines turbulence behavior settings.
type TurbulenceConfig struct {
	Scale    float64 // Noise frequency
	Strength float64 // Force magnitude
}

// TransitionType defines how particles move to their targets.
type TransitionType int

const (
	TransitionDirect TransitionType = iota // Go straight to target
	TransitionBurst                        // Burst outward from center, then to target
)

// MorphTarget defines a single morph destination with timing and behavior settings.
type MorphTarget struct {
	Cloud          []fmath.Vec2         // Target positions
	Duration       float64              // Time to complete this morph (seconds)
	Strategy       DistributionStrategy // How to assign particles to targets
	Turbulence     *TurbulenceConfig    // nil = no turbulence for this morph
	TransitionType TransitionType       // How particles transition to targets
	BurstDistance  float64              // How far to burst outward (for TransitionBurst)
	BurstDuration  float64              // What fraction of duration to burst (0.0-1.0, default 0.3)
}

// PointCloudSequence manages a particle system that morphs through multiple targets.
// Handles particle spawning, behavior management, and transition timing.
type PointCloudSequence struct {
	world  *core.World
	entity core.Entity // Container entity for the sequence behavior

	// Particle management
	particles           []core.Entity
	movementBehaviors   []core.Behavior      // Track movement behaviors to disable on morph
	turbulenceBehaviors []*core.FuncBehavior // Track turbulence behaviors
	burstBehaviors      []*core.FuncBehavior // Track burst behaviors (for TransitionBurst)

	// Morph sequence
	targets            []MorphTarget
	currentTargetIndex int
	loop               bool // Whether to loop back to first target

	// Timing
	transitionStartTime float64
	burstEndTime        float64 // When burst phase ends (for TransitionBurst)
	isTransitioning     bool
	isBursting          bool // Whether currently in burst phase

	// Template for spawning new particles
	templateDrawable core.Drawable
	templateMaterial core.Material
	templateLayer    int
}

// NewPointCloudSequence creates a particle system from an initial cloud.
// drawable and material are used as templates when spawning new particles.
func NewPointCloudSequence(
	world *core.World,
	initialCloud []fmath.Vec2,
	drawable core.Drawable,
	material core.Material,
	layer int,
) *PointCloudSequence {
	seq := &PointCloudSequence{
		world:               world,
		particles:           make([]core.Entity, len(initialCloud)),
		movementBehaviors:   make([]core.Behavior, 0),
		turbulenceBehaviors: make([]*core.FuncBehavior, 0),
		burstBehaviors:      make([]*core.FuncBehavior, 0),
		targets:             make([]MorphTarget, 0),
		currentTargetIndex:  -1,
		loop:                true,
		templateDrawable:    drawable,
		templateMaterial:    material,
		templateLayer:       layer,
	}

	// Spawn initial particles
	for i, pos := range initialCloud {
		p := world.Spawn()
		seq.particles[i] = p

		world.AddTransform(p, &core.Transform{
			Position: fmath.Vec3{X: pos.X, Y: pos.Y},
			Scale:    fmath.Vec3{X: 1, Y: 1, Z: 1},
		})
		world.AddBody(p, &core.Body{})
		world.AddDrawable(p, drawable)
		world.AddMaterial(p, material)
		world.AddLayer(p, layer)
		world.AddRoot(p)
	}

	// Create container entity for sequence update behavior
	seq.entity = world.Spawn()
	world.AddBehavior(seq.entity, core.NewBehavior(seq.update))

	return seq
}

// burstOutward creates a behavior that moves particles radially outward from a center point.
func burstOutward(center fmath.Vec2, speed float64) core.BehaviorFunc {
	return func(t core.Time, e core.Entity, w *core.World) {
		transform := w.Transform(e)
		body := w.Body(e)
		if transform == nil {
			return
		}

		pos := fmath.Vec2{X: transform.Position.X, Y: transform.Position.Y}
		delta := pos.Sub(center) // Direction away from center

		// Handle particles at exact center
		if delta.X == 0 && delta.Y == 0 {
			delta = fmath.Vec2{X: 1, Y: 0} // Default direction
		}

		dir := delta.Normalize()
		dt := t.Delta

		// Move outward
		transform.Position.X += dir.X * speed * dt
		transform.Position.Y += dir.Y * speed * dt

		// Update velocity for directional materials
		if body != nil {
			body.Velocity = fmath.Vec2{X: dir.X * speed, Y: dir.Y * speed}
		}
	}
}

// AddTarget adds a morph target to the sequence.
func (s *PointCloudSequence) AddTarget(target MorphTarget) {
	s.targets = append(s.targets, target)
}

// SetLoop controls whether the sequence loops back to the first target.
func (s *PointCloudSequence) SetLoop(loop bool) {
	s.loop = loop
}

// Particles returns the current particle entities.
func (s *PointCloudSequence) Particles() []core.Entity {
	return s.particles
}

// update is the behavior function that handles transitions.
func (s *PointCloudSequence) update(t core.Time, e core.Entity, w *core.World) {
	if len(s.targets) == 0 {
		return
	}

	// Initialize transition on first frame
	if !s.isTransitioning {
		s.currentTargetIndex = 0 // Start at first target
		s.startNextTransition(t.Total)
	}

	currentTarget := s.targets[s.currentTargetIndex]

	// Check if we're in burst phase and it's time to transition to seek phase
	if s.isBursting && t.Total >= s.burstEndTime {
		s.isBursting = false

		// Disable burst behaviors
		for _, bb := range s.burstBehaviors {
			if bb != nil {
				bb.SetEnabled(false)
			}
		}

		// Enable movement behaviors to start seeking targets
		for _, mb := range s.movementBehaviors {
			if mb != nil {
				if fb, ok := mb.(*core.FuncBehavior); ok {
					fb.SetEnabled(true)
				}
			}
		}
	}

	// Check if current transition is complete
	elapsed := t.Total - s.transitionStartTime

	if elapsed >= currentTarget.Duration {
		// Move to next target
		s.currentTargetIndex++

		// Handle looping or stopping
		if s.currentTargetIndex >= len(s.targets) {
			if s.loop {
				s.currentTargetIndex = 0
			} else {
				s.isTransitioning = false
				return
			}
		}

		s.startNextTransition(t.Total)
	}
}

// startNextTransition initiates transition to the current target.
func (s *PointCloudSequence) startNextTransition(currentTime float64) {
	target := s.targets[s.currentTargetIndex]
	s.transitionStartTime = currentTime
	s.isTransitioning = true

	// Disable all old movement behaviors
	for _, mb := range s.movementBehaviors {
		if mb != nil {
			if fb, ok := mb.(*core.FuncBehavior); ok {
				fb.SetEnabled(false)
			}
		}
	}
	s.movementBehaviors = s.movementBehaviors[:0] // Clear slice

	// Update turbulence state for all particles
	turbulenceEnabled := target.Turbulence != nil
	for _, tb := range s.turbulenceBehaviors {
		if tb != nil {
			tb.SetEnabled(turbulenceEnabled)
		}
	}

	// Calculate speed to complete in target duration
	completionTime := target.Duration * 0.9 // 90% buffer
	speed := CalculateSpeedForDuration(s.particles, target.Cloud, completionTime, s.world)

	// Apply distribution strategy
	oldParticleCount := len(s.particles)
	s.particles = DistributeTargets(
		s.particles,
		target.Cloud,
		speed,
		target.Strategy,
		s.world,
	)

	// Track newly added movement behaviors (one per existing particle)
	// Note: DistributeTargets adds behaviors internally, we need to retrieve them
	for _, p := range s.particles[:oldParticleCount] {
		behaviors := s.world.Behaviors(p)
		if len(behaviors) > 0 {
			// Get the last added behavior (the movement behavior)
			s.movementBehaviors = append(s.movementBehaviors, behaviors[len(behaviors)-1])
		}
	}

	// Handle newly spawned particles
	if len(s.particles) > oldParticleCount {
		newParticleCount := len(s.particles) - oldParticleCount

		// Add turbulence to new particles
		for i := 0; i < newParticleCount; i++ {
			particleIdx := oldParticleCount + i
			p := s.particles[particleIdx]

			// Add turbulence behavior (enabled based on current target)
			var tb *core.FuncBehavior
			if target.Turbulence != nil {
				tb = s.world.AddBehavior(
					p,
					core.NewBehavior(physics.Turbulence(target.Turbulence.Scale, target.Turbulence.Strength)),
				).(*core.FuncBehavior)
				tb.SetEnabled(turbulenceEnabled)
			} else {
				// Add disabled turbulence for consistency
				tb = s.world.AddBehavior(
					p,
					core.NewBehavior(physics.Turbulence(0.05, 30.0)),
				).(*core.FuncBehavior)
				tb.SetEnabled(false)
			}
			s.turbulenceBehaviors = append(s.turbulenceBehaviors, tb)

			// Track movement behavior for new particle
			behaviors := s.world.Behaviors(p)
			if len(behaviors) > 0 {
				// Find the InterpolateToTarget behavior (not the turbulence one we just added)
				for _, b := range behaviors {
					// Skip turbulence behavior
					if b == tb {
						continue
					}
					s.movementBehaviors = append(s.movementBehaviors, b)
					break
				}
			}
		}
	}

	// Handle burst transition
	if target.TransitionType == TransitionBurst {
		s.isBursting = true

		// Calculate burst parameters
		burstFraction := target.BurstDuration
		if burstFraction <= 0 {
			burstFraction = 0.3 // Default: 30% of duration for burst
		}
		burstTime := target.Duration * burstFraction
		s.burstEndTime = currentTime + burstTime

		// Calculate center point (centroid of current particle positions)
		center := fmath.Vec2{X: 0, Y: 0}
		for _, p := range s.particles {
			if tr := s.world.Transform(p); tr != nil {
				center.X += tr.Position.X
				center.Y += tr.Position.Y
			}
		}
		if len(s.particles) > 0 {
			center.X /= float64(len(s.particles))
			center.Y /= float64(len(s.particles))
		}

		// Determine burst speed
		burstSpeed := target.BurstDistance / burstTime
		if burstSpeed < 10 {
			burstSpeed = 50.0 // Default burst speed
		}

		// Disable movement behaviors initially (will be enabled after burst)
		for _, mb := range s.movementBehaviors {
			if mb != nil {
				if fb, ok := mb.(*core.FuncBehavior); ok {
					fb.SetEnabled(false)
				}
			}
		}

		// Clear old burst behaviors
		for _, bb := range s.burstBehaviors {
			if bb != nil {
				bb.SetEnabled(false)
			}
		}
		s.burstBehaviors = s.burstBehaviors[:0]

		// Add burst behaviors to all particles
		for _, p := range s.particles {
			bb := s.world.AddBehavior(
				p,
				core.NewBehavior(burstOutward(center, burstSpeed)),
			).(*core.FuncBehavior)
			s.burstBehaviors = append(s.burstBehaviors, bb)
		}
	} else {
		// Direct transition - ensure movement behaviors are enabled
		s.isBursting = false
		for _, mb := range s.movementBehaviors {
			if mb != nil {
				if fb, ok := mb.(*core.FuncBehavior); ok {
					fb.SetEnabled(true)
				}
			}
		}
	}
}

// NewPointCloudSequenceFromBitmaps is a convenience constructor that converts bitmaps to clouds.
func NewPointCloudSequenceFromBitmaps(
	world *core.World,
	initialBitmap *bitmap.Bitmap,
	drawable core.Drawable,
	material core.Material,
	layer int,
	offsetX, offsetY float64,
) *PointCloudSequence {
	cloud := BitmapToCloud(initialBitmap)

	// Apply offset to cloud
	offsetCloud := make([]fmath.Vec2, len(cloud))
	for i, pos := range cloud {
		offsetCloud[i] = fmath.Vec2{
			X: pos.X + offsetX,
			Y: pos.Y + offsetY,
		}
	}

	return NewPointCloudSequence(world, offsetCloud, drawable, material, layer)
}
