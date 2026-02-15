package particle

import (
	"flicker/core"
	"flicker/fmath"
)

// InterpolateToTarget returns a behavior that moves an entity towards a target position at a given speed.
// Moves at speed units/second. Stops when within epsilon distance.
// Updates Body.Velocity to reflect direction of motion (for directional materials).
// No physics - directly modifies Transform.Position. Use for deterministic motion (point cloud morphing).
func InterpolateToTarget(target fmath.Vec2, speed float64) core.Behavior {
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
