package core

import "flicker/fmath"

// RenderFunc iterates over visible cells and calls emit for each.
// dx, dy are local drawable coords; sx, sy are screen coords.
type RenderFunc func(world fmath.Mat3, emit func(dx, dy, sx, sy int, cell Cell))

type Drawable interface {
	Draw(canvas *Canvas, x, y int)
	CellAt(x, y int) Cell
	Renderer() RenderFunc
	// BitmapToScreen converts bitmap pixel coordinates to screen character cell coordinates.
	// Each drawable has its own compression ratio (e.g., HalfBlock is 1:1 horizontal, 2:1 vertical).
	BitmapToScreen(bitmapCoord fmath.Vec2) fmath.Vec2
}
