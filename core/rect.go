package core

import "flicker/fmath"

type Rect struct {
	Width  int
	Height int
	Rune   rune
	FG, BG Color
}

func (r *Rect) Draw(canvas *Canvas, x, y int) {
	for dy := 0; dy < r.Height; dy++ {
		for dx := 0; dx < r.Width; dx++ {
			canvas.Set(x+dx, y+dy, Cell{
				Rune:    r.Rune,
				FG:      r.FG,
				BG:      r.BG,
				FGAlpha: 1,
				BGAlpha: 1,
			})
		}
	}
}

func (r *Rect) Bounds() (int, int) {
	return r.Width, r.Height
}

func (r *Rect) CellAt(x, y int) Cell {
	return Cell{Rune: r.Rune, FG: r.FG, BG: r.BG, FGAlpha: 1, BGAlpha: 1}
}

func (r *Rect) Renderer() RenderFunc {
	return func(world fmath.Mat3, emit func(dx, dy, sx, sy int, cell Cell)) {
		bw, bh := r.Bounds()
		cx, cy := float64(bw)/2.0, float64(bh)/2.0

		for dy := range bh {
			for dx := range bw {
				cell := r.CellAt(dx, dy)
				if cell.FGAlpha == 0 && cell.BGAlpha == 0 {
					continue
				}
				relX := float64(dx) - cx
				relY := float64(dy) - cy
				sx := int(world[0]*relX + world[1]*relY + world[2] + cx)
				sy := int(world[3]*relX + world[4]*relY + world[5] + cy)
				emit(dx, dy, sx, sy, cell)
			}
		}
	}
}
