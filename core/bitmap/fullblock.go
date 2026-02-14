package bitmap

import (
	"flicker/core"
	"flicker/fmath"
)

// FullBlock wraps a Bitmap and implements core.Drawable using full-block encoding.
// Each pixel maps 1:1 to one full-block character (█).
type FullBlock struct {
	Bitmap *Bitmap
}

// Draw renders the bitmap onto the canvas at the given offset.
func (fb *FullBlock) Draw(canvas *core.Canvas, cx, cy int) {
	if fb.Bitmap == nil {
		return
	}
	b := fb.Bitmap
	for y := range b.Height {
		for x := range b.Width {
			c, a := b.Get(x, y)
			if a == 0 {
				continue
			}
			canvas.Set(cx+x, cy+y, core.Cell{
				Rune:    '█',
				FG:      c,
				FGAlpha: a,
			})
		}
	}
}

// CellAt returns the full-block-encoded Cell for the pixel at (col, row).
func (fb *FullBlock) CellAt(x, y int) core.Cell {
	if fb.Bitmap == nil {
		return core.Cell{}
	}
	c, a := fb.Bitmap.Get(x, y)
	if a == 0 {
		return core.Cell{}
	}
	return core.Cell{
		Rune:    '█',
		FG:      c,
		FGAlpha: a,
	}
}

// Bounds returns the cell-space dimensions of the bitmap in full-block encoding.
func (fb *FullBlock) Bounds() (int, int) {
	if fb.Bitmap == nil {
		return 0, 0
	}
	return fb.Bitmap.Width, fb.Bitmap.Height
}

// Renderer returns a forward-mapping RenderFunc for full-block mode.
func (fb *FullBlock) Renderer() core.RenderFunc {
	if fb.Bitmap == nil {
		return func(world fmath.Mat3, emit func(dx, dy, sx, sy int, cell core.Cell)) {}
	}
	return forwardRenderer(fb)
}
