package bitmap

import (
	"math"

	"flicker/core"
	"flicker/fmath"
)

// inverseRenderer returns an inverse-mapping RenderFunc. For each screen cell
// in the transformed bounding box it calls sample with the inverse matrix and
// screen position. The sampler maps back to source coordinates and returns the
// cell to emit (or false to skip).
func inverseRenderer(
	bw, bh int,
	sample func(inv [4]float64, tx, ty float64, sx, sy int) (lx, ly int, cell core.Cell, ok bool),
) core.RenderFunc {
	return func(world fmath.Mat3, emit func(dx, dy, sx, sy int, cell core.Cell)) {
		cx, cy := float64(bw)/2.0, float64(bh)/2.0

		det := world[0]*world[4] - world[1]*world[3]
		if det == 0 {
			return
		}
		invDet := 1.0 / det

		// Inverse of the 2x2 rotation/scale sub-matrix, pre-divided by det.
		inv := [4]float64{
			world[4] * invDet,
			-world[1] * invDet,
			-world[3] * invDet,
			world[0] * invDet,
		}

		// Translation offset: world[2]+cx, world[5]+cy.
		tx := world[2] + cx
		ty := world[5] + cy

		// Transform 4 corners of the drawable to find the screen bounding box.
		corners := [4][2]float64{
			{-cx, -cy},
			{float64(bw) - cx, -cy},
			{-cx, float64(bh) - cy},
			{float64(bw) - cx, float64(bh) - cy},
		}

		minSX := math.Inf(1)
		minSY := math.Inf(1)
		maxSX := math.Inf(-1)
		maxSY := math.Inf(-1)
		for _, cr := range corners {
			scrX := world[0]*cr[0] + world[1]*cr[1] + tx
			scrY := world[3]*cr[0] + world[4]*cr[1] + ty
			if scrX < minSX {
				minSX = scrX
			}
			if scrX > maxSX {
				maxSX = scrX
			}
			if scrY < minSY {
				minSY = scrY
			}
			if scrY > maxSY {
				maxSY = scrY
			}
		}

		startX := int(math.Floor(minSX)) - 1
		startY := int(math.Floor(minSY)) - 1
		endX := int(math.Ceil(maxSX)) + 1
		endY := int(math.Ceil(maxSY)) + 1

		for sy := startY; sy <= endY; sy++ {
			for sx := startX; sx <= endX; sx++ {
				lx, ly, cell, ok := sample(inv, tx, ty, sx, sy)
				if !ok {
					continue
				}
				emit(lx, ly, sx, sy, cell)
			}
		}
	}
}
