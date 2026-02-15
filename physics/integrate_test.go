package physics

import (
	"math"
	"testing"

	"flicker/core"
	"flicker/fmath"
)

func TestEulerIntegration(t *testing.T) {
	w := core.NewWorld()
	e := w.Spawn()

	// Start at origin with zero velocity.
	w.AddTransform(e, &core.Transform{Position: fmath.Vec3{X: 0, Y: 0}})
	w.AddBody(e, &core.Body{
		Velocity:     fmath.Vec2{X: 0, Y: 0},
		Acceleration: fmath.Vec2{X: 10, Y: 0}, // constant acceleration
	})
	w.AddBehavior(e, core.NewBehavior(EulerIntegration()))

	// Step 1: dt=0.1
	// vel = 0 + 10*0.1 = 1
	// pos = 0 + 0*0.1 = 0
	behavior := w.Behaviors(e)[0]
	behavior.Update(core.Time{Delta: 0.1}, e, w)

	body := w.Body(e)
	transform := w.Transform(e)

	if math.Abs(transform.Position.X-0.0) > 0.001 {
		t.Errorf("Expected position.X=0.0 after step 1, got %f", transform.Position.X)
	}
	if math.Abs(body.Velocity.X-1.0) > 0.001 {
		t.Errorf("Expected velocity.X=1.0 after step 1, got %f", body.Velocity.X)
	}
	if math.Abs(body.Acceleration.X-0.0) > 0.001 {
		t.Errorf("Expected acceleration.X=0.0 (reset) after step 1, got %f", body.Acceleration.X)
	}

	// Step 2: Apply acceleration again and step.
	body.Acceleration = fmath.Vec2{X: 10, Y: 0}
	behavior.Update(core.Time{Delta: 0.1}, e, w)

	// vel = 1 + 10*0.1 = 2
	// pos = 0 + 1*0.1 = 0.1
	if math.Abs(transform.Position.X-0.1) > 0.001 {
		t.Errorf("Expected position.X=0.1 after step 2, got %f", transform.Position.X)
	}
	if math.Abs(body.Velocity.X-2.0) > 0.001 {
		t.Errorf("Expected velocity.X=2.0 after step 2, got %f", body.Velocity.X)
	}
}

func TestVerletIntegration(t *testing.T) {
	w := core.NewWorld()
	e := w.Spawn()

	// Start at origin with zero velocity.
	w.AddTransform(e, &core.Transform{Position: fmath.Vec3{X: 0, Y: 0}})
	w.AddBody(e, &core.Body{
		Velocity:     fmath.Vec2{X: 0, Y: 0},
		Acceleration: fmath.Vec2{X: 10, Y: 0}, // constant acceleration
	})
	w.AddBehavior(e, core.NewBehavior(VerletIntegration()))

	behavior := w.Behaviors(e)[0]

	// Step 1: dt=0.1
	behavior.Update(core.Time{Delta: 0.1}, e, w)

	transform := w.Transform(e)
	body := w.Body(e)

	// First step initializes prevPos to current pos.
	// newPos = 2*0 - 0 + 10*0.1² = 0.1
	if math.Abs(transform.Position.X-0.1) > 0.001 {
		t.Errorf("Expected position.X=0.1 after step 1, got %f", transform.Position.X)
	}

	// Step 2: Apply acceleration again and step.
	body.Acceleration = fmath.Vec2{X: 10, Y: 0}
	behavior.Update(core.Time{Delta: 0.1}, e, w)

	// prevPos = 0, pos = 0.1
	// newPos = 2*0.1 - 0 + 10*0.1² = 0.2 + 0.1 = 0.3
	if math.Abs(transform.Position.X-0.3) > 0.001 {
		t.Errorf("Expected position.X=0.3 after step 2, got %f", transform.Position.X)
	}
}

func TestVerletStateSeparation(t *testing.T) {
	w := core.NewWorld()

	// Create two entities with the same Verlet behavior function.
	behaviorFn := VerletIntegration()

	e1 := w.Spawn()
	w.AddTransform(e1, &core.Transform{Position: fmath.Vec3{X: 0, Y: 0}})
	w.AddBody(e1, &core.Body{Acceleration: fmath.Vec2{X: 10, Y: 0}})

	e2 := w.Spawn()
	w.AddTransform(e2, &core.Transform{Position: fmath.Vec3{X: 100, Y: 0}})
	w.AddBody(e2, &core.Body{Acceleration: fmath.Vec2{X: 20, Y: 0}})

	// Step both entities using the raw function (testing closure state separation).
	behaviorFn(core.Time{Delta: 0.1}, e1, w)
	behaviorFn(core.Time{Delta: 0.1}, e2, w)

	// Each entity should maintain separate state.
	t1 := w.Transform(e1)
	t2 := w.Transform(e2)

	if math.Abs(t1.Position.X-0.1) > 0.001 {
		t.Errorf("Expected e1.X=0.1, got %f", t1.Position.X)
	}
	if math.Abs(t2.Position.X-100.2) > 0.001 {
		t.Errorf("Expected e2.X=100.2, got %f", t2.Position.X)
	}
}

func TestIntegrationWithoutComponents(t *testing.T) {
	w := core.NewWorld()
	e := w.Spawn()

	// Entity without Body or Transform should not crash.
	behavior := core.NewBehavior(EulerIntegration())
	behavior.Update(core.Time{Delta: 0.1}, e, w)

	// No panic = success.
}
