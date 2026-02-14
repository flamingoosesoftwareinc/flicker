package bitmap

import (
	"math"
	"testing"

	"flicker/core"
)

func TestComputeSDF(t *testing.T) {
	// Create a 10x10 bitmap with a 6x6 filled rect at (2,2)-(7,7).
	bm := New(10, 10)
	for y := 2; y <= 7; y++ {
		for x := 2; x <= 7; x++ {
			bm.Set(x, y, core.Color{R: 255, G: 255, B: 255}, 1.0)
		}
	}

	sdf := ComputeSDF(bm, 10)

	// Center pixel (5,5) should be inside (negative distance).
	center := sdf.At(5, 5)
	if center >= 0 {
		t.Errorf("center (5,5): expected negative, got %f", center)
	}

	// Corner pixel (0,0) should be outside (positive distance).
	corner := sdf.At(0, 0)
	if corner <= 0 {
		t.Errorf("corner (0,0): expected positive, got %f", corner)
	}

	// Edge pixel (2,2) should be approximately 0 (on the boundary).
	edge := sdf.At(2, 2)
	if math.Abs(edge) > 1.5 {
		t.Errorf("edge (2,2): expected near 0, got %f", edge)
	}

	// Gradient at top-center edge (5,2) should point upward (Y < 0).
	grad := sdf.Gradient(5, 2)
	if grad.Y >= 0 {
		t.Errorf("gradient at (5,2): expected Y < 0 (points up toward outside), got Y=%f", grad.Y)
	}

	// OOB returns MaxDist.
	oob := sdf.At(-1, -1)
	if oob != sdf.MaxDist {
		t.Errorf("OOB At(-1,-1): expected %f, got %f", sdf.MaxDist, oob)
	}
	oob2 := sdf.At(10, 10)
	if oob2 != sdf.MaxDist {
		t.Errorf("OOB At(10,10): expected %f, got %f", sdf.MaxDist, oob2)
	}

	// ComputeSDF with maxDist=3 should clamp exterior values.
	sdfClamped := ComputeSDF(bm, 3)
	farCorner := sdfClamped.At(0, 0)
	if farCorner > 3.0 {
		t.Errorf("clamped corner (0,0): expected <= 3.0, got %f", farCorner)
	}
}
