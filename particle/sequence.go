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

// MorphTarget defines a single morph destination with timing and behavior settings.
type MorphTarget struct {
	Cloud      []fmath.Vec2         // Target positions
	Duration   float64              // Time to complete this morph (seconds)
	Strategy   DistributionStrategy // How to assign particles to targets
	Turbulence *TurbulenceConfig    // nil = no turbulence for this morph
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

	// Morph sequence
	targets            []MorphTarget
	currentTargetIndex int
	loop               bool // Whether to loop back to first target

	// Timing
	transitionStartTime float64
	isTransitioning     bool

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

	// Check if current transition is complete
	currentTarget := s.targets[s.currentTargetIndex]
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
