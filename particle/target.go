package particle

import (
	"math"

	"flicker/core"
	"flicker/fmath"
)

// CalculateSpeedForDuration computes the speed needed for all entities to reach
// their targets within the given duration. Returns speed based on the maximum distance
// any entity needs to travel.
func CalculateSpeedForDuration(
	entities []core.Entity,
	targets []fmath.Vec2,
	duration float64,
	world *core.World,
) float64 {
	if len(entities) == 0 || len(targets) == 0 || duration <= 0 {
		return 1.0 // safe default
	}

	maxDistance := 0.0

	// Find maximum distance any entity needs to travel
	for i, e := range entities {
		tr := world.Transform(e)
		if tr == nil {
			continue
		}

		targetIdx := i % len(targets)
		target := targets[targetIdx]

		dx := target.X - tr.Position.X
		dy := target.Y - tr.Position.Y
		dist := math.Sqrt(dx*dx + dy*dy)

		if dist > maxDistance {
			maxDistance = dist
		}
	}

	// Speed = distance / time (with small buffer for safety)
	speed := maxDistance / (duration * 0.9) // 90% of duration to ensure completion
	if speed < 1.0 {
		speed = 1.0 // minimum speed
	}

	return speed
}

// InterpolateToTarget returns a behavior that moves an entity towards a target position at a given speed.
// Moves at speed units/second. Stops when within epsilon distance.
// Updates Body.Velocity to reflect direction of motion (for directional materials).
// No physics - directly modifies Transform.Position. Use for deterministic motion (point cloud morphing).
func InterpolateToTarget(target fmath.Vec2, speed float64) core.BehaviorFunc {
	return func(t core.Time, e core.Entity, w *core.World) {
		transform := w.Transform(e)
		body := w.Body(e)
		if transform == nil {
			return
		}

		pos := fmath.Vec2{X: transform.Position.X, Y: transform.Position.Y}
		delta := target.Sub(pos)
		dist := delta.Length()

		// Stop if close enough (epsilon threshold).
		const epsilon = 0.01
		if dist < epsilon {
			// Stopped at target - zero velocity.
			if body != nil {
				body.Velocity = fmath.Vec2{X: 0, Y: 0}
			}
			return
		}

		// Move towards target at speed.
		dt := t.Delta
		maxDist := speed * dt

		if dist <= maxDist {
			// Snap to target if we'd overshoot.
			transform.Position.X = target.X
			transform.Position.Y = target.Y
			// Set velocity to zero when reaching target.
			if body != nil {
				body.Velocity = fmath.Vec2{X: 0, Y: 0}
			}
		} else {
			// Move towards target.
			dir := delta.Normalize()
			transform.Position.X += dir.X * maxDist
			transform.Position.Y += dir.Y * maxDist
			// Update velocity to reflect current motion direction.
			if body != nil {
				body.Velocity = fmath.Vec2{X: dir.X * speed, Y: dir.Y * speed}
			}
		}
	}
}
