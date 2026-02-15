package particle

import (
	"flicker/core"
	"flicker/core/bitmap"
	"flicker/fmath"
)

// MorphTarget defines a single morph destination with timing and transition settings.
type MorphTarget struct {
	Cloud    []fmath.Vec2         // Target positions
	Duration float64              // Time to complete this morph (seconds)
	Strategy DistributionStrategy // How to assign particles to targets
	Phases   []TransitionPhase    // Sequence of transition phases (agnostic of implementation)
}

// PointCloudSequence manages a particle system that morphs through multiple targets.
// Orchestrates phases without knowledge of their implementation (behaviors, keyframes, curves, etc.)
type PointCloudSequence struct {
	world  *core.World
	entity core.Entity // Container entity for the sequence behavior

	// Particle management
	particles []core.Entity

	// Morph sequence
	targets            []MorphTarget
	currentTargetIndex int
	currentPhaseIndex  int
	loop               bool // Whether to loop back to first target

	// Phase orchestration
	currentController PhaseController
	phaseStartTime    float64
	targetStartTime   float64
	isTransitioning   bool

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
		world:              world,
		particles:          make([]core.Entity, len(initialCloud)),
		targets:            make([]MorphTarget, 0),
		currentTargetIndex: -1,
		currentPhaseIndex:  -1,
		loop:               true,
		templateDrawable:   drawable,
		templateMaterial:   material,
		templateLayer:      layer,
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

// update orchestrates phase transitions.
func (s *PointCloudSequence) update(t core.Time, e core.Entity, w *core.World) {
	if len(s.targets) == 0 {
		return
	}

	// Initialize on first frame
	if !s.isTransitioning {
		s.currentTargetIndex = 0
		s.startNextTarget(t.Total)
	}

	currentTarget := s.targets[s.currentTargetIndex]
	phaseElapsed := t.Total - s.phaseStartTime

	// Update current phase
	if s.currentController != nil {
		phaseComplete := s.currentController.Update(phaseElapsed)

		if phaseComplete {
			// End current phase
			s.currentController.End()
			s.currentController = nil

			// Move to next phase
			s.currentPhaseIndex++

			if s.currentPhaseIndex >= len(currentTarget.Phases) {
				// All phases complete - move to next target
				s.moveToNextTarget(t.Total)
			} else {
				// Start next phase
				s.startNextPhase(t.Total)
			}
		}
	}
}

// startNextTarget begins transition to the next target in the sequence.
func (s *PointCloudSequence) startNextTarget(currentTime float64) {
	target := s.targets[s.currentTargetIndex]
	s.targetStartTime = currentTime
	s.isTransitioning = true

	// Apply distribution strategy to assign particles to targets
	oldParticleCount := len(s.particles)

	// Phase system handles movement, so skip behavior creation (speed <= 0)
	s.particles = DistributeTargets(
		s.particles,
		target.Cloud,
		0, // Speed 0 = don't add InterpolateToTarget behaviors (phases control movement)
		target.Strategy,
		s.world,
	)

	// Handle newly spawned particles (copy template components)
	if len(s.particles) > oldParticleCount {
		for i := oldParticleCount; i < len(s.particles); i++ {
			p := s.particles[i]
			// Components already added by DistributeTargets
			// Just ensure they have the template material
			if s.world.Material(p) == nil {
				s.world.AddMaterial(p, s.templateMaterial)
			}
		}
	}

	// Start first phase
	s.currentPhaseIndex = 0
	s.startNextPhase(currentTime)
}

// startNextPhase begins the current phase.
func (s *PointCloudSequence) startNextPhase(currentTime float64) {
	target := s.targets[s.currentTargetIndex]
	phase := target.Phases[s.currentPhaseIndex]

	s.phaseStartTime = currentTime

	// Calculate phase duration (fraction of total target duration)
	phaseDuration := target.Duration / float64(len(target.Phases))

	// Create phase context
	ctx := PhaseContext{
		Particles: s.particles,
		Targets:   target.Cloud,
		Duration:  phaseDuration,
		World:     s.world,
	}

	// Start the phase
	s.currentController = phase.Start(ctx)
}

// moveToNextTarget advances to the next target in the sequence.
func (s *PointCloudSequence) moveToNextTarget(currentTime float64) {
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

	s.startNextTarget(currentTime)
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
