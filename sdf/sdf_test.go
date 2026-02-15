package sdf

import (
	"math"
	"testing"

	"flicker/fmath"
)

const epsilon = 1e-9

func approxEqual(a, b float64) bool {
	return math.Abs(a-b) < epsilon
}

// Test Circle primitive
func TestCircle(t *testing.T) {
	tests := []struct {
		name     string
		p        fmath.Vec2
		radius   float64
		expected float64
	}{
		{"origin", fmath.Vec2{X: 0, Y: 0}, 1.0, -1.0},
		{"on boundary", fmath.Vec2{X: 1, Y: 0}, 1.0, 0.0},
		{"outside", fmath.Vec2{X: 2, Y: 0}, 1.0, 1.0},
		{"inside", fmath.Vec2{X: 0.5, Y: 0}, 1.0, -0.5},
		{"diagonal on boundary", fmath.Vec2{X: 0.707106781, Y: 0.707106781}, 1.0, 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Circle(tt.p, tt.radius)
			if !approxEqual(result, tt.expected) {
				t.Errorf("Circle(%v, %v) = %v, want %v", tt.p, tt.radius, result, tt.expected)
			}
		})
	}
}

// Test Box primitive
func TestBox(t *testing.T) {
	tests := []struct {
		name     string
		p        fmath.Vec2
		size     fmath.Vec2
		expected float64
	}{
		{"origin", fmath.Vec2{X: 0, Y: 0}, fmath.Vec2{X: 1, Y: 1}, -1.0},
		{"on edge", fmath.Vec2{X: 1, Y: 0}, fmath.Vec2{X: 1, Y: 1}, 0.0},
		{"on corner", fmath.Vec2{X: 1, Y: 1}, fmath.Vec2{X: 1, Y: 1}, 0.0},
		{"outside", fmath.Vec2{X: 2, Y: 0}, fmath.Vec2{X: 1, Y: 1}, 1.0},
		{"inside", fmath.Vec2{X: 0.5, Y: 0.5}, fmath.Vec2{X: 1, Y: 1}, -0.5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Box(tt.p, tt.size)
			if !approxEqual(result, tt.expected) {
				t.Errorf("Box(%v, %v) = %v, want %v", tt.p, tt.size, result, tt.expected)
			}
		})
	}
}

// Test RoundedBox primitive
func TestRoundedBox(t *testing.T) {
	tests := []struct {
		name         string
		p            fmath.Vec2
		size         fmath.Vec2
		cornerRadius float64
		checkSign    bool // just check if sign is correct
	}{
		{"origin inside", fmath.Vec2{X: 0, Y: 0}, fmath.Vec2{X: 1, Y: 1}, 0.2, true},
		{"outside", fmath.Vec2{X: 2, Y: 2}, fmath.Vec2{X: 1, Y: 1}, 0.2, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RoundedBox(tt.p, tt.size, tt.cornerRadius)
			if tt.checkSign {
				if tt.name == "origin inside" && result >= 0 {
					t.Errorf(
						"RoundedBox(%v, %v, %v) = %v, expected negative (inside)",
						tt.p,
						tt.size,
						tt.cornerRadius,
						result,
					)
				}
				if tt.name == "outside" && result <= 0 {
					t.Errorf(
						"RoundedBox(%v, %v, %v) = %v, expected positive (outside)",
						tt.p,
						tt.size,
						tt.cornerRadius,
						result,
					)
				}
			}
		})
	}
}

// Test Segment primitive
func TestSegment(t *testing.T) {
	tests := []struct {
		name     string
		p        fmath.Vec2
		a        fmath.Vec2
		b        fmath.Vec2
		expected float64
	}{
		{
			"on start point",
			fmath.Vec2{X: 0, Y: 0},
			fmath.Vec2{X: 0, Y: 0},
			fmath.Vec2{X: 1, Y: 0},
			0.0,
		},
		{
			"on end point",
			fmath.Vec2{X: 1, Y: 0},
			fmath.Vec2{X: 0, Y: 0},
			fmath.Vec2{X: 1, Y: 0},
			0.0,
		},
		{
			"on midpoint",
			fmath.Vec2{X: 0.5, Y: 0},
			fmath.Vec2{X: 0, Y: 0},
			fmath.Vec2{X: 1, Y: 0},
			0.0,
		},
		{
			"perpendicular distance",
			fmath.Vec2{X: 0.5, Y: 1},
			fmath.Vec2{X: 0, Y: 0},
			fmath.Vec2{X: 1, Y: 0},
			1.0,
		},
		{
			"beyond start",
			fmath.Vec2{X: -1, Y: 0},
			fmath.Vec2{X: 0, Y: 0},
			fmath.Vec2{X: 1, Y: 0},
			1.0,
		},
		{"beyond end", fmath.Vec2{X: 2, Y: 0}, fmath.Vec2{X: 0, Y: 0}, fmath.Vec2{X: 1, Y: 0}, 1.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Segment(tt.p, tt.a, tt.b)
			if !approxEqual(result, tt.expected) {
				t.Errorf("Segment(%v, %v, %v) = %v, want %v", tt.p, tt.a, tt.b, result, tt.expected)
			}
		})
	}
}

// Test Triangle primitive
func TestTriangle(t *testing.T) {
	// Equilateral triangle centered at origin
	p0 := fmath.Vec2{X: 0, Y: 1}
	p1 := fmath.Vec2{X: -0.866025404, Y: -0.5}
	p2 := fmath.Vec2{X: 0.866025404, Y: -0.5}

	tests := []struct {
		name      string
		p         fmath.Vec2
		checkSign bool
	}{
		{"origin inside", fmath.Vec2{X: 0, Y: 0}, true},
		{"on vertex", p0, true},
		{"outside", fmath.Vec2{X: 0, Y: 2}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Triangle(tt.p, p0, p1, p2)
			if tt.checkSign {
				if tt.name == "origin inside" && result >= 0 {
					t.Errorf("Triangle(%v) = %v, expected negative (inside)", tt.p, result)
				}
				if tt.name == "outside" && result <= 0 {
					t.Errorf("Triangle(%v) = %v, expected positive (outside)", tt.p, result)
				}
			}
		})
	}
}

// Test EquilateralTriangle primitive
func TestEquilateralTriangle(t *testing.T) {
	tests := []struct {
		name      string
		p         fmath.Vec2
		radius    float64
		checkSign bool
	}{
		{"origin inside", fmath.Vec2{X: 0, Y: 0}, 1.0, true},
		{"outside above", fmath.Vec2{X: 0, Y: 2}, 1.0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EquilateralTriangle(tt.p, tt.radius)
			if tt.checkSign {
				if tt.name == "origin inside" && result >= 0 {
					t.Errorf(
						"EquilateralTriangle(%v, %v) = %v, expected negative (inside)",
						tt.p,
						tt.radius,
						result,
					)
				}
				if tt.name == "outside above" && result <= 0 {
					t.Errorf(
						"EquilateralTriangle(%v, %v) = %v, expected positive (outside)",
						tt.p,
						tt.radius,
						result,
					)
				}
			}
		})
	}
}

// Test Rhombus primitive
func TestRhombus(t *testing.T) {
	tests := []struct {
		name      string
		p         fmath.Vec2
		b         fmath.Vec2
		checkSign bool
	}{
		{"origin inside", fmath.Vec2{X: 0, Y: 0}, fmath.Vec2{X: 1, Y: 1}, true},
		{"outside", fmath.Vec2{X: 2, Y: 2}, fmath.Vec2{X: 1, Y: 1}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Rhombus(tt.p, tt.b)
			if tt.checkSign {
				if tt.name == "origin inside" && result >= 0 {
					t.Errorf("Rhombus(%v, %v) = %v, expected negative (inside)", tt.p, tt.b, result)
				}
				if tt.name == "outside" && result <= 0 {
					t.Errorf(
						"Rhombus(%v, %v) = %v, expected positive (outside)",
						tt.p,
						tt.b,
						result,
					)
				}
			}
		})
	}
}

// Test Pentagon primitive
func TestPentagon(t *testing.T) {
	tests := []struct {
		name      string
		p         fmath.Vec2
		radius    float64
		checkSign bool
	}{
		{"origin inside", fmath.Vec2{X: 0, Y: 0}, 1.0, true},
		{"outside", fmath.Vec2{X: 0, Y: 2}, 1.0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Pentagon(tt.p, tt.radius)
			if tt.checkSign {
				if tt.name == "origin inside" && result >= 0 {
					t.Errorf(
						"Pentagon(%v, %v) = %v, expected negative (inside)",
						tt.p,
						tt.radius,
						result,
					)
				}
				if tt.name == "outside" && result <= 0 {
					t.Errorf(
						"Pentagon(%v, %v) = %v, expected positive (outside)",
						tt.p,
						tt.radius,
						result,
					)
				}
			}
		})
	}
}

// Test Hexagon primitive
func TestHexagon(t *testing.T) {
	tests := []struct {
		name      string
		p         fmath.Vec2
		radius    float64
		checkSign bool
	}{
		{"origin inside", fmath.Vec2{X: 0, Y: 0}, 1.0, true},
		{"outside", fmath.Vec2{X: 0, Y: 2}, 1.0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Hexagon(tt.p, tt.radius)
			if tt.checkSign {
				if tt.name == "origin inside" && result >= 0 {
					t.Errorf(
						"Hexagon(%v, %v) = %v, expected negative (inside)",
						tt.p,
						tt.radius,
						result,
					)
				}
				if tt.name == "outside" && result <= 0 {
					t.Errorf(
						"Hexagon(%v, %v) = %v, expected positive (outside)",
						tt.p,
						tt.radius,
						result,
					)
				}
			}
		})
	}
}

// Test Ellipse primitive
func TestEllipse(t *testing.T) {
	tests := []struct {
		name      string
		p         fmath.Vec2
		ab        fmath.Vec2
		checkSign bool
	}{
		{"origin inside", fmath.Vec2{X: 0, Y: 0}, fmath.Vec2{X: 2, Y: 1}, true},
		{"on boundary X", fmath.Vec2{X: 2, Y: 0}, fmath.Vec2{X: 2, Y: 1}, true},
		{"on boundary Y", fmath.Vec2{X: 0, Y: 1}, fmath.Vec2{X: 2, Y: 1}, true},
		{"outside", fmath.Vec2{X: 3, Y: 2}, fmath.Vec2{X: 2, Y: 1}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Ellipse(tt.p, tt.ab)
			if tt.checkSign {
				if tt.name == "origin inside" && result >= 0 {
					t.Errorf(
						"Ellipse(%v, %v) = %v, expected negative (inside)",
						tt.p,
						tt.ab,
						result,
					)
				}
				if tt.name == "outside" && result <= 0 {
					t.Errorf(
						"Ellipse(%v, %v) = %v, expected positive (outside)",
						tt.p,
						tt.ab,
						result,
					)
				}
				// For "on boundary" cases, check that the result is close to 0
				if (tt.name == "on boundary X" || tt.name == "on boundary Y") &&
					math.Abs(result) > 0.01 {
					t.Errorf(
						"Ellipse(%v, %v) = %v, expected close to 0 (on boundary)",
						tt.p,
						tt.ab,
						result,
					)
				}
			}
		})
	}
}

// Test Arc primitive
func TestArc(t *testing.T) {
	// sc represents sin/cos of 45 degrees
	sc := fmath.Vec2{X: 0.707106781, Y: 0.707106781}
	tests := []struct {
		name string
		p    fmath.Vec2
		sc   fmath.Vec2
		ra   float64
		rb   float64
	}{
		{"basic arc", fmath.Vec2{X: 1, Y: 0}, sc, 1.0, 0.1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Arc(tt.p, tt.sc, tt.ra, tt.rb)
			// Just verify it returns a finite value
			if math.IsNaN(result) || math.IsInf(result, 0) {
				t.Errorf("Arc returned invalid value: %v", result)
			}
		})
	}
}

// Test Pie primitive
func TestPie(t *testing.T) {
	// c represents sin/cos of 45 degrees
	c := fmath.Vec2{X: 0.707106781, Y: 0.707106781}
	tests := []struct {
		name string
		p    fmath.Vec2
		c    fmath.Vec2
		r    float64
	}{
		{"basic pie", fmath.Vec2{X: 0.3, Y: 0.3}, c, 1.0},
		{"outside radius", fmath.Vec2{X: 2, Y: 0}, c, 1.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Pie(tt.p, tt.c, tt.r)
			// Just verify it returns a finite value
			if math.IsNaN(result) || math.IsInf(result, 0) {
				t.Errorf("Pie returned invalid value: %v", result)
			}
		})
	}
}

// Test Union operation
func TestUnion(t *testing.T) {
	tests := []struct {
		name     string
		d1       float64
		d2       float64
		expected float64
	}{
		{"both positive", 1.0, 2.0, 1.0},
		{"both negative", -1.0, -2.0, -2.0},
		{"mixed", -1.0, 2.0, -1.0},
		{"equal", 1.0, 1.0, 1.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Union(tt.d1, tt.d2)
			if !approxEqual(result, tt.expected) {
				t.Errorf("Union(%v, %v) = %v, want %v", tt.d1, tt.d2, result, tt.expected)
			}
		})
	}
}

// Test Subtract operation
func TestSubtract(t *testing.T) {
	tests := []struct {
		name     string
		d1       float64
		d2       float64
		expected float64
	}{
		{"remove from positive", 1.0, -2.0, 2.0},
		{"no overlap", 2.0, 3.0, 2.0},
		{"complete removal", -1.0, -2.0, 2.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Subtract(tt.d1, tt.d2)
			if !approxEqual(result, tt.expected) {
				t.Errorf("Subtract(%v, %v) = %v, want %v", tt.d1, tt.d2, result, tt.expected)
			}
		})
	}
}

// Test Intersect operation
func TestIntersect(t *testing.T) {
	tests := []struct {
		name     string
		d1       float64
		d2       float64
		expected float64
	}{
		{"both positive", 1.0, 2.0, 2.0},
		{"both negative", -1.0, -2.0, -1.0},
		{"mixed", -1.0, 2.0, 2.0},
		{"equal", 1.0, 1.0, 1.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Intersect(tt.d1, tt.d2)
			if !approxEqual(result, tt.expected) {
				t.Errorf("Intersect(%v, %v) = %v, want %v", tt.d1, tt.d2, result, tt.expected)
			}
		})
	}
}

// Test SmoothUnion operation
func TestSmoothUnion(t *testing.T) {
	tests := []struct {
		name string
		d1   float64
		d2   float64
		k    float64
	}{
		{"basic smooth union", 1.0, 2.0, 0.5},
		{"equal distances", 1.0, 1.0, 0.5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SmoothUnion(tt.d1, tt.d2, tt.k)
			// Result should be less than or equal to the minimum (smooth union blends inward)
			minVal := math.Min(tt.d1, tt.d2)
			if result > minVal+epsilon {
				t.Errorf("SmoothUnion(%v, %v, %v) = %v, expected <= %v",
					tt.d1, tt.d2, tt.k, result, minVal)
			}
			// Also verify it's not too far below (sanity check)
			if result < minVal-tt.k {
				t.Errorf("SmoothUnion(%v, %v, %v) = %v, expected >= %v",
					tt.d1, tt.d2, tt.k, result, minVal-tt.k)
			}
		})
	}
}

// Test SmoothSubtract operation
func TestSmoothSubtract(t *testing.T) {
	result := SmoothSubtract(1.0, -2.0, 0.5)
	// Just verify it returns a reasonable value
	if math.IsNaN(result) || math.IsInf(result, 0) {
		t.Errorf("SmoothSubtract returned invalid value: %v", result)
	}
}

// Test SmoothIntersect operation
func TestSmoothIntersect(t *testing.T) {
	result := SmoothIntersect(1.0, 2.0, 0.5)
	// Just verify it returns a reasonable value
	if math.IsNaN(result) || math.IsInf(result, 0) {
		t.Errorf("SmoothIntersect returned invalid value: %v", result)
	}
}

// Test helper functions
func TestDot(t *testing.T) {
	tests := []struct {
		name     string
		a        fmath.Vec2
		b        fmath.Vec2
		expected float64
	}{
		{"perpendicular", fmath.Vec2{X: 1, Y: 0}, fmath.Vec2{X: 0, Y: 1}, 0.0},
		{"parallel", fmath.Vec2{X: 1, Y: 0}, fmath.Vec2{X: 2, Y: 0}, 2.0},
		{"general", fmath.Vec2{X: 1, Y: 2}, fmath.Vec2{X: 3, Y: 4}, 11.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := dot(tt.a, tt.b)
			if !approxEqual(result, tt.expected) {
				t.Errorf("dot(%v, %v) = %v, want %v", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}

func TestAbs2(t *testing.T) {
	tests := []struct {
		name     string
		v        fmath.Vec2
		expected fmath.Vec2
	}{
		{"all positive", fmath.Vec2{X: 1, Y: 2}, fmath.Vec2{X: 1, Y: 2}},
		{"all negative", fmath.Vec2{X: -1, Y: -2}, fmath.Vec2{X: 1, Y: 2}},
		{"mixed", fmath.Vec2{X: -1, Y: 2}, fmath.Vec2{X: 1, Y: 2}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := abs2(tt.v)
			if !approxEqual(result.X, tt.expected.X) || !approxEqual(result.Y, tt.expected.Y) {
				t.Errorf("abs2(%v) = %v, want %v", tt.v, result, tt.expected)
			}
		})
	}
}

func TestMax2(t *testing.T) {
	tests := []struct {
		name     string
		a        fmath.Vec2
		b        fmath.Vec2
		expected fmath.Vec2
	}{
		{"first larger", fmath.Vec2{X: 2, Y: 3}, fmath.Vec2{X: 1, Y: 2}, fmath.Vec2{X: 2, Y: 3}},
		{"second larger", fmath.Vec2{X: 1, Y: 2}, fmath.Vec2{X: 2, Y: 3}, fmath.Vec2{X: 2, Y: 3}},
		{"mixed", fmath.Vec2{X: 1, Y: 3}, fmath.Vec2{X: 2, Y: 2}, fmath.Vec2{X: 2, Y: 3}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := max2(tt.a, tt.b)
			if !approxEqual(result.X, tt.expected.X) || !approxEqual(result.Y, tt.expected.Y) {
				t.Errorf("max2(%v, %v) = %v, want %v", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}

func TestMin2(t *testing.T) {
	tests := []struct {
		name     string
		a        fmath.Vec2
		b        fmath.Vec2
		expected fmath.Vec2
	}{
		{"first smaller", fmath.Vec2{X: 1, Y: 2}, fmath.Vec2{X: 2, Y: 3}, fmath.Vec2{X: 1, Y: 2}},
		{"second smaller", fmath.Vec2{X: 2, Y: 3}, fmath.Vec2{X: 1, Y: 2}, fmath.Vec2{X: 1, Y: 2}},
		{"mixed", fmath.Vec2{X: 1, Y: 3}, fmath.Vec2{X: 2, Y: 2}, fmath.Vec2{X: 1, Y: 2}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := min2(tt.a, tt.b)
			if !approxEqual(result.X, tt.expected.X) || !approxEqual(result.Y, tt.expected.Y) {
				t.Errorf("min2(%v, %v) = %v, want %v", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}

func TestSign(t *testing.T) {
	tests := []struct {
		name     string
		x        float64
		expected float64
	}{
		{"positive", 5.0, 1.0},
		{"negative", -5.0, -1.0},
		{"zero", 0.0, 0.0},
		{"small positive", 0.0001, 1.0},
		{"small negative", -0.0001, -1.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sign(tt.x)
			if result != tt.expected {
				t.Errorf("sign(%v) = %v, want %v", tt.x, result, tt.expected)
			}
		})
	}
}
