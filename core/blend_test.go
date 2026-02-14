package core

import (
	"fmt"
	"math"
	"testing"
)

func TestBlendModes(t *testing.T) {
	type testCase struct {
		name  string
		mode  BlendMode
		d, s  uint8
		alpha float64
		want  uint8
	}

	cases := []testCase{
		// --- BlendNormal ---
		{"Normal black+white a=1", BlendNormal, 0, 255, 1.0, 255},
		{"Normal white+black a=1", BlendNormal, 255, 0, 1.0, 0},
		{"Normal mid+mid a=1", BlendNormal, 128, 128, 1.0, 128},
		{"Normal a=0 no effect", BlendNormal, 100, 200, 0.0, 100},
		{"Normal a=0.5 half", BlendNormal, 0, 255, 0.5, 128},

		// --- BlendMultiply ---
		{"Multiply black*white a=1", BlendMultiply, 0, 255, 1.0, 0},
		{"Multiply white*white a=1", BlendMultiply, 255, 255, 1.0, 255},
		{"Multiply 128*128 a=1", BlendMultiply, 128, 128, 1.0, 64},
		{"Multiply a=0 no effect", BlendMultiply, 128, 128, 0.0, 128},
		{"Multiply white*black a=1", BlendMultiply, 255, 0, 1.0, 0},

		// --- BlendScreen ---
		{"Screen black+black a=1", BlendScreen, 0, 0, 1.0, 0},
		{"Screen white+white a=1", BlendScreen, 255, 255, 1.0, 255},
		{"Screen 128+128 a=1", BlendScreen, 128, 128, 1.0, 191},
		{"Screen a=0 no effect", BlendScreen, 128, 128, 0.0, 128},
		{"Screen black+white a=1", BlendScreen, 0, 255, 1.0, 255},

		// --- BlendOverlay ---
		{"Overlay black+mid a=1", BlendOverlay, 0, 128, 1.0, 0},
		{"Overlay white+mid a=1", BlendOverlay, 255, 128, 1.0, 255},
		{"Overlay mid+mid a=1", BlendOverlay, 128, 128, 1.0, 128},
		{"Overlay a=0 no effect", BlendOverlay, 100, 200, 0.0, 100},

		// --- BlendHardLight ---
		{"HardLight mid+black a=1", BlendHardLight, 128, 0, 1.0, 0},
		{"HardLight mid+white a=1", BlendHardLight, 128, 255, 1.0, 255},
		{"HardLight a=0 no effect", BlendHardLight, 100, 200, 0.0, 100},

		// --- BlendSoftLight ---
		{"SoftLight mid+mid a=1", BlendSoftLight, 128, 128, 1.0, 128},
		{"SoftLight a=0 no effect", BlendSoftLight, 100, 200, 0.0, 100},
		{"SoftLight black+white a=1", BlendSoftLight, 0, 255, 1.0, 0},

		// --- BlendDifference ---
		{"Difference same a=1", BlendDifference, 128, 128, 1.0, 0},
		{"Difference black+white a=1", BlendDifference, 0, 255, 1.0, 255},
		{"Difference white+black a=1", BlendDifference, 255, 0, 1.0, 255},
		{"Difference a=0 no effect", BlendDifference, 0, 255, 0.0, 0},

		// --- BlendExclusion ---
		{"Exclusion black+black a=1", BlendExclusion, 0, 0, 1.0, 0},
		{"Exclusion white+white a=1", BlendExclusion, 255, 255, 1.0, 0},
		{"Exclusion black+white a=1", BlendExclusion, 0, 255, 1.0, 255},
		{"Exclusion mid+mid a=1", BlendExclusion, 128, 128, 1.0, 128},

		// --- BlendHardMix ---
		{"HardMix 200+200 a=1", BlendHardMix, 200, 200, 1.0, 255},
		{"HardMix 50+50 a=1", BlendHardMix, 50, 50, 1.0, 0},
		{"HardMix a=0 no effect", BlendHardMix, 200, 200, 0.0, 200},

		// --- BlendDarken ---
		{"Darken 100+200 a=1", BlendDarken, 100, 200, 1.0, 100},
		{"Darken 200+100 a=1", BlendDarken, 200, 100, 1.0, 100},
		{"Darken a=0 no effect", BlendDarken, 200, 100, 0.0, 200},

		// --- BlendLighten ---
		{"Lighten 100+200 a=1", BlendLighten, 100, 200, 1.0, 200},
		{"Lighten 200+100 a=1", BlendLighten, 200, 100, 1.0, 200},
		{"Lighten a=0 no effect", BlendLighten, 200, 100, 0.0, 200},

		// --- BlendLinearDodge ---
		{"LinearDodge 128+128 a=1", BlendLinearDodge, 128, 128, 1.0, 255},
		{"LinearDodge 0+0 a=1", BlendLinearDodge, 0, 0, 1.0, 0},
		{"LinearDodge a=0 no effect", BlendLinearDodge, 128, 128, 0.0, 128},

		// --- BlendLinearBurn ---
		{"LinearBurn 255+255 a=1", BlendLinearBurn, 255, 255, 1.0, 255},
		{"LinearBurn 128+128 a=1", BlendLinearBurn, 128, 128, 1.0, 1},
		{"LinearBurn 0+0 a=1", BlendLinearBurn, 0, 0, 1.0, 0},
		{"LinearBurn a=0 no effect", BlendLinearBurn, 128, 128, 0.0, 128},

		// --- BlendColorDodge ---
		{"ColorDodge 128+0 a=1", BlendColorDodge, 128, 0, 1.0, 128},
		{"ColorDodge 128+255 a=1", BlendColorDodge, 128, 255, 1.0, 255},
		{"ColorDodge a=0 no effect", BlendColorDodge, 128, 128, 0.0, 128},

		// --- BlendColorBurn ---
		{"ColorBurn 128+255 a=1", BlendColorBurn, 128, 255, 1.0, 128},
		{"ColorBurn 128+0 a=1", BlendColorBurn, 128, 0, 1.0, 0},
		{"ColorBurn a=0 no effect", BlendColorBurn, 128, 128, 0.0, 128},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.mode(tc.d, tc.s, tc.alpha)
			// Allow ±1 for rounding.
			diff := int(got) - int(tc.want)
			if diff < -1 || diff > 1 {
				t.Errorf("got %d, want %d (±1)", got, tc.want)
			}
		})
	}
}

func TestColorBlendWrappers(t *testing.T) {
	dst := Color{R: 200, G: 100, B: 50}
	src := Color{R: 100, G: 200, B: 150}
	alpha := 0.7

	wrappers := []struct {
		name  string
		blend ColorBlend
		mode  BlendMode
	}{
		{"NormalColorBlend", NormalColorBlend, BlendNormal},
		{"MultiplyColorBlend", MultiplyColorBlend, BlendMultiply},
		{"ScreenColorBlend", ScreenColorBlend, BlendScreen},
		{"OverlayColorBlend", OverlayColorBlend, BlendOverlay},
		{"HardLightColorBlend", HardLightColorBlend, BlendHardLight},
		{"SoftLightColorBlend", SoftLightColorBlend, BlendSoftLight},
		{"DifferenceColorBlend", DifferenceColorBlend, BlendDifference},
		{"ExclusionColorBlend", ExclusionColorBlend, BlendExclusion},
		{"HardMixColorBlend", HardMixColorBlend, BlendHardMix},
		{"DarkenColorBlend", DarkenColorBlend, BlendDarken},
		{"LightenColorBlend", LightenColorBlend, BlendLighten},
		{"LinearDodgeColorBlend", LinearDodgeColorBlend, BlendLinearDodge},
		{"LinearBurnColorBlend", LinearBurnColorBlend, BlendLinearBurn},
		{"ColorDodgeColorBlend", ColorDodgeColorBlend, BlendColorDodge},
		{"ColorBurnColorBlend", ColorBurnColorBlend, BlendColorBurn},
	}

	for _, w := range wrappers {
		t.Run(w.name, func(t *testing.T) {
			got := w.blend(dst, src, alpha)
			want := BlendColor(dst, src, alpha, w.mode)
			if got != want {
				t.Errorf("got %v, want %v", got, want)
			}
		})
	}
}

func TestBlendAlphaHalf(t *testing.T) {
	// With alpha=0.5, the result should be halfway between dst and the
	// fully-blended value.
	modes := []struct {
		name string
		mode BlendMode
	}{
		{"Multiply", BlendMultiply},
		{"Screen", BlendScreen},
		{"Overlay", BlendOverlay},
		{"Difference", BlendDifference},
	}

	d, s := uint8(200), uint8(100)
	for _, m := range modes {
		t.Run(m.name, func(t *testing.T) {
			full := m.mode(d, s, 1.0)
			half := m.mode(d, s, 0.5)
			// half should be approximately midpoint between d and full.
			expected := (float64(d) + float64(full)) / 2
			if math.Abs(float64(half)-expected) > 2 {
				t.Errorf("half=%d, expected≈%.0f (d=%d, full=%d)",
					half, expected, d, full)
			}
		})
	}
}

func TestClamp01(t *testing.T) {
	cases := []struct {
		in, want float64
	}{
		{-0.5, 0},
		{0, 0},
		{0.5, 0.5},
		{1, 1},
		{1.5, 1},
	}
	for _, tc := range cases {
		t.Run(fmt.Sprintf("%v", tc.in), func(t *testing.T) {
			got := clamp01(tc.in)
			if got != tc.want {
				t.Errorf("clamp01(%v) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}
