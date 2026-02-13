package core

func Render(world *World, canvas *Canvas, t Time) {
	for _, root := range world.Roots() {
		renderEntity(world, canvas, root, 0, 0, 0, t)
	}
}

func renderEntity(w *World, c *Canvas, e Entity, ox, oy, oz float64, t Time) {
	tr := w.Transform(e)
	if tr == nil {
		return
	}

	ax := ox + tr.Position.X
	ay := oy + tr.Position.Y
	az := oz + tr.Position.Z

	if d := w.Drawable(e); d != nil {
		sx, sy := int(ax), int(ay)
		d.Draw(c, sx, sy)

		if m := w.Material(e); m != nil {
			bw, bh := d.Bounds()
			for dy := range bh {
				for dx := range bw {
					cx, cy := sx+dx, sy+dy
					cell := c.Get(cx, cy)
					c.Set(cx, cy, m(dx, dy, t, cell))
				}
			}
		}
	}

	for _, child := range w.Children(e) {
		renderEntity(w, c, child, ax, ay, az, t)
	}
}
