package fmath

func BezierQuadratic(p0, p1, p2 Vec2, t float64) Vec2 {
	u := 1 - t
	return Vec2{
		X: u*u*p0.X + 2*u*t*p1.X + t*t*p2.X,
		Y: u*u*p0.Y + 2*u*t*p1.Y + t*t*p2.Y,
	}
}

func BezierCubic(p0, p1, p2, p3 Vec2, t float64) Vec2 {
	u := 1 - t
	u2 := u * u
	t2 := t * t
	return Vec2{
		X: u2*u*p0.X + 3*u2*t*p1.X + 3*u*t2*p2.X + t2*t*p3.X,
		Y: u2*u*p0.Y + 3*u2*t*p1.Y + 3*u*t2*p2.Y + t2*t*p3.Y,
	}
}
