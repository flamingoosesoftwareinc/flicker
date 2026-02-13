package fmath

import (
	"math"
	"testing"
)

const epsilon = 1e-9

func approxEqual(a, b float64) bool {
	return math.Abs(a-b) < epsilon
}

func TestFrac(t *testing.T) {
	tests := []struct {
		in, want float64
	}{
		{0.0, 0.0},
		{0.25, 0.25},
		{0.99, 0.99},
		{1.0, 0.0},
		{1.75, 0.75},
		{-0.3, 0.7},
		{-1.0, 0.0},
		{-1.25, 0.75},
		{3.5, 0.5},
	}
	for _, tc := range tests {
		got := frac(tc.in)
		if !approxEqual(got, tc.want) {
			t.Errorf("frac(%v) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestSaw(t *testing.T) {
	tests := []struct {
		in, want float64
	}{
		{0.0, 0.0},
		{0.25, 0.25},
		{0.5, 0.5},
		{0.75, 0.75},
		{1.0, 0.0},
		{-0.25, 0.75},
	}
	for _, tc := range tests {
		got := Saw(tc.in)
		if !approxEqual(got, tc.want) {
			t.Errorf("Saw(%v) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestSine(t *testing.T) {
	tests := []struct {
		in, want float64
	}{
		{0.0, 0.0},
		{0.25, 1.0},
		{0.5, 0.0},
		{0.75, -1.0},
		{1.0, 0.0},
	}
	for _, tc := range tests {
		got := Sine(tc.in)
		if !approxEqual(got, tc.want) {
			t.Errorf("Sine(%v) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestTriangle(t *testing.T) {
	tests := []struct {
		in, want float64
	}{
		{0.0, 0.0},
		{0.25, 0.5},
		{0.5, 1.0},
		{0.75, 0.5},
		{1.0, 0.0},
		{-0.25, 0.5},
	}
	for _, tc := range tests {
		got := Triangle(tc.in)
		if !approxEqual(got, tc.want) {
			t.Errorf("Triangle(%v) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestSquare(t *testing.T) {
	tests := []struct {
		in, want float64
	}{
		{0.0, 1.0},
		{0.25, 1.0},
		{0.5, 0.0},
		{0.75, 0.0},
		{1.0, 1.0},
	}
	for _, tc := range tests {
		got := Square(tc.in)
		if !approxEqual(got, tc.want) {
			t.Errorf("Square(%v) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestPulse(t *testing.T) {
	tests := []struct {
		in, duty, want float64
	}{
		{0.0, 0.25, 1.0},
		{0.1, 0.25, 1.0},
		{0.25, 0.25, 0.0},
		{0.5, 0.25, 0.0},
		{0.0, 0.75, 1.0},
		{0.5, 0.75, 1.0},
		{0.75, 0.75, 0.0},
		{0.0, 0.0, 0.0},
		{0.0, 1.0, 1.0},
		{0.99, 1.0, 1.0},
	}
	for _, tc := range tests {
		got := Pulse(tc.in, tc.duty)
		if !approxEqual(got, tc.want) {
			t.Errorf("Pulse(%v, %v) = %v, want %v", tc.in, tc.duty, got, tc.want)
		}
	}
}
