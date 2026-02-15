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

// DistributeTargets assigns target positions from cloud to entities.
// If cloud has more points than entities, spawns additional particles to fill the gaps.
// Returns all entities (original + newly spawned).
// Adds InterpolateToTarget behavior to each entity.
func DistributeTargets(
	entities []core.Entity,
	cloud []fmath.Vec2,
	speed float64,
	world *core.World,
) []core.Entity {
	if len(cloud) == 0 {
		return entities
	}

	// Assign targets to existing entities.
	for i, e := range entities {
		if i < len(cloud) {
			world.AddBehavior(e, core.NewBehavior(InterpolateToTarget(cloud[i], speed)))
		} else {
			// More entities than targets - round-robin wrap.
			world.AddBehavior(e, core.NewBehavior(InterpolateToTarget(cloud[i%len(cloud)], speed)))
		}
	}

	// If cloud has more points than entities, spawn new particles for the extra points.
	if len(cloud) > len(entities) {
		for i := len(entities); i < len(cloud); i++ {
			// Get template from a random existing entity to distribute spawn positions
			// across the source cloud instead of all spawning from the same position.
			randomIdx := rand.Intn(len(entities))
			template := entities[randomIdx]
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

			// Assign target.
			world.AddBehavior(p, core.NewBehavior(InterpolateToTarget(cloud[i], speed)))

			// Add to entities list.
			entities = append(entities, p)
		}
	}

	return entities
}
