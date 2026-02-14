package core

type Color struct {
	R, G, B uint8
}

type Cell struct {
	Rune  rune
	FG    Color
	BG    Color
	Alpha float64
}

type Canvas struct {
	Width, Height int
	Cells         [][]Cell
}

func NewCanvas(w, h int) *Canvas {
	cells := make([][]Cell, h)
	for y := range cells {
		cells[y] = make([]Cell, w)
	}
	return &Canvas{Width: w, Height: h, Cells: cells}
}

func (c *Canvas) Set(x, y int, cell Cell) {
	if x < 0 || x >= c.Width || y < 0 || y >= c.Height {
		return
	}
	c.Cells[y][x] = cell
}

func (c *Canvas) Get(x, y int) Cell {
	if x < 0 || x >= c.Width || y < 0 || y >= c.Height {
		return Cell{}
	}
	return c.Cells[y][x]
}

func (c *Canvas) Clear() {
	for y := range c.Cells {
		for x := range c.Cells[y] {
			c.Cells[y][x] = Cell{}
		}
	}
}

// BlendMode computes a blended channel value from dst and src channel values
// at the given alpha. Both d and s are in [0,255]; alpha is in [0,1].
type BlendMode func(d, s uint8, alpha float64) uint8

// BlendNormal is the standard linear interpolation: dst*(1-a) + src*a.
func BlendNormal(d, s uint8, alpha float64) uint8 {
	return uint8(float64(d)*(1-alpha) + float64(s)*alpha)
}

// ColorBlend blends two colors at a given alpha. BlendCell and
// Canvas.Composite accept this type so callers control color mixing.
type ColorBlend func(dst, src Color, alpha float64) Color

// NormalColorBlend applies BlendNormal per-channel.
func NormalColorBlend(dst, src Color, alpha float64) Color {
	return BlendColor(dst, src, alpha, BlendNormal)
}

// BlendColor applies a BlendMode per-channel to produce a blended color.
func BlendColor(dst, src Color, alpha float64, mode BlendMode) Color {
	a := alpha
	if a < 0 {
		a = 0
	}
	if a > 1 {
		a = 1
	}
	return Color{
		R: mode(dst.R, src.R, a),
		G: mode(dst.G, src.G, a),
		B: mode(dst.B, src.B, a),
	}
}

// BlendCell composites src over dst using the "over" operator. The
// ColorBlend function controls how FG/BG colors are mixed.
func BlendCell(dst, src Cell, blend ColorBlend) Cell {
	if src.Alpha == 0 {
		return dst
	}
	if src.Alpha >= 1 && dst.Alpha == 0 {
		return src
	}

	a := src.Alpha
	out := Cell{
		FG:    blend(dst.FG, src.FG, a),
		BG:    blend(dst.BG, src.BG, a),
		Alpha: dst.Alpha + src.Alpha*(1-dst.Alpha),
	}

	// Rune rule: real src rune wins; empty src preserves dst text.
	if src.Rune != 0 {
		out.Rune = src.Rune
	} else {
		out.Rune = dst.Rune
	}
	return out
}

// Composite applies BlendCell cell-by-cell, compositing src on top of c
// using the given ColorBlend.
func (c *Canvas) Composite(src *Canvas, blend ColorBlend) {
	for y := 0; y < c.Height && y < src.Height; y++ {
		for x := 0; x < c.Width && x < src.Width; x++ {
			c.Cells[y][x] = BlendCell(c.Cells[y][x], src.Cells[y][x], blend)
		}
	}
}

func (c *Canvas) DrawBorder() {
	if c.Width < 2 || c.Height < 2 {
		return
	}

	last := c.Width - 1
	bottom := c.Height - 1

	c.Set(0, 0, Cell{Rune: '┌'})
	c.Set(last, 0, Cell{Rune: '┐'})
	c.Set(0, bottom, Cell{Rune: '└'})
	c.Set(last, bottom, Cell{Rune: '┘'})

	for x := 1; x < last; x++ {
		c.Set(x, 0, Cell{Rune: '─'})
		c.Set(x, bottom, Cell{Rune: '─'})
	}
	for y := 1; y < bottom; y++ {
		c.Set(0, y, Cell{Rune: '│'})
		c.Set(last, y, Cell{Rune: '│'})
	}
}
