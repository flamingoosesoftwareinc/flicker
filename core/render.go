package core

func Render(world *World, canvas *Canvas) {
	for _, root := range world.Roots() {
		renderEntity(world, canvas, root, 0, 0, 0)
	}
}

func renderEntity(w *World, c *Canvas, e Entity, ox, oy, oz float64) {
	t := w.Transform(e)
	if t == nil {
		return
	}

	ax := ox + t.Position.X
	ay := oy + t.Position.Y
	az := oz + t.Position.Z

	if d := w.Drawable(e); d != nil {
		d.Draw(c, int(ax), int(ay))
	}

	for _, child := range w.Children(e) {
		renderEntity(w, c, child, ax, ay, az)
	}
}
