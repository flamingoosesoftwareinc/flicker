package fmath

import "math"

// Mat4 is a row-major 4x4 matrix for 3D transforms and projection.
type Mat4 [16]float64

func Mat4Identity() Mat4 {
	return Mat4{
		1, 0, 0, 0,
		0, 1, 0, 0,
		0, 0, 1, 0,
		0, 0, 0, 1,
	}
}

func (m Mat4) Multiply(o Mat4) Mat4 {
	var r Mat4
	for row := range 4 {
		for col := range 4 {
			for k := range 4 {
				r[row*4+col] += m[row*4+k] * o[k*4+col]
			}
		}
	}
	return r
}

func (m Mat4) Transpose() Mat4 {
	return Mat4{
		m[0], m[4], m[8], m[12],
		m[1], m[5], m[9], m[13],
		m[2], m[6], m[10], m[14],
		m[3], m[7], m[11], m[15],
	}
}

func (m Mat4) Determinant() float64 {
	// Cofactor expansion along the first row.
	return m[0]*m.cofactor(0, 0) - m[1]*m.cofactor(0, 1) +
		m[2]*m.cofactor(0, 2) - m[3]*m.cofactor(0, 3)
}

// cofactor returns the 3x3 minor determinant for element (row, col).
func (m Mat4) cofactor(row, col int) float64 {
	var sub [9]float64
	idx := 0
	for r := range 4 {
		if r == row {
			continue
		}
		for c := range 4 {
			if c == col {
				continue
			}
			sub[idx] = m[r*4+c]
			idx++
		}
	}
	return sub[0]*(sub[4]*sub[8]-sub[5]*sub[7]) -
		sub[1]*(sub[3]*sub[8]-sub[5]*sub[6]) +
		sub[2]*(sub[3]*sub[7]-sub[4]*sub[6])
}

func (m Mat4) Inverse() Mat4 {
	det := m.Determinant()
	if det == 0 {
		return Mat4Identity()
	}
	invDet := 1.0 / det

	var r Mat4
	for row := range 4 {
		for col := range 4 {
			sign := 1.0
			if (row+col)%2 != 0 {
				sign = -1.0
			}
			// Adjugate = transpose of cofactor matrix.
			r[col*4+row] = sign * m.cofactor(row, col) * invDet
		}
	}
	return r
}

// Apply transforms a 3D point (x,y,z,1) by this matrix with perspective divide.
func (m Mat4) Apply(v Vec3) Vec3 {
	x := m[0]*v.X + m[1]*v.Y + m[2]*v.Z + m[3]
	y := m[4]*v.X + m[5]*v.Y + m[6]*v.Z + m[7]
	z := m[8]*v.X + m[9]*v.Y + m[10]*v.Z + m[11]
	w := m[12]*v.X + m[13]*v.Y + m[14]*v.Z + m[15]
	if w != 0 && w != 1 {
		x /= w
		y /= w
		z /= w
	}
	return Vec3{X: x, Y: y, Z: z}
}

// Mat4Ortho returns an orthographic projection matrix.
func Mat4Ortho(left, right, bottom, top, near, far float64) Mat4 {
	rl := right - left
	tb := top - bottom
	fn := far - near
	return Mat4{
		2 / rl, 0, 0, -(right + left) / rl,
		0, 2 / tb, 0, -(top + bottom) / tb,
		0, 0, -2 / fn, -(far + near) / fn,
		0, 0, 0, 1,
	}
}

// Mat4Translate returns a translation matrix.
func Mat4Translate(x, y, z float64) Mat4 {
	return Mat4{
		1, 0, 0, x,
		0, 1, 0, y,
		0, 0, 1, z,
		0, 0, 0, 1,
	}
}

// Mat4RotateX returns a rotation matrix around the X axis.
func Mat4RotateX(angle float64) Mat4 {
	c := math.Cos(angle)
	s := math.Sin(angle)
	return Mat4{
		1, 0, 0, 0,
		0, c, -s, 0,
		0, s, c, 0,
		0, 0, 0, 1,
	}
}

// Mat4RotateY returns a rotation matrix around the Y axis.
func Mat4RotateY(angle float64) Mat4 {
	c := math.Cos(angle)
	s := math.Sin(angle)
	return Mat4{
		c, 0, s, 0,
		0, 1, 0, 0,
		-s, 0, c, 0,
		0, 0, 0, 1,
	}
}

// Mat4RotateZ returns a rotation matrix around the Z axis.
func Mat4RotateZ(angle float64) Mat4 {
	c := math.Cos(angle)
	s := math.Sin(angle)
	return Mat4{
		c, -s, 0, 0,
		s, c, 0, 0,
		0, 0, 1, 0,
		0, 0, 0, 1,
	}
}

// Mat4Perspective returns a perspective projection matrix.
func Mat4Perspective(fovY, aspect, near, far float64) Mat4 {
	f := 1.0 / math.Tan(fovY/2)
	fn := near - far
	return Mat4{
		f / aspect, 0, 0, 0,
		0, f, 0, 0,
		0, 0, (far + near) / fn, 2 * far * near / fn,
		0, 0, -1, 0,
	}
}
