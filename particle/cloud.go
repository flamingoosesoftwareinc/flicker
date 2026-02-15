package particle

import (
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
// Simple strategy: round-robin assignment (entities[i] gets cloud[i % len(cloud)]).
// Adds InterpolateToTarget behavior to each entity.
func DistributeTargets(
	entities []core.Entity,
	cloud []fmath.Vec2,
	speed float64,
	world *core.World,
) {
	if len(cloud) == 0 {
		return
	}

	for i, e := range entities {
		target := cloud[i%len(cloud)]
		world.AddBehavior(e, InterpolateToTarget(target, speed))
	}
}
