package bitmap

import (
	"flicker/core"
	"flicker/fmath"
)

// forwardRenderer returns a forward-mapping RenderFunc shared by
// HalfBlock, FullBlock, and BGBlock drawables.
func forwardRenderer(d core.Drawable) core.RenderFunc {
	return func(world fmath.Mat3, emit func(dx, dy, sx, sy int, cell core.Cell)) {
		bw, bh := d.Bounds()
		cx, cy := float64(bw)/2.0, float64(bh)/2.0

		for dy := range bh {
			for dx := range bw {
				cell := d.CellAt(dx, dy)
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
