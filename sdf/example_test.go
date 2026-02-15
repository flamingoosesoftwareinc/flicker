package sdf_test

import (
	"fmt"

	"flicker/fmath"
	"flicker/sdf"
)

// Example demonstrates basic SDF primitive usage.
func ExampleCircle() {
	// Query point
	p := fmath.Vec2{X: 1.5, Y: 0}

	// Circle with radius 1.0 centered at origin
	distance := sdf.Circle(p, 1.0)

	// Positive distance means outside
	fmt.Printf("Distance: %.1f (outside)\n", distance)
	// Output: Distance: 0.5 (outside)
}

// Example demonstrates combining shapes with boolean operations.
func ExampleUnion() {
	p := fmath.Vec2{X: 0.5, Y: 0}

	// Two circles
	circle1 := sdf.Circle(p, 1.0)                             // Centered at origin
	circle2 := sdf.Circle(p.Sub(fmath.Vec2{X: 2, Y: 0}), 1.0) // Centered at (2,0)

	// Union combines both shapes
	unionDist := sdf.Union(circle1, circle2)

	fmt.Printf("Distance to union: %.1f\n", unionDist)
	// Output: Distance to union: -0.5
}

// Example demonstrates creating a composite shape using operations.
func ExampleSubtract() {
	p := fmath.Vec2{X: 0, Y: 0}

	// Create a box
	box := sdf.Box(p, fmath.Vec2{X: 2, Y: 2})

	// Create a circle to subtract
	circle := sdf.Circle(p, 1.0)

	// Subtract circle from box (box with circular hole)
	result := sdf.Subtract(box, circle)

	fmt.Printf("Distance: %.1f\n", result)
	// Output: Distance: 1.0
}

// Example demonstrates smooth blending between shapes.
func ExampleSmoothUnion() {
	p := fmath.Vec2{X: 1, Y: 0}

	// Two circles
	circle1 := sdf.Circle(p, 1.0)
	circle2 := sdf.Circle(p.Sub(fmath.Vec2{X: 2, Y: 0}), 1.0)

	// Smooth union with blending factor 0.5
	smoothDist := sdf.SmoothUnion(circle1, circle2, 0.5)

	// Regular union for comparison
	regularDist := sdf.Union(circle1, circle2)

	// Smooth union creates a blend, resulting in a slightly different distance
	fmt.Printf("Regular: %.1f, Smooth: %.3f\n", regularDist, smoothDist)
	// Output: Regular: 0.0, Smooth: -0.125
}

// Example demonstrates creating complex shapes with multiple primitives.
func ExampleTriangle() {
	// Define triangle vertices
	p0 := fmath.Vec2{X: 0, Y: 1}
	p1 := fmath.Vec2{X: -1, Y: -1}
	p2 := fmath.Vec2{X: 1, Y: -1}

	// Query point at origin
	p := fmath.Vec2{X: 0, Y: 0}

	distance := sdf.Triangle(p, p0, p1, p2)

	// Negative distance means inside
	fmt.Printf("Distance: %.2f (inside)\n", distance)
	// Output: Distance: -0.45 (inside)
}

// Example demonstrates transforming shapes by transforming the query point.
func ExampleBox() {
	// Original point
	p := fmath.Vec2{X: 3, Y: 1}

	// To translate a shape, subtract the translation from the query point
	// This moves the box to be centered at (2, 0) instead of origin
	translation := fmath.Vec2{X: 2, Y: 0}
	transformedP := p.Sub(translation)

	distance := sdf.Box(transformedP, fmath.Vec2{X: 1, Y: 1})

	fmt.Printf("Distance: %.1f\n", distance)
	// Output: Distance: 0.0
}
