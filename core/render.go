package core

func Render(world *World, canvas *Canvas) {
	for _, root := range world.Roots() {
		renderEntity(world, canvas, root, 0, 0)
	}
}

func renderEntity(w *World, c *Canvas, e Entity, ox, oy float64) {
	t := w.Transform(e)
	if t == nil {
		return
	}

	ax := ox + t.Position.X
	ay := oy + t.Position.Y

	if g := w.Geometry(e); g != nil {
		drawGeometry(c, g, int(ax), int(ay))
	}

	for _, child := range w.Children(e) {
		renderEntity(w, c, child, ax, ay)
	}
}

func drawGeometry(c *Canvas, g *Geometry, x, y int) {
	switch g.Kind {
	case GeoRect:
		for dy := 0; dy < g.Height; dy++ {
			for dx := 0; dx < g.Width; dx++ {
				c.Set(x+dx, y+dy, Cell{Rune: g.Rune})
			}
		}
	}
}
