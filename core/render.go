package core

import "flicker/fmath"

func Render(world *World, canvas *Canvas, t Time) {
	view := viewMatrix(world, canvas.Width, canvas.Height)
	for _, root := range world.Roots() {
		renderEntity(world, canvas, root, view, t)
	}
}

// viewMatrix returns the camera's view matrix, or identity if no active camera.
func viewMatrix(w *World, canvasW, canvasH int) fmath.Mat3 {
	e := w.ActiveCamera()
	if e == 0 {
		return fmath.Mat3Identity()
	}
	return ViewMatrix(w.Camera(e), w.Transform(e), canvasW, canvasH)
}

func renderEntity(w *World, c *Canvas, e Entity, parent fmath.Mat3, t Time) {
	tr := w.Transform(e)
	if tr == nil {
		return
	}

	world := parent.Multiply(tr.LocalMatrix())

	if d := w.Drawable(e); d != nil {
		render := d.Renderer()
		m := w.Material(e)
		render(world, func(dx, dy, sx, sy int, cell Cell) {
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
			if cell.BGAlpha == 0 {
				existing := c.Get(sx, sy)
				cell.BG = existing.BG
				cell.BGAlpha = existing.BGAlpha
			}
			c.Set(sx, sy, cell)
		})
	}

	for _, child := range w.Children(e) {
		renderEntity(w, c, child, world, t)
	}
}
