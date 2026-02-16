package bitmap

import (
	"math"

	"flicker/core"
	"flicker/fmath"
)

// BGBlock wraps a Bitmap and implements core.Drawable using BG-only encoding.
// Each pixel maps to one space character with BG color; FGAlpha is 0.
type BGBlock struct {
	Bitmap *Bitmap
}

// Draw renders the bitmap onto the canvas at the given offset.
func (bg *BGBlock) Draw(canvas *core.Canvas, cx, cy int) {
	if bg.Bitmap == nil {
		return
	}
	b := bg.Bitmap
	for y := range b.Height {
		for x := range b.Width {
			c, a := b.Get(x, y)
			if a == 0 {
				continue
			}
			canvas.Set(cx+x, cy+y, core.Cell{
				Rune:    ' ',
				BG:      c,
				BGAlpha: a,
			})
		}
	}
}

// CellAt returns the BG-only Cell for the pixel at (col, row).
func (bg *BGBlock) CellAt(x, y int) core.Cell {
	if bg.Bitmap == nil {
		return core.Cell{}
	}
	c, a := bg.Bitmap.Get(x, y)
	if a == 0 {
		return core.Cell{}
	}
	return core.Cell{
		Rune:    ' ',
		BG:      c,
		BGAlpha: a,
	}
}

// Bounds returns the cell-space dimensions of the bitmap in BG-block encoding.
func (bg *BGBlock) Bounds() (int, int) {
	if bg.Bitmap == nil {
		return 0, 0
	}
	return bg.Bitmap.Width, bg.Bitmap.Height
}

// Renderer returns an inverse-mapping RenderFunc for BG-block mode.
func (bg *BGBlock) Renderer() core.RenderFunc {
	if bg.Bitmap == nil {
		return func(world fmath.Mat3, emit func(dx, dy, sx, sy int, cell core.Cell)) {}
	}
	bw, bh := bg.Bounds()
	bm := bg.Bitmap
	cx, cy := float64(bw)/2.0, float64(bh)/2.0
	return inverseRenderer(
		bw,
		bh,
		func(inv [4]float64, tx, ty float64, sx, sy int) (int, int, core.Cell, bool) {
			P := float64(sx) - tx + 0.5
			Q := float64(sy) - ty + 0.5
			localX := inv[0]*P + inv[1]*Q + cx
			localY := inv[2]*P + inv[3]*Q + cy

			px := int(math.Floor(localX))
			py := int(math.Floor(localY))
			if px < 0 || px >= bm.Width || py < 0 || py >= bm.Height {
				return 0, 0, core.Cell{}, false
			}
			c, a := bm.Get(px, py)
			if a == 0 {
				return 0, 0, core.Cell{}, false
			}
			return px, py, core.Cell{Rune: ' ', BG: c, BGAlpha: a}, true
		},
	)
}
