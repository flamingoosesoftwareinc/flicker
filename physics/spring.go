package physics

import (
	"flicker/core"
	"flicker/fmath"
)

// Spring returns a behavior that applies spring force towards an anchor point.
// F = -k*(pos - anchor) - damping*vel
// Classic Hooke's law with damping. Works well with Verlet integration.
func Spring(anchor fmath.Vec2, k, damping float64) core.BehaviorFunc {
	return func(t core.Time, e core.Entity, w *core.World) {
		body := w.Body(e)
		transform := w.Transform(e)
		if body == nil || transform == nil {
			return
		}

		pos := fmath.Vec2{X: transform.Position.X, Y: transform.Position.Y}

		// Spring force: F = -k * (pos - anchor)
		displacement := pos.Sub(anchor)
		springForce := displacement.Scale(-k)

		// Damping force: F = -damping * vel
		dampingForce := body.Velocity.Scale(-damping)

		// Total force.
		totalForce := springForce.Add(dampingForce)

		body.Acceleration.X += totalForce.X
		body.Acceleration.Y += totalForce.Y
	}
}
