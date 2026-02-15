package physics

import (
	"flicker/core"
	"flicker/fmath"
)

// EulerIntegration returns a behavior that performs basic Euler integration on entities with Body and Transform components.
// Algorithm:
//
//	pos += vel * dt
//	vel += acc * dt
//	acc = 0  // reset for next frame
func EulerIntegration() core.BehaviorFunc {
	return func(t core.Time, e core.Entity, w *core.World) {
		body := w.Body(e)
		transform := w.Transform(e)
		if body == nil || transform == nil {
			return
		}

		dt := t.Delta

		// Update position based on velocity.
		transform.Position.X += body.Velocity.X * dt
		transform.Position.Y += body.Velocity.Y * dt

		// Update velocity based on acceleration.
		body.Velocity.X += body.Acceleration.X * dt
		body.Velocity.Y += body.Acceleration.Y * dt

		// Reset acceleration for next frame.
		body.Acceleration = fmath.Vec2{}
	}
}

// VerletIntegration returns a behavior that performs Verlet integration on entities with Body and Transform components.
// Verlet integration is more stable than Euler for physics simulation.
// Algorithm (maintains previous positions in closure):
//
//	newPos = 2*pos - prevPos + acc*dt²
//	prevPos = pos
//	pos = newPos
//	vel = (pos - prevPos) / dt  // compute velocity for other behaviors
//	acc = 0
func VerletIntegration() core.BehaviorFunc {
	prevPositions := make(map[core.Entity]fmath.Vec2)

	return func(t core.Time, e core.Entity, w *core.World) {
		body := w.Body(e)
		transform := w.Transform(e)
		if body == nil || transform == nil {
			return
		}

		dt := t.Delta
		pos := fmath.Vec2{X: transform.Position.X, Y: transform.Position.Y}

		// Initialize previous position on first frame.
		prev, exists := prevPositions[e]
		if !exists {
			prev = pos
		}

		// Verlet integration: newPos = 2*pos - prevPos + acc*dt²
		dtSq := dt * dt
		newPos := fmath.Vec2{
			X: 2*pos.X - prev.X + body.Acceleration.X*dtSq,
			Y: 2*pos.Y - prev.Y + body.Acceleration.Y*dtSq,
		}

		// Update previous position.
		prevPositions[e] = pos

		// Update transform position.
		transform.Position.X = newPos.X
		transform.Position.Y = newPos.Y

		// Compute velocity for other behaviors: vel = (pos - prevPos) / dt
		if dt > 0 {
			body.Velocity.X = (newPos.X - pos.X) / dt
			body.Velocity.Y = (newPos.Y - pos.Y) / dt
		}

		// Reset acceleration for next frame.
		body.Acceleration = fmath.Vec2{}
	}
}
