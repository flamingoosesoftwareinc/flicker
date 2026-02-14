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
