package fmath

import "math"

// frac returns the fractional part of t, wrapped to [0, 1).
// Handles negative values correctly: frac(-0.3) == 0.7.
func frac(t float64) float64 {
	t = math.Mod(t, 1.0)
	if t < 0 {
		t += 1.0
	}
	return t
}

// Saw returns a sawtooth ramp in [0, 1) with period 1.
func Saw(t float64) float64 {
	return frac(t)
}

// Sine returns a sine wave in [-1, 1] with period 1.
func Sine(t float64) float64 {
	return math.Sin(2 * math.Pi * t)
}

// Triangle returns a triangle wave in [0, 1] with period 1.
func Triangle(t float64) float64 {
	f := frac(t)
	if f < 0.5 {
		return 2 * f
	}
	return 2 * (1 - f)
}

// Square returns a square wave (0 or 1) with period 1 and 50% duty cycle.
func Square(t float64) float64 {
	return Pulse(t, 0.5)
}

// Pulse returns a pulse wave (0 or 1) with period 1 and configurable duty cycle.
// duty is the fraction of the period spent at 1, in [0, 1].
func Pulse(t, duty float64) float64 {
	if frac(t) < duty {
		return 1
	}
	return 0
}
