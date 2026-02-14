package core

import "flicker/fmath"

// RenderFunc iterates over visible cells and calls emit for each.
// dx, dy are local drawable coords; sx, sy are screen coords.
type RenderFunc func(world fmath.Mat3, emit func(dx, dy, sx, sy int, cell Cell))

type Drawable interface {
	Draw(canvas *Canvas, x, y int)
	Bounds() (width, height int)
	CellAt(x, y int) Cell
	Renderer() RenderFunc
}
