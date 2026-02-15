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
	SpawnFrom     []int // SpawnFrom[i] = entity to clone from for spawn i
	SpawnTargets  []int // SpawnTargets[i] = target index for spawn i
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
			extraCount := targetCount - entityCount
			mapping.SpawnFrom = make([]int, extraCount)
			mapping.SpawnTargets = make([]int, extraCount)
			for i := 0; i < extraCount; i++ {
				mapping.SpawnFrom[i] = i % entityCount
				mapping.SpawnTargets[i] = entityCount + i // Targets sequentially after assigned ones
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
			extraCount := targetCount - entityCount
			mapping.SpawnFrom = make([]int, extraCount)
			mapping.SpawnTargets = make([]int, extraCount)
			for i := 0; i < extraCount; i++ {
				mapping.SpawnFrom[i] = rand.Intn(entityCount)
				mapping.SpawnTargets[i] = entityCount + i // Targets sequentially after assigned ones
			}
		}

		return mapping
	}
}

// ClosestPointDistribution assigns each entity to its nearest target using greedy matching.
// This minimizes total travel distance and creates more natural-looking morphs.
func ClosestPointDistribution(
	entities []core.Entity,
	targets []fmath.Vec2,
	world *core.World,
) DistributionStrategy {
	return func(entityCount, targetCount int) TargetMapping {
		mapping := TargetMapping{
			EntityTargets: make([]int, entityCount),
		}

		// Use min of entityCount and captured entities length
		// (particle count may have grown since strategy was created)
		availableEntities := entityCount
		if availableEntities > len(entities) {
			availableEntities = len(entities)
		}

		// Build position list for entities we have
		entityPositions := make([]fmath.Vec2, availableEntities)
		for i := 0; i < availableEntities; i++ {
			tr := world.Transform(entities[i])
			if tr != nil {
				entityPositions[i] = fmath.Vec2{X: tr.Position.X, Y: tr.Position.Y}
			}
		}

		// Track which targets are already assigned
		targetAssigned := make([]bool, targetCount)

		// Greedy assignment: for each entity we have positions for, assign to nearest unassigned target
		for i := 0; i < availableEntities; i++ {
			bestTarget := -1
			bestDist := 1e9

			for t := 0; t < targetCount; t++ {
				if targetAssigned[t] {
					continue
				}

				dx := entityPositions[i].X - targets[t].X
				dy := entityPositions[i].Y - targets[t].Y
				dist := dx*dx + dy*dy // squared distance (avoids sqrt)

				if dist < bestDist {
					bestDist = dist
					bestTarget = t
				}
			}

			if bestTarget >= 0 {
				mapping.EntityTargets[i] = bestTarget
				targetAssigned[bestTarget] = true
			} else {
				// All targets assigned, wrap to first target
				mapping.EntityTargets[i] = 0
			}
		}

		// For entities beyond what we captured, use round-robin
		for i := availableEntities; i < entityCount; i++ {
			targetIdx := i % targetCount
			mapping.EntityTargets[i] = targetIdx
			targetAssigned[targetIdx] = true // Mark these targets as assigned
		}

		// If more targets than entities, spawn from nearest entity to each unassigned target
		if targetCount > entityCount {
			// Count actual unassigned targets first
			unassignedCount := 0
			for t := 0; t < targetCount; t++ {
				if !targetAssigned[t] {
					unassignedCount++
				}
			}

			mapping.SpawnFrom = make([]int, unassignedCount)
			mapping.SpawnTargets = make([]int, unassignedCount)
			spawnIdx := 0

			for t := 0; t < targetCount; t++ {
				if targetAssigned[t] {
					continue
				}

				// Find nearest entity to this target (from entities we have positions for)
				bestEntity := 0
				bestDist := 1e9

				for e := 0; e < availableEntities; e++ {
					dx := entityPositions[e].X - targets[t].X
					dy := entityPositions[e].Y - targets[t].Y
					dist := dx*dx + dy*dy

					if dist < bestDist {
						bestDist = dist
						bestEntity = e
					}
				}

				mapping.SpawnFrom[spawnIdx] = bestEntity
				mapping.SpawnTargets[spawnIdx] = t // Use the actual unassigned target index
				spawnIdx++
			}
		}

		return mapping
	}
}

// DistributeParticlesToTargets assigns particles to target positions using a distribution strategy.
// This is a structural operation that handles particle-to-target assignment and spawning,
// performed at the boundary between morph targets (not during phase execution).
//
// If cloud has more points than entities, spawns additional particles to fill the gaps.
// Returns all entities (original + newly spawned).
//
// Note: This function does NOT add movement behaviors. Phases are responsible for
// adding whatever behaviors/keyframes/curves they need to move particles to their targets.
func DistributeParticlesToTargets(
	entities []core.Entity,
	cloud []fmath.Vec2,
	strategy DistributionStrategy,
	world *core.World,
) []core.Entity {
	if len(cloud) == 0 {
		return entities
	}

	// Compute target mapping using strategy.
	mapping := strategy(len(entities), len(cloud))

	// If cloud has more points than entities, spawn new particles for the extra points.
	if len(cloud) > len(entities) {
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

			// Add to entities list.
			// Note: Target assignment is recorded in mapping.SpawnTargets[i],
			// but phases are responsible for adding movement behaviors.
			entities = append(entities, p)
		}
	}

	return entities
}
