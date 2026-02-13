package core

type Drawable interface {
	Draw(canvas *Canvas, x, y int)
	Bounds() (width, height int)
}
