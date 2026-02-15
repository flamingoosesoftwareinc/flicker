package particle

import (
	"math"
	"testing"

	"flicker/core"
	"flicker/fmath"
)

func TestInterpolateToTarget(t *testing.T) {
	w := core.NewWorld()
	e := w.Spawn()

	// Start at (0, 0), target at (10, 0), speed 1.0 unit/sec.
	w.AddTransform(e, &core.Transform{Position: fmath.Vec3{X: 0, Y: 0}})
	w.AddBody(e, &core.Body{})

	target := fmath.Vec2{X: 10, Y: 0}
	speed := 1.0
	behaviorFn := InterpolateToTarget(target, speed)

	// Step 1: dt=0.5, should move 0.5 units.
	behaviorFn(core.Time{Delta: 0.5}, e, w)

	transform := w.Transform(e)
	body := w.Body(e)
	if math.Abs(transform.Position.X-0.5) > 0.001 {
		t.Errorf("Expected position.X=0.5 after step 1, got %f", transform.Position.X)
	}
	// Velocity should be set to (1.0, 0) = speed * direction.
	if math.Abs(body.Velocity.X-1.0) > 0.001 || math.Abs(body.Velocity.Y) > 0.001 {
		t.Errorf("Expected velocity=(1.0, 0.0), got (%f, %f)", body.Velocity.X, body.Velocity.Y)
	}

	// Step 2: dt=0.5, should move another 0.5 units.
	behaviorFn(core.Time{Delta: 0.5}, e, w)

	if math.Abs(transform.Position.X-1.0) > 0.001 {
		t.Errorf("Expected position.X=1.0 after step 2, got %f", transform.Position.X)
	}
}

func TestInterpolateToTargetReaches(t *testing.T) {
	w := core.NewWorld()
	e := w.Spawn()

	// Start at (0, 0), target at (1, 0), speed 10.0 unit/sec.
	w.AddTransform(e, &core.Transform{Position: fmath.Vec3{X: 0, Y: 0}})
	w.AddBody(e, &core.Body{})

	target := fmath.Vec2{X: 1, Y: 0}
	speed := 10.0
	behaviorFn := InterpolateToTarget(target, speed)

	// Step with dt=0.5. Max movement = 10*0.5 = 5 units, but target is only 1 unit away.
	// Should snap to target.
	behaviorFn(core.Time{Delta: 0.5}, e, w)

	transform := w.Transform(e)
	body := w.Body(e)
	if math.Abs(transform.Position.X-1.0) > 0.001 {
		t.Errorf("Expected to snap to target.X=1.0, got %f", transform.Position.X)
	}
	// Velocity should be zero when reaching target.
	if math.Abs(body.Velocity.X) > 0.001 || math.Abs(body.Velocity.Y) > 0.001 {
		t.Errorf(
			"Expected velocity=(0, 0) when at target, got (%f, %f)",
			body.Velocity.X,
			body.Velocity.Y,
		)
	}

	// Step again - should not move past target.
	behaviorFn(core.Time{Delta: 0.5}, e, w)

	if math.Abs(transform.Position.X-1.0) > 0.001 {
		t.Errorf("Expected to stay at target.X=1.0, got %f", transform.Position.X)
	}
}

func TestInterpolateToTargetWithoutTransform(t *testing.T) {
	w := core.NewWorld()
	e := w.Spawn()

	// Entity without Transform should not crash.
	behaviorFn := InterpolateToTarget(fmath.Vec2{X: 10, Y: 10}, 1.0)
	behaviorFn(core.Time{Delta: 0.1}, e, w)

	// No panic = success.
}
