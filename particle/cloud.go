package particle

import (
	"math/rand"

	"flicker/core"
	"flicker/core/bitmap"
	"flicker/fmath"
)

// BitmapToCloud samples all non-transparent pixels (alpha > 0) from bitmap.
// Returns their positions as []fmath.Vec2.
func BitmapToCloud(bm *bitmap.Bitmap) []fmath.Vec2 {
	cloud := make([]fmath.Vec2, 0)

	for y := 0; y < bm.Height; y++ {
		for x := 0; x < bm.Width; x++ {
			_, alpha := bm.Get(x, y)
			if alpha > 0 {
				cloud = append(cloud, fmath.Vec2{X: float64(x), Y: float64(y)})
			}
		}
	}

	return cloud
}

// TargetMapping defines how entities map to targets and where to spawn new particles.
type TargetMapping struct {
	EntityTargets []int // EntityTargets[i] = target index for entity i
	SpawnFrom     []int // For extra targets, spawn from entity SpawnFrom[i]
}

// DistributionStrategy computes target assignments for entities and spawn sources.
type DistributionStrategy func(entityCount, targetCount int) TargetMapping

// LinearDistribution creates 1:1 mapping between entities and targets.
// Extra entities wrap to beginning of targets. Extra targets spawn from sequential entities.
func LinearDistribution() DistributionStrategy {
	return func(entityCount, targetCount int) TargetMapping {
		mapping := TargetMapping{
			EntityTargets: make([]int, entityCount),
		}

		// Map entities to targets linearly (with wraparound if more entities)
		for i := 0; i < entityCount; i++ {
			mapping.EntityTargets[i] = i % targetCount
		}

		// If more targets than entities, spawn from sequential entities
		if targetCount > entityCount {
			mapping.SpawnFrom = make([]int, targetCount-entityCount)
			for i := 0; i < len(mapping.SpawnFrom); i++ {
				mapping.SpawnFrom[i] = i % entityCount
			}
		}

		return mapping
	}
}

// RoundRobinDistribution creates round-robin mapping with random spawn positions.
// This is the original DistributeTargets behavior.
func RoundRobinDistribution() DistributionStrategy {
	return func(entityCount, targetCount int) TargetMapping {
		mapping := TargetMapping{
			EntityTargets: make([]int, entityCount),
		}

		// Round-robin mapping
		for i := 0; i < entityCount; i++ {
			mapping.EntityTargets[i] = i % targetCount
		}

		// If more targets than entities, spawn from random entities
		if targetCount > entityCount {
			mapping.SpawnFrom = make([]int, targetCount-entityCount)
			for i := 0; i < len(mapping.SpawnFrom); i++ {
				mapping.SpawnFrom[i] = rand.Intn(entityCount)
			}
		}

		return mapping
	}
}

// DistributeTargets assigns target positions from cloud to entities using a distribution strategy.
// If cloud has more points than entities, spawns additional particles to fill the gaps.
// Returns all entities (original + newly spawned).
// Adds InterpolateToTarget behavior to each entity.
func DistributeTargets(
	entities []core.Entity,
	cloud []fmath.Vec2,
	speed float64,
	strategy DistributionStrategy,
	world *core.World,
) []core.Entity {
	if len(cloud) == 0 {
		return entities
	}

	// Compute target mapping using strategy.
	mapping := strategy(len(entities), len(cloud))

	// Assign targets to existing entities using strategy mapping.
	for i, e := range entities {
		targetIdx := mapping.EntityTargets[i]
		world.AddBehavior(e, core.NewBehavior(InterpolateToTarget(cloud[targetIdx], speed)))
	}

	// If cloud has more points than entities, spawn new particles for the extra points.
	if len(cloud) > len(entities) {
		initialEntityCount := len(entities) // Capture before appending
		for i := 0; i < len(mapping.SpawnFrom); i++ {
			// Get template from entity specified by strategy.
			sourceIdx := mapping.SpawnFrom[i]
			template := entities[sourceIdx]
			templateTransform := world.Transform(template)
			templateDrawable := world.Drawable(template)
			templateMaterial := world.Material(template)
			templateBody := world.Body(template)
			templateLayer := world.Layer(template)

			// Spawn new particle.
			p := world.Spawn()

			// Copy transform (will be moved to target by behavior).
			if templateTransform != nil {
				world.AddTransform(p, &core.Transform{
					Position: templateTransform.Position,
					Rotation: templateTransform.Rotation,
					Scale:    templateTransform.Scale,
				})
			}

			// Copy body.
			if templateBody != nil {
				world.AddBody(p, &core.Body{
					Velocity:     templateBody.Velocity,
					Acceleration: templateBody.Acceleration,
				})
			}

			// Copy drawable (reuse same drawable instance - safe for read-only data).
			if templateDrawable != nil {
				world.AddDrawable(p, templateDrawable)
			}

			// Copy material.
			if templateMaterial != nil {
				world.AddMaterial(p, templateMaterial)
			}

			// Copy layer.
			world.AddLayer(p, templateLayer)

			// Add to roots.
			world.AddRoot(p)

			// Assign target from remaining cloud points.
			targetIdx := initialEntityCount + i
			world.AddBehavior(p, core.NewBehavior(InterpolateToTarget(cloud[targetIdx], speed)))

			// Add to entities list.
			entities = append(entities, p)
		}
	}

	return entities
}
