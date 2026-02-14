package bitmap

import (
	"math"

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

// Renderer returns an inverse-mapping RenderFunc for half-block mode.
// Each screen cell samples two bitmap rows (top and bottom halves).
func (hb *HalfBlock) Renderer() core.RenderFunc {
	if hb.Bitmap == nil {
		return func(world fmath.Mat3, emit func(dx, dy, sx, sy int, cell core.Cell)) {}
	}
	bw, bh := hb.Bounds()
	bm := hb.Bitmap
	cx, cy := float64(bw)/2.0, float64(bh)/2.0
	return inverseRenderer(
		bw,
		bh,
		func(inv [4]float64, tx, ty float64, sx, sy int) (core.Cell, bool) {
			// Sample top half (cell center at y+0.25) and bottom half (y+0.75).
			P := float64(sx) - tx + 0.5
			Qtop := float64(sy) - ty + 0.25
			Qbot := float64(sy) - ty + 0.75

			topLX := inv[0]*P + inv[1]*Qtop + cx
			topLY := inv[2]*P + inv[3]*Qtop + cy
			botLX := inv[0]*P + inv[1]*Qbot + cx
			botLY := inv[2]*P + inv[3]*Qbot + cy

			// Convert cell-space to bitmap pixel coordinates (2 bitmap rows per cell row).
			topPX := int(math.Floor(topLX))
			topPY := int(math.Floor(topLY * 2))
			botPX := int(math.Floor(botLX))
			botPY := int(math.Floor(botLY * 2))

			var topC core.Color
			var topA float64
			if topPX >= 0 && topPX < bm.Width && topPY >= 0 && topPY < bm.Height {
				topC, topA = bm.Get(topPX, topPY)
			}

			var botC core.Color
			var botA float64
			if botPX >= 0 && botPX < bm.Width && botPY >= 0 && botPY < bm.Height {
				botC, botA = bm.Get(botPX, botPY)
			}

			topOn := topA > 0
			botOn := botA > 0

			if !topOn && !botOn {
				return core.Cell{}, false
			}

			var cell core.Cell
			switch {
			case topOn && botOn:
				cell.Rune = '▀'
				cell.FG = topC
				cell.BG = botC
				cell.FGAlpha = topA
				cell.BGAlpha = botA
			case topOn:
				cell.Rune = '▀'
				cell.FG = topC
				cell.FGAlpha = topA
			case botOn:
				cell.Rune = '▄'
				cell.FG = botC
				cell.FGAlpha = botA
			}

			return cell, true
		},
	)
}
