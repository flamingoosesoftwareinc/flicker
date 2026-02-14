package fmath

import (
	"math"
	"testing"
)

func TestDegToRad(t *testing.T) {
	tests := []struct {
		deg, want float64
	}{
		{0, 0},
		{90, math.Pi / 2},
		{180, math.Pi},
		{360, 2 * math.Pi},
		{-90, -math.Pi / 2},
		{45, math.Pi / 4},
	}
	for _, tc := range tests {
		got := DegToRad(tc.deg)
		if !approxEqual(got, tc.want) {
			t.Errorf("DegToRad(%v) = %v, want %v", tc.deg, got, tc.want)
		}
	}
}

func TestRadToDeg(t *testing.T) {
	tests := []struct {
		rad, want float64
	}{
		{0, 0},
		{math.Pi / 2, 90},
		{math.Pi, 180},
		{2 * math.Pi, 360},
		{-math.Pi / 2, -90},
	}
	for _, tc := range tests {
		got := RadToDeg(tc.rad)
		if !approxEqual(got, tc.want) {
			t.Errorf("RadToDeg(%v) = %v, want %v", tc.rad, got, tc.want)
		}
	}
}

func TestAngleRoundtrip(t *testing.T) {
	for _, deg := range []float64{0, 45, 90, 135, 180, 270, 360, -45} {
		got := RadToDeg(DegToRad(deg))
		if !approxEqual(got, deg) {
			t.Errorf("roundtrip(%v) = %v", deg, got)
		}
	}
}
