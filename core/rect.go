package core

type Rect struct {
	Width, Height    int
	FG, BG           Color
	FGAlpha, BGAlpha float64 // 0 means opaque (1.0)

	bd *BitmapDrawable // lazily built
}

func (r *Rect) ensureBitmap() {
	if r.bd != nil {
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
	bm := NewBitmap(r.Width, r.Height*2)
	for row := range r.Height {
		for x := range r.Width {
			bm.Set(x, row*2, r.FG, fgA)
			bm.Set(x, row*2+1, r.BG, bgA)
		}
	}
	r.bd = &BitmapDrawable{Bitmap: bm, Mode: EncodeHalfBlock}
}

func (r *Rect) Draw(canvas *Canvas, x, y int) {
	r.ensureBitmap()
	r.bd.Draw(canvas, x, y)
}

func (r *Rect) Bounds() (int, int) {
	r.ensureBitmap()
	return r.bd.Bounds()
}

func (r *Rect) CellAt(x, y int) Cell {
	r.ensureBitmap()
	return r.bd.CellAt(x, y)
}

func (r *Rect) Renderer() RenderFunc {
	r.ensureBitmap()
	return r.bd.Renderer()
}
