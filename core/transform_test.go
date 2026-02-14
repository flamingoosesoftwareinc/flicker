package core

import (
	"math"
	"testing"

	"flicker/fmath"
)

const epsilon = 1e-9

func approxEqual(a, b float64) bool {
	return math.Abs(a-b) < epsilon
}

func TestTransformZeroValue(t *testing.T) {
	// Zero-value Transform has zero scale — everything maps to origin.
	tr := &Transform{}
	m := tr.LocalMatrix()
	got := m.Apply(fmath.Vec2{X: 5, Y: 10})
	if !approxEqual(got.X, 0) || !approxEqual(got.Y, 0) {
		t.Errorf("zero-value Transform maps (5,10) to %v, want (0,0)", got)
	}
}

func TestTransformPositionUnitScale(t *testing.T) {
	tr := &Transform{
		Position: fmath.Vec3{X: 10, Y: 20},
		Scale:    fmath.Vec3{X: 1, Y: 1, Z: 1},
	}
	m := tr.LocalMatrix()
	got := m.Apply(fmath.Vec2{})
	if !approxEqual(got.X, 10) || !approxEqual(got.Y, 20) {
		t.Errorf("position+unit scale: origin maps to %v, want (10,20)", got)
	}
}

func TestTransformRotation(t *testing.T) {
	tr := &Transform{
		Rotation: math.Pi / 2, // 90° counterclockwise
		Scale:    fmath.Vec3{X: 1, Y: 1, Z: 1},
	}
	m := tr.LocalMatrix()
	got := m.Apply(fmath.Vec2{X: 1, Y: 0})
	if !approxEqual(got.X, 0) || !approxEqual(got.Y, 1) {
		t.Errorf("90° rotation maps (1,0) to %v, want (0,1)", got)
	}
}

func TestTransformScaleOnly(t *testing.T) {
	tr := &Transform{
		Scale: fmath.Vec3{X: 3, Y: 2, Z: 1},
	}
	m := tr.LocalMatrix()
	got := m.Apply(fmath.Vec2{X: 4, Y: 5})
	if !approxEqual(got.X, 12) || !approxEqual(got.Y, 10) {
		t.Errorf("scale(3,2) maps (4,5) to %v, want (12,10)", got)
	}
}

func TestTransformCombinedTRS(t *testing.T) {
	tr := &Transform{
		Position: fmath.Vec3{X: 10, Y: 20},
		Rotation: math.Pi / 2,
		Scale:    fmath.Vec3{X: 2, Y: 2, Z: 1},
	}
	m := tr.LocalMatrix()
	// (1,0) → Scale(2,2) → (2,0) → Rotate(90°) → (0,2) → Translate(10,20) → (10,22)
	got := m.Apply(fmath.Vec2{X: 1, Y: 0})
	if !approxEqual(got.X, 10) || !approxEqual(got.Y, 22) {
		t.Errorf("TRS maps (1,0) to %v, want (10,22)", got)
	}
}

func TestTransformZeroScale(t *testing.T) {
	tr := &Transform{
		Position: fmath.Vec3{X: 5, Y: 5},
		Scale:    fmath.Vec3{}, // zero scale
	}
	m := tr.LocalMatrix()
	// Any point should map to the translated origin (5,5).
	got := m.Apply(fmath.Vec2{X: 100, Y: 200})
	if !approxEqual(got.X, 5) || !approxEqual(got.Y, 5) {
		t.Errorf("zero scale maps (100,200) to %v, want (5,5)", got)
	}
}
