package bitmap

import (
	"math"

	"flicker/core"
	"flicker/fmath"
)

// Adaptive wraps a Bitmap and implements core.Drawable using adaptive block
// encoding. Each 6×9 pixel block is matched against ~130 Unicode character
// templates (sextants, diagonal blocks, triangular blocks, and standard block
// elements) to find the best-fit character via minimum hamming distance.
type Adaptive struct {
	Bitmap         *Bitmap
	AlphaThreshold float64 // pixels with alpha ≤ threshold are treated as empty
}

// Draw renders the bitmap onto the canvas at the given offset.
func (ad *Adaptive) Draw(canvas *core.Canvas, cx, cy int) {
	if ad.Bitmap == nil {
		return
	}
	bm := ad.Bitmap
	cols := (bm.Width + 5) / 6
	rows := (bm.Height + 8) / 9

	for row := range rows {
		for col := range cols {
			cell := ad.cellFromBitmap(bm, col, row)
			if cell.Rune == 0 {
				continue
			}
			canvas.Set(cx+col, cy+row, cell)
		}
	}
}

// CellAt returns the adaptive-encoded Cell for the cell-grid position (col, row).
func (ad *Adaptive) CellAt(x, y int) core.Cell {
	if ad.Bitmap == nil {
		return core.Cell{}
	}
	return ad.cellFromBitmap(ad.Bitmap, x, y)
}

// Bounds returns the cell-space dimensions of the bitmap in adaptive encoding.
func (ad *Adaptive) Bounds() (int, int) {
	if ad.Bitmap == nil {
		return 0, 0
	}
	return (ad.Bitmap.Width + 5) / 6, (ad.Bitmap.Height + 8) / 9
}

// Renderer returns an inverse-mapping RenderFunc for adaptive mode.
// Maps each screen cell center back to the local cell grid and uses the
// pre-computed adaptive encoding, avoiding cross-cell sampling artifacts.
func (ad *Adaptive) Renderer() core.RenderFunc {
	if ad.Bitmap == nil {
		return func(world fmath.Mat3, emit func(dx, dy, sx, sy int, cell core.Cell)) {}
	}
	bw, bh := ad.Bounds()
	bm := ad.Bitmap
	cx, cy := float64(bw)/2.0, float64(bh)/2.0
	return inverseRenderer(
		bw,
		bh,
		func(inv [4]float64, tx, ty float64, sx, sy int) (int, int, core.Cell, bool) {
			// Map screen cell center to local cell-space coordinates.
			P := float64(sx) - tx + 0.5
			Q := float64(sy) - ty + 0.5
			localX := inv[0]*P + inv[1]*Q + cx
			localY := inv[2]*P + inv[3]*Q + cy

			cellX := int(math.Floor(localX))
			cellY := int(math.Floor(localY))

			if cellX < 0 || cellX >= bw || cellY < 0 || cellY >= bh {
				return 0, 0, core.Cell{}, false
			}

			cell := ad.cellFromBitmap(bm, cellX, cellY)
			if cell.Rune == 0 {
				return 0, 0, core.Cell{}, false
			}
			return cellX, cellY, cell, true
		},
	)
}

// cellFromBitmap samples a 6×9 region from the bitmap and returns the best-fit cell.
func (ad *Adaptive) cellFromBitmap(bm *Bitmap, col, row int) core.Cell {
	sample := sampleCellThreshold(bm, col, row, ad.AlphaThreshold)
	if sample == 0 {
		return core.Cell{}
	}

	r, _ := bestMatch(sample)
	if r == ' ' {
		return core.Cell{}
	}

	// Compute average color of filled pixels.
	thresh := ad.AlphaThreshold
	var rSum, gSum, bSum int
	var count int
	var maxAlpha float64
	for dy := range 9 {
		for dx := range 6 {
			px := col*6 + dx
			py := row*9 + dy
			if px >= bm.Width || py >= bm.Height {
				continue
			}
			a := bm.Alpha[py*bm.Width+px]
			if a > thresh {
				c := bm.Pix[py*bm.Width+px]
				rSum += int(c.R)
				gSum += int(c.G)
				bSum += int(c.B)
				count++
				if a > maxAlpha {
					maxAlpha = a
				}
			}
		}
	}

	if count == 0 {
		return core.Cell{}
	}

	return core.Cell{
		Rune: r,
		FG: core.Color{
			R: uint8(rSum / count),
			G: uint8(gSum / count),
			B: uint8(bSum / count),
		},
		FGAlpha: maxAlpha,
	}
}
