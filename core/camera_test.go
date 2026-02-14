package core

import (
	"testing"

	"flicker/fmath"
)

func TestViewMatrixIdentityAtOrigin(t *testing.T) {
	// Camera at origin with zoom 1 should map origin to screen center.
	cam := &Camera{Zoom: 1}
	tr := &Transform{
		Scale: fmath.Vec3{X: 1, Y: 1, Z: 1},
	}
	m := ViewMatrix(cam, tr, 80, 24)
	got := m.Apply(fmath.Vec2{X: 0, Y: 0})
	if !approxEqual(got.X, 40) || !approxEqual(got.Y, 12) {
		t.Errorf("origin maps to %v, want (40,12)", got)
	}
}

func TestViewMatrixPan(t *testing.T) {
	// Camera panned to (10, 5); world origin should appear offset.
	cam := &Camera{Zoom: 1}
	tr := &Transform{
		Position: fmath.Vec3{X: 10, Y: 5},
		Scale:    fmath.Vec3{X: 1, Y: 1, Z: 1},
	}
	m := ViewMatrix(cam, tr, 80, 24)

	// Camera position should map to screen center.
	got := m.Apply(fmath.Vec2{X: 10, Y: 5})
	if !approxEqual(got.X, 40) || !approxEqual(got.Y, 12) {
		t.Errorf("camera pos maps to %v, want (40,12)", got)
	}

	// World origin should appear shifted left and up by camera offset.
	got = m.Apply(fmath.Vec2{X: 0, Y: 0})
	if !approxEqual(got.X, 30) || !approxEqual(got.Y, 7) {
		t.Errorf("world origin maps to %v, want (30,7)", got)
	}
}

func TestViewMatrixZoom(t *testing.T) {
	// Zoom=2 should make distances from center 2× larger on screen.
	cam := &Camera{Zoom: 2}
	tr := &Transform{
		Scale: fmath.Vec3{X: 1, Y: 1, Z: 1},
	}
	m := ViewMatrix(cam, tr, 80, 24)

	// Origin still maps to screen center.
	got := m.Apply(fmath.Vec2{X: 0, Y: 0})
	if !approxEqual(got.X, 40) || !approxEqual(got.Y, 12) {
		t.Errorf("zoom2 origin maps to %v, want (40,12)", got)
	}

	// Point (5,0) should be 10 pixels from center (5 * zoom=2).
	got = m.Apply(fmath.Vec2{X: 5, Y: 0})
	if !approxEqual(got.X, 50) || !approxEqual(got.Y, 12) {
		t.Errorf("zoom2 (5,0) maps to %v, want (50,12)", got)
	}
}

func TestViewMatrixZeroZoom(t *testing.T) {
	// Zero-value Camera (Zoom=0) should be treated as zoom 1.
	cam := &Camera{}
	tr := &Transform{
		Scale: fmath.Vec3{X: 1, Y: 1, Z: 1},
	}
	m := ViewMatrix(cam, tr, 80, 24)
	got := m.Apply(fmath.Vec2{X: 0, Y: 0})
	if !approxEqual(got.X, 40) || !approxEqual(got.Y, 12) {
		t.Errorf("zero zoom origin maps to %v, want (40,12)", got)
	}

	// Ensure it behaves identically to zoom=1.
	cam1 := &Camera{Zoom: 1}
	m1 := ViewMatrix(cam1, tr, 80, 24)
	got1 := m1.Apply(fmath.Vec2{X: 7, Y: 3})
	got0 := m.Apply(fmath.Vec2{X: 7, Y: 3})
	if !approxEqual(got0.X, got1.X) || !approxEqual(got0.Y, got1.Y) {
		t.Errorf("zero zoom %v != zoom=1 %v", got0, got1)
	}
}

func TestNoCameraReturnsIdentity(t *testing.T) {
	// ViewMatrix with nil camera or nil transform returns identity.
	id := fmath.Mat3Identity()

	m := ViewMatrix(nil, &Transform{}, 80, 24)
	if m != id {
		t.Errorf("nil camera: got %v, want identity", m)
	}

	m = ViewMatrix(&Camera{Zoom: 1}, nil, 80, 24)
	if m != id {
		t.Errorf("nil transform: got %v, want identity", m)
	}
}

func TestViewMatrixNoActiveCamera(t *testing.T) {
	// World with no active camera should produce identity view matrix.
	w := NewWorld()
	m := viewMatrix(w, 80, 24)
	if m != fmath.Mat3Identity() {
		t.Errorf("no active camera: got %v, want identity", m)
	}
}

func TestViewMatrixActiveCamera(t *testing.T) {
	// World with active camera should produce camera's view matrix.
	w := NewWorld()
	cam := w.Spawn()
	w.AddTransform(cam, &Transform{
		Position: fmath.Vec3{X: 10, Y: 5},
		Scale:    fmath.Vec3{X: 1, Y: 1, Z: 1},
	})
	w.AddCamera(cam, &Camera{Zoom: 1})
	w.SetActiveCamera(cam)

	m := viewMatrix(w, 80, 24)
	got := m.Apply(fmath.Vec2{X: 10, Y: 5})
	if !approxEqual(got.X, 40) || !approxEqual(got.Y, 12) {
		t.Errorf("active camera: pos maps to %v, want (40,12)", got)
	}
}
