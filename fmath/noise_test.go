package fmath

import (
	"math"
	"testing"
)

func TestNoise2DRange(t *testing.T) {
	for i := range 1000 {
		x := float64(i) * 0.1
		y := float64(i) * 0.073
		v := Noise2D(x, y)
		if v < -1 || v > 1 {
			t.Errorf("Noise2D(%v, %v) = %v, out of [-1,1]", x, y, v)
		}
	}
}

func TestNoise2DDeterminism(t *testing.T) {
	for i := range 100 {
		x := float64(i) * 0.37
		y := float64(i) * 0.53
		a := Noise2D(x, y)
		b := Noise2D(x, y)
		if a != b {
			t.Errorf("Noise2D(%v, %v) not deterministic: %v != %v", x, y, a, b)
		}
	}
}

func TestNoise2DContinuity(t *testing.T) {
	// Nearby points should produce similar values.
	x, y := 5.0, 5.0
	base := Noise2D(x, y)
	delta := 0.001
	nearby := Noise2D(x+delta, y+delta)
	diff := math.Abs(base - nearby)
	if diff > 0.1 {
		t.Errorf("Noise2D not continuous: diff=%v between (%v,%v) and (%v,%v)",
			diff, x, y, x+delta, y+delta)
	}
}
