package fmath

import "math"

func EaseLinear(t float64) float64 {
	return t
}

func EaseInQuad(t float64) float64 {
	return t * t
}

func EaseOutQuad(t float64) float64 {
	return 1 - (1-t)*(1-t)
}

func EaseInOutQuad(t float64) float64 {
	if t < 0.5 {
		return 2 * t * t
	}
	return 1 - (-2*t+2)*(-2*t+2)/2
}

func EaseInCubic(t float64) float64 {
	return t * t * t
}

func EaseOutCubic(t float64) float64 {
	return 1 - (1-t)*(1-t)*(1-t)
}

func EaseInOutCubic(t float64) float64 {
	if t < 0.5 {
		return 4 * t * t * t
	}
	return 1 - (-2*t+2)*(-2*t+2)*(-2*t+2)/2
}

func EaseInElastic(t float64) float64 {
	if t == 0 || t == 1 {
		return t
	}
	return -math.Pow(2, 10*(t-1)) * math.Sin((t-1.1)*5*math.Pi)
}

func EaseOutElastic(t float64) float64 {
	if t == 0 || t == 1 {
		return t
	}
	return math.Pow(2, -10*t)*math.Sin((t-0.1)*5*math.Pi) + 1
}

func EaseOutBounce(t float64) float64 {
	switch {
	case t < 1/2.75:
		return 7.5625 * t * t
	case t < 2/2.75:
		t -= 1.5 / 2.75
		return 7.5625*t*t + 0.75
	case t < 2.5/2.75:
		t -= 2.25 / 2.75
		return 7.5625*t*t + 0.9375
	default:
		t -= 2.625 / 2.75
		return 7.5625*t*t + 0.984375
	}
}
