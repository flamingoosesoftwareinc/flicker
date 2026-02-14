package fmath

import (
	"math"
	"testing"
)

func TestBezierQuadraticEndpoints(t *testing.T) {
	p0 := Vec2{X: 0, Y: 0}
	p1 := Vec2{X: 5, Y: 10}
	p2 := Vec2{X: 10, Y: 0}

	got0 := BezierQuadratic(p0, p1, p2, 0)
	if !approxEqual(got0.X, p0.X) || !approxEqual(got0.Y, p0.Y) {
		t.Errorf("t=0: %v, want %v", got0, p0)
	}

	got1 := BezierQuadratic(p0, p1, p2, 1)
	if !approxEqual(got1.X, p2.X) || !approxEqual(got1.Y, p2.Y) {
		t.Errorf("t=1: %v, want %v", got1, p2)
	}
}

func TestBezierQuadraticMidpoint(t *testing.T) {
	p0 := Vec2{X: 0, Y: 0}
	p1 := Vec2{X: 5, Y: 10}
	p2 := Vec2{X: 10, Y: 0}
	// At t=0.5: x = 0.25*0 + 0.5*5 + 0.25*10 = 5
	//           y = 0.25*0 + 0.5*10 + 0.25*0 = 5
	got := BezierQuadratic(p0, p1, p2, 0.5)
	if !approxEqual(got.X, 5) || !approxEqual(got.Y, 5) {
		t.Errorf("t=0.5: %v, want (5,5)", got)
	}
}

func TestBezierCubicEndpoints(t *testing.T) {
	p0 := Vec2{X: 0, Y: 0}
	p1 := Vec2{X: 1, Y: 2}
	p2 := Vec2{X: 3, Y: 2}
	p3 := Vec2{X: 4, Y: 0}

	got0 := BezierCubic(p0, p1, p2, p3, 0)
	if !approxEqual(got0.X, p0.X) || !approxEqual(got0.Y, p0.Y) {
		t.Errorf("t=0: %v, want %v", got0, p0)
	}

	got1 := BezierCubic(p0, p1, p2, p3, 1)
	if !approxEqual(got1.X, p3.X) || !approxEqual(got1.Y, p3.Y) {
		t.Errorf("t=1: %v, want %v", got1, p3)
	}
}

func TestBezierCubicMidpoint(t *testing.T) {
	// Symmetric curve: midpoint should be at x=2
	p0 := Vec2{X: 0, Y: 0}
	p1 := Vec2{X: 1, Y: 2}
	p2 := Vec2{X: 3, Y: 2}
	p3 := Vec2{X: 4, Y: 0}
	got := BezierCubic(p0, p1, p2, p3, 0.5)
	if !approxEqual(got.X, 2) {
		t.Errorf("t=0.5: X=%v, want 2", got.X)
	}
}

func TestBezierCubicContinuity(t *testing.T) {
	p0 := Vec2{X: 0, Y: 0}
	p1 := Vec2{X: 1, Y: 3}
	p2 := Vec2{X: 3, Y: 3}
	p3 := Vec2{X: 4, Y: 0}

	prev := BezierCubic(p0, p1, p2, p3, 0)
	for i := 1; i <= 100; i++ {
		s := float64(i) / 100.0
		cur := BezierCubic(p0, p1, p2, p3, s)
		dist := math.Sqrt((cur.X-prev.X)*(cur.X-prev.X) + (cur.Y-prev.Y)*(cur.Y-prev.Y))
		if dist > 0.2 {
			t.Errorf("discontinuity at t=%v: dist=%v", s, dist)
		}
		prev = cur
	}
}
