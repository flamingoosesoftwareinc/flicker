package core

import "math"

// BlendMode computes a blended channel value from dst and src channel values
// at the given alpha. Both d and s are in [0,255]; alpha is in [0,1].
type BlendMode func(d, s uint8, alpha float64) uint8

// ColorBlend blends two colors at a given alpha. BlendCell and
// Canvas.Composite accept this type so callers control color mixing.
type ColorBlend func(dst, src Color, alpha float64) Color

// BlendColor applies a BlendMode per-channel to produce a blended color.
func BlendColor(dst, src Color, alpha float64, mode BlendMode) Color {
	a := alpha
	if a < 0 {
		a = 0
	}
	if a > 1 {
		a = 1
	}
	return Color{
		R: mode(dst.R, src.R, a),
		G: mode(dst.G, src.G, a),
		B: mode(dst.B, src.B, a),
	}
}

// clamp01 clamps v to [0, 1].
func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

// blendLerp normalizes d,s to [0,1], applies the raw blend function,
// lerps between d and the result by alpha, and converts back to uint8.
func blendLerp(d, s uint8, alpha float64, fn func(d, s float64) float64) uint8 {
	df, sf := float64(d)/255, float64(s)/255
	raw := fn(df, sf)
	result := df*(1-alpha) + raw*alpha
	return uint8(clamp01(result)*255 + 0.5)
}

// --- Normal ---

// BlendNormal is the standard linear interpolation: dst*(1-a) + src*a.
func BlendNormal(d, s uint8, alpha float64) uint8 {
	return uint8(float64(d)*(1-alpha) + float64(s)*alpha + 0.5)
}

// NormalColorBlend applies BlendNormal per-channel.
func NormalColorBlend(dst, src Color, alpha float64) Color {
	return BlendColor(dst, src, alpha, BlendNormal)
}

// --- Multiply (Darken) ---

// BlendMultiply: d * s
func BlendMultiply(d, s uint8, alpha float64) uint8 {
	return blendLerp(d, s, alpha, func(d, s float64) float64 {
		return d * s
	})
}

// MultiplyColorBlend applies BlendMultiply per-channel.
func MultiplyColorBlend(dst, src Color, alpha float64) Color {
	return BlendColor(dst, src, alpha, BlendMultiply)
}

// --- Screen (Lighten) ---

// BlendScreen: 1 - (1-d)(1-s)
func BlendScreen(d, s uint8, alpha float64) uint8 {
	return blendLerp(d, s, alpha, func(d, s float64) float64 {
		return 1 - (1-d)*(1-s)
	})
}

// ScreenColorBlend applies BlendScreen per-channel.
func ScreenColorBlend(dst, src Color, alpha float64) Color {
	return BlendColor(dst, src, alpha, BlendScreen)
}

// --- Overlay (Contrast) ---

// BlendOverlay: d < 0.5 ? 2*d*s : 1 - 2*(1-d)*(1-s)
func BlendOverlay(d, s uint8, alpha float64) uint8 {
	return blendLerp(d, s, alpha, func(d, s float64) float64 {
		if d < 0.5 {
			return 2 * d * s
		}
		return 1 - 2*(1-d)*(1-s)
	})
}

// OverlayColorBlend applies BlendOverlay per-channel.
func OverlayColorBlend(dst, src Color, alpha float64) Color {
	return BlendColor(dst, src, alpha, BlendOverlay)
}

// --- HardLight (Contrast) ---

// BlendHardLight: s < 0.5 ? 2*d*s : 1 - 2*(1-d)*(1-s)
func BlendHardLight(d, s uint8, alpha float64) uint8 {
	return blendLerp(d, s, alpha, func(d, s float64) float64 {
		if s < 0.5 {
			return 2 * d * s
		}
		return 1 - 2*(1-d)*(1-s)
	})
}

// HardLightColorBlend applies BlendHardLight per-channel.
func HardLightColorBlend(dst, src Color, alpha float64) Color {
	return BlendColor(dst, src, alpha, BlendHardLight)
}

// --- SoftLight (Contrast) ---

// BlendSoftLight uses the Pegtop formula: (1-2s)*d² + 2*s*d
func BlendSoftLight(d, s uint8, alpha float64) uint8 {
	return blendLerp(d, s, alpha, func(d, s float64) float64 {
		return (1-2*s)*d*d + 2*s*d
	})
}

// SoftLightColorBlend applies BlendSoftLight per-channel.
func SoftLightColorBlend(dst, src Color, alpha float64) Color {
	return BlendColor(dst, src, alpha, BlendSoftLight)
}

// --- Difference (Inversion) ---

// BlendDifference: |d - s|
func BlendDifference(d, s uint8, alpha float64) uint8 {
	return blendLerp(d, s, alpha, func(d, s float64) float64 {
		return math.Abs(d - s)
	})
}

// DifferenceColorBlend applies BlendDifference per-channel.
func DifferenceColorBlend(dst, src Color, alpha float64) Color {
	return BlendColor(dst, src, alpha, BlendDifference)
}

// --- Exclusion (Inversion) ---

// BlendExclusion: d + s - 2*d*s
func BlendExclusion(d, s uint8, alpha float64) uint8 {
	return blendLerp(d, s, alpha, func(d, s float64) float64 {
		return d + s - 2*d*s
	})
}

// ExclusionColorBlend applies BlendExclusion per-channel.
func ExclusionColorBlend(dst, src Color, alpha float64) Color {
	return BlendColor(dst, src, alpha, BlendExclusion)
}

// --- HardMix (Posterize) ---

// BlendHardMix: d + s >= 1 ? 1 : 0
func BlendHardMix(d, s uint8, alpha float64) uint8 {
	return blendLerp(d, s, alpha, func(d, s float64) float64 {
		if d+s >= 1 {
			return 1
		}
		return 0
	})
}

// HardMixColorBlend applies BlendHardMix per-channel.
func HardMixColorBlend(dst, src Color, alpha float64) Color {
	return BlendColor(dst, src, alpha, BlendHardMix)
}

// --- Darken ---

// BlendDarken: min(d, s)
func BlendDarken(d, s uint8, alpha float64) uint8 {
	return blendLerp(d, s, alpha, func(d, s float64) float64 {
		return math.Min(d, s)
	})
}

// DarkenColorBlend applies BlendDarken per-channel.
func DarkenColorBlend(dst, src Color, alpha float64) Color {
	return BlendColor(dst, src, alpha, BlendDarken)
}

// --- Lighten ---

// BlendLighten: max(d, s)
func BlendLighten(d, s uint8, alpha float64) uint8 {
	return blendLerp(d, s, alpha, func(d, s float64) float64 {
		return math.Max(d, s)
	})
}

// LightenColorBlend applies BlendLighten per-channel.
func LightenColorBlend(dst, src Color, alpha float64) Color {
	return BlendColor(dst, src, alpha, BlendLighten)
}

// --- LinearDodge (Lighten) ---

// BlendLinearDodge: min(1, d + s)
func BlendLinearDodge(d, s uint8, alpha float64) uint8 {
	return blendLerp(d, s, alpha, func(d, s float64) float64 {
		return math.Min(1, d+s)
	})
}

// LinearDodgeColorBlend applies BlendLinearDodge per-channel.
func LinearDodgeColorBlend(dst, src Color, alpha float64) Color {
	return BlendColor(dst, src, alpha, BlendLinearDodge)
}

// --- LinearBurn (Darken) ---

// BlendLinearBurn: max(0, d + s - 1)
func BlendLinearBurn(d, s uint8, alpha float64) uint8 {
	return blendLerp(d, s, alpha, func(d, s float64) float64 {
		return math.Max(0, d+s-1)
	})
}

// LinearBurnColorBlend applies BlendLinearBurn per-channel.
func LinearBurnColorBlend(dst, src Color, alpha float64) Color {
	return BlendColor(dst, src, alpha, BlendLinearBurn)
}

// --- ColorDodge (Lighten) ---

// BlendColorDodge: s >= 1 ? 1 : min(1, d/(1-s))
func BlendColorDodge(d, s uint8, alpha float64) uint8 {
	return blendLerp(d, s, alpha, func(d, s float64) float64 {
		if s >= 1 {
			return 1
		}
		return math.Min(1, d/(1-s))
	})
}

// ColorDodgeColorBlend applies BlendColorDodge per-channel.
func ColorDodgeColorBlend(dst, src Color, alpha float64) Color {
	return BlendColor(dst, src, alpha, BlendColorDodge)
}

// --- ColorBurn (Darken) ---

// BlendColorBurn: s <= 0 ? 0 : max(0, 1-(1-d)/s)
func BlendColorBurn(d, s uint8, alpha float64) uint8 {
	return blendLerp(d, s, alpha, func(d, s float64) float64 {
		if s <= 0 {
			return 0
		}
		return math.Max(0, 1-(1-d)/s)
	})
}

// ColorBurnColorBlend applies BlendColorBurn per-channel.
func ColorBurnColorBlend(dst, src Color, alpha float64) Color {
	return BlendColor(dst, src, alpha, BlendColorBurn)
}
