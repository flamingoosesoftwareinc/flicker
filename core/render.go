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
	pos := world.Apply(fmath.Vec2{})

	if d := w.Drawable(e); d != nil {
		sx, sy := int(pos.X), int(pos.Y)
		d.Draw(c, sx, sy)

		if m := w.Material(e); m != nil {
			bw, bh := d.Bounds()
			for dy := range bh {
				for dx := range bw {
					cx, cy := sx+dx, sy+dy
					f := Fragment{
						X: dx, Y: dy,
						ScreenX: cx, ScreenY: cy,
						Time:   t,
						Cell:   c.Get(cx, cy),
						Source: c,
					}
					c.Set(cx, cy, m(f))
				}
			}
		}
	}

	for _, child := range w.Children(e) {
		renderEntity(w, c, child, world, t)
	}
}
