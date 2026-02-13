package core

type Rect struct {
	Width  int
	Height int
	Rune   rune
}

func (r *Rect) Draw(canvas *Canvas, x, y int) {
	for dy := 0; dy < r.Height; dy++ {
		for dx := 0; dx < r.Width; dx++ {
			canvas.Set(x+dx, y+dy, Cell{Rune: r.Rune})
		}
	}
}
