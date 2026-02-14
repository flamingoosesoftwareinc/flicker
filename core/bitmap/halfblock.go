package bitmap

import (
	"flicker/core"
	"flicker/fmath"
)

// HalfBlock wraps a Bitmap and implements core.Drawable using half-block encoding.
// Each 1x2 pixel pair maps to one half-block character (▀/▄).
type HalfBlock struct {
	Bitmap *Bitmap
}

// Draw renders the bitmap onto the canvas at the given offset.
func (hb *HalfBlock) Draw(canvas *core.Canvas, cx, cy int) {
	if hb.Bitmap == nil {
		return
	}
	b := hb.Bitmap
	cols := b.Width
	rows := (b.Height + 1) / 2

	for row := range rows {
		for col := range cols {
			topY := row * 2
			botY := row*2 + 1

			_, topA := b.Get(col, topY)
			_, botA := b.Get(col, botY)
			topOn := topA > 0
			botOn := botA > 0

			if !topOn && !botOn {
				continue
			}

			var cell core.Cell

			switch {
			case topOn && botOn:
				topC, _ := b.Get(col, topY)
				botC, _ := b.Get(col, botY)
				cell.Rune = '▀'
				cell.FG = topC
				cell.BG = botC
				cell.FGAlpha = topA
				cell.BGAlpha = botA
			case topOn:
				topC, _ := b.Get(col, topY)
				cell.Rune = '▀'
				cell.FG = topC
				cell.FGAlpha = topA
			case botOn:
				botC, _ := b.Get(col, botY)
				cell.Rune = '▄'
				cell.FG = botC
				cell.FGAlpha = botA
			}

			canvas.Set(cx+col, cy+row, cell)
		}
	}
}

// CellAt returns the half-block-encoded Cell for the cell-grid position (col, row).
func (hb *HalfBlock) CellAt(x, y int) core.Cell {
	if hb.Bitmap == nil {
		return core.Cell{}
	}
	b := hb.Bitmap
	topY := y * 2
	botY := y*2 + 1

	_, topA := b.Get(x, topY)
	_, botA := b.Get(x, botY)
	topOn := topA > 0
	botOn := botA > 0

	if !topOn && !botOn {
		return core.Cell{}
	}

	var cell core.Cell

	switch {
	case topOn && botOn:
		topC, _ := b.Get(x, topY)
		botC, _ := b.Get(x, botY)
		cell.Rune = '▀'
		cell.FG = topC
		cell.BG = botC
		cell.FGAlpha = topA
		cell.BGAlpha = botA
	case topOn:
		topC, _ := b.Get(x, topY)
		cell.Rune = '▀'
		cell.FG = topC
		cell.FGAlpha = topA
	case botOn:
		botC, _ := b.Get(x, botY)
		cell.Rune = '▄'
		cell.FG = botC
		cell.FGAlpha = botA
	}

	return cell
}

// Bounds returns the cell-space dimensions of the bitmap in half-block encoding.
func (hb *HalfBlock) Bounds() (int, int) {
	if hb.Bitmap == nil {
		return 0, 0
	}
	return hb.Bitmap.Width, (hb.Bitmap.Height + 1) / 2
}

// Renderer returns a forward-mapping RenderFunc for half-block mode.
func (hb *HalfBlock) Renderer() core.RenderFunc {
	if hb.Bitmap == nil {
		return func(world fmath.Mat3, emit func(dx, dy, sx, sy int, cell core.Cell)) {}
	}
	return forwardRenderer(hb)
}
