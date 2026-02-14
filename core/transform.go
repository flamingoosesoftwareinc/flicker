package core

import "flicker/fmath"

type Transform struct {
	Position fmath.Vec3
	Rotation float64    // radians, 2D rotation around Z axis
	Scale    fmath.Vec3 // per-axis scale; {0,0,0} means zero (scales to nothing)
}

func (t *Transform) LocalMatrix() fmath.Mat3 {
	return fmath.Mat3Translate(t.Position.X, t.Position.Y).
		Multiply(fmath.Mat3Rotate(t.Rotation)).
		Multiply(fmath.Mat3Scale(t.Scale.X, t.Scale.Y))
}
