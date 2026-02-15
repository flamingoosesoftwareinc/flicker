package physics

import (
	"math"
	"testing"

	"flicker/core"
	"flicker/fmath"
)

func TestSpring(t *testing.T) {
	w := core.NewWorld()
	e := w.Spawn()

	// Entity at (10, 0), anchor at (0, 0), no velocity.
	w.AddTransform(e, &core.Transform{Position: fmath.Vec3{X: 10, Y: 0}})
	w.AddBody(e, &core.Body{Velocity: fmath.Vec2{X: 0, Y: 0}})

	anchor := fmath.Vec2{X: 0, Y: 0}
	k := 1.0
	damping := 0.0
	behaviorFn := Spring(anchor, k, damping)

	behaviorFn(core.Time{Delta: 0.1}, e, w)

	body := w.Body(e)

	// Spring force: F = -k * (pos - anchor) = -1.0 * (10, 0) = (-10, 0)
	// No damping, so acceleration should be (-10, 0).
	if math.Abs(body.Acceleration.X-(-10.0)) > 0.001 {
		t.Errorf("Expected acceleration.X=-10.0, got %f", body.Acceleration.X)
	}
	if math.Abs(body.Acceleration.Y) > 0.001 {
		t.Errorf("Expected acceleration.Y=0.0, got %f", body.Acceleration.Y)
	}
}

func TestSpringDamping(t *testing.T) {
	w := core.NewWorld()
	e := w.Spawn()

	// Entity at (10, 0), anchor at (0, 0), with velocity towards anchor.
	w.AddTransform(e, &core.Transform{Position: fmath.Vec3{X: 10, Y: 0}})
	w.AddBody(e, &core.Body{Velocity: fmath.Vec2{X: -5, Y: 0}})

	anchor := fmath.Vec2{X: 0, Y: 0}
	k := 1.0
	damping := 0.5
	behaviorFn := Spring(anchor, k, damping)

	behaviorFn(core.Time{Delta: 0.1}, e, w)

	body := w.Body(e)

	// Spring force: F = -1.0 * (10, 0) = (-10, 0)
	// Damping force: F = -0.5 * (-5, 0) = (2.5, 0)
	// Total: (-10, 0) + (2.5, 0) = (-7.5, 0)
	if math.Abs(body.Acceleration.X-(-7.5)) > 0.001 {
		t.Errorf("Expected acceleration.X=-7.5 with damping, got %f", body.Acceleration.X)
	}
}

func TestSpringWithoutComponents(t *testing.T) {
	w := core.NewWorld()
	e := w.Spawn()

	// Entity without Body or Transform should not crash.
	behaviorFn := Spring(fmath.Vec2{X: 0, Y: 0}, 1.0, 0.5)
	behaviorFn(core.Time{Delta: 0.1}, e, w)

	// No panic = success.
}
