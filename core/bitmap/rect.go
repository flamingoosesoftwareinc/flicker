package bitmap

import (
	"flicker/core"
	"flicker/fmath"
)

// Rect is a rectangular drawable that uses half-block encoding internally.
type Rect struct {
	Width, Height    int
	FG, BG           core.Color
	FGAlpha, BGAlpha float64 // 0 means opaque (1.0)

	hb *HalfBlock // lazily built
}

func (r *Rect) ensureBitmap() {
	if r.hb != nil {
		return
	}
	fgA := r.FGAlpha
	if fgA == 0 {
		fgA = 1
	}
	bgA := r.BGAlpha
	if bgA == 0 {
		bgA = 1
	}
	bm := New(r.Width, r.Height*2)
	for row := range r.Height {
		for x := range r.Width {
			bm.Set(x, row*2, r.FG, fgA)
			bm.Set(x, row*2+1, r.BG, bgA)
		}
	}
	r.hb = &HalfBlock{Bitmap: bm}
}

func (r *Rect) Draw(canvas *core.Canvas, x, y int) {
	r.ensureBitmap()
	r.hb.Draw(canvas, x, y)
}

func (r *Rect) Bounds() (int, int) {
	r.ensureBitmap()
	return r.hb.Bounds()
}

func (r *Rect) CellAt(x, y int) core.Cell {
	r.ensureBitmap()
	return r.hb.CellAt(x, y)
}

// BitmapToScreen converts bitmap pixel coordinates to screen character cell coordinates.
// Rect delegates to its internal HalfBlock implementation.
func (r *Rect) BitmapToScreen(coord fmath.Vec2) fmath.Vec2 {
	r.ensureBitmap()
	return r.hb.BitmapToScreen(coord)
}

func (r *Rect) Renderer() core.RenderFunc {
	r.ensureBitmap()
	return r.hb.Renderer()
}
