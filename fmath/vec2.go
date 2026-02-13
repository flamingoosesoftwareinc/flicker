package fmath

import "math"

type Vec2 struct {
	X, Y float64
}

func (v Vec2) Add(o Vec2) Vec2 {
	return Vec2{v.X + o.X, v.Y + o.Y}
}

func (v Vec2) Sub(o Vec2) Vec2 {
	return Vec2{v.X - o.X, v.Y - o.Y}
}

func (v Vec2) Scale(s float64) Vec2 {
	return Vec2{v.X * s, v.Y * s}
}

func (v Vec2) Length() float64 {
	return math.Sqrt(v.X*v.X + v.Y*v.Y)
}

func (v Vec2) Normalize() Vec2 {
	l := v.Length()
	if l == 0 {
		return Vec2{}
	}
	return Vec2{v.X / l, v.Y / l}
}

func (v Vec2) Lerp(to Vec2, t float64) Vec2 {
	return Vec2{
		X: v.X + (to.X-v.X)*t,
		Y: v.Y + (to.Y-v.Y)*t,
	}
}
