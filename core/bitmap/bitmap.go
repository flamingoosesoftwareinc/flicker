package bitmap

import "flicker/core"

// Bitmap is a high-resolution pixel buffer that maps back to terminal cells
// via braille or half-block encoding.
type Bitmap struct {
	Width, Height int
	Pix           []core.Color // flat row-major: Pix[y*Width+x]
	Alpha         []float64    // flat row-major: Alpha[y*Width+x]
}

// New creates a bitmap of the given pixel dimensions.
func New(w, h int) *Bitmap {
	n := w * h
	return &Bitmap{
		Width:  w,
		Height: h,
		Pix:    make([]core.Color, n),
		Alpha:  make([]float64, n),
	}
}

// Set sets the pixel at (x, y) to the given color and alpha.
func (b *Bitmap) Set(x, y int, c core.Color, alpha float64) {
	if x < 0 || x >= b.Width || y < 0 || y >= b.Height {
		return
	}
	i := y*b.Width + x
	b.Pix[i] = c
	b.Alpha[i] = alpha
}

// Get returns the color and alpha at (x, y). Out-of-bounds returns zero values.
func (b *Bitmap) Get(x, y int) (core.Color, float64) {
	if x < 0 || x >= b.Width || y < 0 || y >= b.Height {
		return core.Color{}, 0
	}
	i := y*b.Width + x
	return b.Pix[i], b.Alpha[i]
}

// SetDot sets the pixel at (x, y) to the given color with alpha=1.
func (b *Bitmap) SetDot(x, y int, c core.Color) {
	b.Set(x, y, c, 1.0)
}

// Clear resets all pixels to zero color and zero alpha.
func (b *Bitmap) Clear() {
	for i := range b.Pix {
		b.Pix[i] = core.Color{}
		b.Alpha[i] = 0
	}
}

// Line draws a line from (x0,y0) to (x1,y1) using Bresenham's algorithm.
func (b *Bitmap) Line(x0, y0, x1, y1 int, c core.Color) {
	dx := x1 - x0
	if dx < 0 {
		dx = -dx
	}
	dy := y1 - y0
	if dy < 0 {
		dy = -dy
	}
	dy = -dy

	sx := 1
	if x0 >= x1 {
		sx = -1
	}
	sy := 1
	if y0 >= y1 {
		sy = -1
	}

	err := dx + dy
	for {
		b.SetDot(x0, y0, c)
		if x0 == x1 && y0 == y1 {
			break
		}
		e2 := 2 * err
		if e2 >= dy {
			err += dy
			x0 += sx
		}
		if e2 <= dx {
			err += dx
			y0 += sy
		}
	}
}
