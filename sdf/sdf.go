// Package sdf provides analytical 2D signed distance field primitives and operations.
//
// Signed Distance Functions (SDFs) represent shapes implicitly through distance fields.
// The SDF convention used throughout this package:
//   - Negative values: point is inside the shape
//   - Positive values: point is outside the shape
//   - Zero: point is exactly on the shape boundary
//
// The returned distance is the shortest Euclidean distance from the query point
// to the shape's surface.
//
// All primitives are centered at the origin unless otherwise specified.
// To create translated, rotated, or scaled shapes, transform the query point
// before passing it to the primitive function.
//
// Based on distance functions from Inigo Quilez:
// https://iquilezles.org/articles/distfunctions2d/
package sdf

import (
	"math"

	"flicker/fmath"
)

// Primitives

// Circle returns the signed distance from point p to a circle with the given radius.
// The circle is centered at the origin.
func Circle(p fmath.Vec2, radius float64) float64 {
	return p.Length() - radius
}

// Box returns the signed distance from point p to an axis-aligned box.
// The box is centered at the origin with half-extents given by size.
func Box(p fmath.Vec2, size fmath.Vec2) float64 {
	d := abs2(p).Sub(size)
	return max2(d, fmath.Vec2{}).Length() + math.Min(math.Max(d.X, d.Y), 0.0)
}

// RoundedBox returns the signed distance from point p to a rounded box.
// The box is centered at the origin with half-extents given by size.
// The corners are rounded with the given corner radii (clockwise from top-right):
// r.X = top-right, r.Y = bottom-right, r.Z = bottom-left, r.W = top-left.
// Note: This is simplified to use uniform rounding for now.
func RoundedBox(p fmath.Vec2, size fmath.Vec2, cornerRadius float64) float64 {
	q := abs2(p).Sub(size).Add(fmath.Vec2{X: cornerRadius, Y: cornerRadius})
	return math.Min(math.Max(q.X, q.Y), 0.0) + max2(q, fmath.Vec2{}).Length() - cornerRadius
}

// Segment returns the signed distance from point p to a line segment from a to b.
func Segment(p, a, b fmath.Vec2) float64 {
	pa := p.Sub(a)
	ba := b.Sub(a)
	h := fmath.Clamp(dot(pa, ba)/dot(ba, ba), 0.0, 1.0)
	return pa.Sub(ba.Scale(h)).Length()
}

// Triangle returns the signed distance from point p to a triangle with vertices p0, p1, p2.
func Triangle(p, p0, p1, p2 fmath.Vec2) float64 {
	e0 := p1.Sub(p0)
	e1 := p2.Sub(p1)
	e2 := p0.Sub(p2)
	v0 := p.Sub(p0)
	v1 := p.Sub(p1)
	v2 := p.Sub(p2)

	pq0 := v0.Sub(e0.Scale(fmath.Clamp(dot(v0, e0)/dot(e0, e0), 0.0, 1.0)))
	pq1 := v1.Sub(e1.Scale(fmath.Clamp(dot(v1, e1)/dot(e1, e1), 0.0, 1.0)))
	pq2 := v2.Sub(e2.Scale(fmath.Clamp(dot(v2, e2)/dot(e2, e2), 0.0, 1.0)))

	s := sign(e0.X*e2.Y - e0.Y*e2.X)
	d := min2(
		min2(
			fmath.Vec2{X: dot(pq0, pq0), Y: s * (v0.X*e0.Y - v0.Y*e0.X)},
			fmath.Vec2{X: dot(pq1, pq1), Y: s * (v1.X*e1.Y - v1.Y*e1.X)},
		),
		fmath.Vec2{X: dot(pq2, pq2), Y: s * (v2.X*e2.Y - v2.Y*e2.X)},
	)
	return -math.Sqrt(d.X) * sign(d.Y)
}

// EquilateralTriangle returns the signed distance from point p to an equilateral triangle
// with the given radius (circumradius).
func EquilateralTriangle(p fmath.Vec2, radius float64) float64 {
	const k = 1.732050808 // sqrt(3)
	p.X = math.Abs(p.X) - radius
	p.Y = p.Y + radius/k
	if p.X+k*p.Y > 0.0 {
		p = fmath.Vec2{X: (p.X - k*p.Y) / 2.0, Y: (-k*p.X - p.Y) / 2.0}
	}
	p.X -= fmath.Clamp(p.X, -2.0*radius, 0.0)
	return -p.Length() * sign(p.Y)
}

// Rhombus returns the signed distance from point p to a rhombus with half-extents b.
func Rhombus(p fmath.Vec2, b fmath.Vec2) float64 {
	b.Y = -b.Y
	p = abs2(p)
	h := fmath.Clamp((dot(b, p)+b.Y*b.Y)/dot(b, b), 0.0, 1.0)
	p = p.Sub(b.Scale(h).Sub(fmath.Vec2{Y: b.Y}))
	return p.Length() * sign(p.X)
}

// Pentagon returns the signed distance from point p to a regular pentagon
// with the given radius (circumradius).
func Pentagon(p fmath.Vec2, radius float64) float64 {
	const (
		kx = 0.809016994
		ky = 0.587785252
		kz = 0.726542528
	)
	k := fmath.Vec2{X: -kx, Y: ky}
	p.X = math.Abs(p.X)
	p = p.Sub(k.Scale(2.0 * math.Min(dot(k.Scale(-1), p), 0.0)))
	k.X = -k.X
	p = p.Sub(k.Scale(2.0 * math.Min(dot(k, p), 0.0)))
	p = p.Sub(fmath.Vec2{X: fmath.Clamp(p.X, -radius*kz, radius*kz), Y: radius})
	return p.Length() * sign(p.Y)
}

// Hexagon returns the signed distance from point p to a regular hexagon
// with the given radius (circumradius).
func Hexagon(p fmath.Vec2, radius float64) float64 {
	const (
		kx = -0.866025404
		ky = 0.5
		kz = 0.577350269
	)
	k := fmath.Vec2{X: kx, Y: ky}
	p = abs2(p)
	p = p.Sub(k.Scale(2.0 * math.Min(dot(k, p), 0.0)))
	p = p.Sub(fmath.Vec2{X: fmath.Clamp(p.X, -kz*radius, kz*radius), Y: radius})
	return p.Length() * sign(p.Y)
}

// Ellipse returns the signed distance from point p to an ellipse with semi-axes ab.
// ab.X is the semi-axis along the X direction, ab.Y is the semi-axis along the Y direction.
// This is an exact distance function using iterative root finding.
func Ellipse(p fmath.Vec2, ab fmath.Vec2) float64 {
	p = abs2(p)
	if p.X > p.Y {
		p = fmath.Vec2{X: p.Y, Y: p.X}
		ab = fmath.Vec2{X: ab.Y, Y: ab.X}
	}
	l := ab.Y*ab.Y - ab.X*ab.X
	m := ab.X * p.X / l
	m2 := m * m
	n := ab.Y * p.Y / l
	n2 := n * n
	c := (m2 + n2 - 1.0) / 3.0
	c3 := c * c * c
	q := c3 + m2*n2*2.0
	d := c3 + m2*n2
	g := m + m*n2
	var co float64

	if d < 0.0 {
		h := math.Acos(q/c3) / 3.0
		s := math.Cos(h)
		t := math.Sin(h) * math.Sqrt(3.0)
		rx := math.Sqrt(-c*(s+t+2.0) + m2)
		ry := math.Sqrt(-c*(s-t+2.0) + m2)
		co = (ry + sign(l)*rx + math.Abs(g)/(rx*ry) - m) / 2.0
	} else {
		h := 2.0 * m * n * math.Sqrt(d)
		s := sign(q+h) * math.Pow(math.Abs(q+h), 1.0/3.0)
		u := sign(q-h) * math.Pow(math.Abs(q-h), 1.0/3.0)
		rx := -s - u - c*4.0 + 2.0*m2
		ry := (s - u) * math.Sqrt(3.0)
		rm := math.Sqrt(rx*rx + ry*ry)
		co = (ry/math.Sqrt(rm-rx) + 2.0*g/rm - m) / 2.0
	}

	r := fmath.Vec2{X: ab.X * co, Y: ab.Y * math.Sqrt(1.0-co*co)}
	return p.Sub(r).Length() * sign(p.Y-r.Y)
}

// Arc returns the signed distance from point p to an arc.
// sc is the sin/cos of the arc's aperture (sc.X = cos, sc.Y = sin).
// ra is the outer radius, rb is the thickness.
func Arc(p fmath.Vec2, sc fmath.Vec2, ra, rb float64) float64 {
	p.X = math.Abs(p.X)
	var dist float64
	if sc.Y*p.X > sc.X*p.Y {
		dist = p.Sub(sc.Scale(ra)).Length()
	} else {
		dist = math.Abs(p.Length() - ra)
	}
	return dist - rb
}

// Pie returns the signed distance from point p to a pie/wedge shape.
// c is the sin/cos of the aperture (c.X = cos, c.Y = sin).
// r is the radius.
func Pie(p fmath.Vec2, c fmath.Vec2, r float64) float64 {
	p.X = math.Abs(p.X)
	l := p.Length() - r
	m := p.Sub(c.Scale(fmath.Clamp(dot(p, c), 0.0, r))).Length()
	return math.Max(l, m*sign(c.Y*p.X-c.X*p.Y))
}

// Operations

// Union returns the signed distance for the union of two shapes.
// This is the minimum distance to either shape.
func Union(d1, d2 float64) float64 {
	return math.Min(d1, d2)
}

// Subtract returns the signed distance for subtracting d2 from d1.
// This removes d2 from d1.
func Subtract(d1, d2 float64) float64 {
	return math.Max(d1, -d2)
}

// Intersect returns the signed distance for the intersection of two shapes.
// This is the region where both shapes overlap.
func Intersect(d1, d2 float64) float64 {
	return math.Max(d1, d2)
}

// SmoothUnion returns a smooth union of two shapes with the given smoothing factor k.
// Larger k values produce smoother blends.
func SmoothUnion(d1, d2, k float64) float64 {
	h := fmath.Clamp(0.5+0.5*(d2-d1)/k, 0.0, 1.0)
	return fmath.Lerp(d2, d1, h) - k*h*(1.0-h)
}

// SmoothSubtract returns a smooth subtraction of d2 from d1 with the given smoothing factor k.
func SmoothSubtract(d1, d2, k float64) float64 {
	h := fmath.Clamp(0.5-0.5*(d2+d1)/k, 0.0, 1.0)
	return fmath.Lerp(d1, -d2, h) + k*h*(1.0-h)
}

// SmoothIntersect returns a smooth intersection of two shapes with the given smoothing factor k.
func SmoothIntersect(d1, d2, k float64) float64 {
	h := fmath.Clamp(0.5-0.5*(d2-d1)/k, 0.0, 1.0)
	return fmath.Lerp(d2, d1, h) + k*h*(1.0-h)
}

// Helper functions

// dot returns the dot product of two Vec2 vectors.
func dot(a, b fmath.Vec2) float64 {
	return a.X*b.X + a.Y*b.Y
}

// abs2 returns a Vec2 with the absolute values of each component.
func abs2(v fmath.Vec2) fmath.Vec2 {
	return fmath.Vec2{X: math.Abs(v.X), Y: math.Abs(v.Y)}
}

// max2 returns a Vec2 with the maximum of each component with the corresponding component in other.
func max2(v, other fmath.Vec2) fmath.Vec2 {
	return fmath.Vec2{X: math.Max(v.X, other.X), Y: math.Max(v.Y, other.Y)}
}

// min2 returns a Vec2 with the minimum of each component with the corresponding component in other.
func min2(a, b fmath.Vec2) fmath.Vec2 {
	return fmath.Vec2{X: math.Min(a.X, b.X), Y: math.Min(a.Y, b.Y)}
}

// sign returns -1 for negative values, 1 for positive values, and 0 for zero.
func sign(x float64) float64 {
	if x < 0 {
		return -1
	}
	if x > 0 {
		return 1
	}
	return 0
}
