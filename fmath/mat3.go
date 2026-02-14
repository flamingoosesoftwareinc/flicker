package fmath

import "math"

// Mat3 is a row-major 3x3 matrix for 2D homogeneous transforms.
//
//	[0 1 2]   [m00 m01 tx]
//	[3 4 5] = [m10 m11 ty]
//	[6 7 8]   [  0   0  1]
type Mat3 [9]float64

func Mat3Identity() Mat3 {
	return Mat3{
		1, 0, 0,
		0, 1, 0,
		0, 0, 1,
	}
}

func (m Mat3) Multiply(o Mat3) Mat3 {
	return Mat3{
		m[0]*o[0] + m[1]*o[3] + m[2]*o[6],
		m[0]*o[1] + m[1]*o[4] + m[2]*o[7],
		m[0]*o[2] + m[1]*o[5] + m[2]*o[8],

		m[3]*o[0] + m[4]*o[3] + m[5]*o[6],
		m[3]*o[1] + m[4]*o[4] + m[5]*o[7],
		m[3]*o[2] + m[4]*o[5] + m[5]*o[8],

		m[6]*o[0] + m[7]*o[3] + m[8]*o[6],
		m[6]*o[1] + m[7]*o[4] + m[8]*o[7],
		m[6]*o[2] + m[7]*o[5] + m[8]*o[8],
	}
}

func (m Mat3) Transpose() Mat3 {
	return Mat3{
		m[0], m[3], m[6],
		m[1], m[4], m[7],
		m[2], m[5], m[8],
	}
}

func (m Mat3) Determinant() float64 {
	return m[0]*(m[4]*m[8]-m[5]*m[7]) -
		m[1]*(m[3]*m[8]-m[5]*m[6]) +
		m[2]*(m[3]*m[7]-m[4]*m[6])
}

func (m Mat3) Inverse() Mat3 {
	det := m.Determinant()
	if det == 0 {
		return Mat3Identity()
	}
	invDet := 1.0 / det
	return Mat3{
		(m[4]*m[8] - m[5]*m[7]) * invDet,
		(m[2]*m[7] - m[1]*m[8]) * invDet,
		(m[1]*m[5] - m[2]*m[4]) * invDet,

		(m[5]*m[6] - m[3]*m[8]) * invDet,
		(m[0]*m[8] - m[2]*m[6]) * invDet,
		(m[2]*m[3] - m[0]*m[5]) * invDet,

		(m[3]*m[7] - m[4]*m[6]) * invDet,
		(m[1]*m[6] - m[0]*m[7]) * invDet,
		(m[0]*m[4] - m[1]*m[3]) * invDet,
	}
}

// Apply transforms a 2D point (x,y,1) by this matrix.
func (m Mat3) Apply(v Vec2) Vec2 {
	return Vec2{
		X: m[0]*v.X + m[1]*v.Y + m[2],
		Y: m[3]*v.X + m[4]*v.Y + m[5],
	}
}

func Mat3Translate(x, y float64) Mat3 {
	return Mat3{
		1, 0, x,
		0, 1, y,
		0, 0, 1,
	}
}

func Mat3Rotate(radians float64) Mat3 {
	c := math.Cos(radians)
	s := math.Sin(radians)
	return Mat3{
		c, -s, 0,
		s, c, 0,
		0, 0, 1,
	}
}

func Mat3Scale(sx, sy float64) Mat3 {
	return Mat3{
		sx, 0, 0,
		0, sy, 0,
		0, 0, 1,
	}
}
