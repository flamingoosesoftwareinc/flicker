package fmath

import (
	"math"
	"testing"
)

func mat4ApproxEqual(a, b Mat4) bool {
	for i := range 16 {
		if !approxEqual(a[i], b[i]) {
			return false
		}
	}
	return true
}

func TestMat4Identity(t *testing.T) {
	id := Mat4Identity()
	p := Vec3{X: 3, Y: 7, Z: 11}
	got := id.Apply(p)
	if !approxEqual(got.X, p.X) || !approxEqual(got.Y, p.Y) || !approxEqual(got.Z, p.Z) {
		t.Errorf("Identity.Apply(%v) = %v", p, got)
	}
}

func TestMat4MultiplyIdentity(t *testing.T) {
	m := Mat4{
		2, 3, 4, 5,
		6, 7, 8, 9,
		10, 11, 12, 13,
		0, 0, 0, 1,
	}
	id := Mat4Identity()
	if !mat4ApproxEqual(m.Multiply(id), m) {
		t.Error("M * I != M")
	}
	if !mat4ApproxEqual(id.Multiply(m), m) {
		t.Error("I * M != M")
	}
}

func TestMat4Transpose(t *testing.T) {
	m := Mat4{
		1, 2, 3, 4,
		5, 6, 7, 8,
		9, 10, 11, 12,
		13, 14, 15, 16,
	}
	got := m.Transpose()
	want := Mat4{
		1, 5, 9, 13,
		2, 6, 10, 14,
		3, 7, 11, 15,
		4, 8, 12, 16,
	}
	if got != want {
		t.Errorf("Transpose = %v, want %v", got, want)
	}
}

func TestMat4Determinant(t *testing.T) {
	id := Mat4Identity()
	if !approxEqual(id.Determinant(), 1) {
		t.Errorf("det(I) = %v, want 1", id.Determinant())
	}
}

func TestMat4Inverse(t *testing.T) {
	m := Mat4{
		1, 0, 0, 5,
		0, 2, 0, 3,
		0, 0, 3, 1,
		0, 0, 0, 1,
	}
	inv := m.Inverse()
	product := m.Multiply(inv)
	if !mat4ApproxEqual(product, Mat4Identity()) {
		t.Errorf("M * M^-1 = %v, want identity", product)
	}
}

func TestMat4InverseSingular(t *testing.T) {
	var m Mat4 // all zeros
	inv := m.Inverse()
	if inv != Mat4Identity() {
		t.Errorf("Inverse of singular = %v, want identity", inv)
	}
}

func TestMat4Ortho(t *testing.T) {
	m := Mat4Ortho(-1, 1, -1, 1, 0.1, 100)
	// Origin should map to (0, 0, near-mapped)
	got := m.Apply(Vec3{})
	if !approxEqual(got.X, 0) || !approxEqual(got.Y, 0) {
		t.Errorf("Ortho.Apply(0,0,0) = %v, want X=0,Y=0", got)
	}
	// Right edge maps to +1
	gotR := m.Apply(Vec3{X: 1})
	if !approxEqual(gotR.X, 1) {
		t.Errorf("Ortho.Apply(1,0,0).X = %v, want 1", gotR.X)
	}
	// Left edge maps to -1
	gotL := m.Apply(Vec3{X: -1})
	if !approxEqual(gotL.X, -1) {
		t.Errorf("Ortho.Apply(-1,0,0).X = %v, want -1", gotL.X)
	}
}

func TestMat4Perspective(t *testing.T) {
	fov := math.Pi / 2 // 90°
	m := Mat4Perspective(fov, 1.0, 0.1, 100)
	// A point on the near plane, centered, should map to (0,0,~-1).
	got := m.Apply(Vec3{Z: -0.1})
	if !approxEqual(got.X, 0) || !approxEqual(got.Y, 0) {
		t.Errorf("Perspective.Apply(0,0,-near) = %v, want X=0,Y=0", got)
	}
}

func TestMat4ApplyPerspectiveDivide(t *testing.T) {
	// Manually construct a matrix with w != 1.
	m := Mat4{
		1, 0, 0, 0,
		0, 1, 0, 0,
		0, 0, 1, 0,
		0, 0, 0, 2, // w = 2
	}
	got := m.Apply(Vec3{X: 4, Y: 6, Z: 8})
	// w = 2, so x/w=2, y/w=3, z/w=4
	if !approxEqual(got.X, 2) || !approxEqual(got.Y, 3) || !approxEqual(got.Z, 4) {
		t.Errorf("Apply with w=2: %v, want (2,3,4)", got)
	}
}
