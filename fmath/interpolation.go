package fmath

func Lerp(a, b, t float64) float64 {
	return a + (b-a)*t
}

func InverseLerp(a, b, v float64) float64 {
	if a == b {
		return 0
	}
	return (v - a) / (b - a)
}

func Remap(inMin, inMax, outMin, outMax, v float64) float64 {
	t := InverseLerp(inMin, inMax, v)
	return Lerp(outMin, outMax, t)
}
