package physics

import (
	"flicker/core"
	"flicker/fmath"
)

// Attractor returns a behavior that applies a force towards a center point with inverse-square falloff.
// F = strength / dist²
func Attractor(center fmath.Vec2, strength float64) core.Behavior {
	return func(t core.Time, e core.Entity, w *core.World) {
		body := w.Body(e)
		transform := w.Transform(e)
		if body == nil || transform == nil {
			return
		}

		pos := fmath.Vec2{X: transform.Position.X, Y: transform.Position.Y}
		delta := center.Sub(pos)
		distSq := delta.X*delta.X + delta.Y*delta.Y

		// Avoid division by zero and extreme forces at very close distances.
		if distSq < 0.01 {
			distSq = 0.01
		}

		// F = strength / dist²
		forceMag := strength / distSq
		dir := delta.Normalize()

		body.Acceleration.X += dir.X * forceMag
		body.Acceleration.Y += dir.Y * forceMag
	}
}

// Repulsor returns a behavior that applies a force away from a center point with inverse-square falloff.
// F = -strength / dist²
func Repulsor(center fmath.Vec2, strength float64) core.Behavior {
	return func(t core.Time, e core.Entity, w *core.World) {
		body := w.Body(e)
		transform := w.Transform(e)
		if body == nil || transform == nil {
			return
		}

		pos := fmath.Vec2{X: transform.Position.X, Y: transform.Position.Y}
		delta := center.Sub(pos)
		distSq := delta.X*delta.X + delta.Y*delta.Y

		// Avoid division by zero and extreme forces at very close distances.
		if distSq < 0.01 {
			distSq = 0.01
		}

		// F = -strength / dist² (negative for repulsion)
		forceMag := -strength / distSq
		dir := delta.Normalize()

		body.Acceleration.X += dir.X * forceMag
		body.Acceleration.Y += dir.Y * forceMag
	}
}

// Drag returns a behavior that applies a drag force opposing velocity.
// vel *= (1 - coefficient * dt)
func Drag(coefficient float64) core.Behavior {
	return func(t core.Time, e core.Entity, w *core.World) {
		body := w.Body(e)
		if body == nil {
			return
		}

		dt := t.Delta
		dampFactor := 1.0 - coefficient*dt
		if dampFactor < 0 {
			dampFactor = 0
		}

		body.Velocity.X *= dampFactor
		body.Velocity.Y *= dampFactor
	}
}

// Gravity returns a behavior that applies constant acceleration.
// acc += force
func Gravity(force fmath.Vec2) core.Behavior {
	return func(t core.Time, e core.Entity, w *core.World) {
		body := w.Body(e)
		if body == nil {
			return
		}

		body.Acceleration.X += force.X
		body.Acceleration.Y += force.Y
	}
}

// Turbulence returns a behavior that applies Perlin noise-based force at entity position.
// scale controls noise frequency, strength controls force magnitude.
func Turbulence(scale, strength float64) core.Behavior {
	return func(t core.Time, e core.Entity, w *core.World) {
		body := w.Body(e)
		transform := w.Transform(e)
		if body == nil || transform == nil {
			return
		}

		pos := fmath.Vec2{X: transform.Position.X, Y: transform.Position.Y}

		// Sample noise at two slightly offset positions to get a 2D force vector.
		noiseX := fmath.Noise2D(pos.X*scale, pos.Y*scale)
		noiseY := fmath.Noise2D(pos.X*scale+100, pos.Y*scale+100)

		// Map noise from [-1, 1] to force.
		forceX := noiseX * strength
		forceY := noiseY * strength

		body.Acceleration.X += forceX
		body.Acceleration.Y += forceY
	}
}
