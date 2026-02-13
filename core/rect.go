package core

type Rect struct {
	Width  int
	Height int
	Rune   rune
	FG, BG Color
}

func (r *Rect) Draw(canvas *Canvas, x, y int) {
	for dy := 0; dy < r.Height; dy++ {
		for dx := 0; dx < r.Width; dx++ {
			canvas.Set(x+dx, y+dy, Cell{
				Rune:  r.Rune,
				FG:    r.FG,
				BG:    r.BG,
				Alpha: 1.0,
			})
		}
	}
}

func (r *Rect) Bounds() (int, int) {
	return r.Width, r.Height
}
