package core

import "flicker/fmath"

func Render(world *World, canvas *Canvas, t Time) {
	identity := fmath.Mat3Identity()
	for _, root := range world.Roots() {
		renderEntity(world, canvas, root, identity, t)
	}
}

func renderEntity(w *World, c *Canvas, e Entity, parent fmath.Mat3, t Time) {
	tr := w.Transform(e)
	if tr == nil {
		return
	}

	world := parent.Multiply(tr.LocalMatrix())

	if d := w.Drawable(e); d != nil {
		bw, bh := d.Bounds()
		cx, cy := float64(bw)/2.0, float64(bh)/2.0
		m := w.Material(e)

		for dy := range bh {
			for dx := range bw {
				cell := d.CellAt(dx, dy)
				if cell.Alpha == 0 {
					continue
				}

				relX := float64(dx) - cx
				relY := float64(dy) - cy
				sx := int(world[0]*relX + world[1]*relY + world[2] + cx)
				sy := int(world[3]*relX + world[4]*relY + world[5] + cy)

				if m != nil {
					f := Fragment{
						X: dx, Y: dy,
						ScreenX: sx, ScreenY: sy,
						Time:   t,
						Cell:   cell,
						Source: c,
					}
					cell = m(f)
				}
				c.Set(sx, sy, cell)
			}
		}
	}

	for _, child := range w.Children(e) {
		renderEntity(w, c, child, world, t)
	}
}
