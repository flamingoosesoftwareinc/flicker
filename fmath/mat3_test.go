package fmath

import (
	"math"
	"testing"
)

func mat3ApproxEqual(a, b Mat3) bool {
	for i := range 9 {
		if !approxEqual(a[i], b[i]) {
			return false
		}
	}
	return true
}

func TestMat3Identity(t *testing.T) {
	id := Mat3Identity()
	p := Vec2{X: 3, Y: 7}
	got := id.Apply(p)
	if !approxEqual(got.X, p.X) || !approxEqual(got.Y, p.Y) {
		t.Errorf("Identity.Apply(%v) = %v, want %v", p, got, p)
	}
}

func TestMat3MultiplyIdentity(t *testing.T) {
	m := Mat3{2, 3, 4, 5, 6, 7, 0, 0, 1}
	id := Mat3Identity()
	if !mat3ApproxEqual(m.Multiply(id), m) {
		t.Error("M * I != M")
	}
	if !mat3ApproxEqual(id.Multiply(m), m) {
		t.Error("I * M != M")
	}
}

func TestMat3Transpose(t *testing.T) {
	m := Mat3{1, 2, 3, 4, 5, 6, 7, 8, 9}
	got := m.Transpose()
	want := Mat3{1, 4, 7, 2, 5, 8, 3, 6, 9}
	if got != want {
		t.Errorf("Transpose = %v, want %v", got, want)
	}
}

func TestMat3Determinant(t *testing.T) {
	id := Mat3Identity()
	if !approxEqual(id.Determinant(), 1) {
		t.Errorf("det(I) = %v, want 1", id.Determinant())
	}
	m := Mat3{1, 2, 3, 0, 1, 4, 5, 6, 0}
	if !approxEqual(m.Determinant(), 1) {
		t.Errorf("det(m) = %v, want 1", m.Determinant())
	}
}

func TestMat3Inverse(t *testing.T) {
	m := Mat3{1, 2, 3, 0, 1, 4, 5, 6, 0}
	inv := m.Inverse()
	product := m.Multiply(inv)
	if !mat3ApproxEqual(product, Mat3Identity()) {
		t.Errorf("M * M^-1 = %v, want identity", product)
	}
}

func TestMat3InverseSingular(t *testing.T) {
	// Singular matrix (all zeros row).
	m := Mat3{1, 2, 3, 0, 0, 0, 4, 5, 6}
	inv := m.Inverse()
	if inv != Mat3Identity() {
		t.Errorf("Inverse of singular = %v, want identity", inv)
	}
}

func TestMat3Translate(t *testing.T) {
	m := Mat3Translate(10, 20)
	got := m.Apply(Vec2{})
	if !approxEqual(got.X, 10) || !approxEqual(got.Y, 20) {
		t.Errorf("Translate(10,20).Apply(0,0) = %v, want (10,20)", got)
	}
	got2 := m.Apply(Vec2{X: 5, Y: 3})
	if !approxEqual(got2.X, 15) || !approxEqual(got2.Y, 23) {
		t.Errorf("Translate(10,20).Apply(5,3) = %v, want (15,23)", got2)
	}
}

func TestMat3Rotate(t *testing.T) {
	m := Mat3Rotate(math.Pi / 2) // 90° counterclockwise
	got := m.Apply(Vec2{X: 1, Y: 0})
	if !approxEqual(got.X, 0) || !approxEqual(got.Y, 1) {
		t.Errorf("Rotate(90°).Apply(1,0) = %v, want (0,1)", got)
	}
}

func TestMat3Scale(t *testing.T) {
	m := Mat3Scale(2, 3)
	got := m.Apply(Vec2{X: 5, Y: 7})
	if !approxEqual(got.X, 10) || !approxEqual(got.Y, 21) {
		t.Errorf("Scale(2,3).Apply(5,7) = %v, want (10,21)", got)
	}
}

func TestMat3TRSComposition(t *testing.T) {
	// Translate(10,20) * Rotate(90°) * Scale(2,2)
	trs := Mat3Translate(10, 20).
		Multiply(Mat3Rotate(math.Pi / 2)).
		Multiply(Mat3Scale(2, 2))
	// Apply to (1, 0):
	// Scale: (2, 0)
	// Rotate 90°: (0, 2)
	// Translate: (10, 22)
	got := trs.Apply(Vec2{X: 1, Y: 0})
	if !approxEqual(got.X, 10) || !approxEqual(got.Y, 22) {
		t.Errorf("TRS.Apply(1,0) = %v, want (10,22)", got)
	}
}
