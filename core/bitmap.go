package core

// EncodeMode selects how a Bitmap is mapped to terminal cells.
type EncodeMode int

const (
	// EncodeBraille maps 2x4 pixel blocks to braille characters (U+2800-U+28FF).
	// One FG color per cell. Best for wireframes, particles, monochrome text.
	EncodeBraille EncodeMode = iota
	// EncodeHalfBlock maps 1x2 pixel pairs to half-block characters (▀/▄).
	// Two independent colors per cell (FG=top, BG=bottom). Best for color images.
	EncodeHalfBlock
)

// brailleBits maps (dx, dy) within a 2x4 block to the corresponding braille dot bit.
//
//	dot1(0x01) dot4(0x08)
//	dot2(0x02) dot5(0x10)
//	dot3(0x04) dot6(0x20)
//	dot7(0x40) dot8(0x80)
var brailleBits = [2][4]byte{
	{0x01, 0x02, 0x04, 0x40}, // left column: dots 1,2,3,7
	{0x08, 0x10, 0x20, 0x80}, // right column: dots 4,5,6,8
}

// Bitmap is a high-resolution pixel buffer that maps back to terminal cells
// via braille or half-block encoding.
type Bitmap struct {
	Width, Height int
	Pix           []Color   // flat row-major: Pix[y*Width+x]
	Alpha         []float64 // flat row-major: Alpha[y*Width+x]
}

// NewBitmap creates a bitmap of the given pixel dimensions.
func NewBitmap(w, h int) *Bitmap {
	n := w * h
	return &Bitmap{
		Width:  w,
		Height: h,
		Pix:    make([]Color, n),
		Alpha:  make([]float64, n),
	}
}

// Set sets the pixel at (x, y) to the given color and alpha.
func (b *Bitmap) Set(x, y int, c Color, alpha float64) {
	if x < 0 || x >= b.Width || y < 0 || y >= b.Height {
		return
	}
	i := y*b.Width + x
	b.Pix[i] = c
	b.Alpha[i] = alpha
}

// Get returns the color and alpha at (x, y). Out-of-bounds returns zero values.
func (b *Bitmap) Get(x, y int) (Color, float64) {
	if x < 0 || x >= b.Width || y < 0 || y >= b.Height {
		return Color{}, 0
	}
	i := y*b.Width + x
	return b.Pix[i], b.Alpha[i]
}

// SetDot sets the pixel at (x, y) to the given color with alpha=1.
func (b *Bitmap) SetDot(x, y int, c Color) {
	b.Set(x, y, c, 1.0)
}

// Clear resets all pixels to zero color and zero alpha.
func (b *Bitmap) Clear() {
	for i := range b.Pix {
		b.Pix[i] = Color{}
		b.Alpha[i] = 0
	}
}

// DrawBraille encodes the bitmap into braille characters on the canvas.
// Each 2x4 pixel block becomes one braille cell at canvas position (cx+col, cy+row).
func (b *Bitmap) DrawBraille(canvas *Canvas, cx, cy int) {
	cols := (b.Width + 1) / 2
	rows := (b.Height + 3) / 4

	for row := range rows {
		for col := range cols {
			var bits byte
			var rSum, gSum, bSum int
			var count int
			var maxAlpha float64

			for dy := range 4 {
				for dx := range 2 {
					px := col*2 + dx
					py := row*4 + dy
					if px >= b.Width || py >= b.Height {
						continue
					}
					_, a := b.Get(px, py)
					if a > 0 {
						bits |= brailleBits[dx][dy]
						c := b.Pix[py*b.Width+px]
						rSum += int(c.R)
						gSum += int(c.G)
						bSum += int(c.B)
						count++
						if a > maxAlpha {
							maxAlpha = a
						}
					}
				}
			}

			if bits == 0 {
				continue
			}

			fg := Color{
				R: uint8(rSum / count),
				G: uint8(gSum / count),
				B: uint8(bSum / count),
			}
			canvas.Set(cx+col, cy+row, Cell{
				Rune:  rune(0x2800 | int(bits)),
				FG:    fg,
				Alpha: maxAlpha,
			})
		}
	}
}

// DrawHalfBlock encodes the bitmap into half-block characters on the canvas.
// Each 1x2 pixel pair becomes one cell at canvas position (cx+x, cy+row).
func (b *Bitmap) DrawHalfBlock(canvas *Canvas, cx, cy int) {
	cols := b.Width
	rows := (b.Height + 1) / 2

	for row := range rows {
		for col := range cols {
			topY := row * 2
			botY := row*2 + 1

			_, topA := b.Get(col, topY)
			_, botA := b.Get(col, botY)
			topOn := topA > 0
			botOn := botA > 0

			if !topOn && !botOn {
				continue
			}

			var cell Cell

			switch {
			case topOn && botOn:
				topC, _ := b.Get(col, topY)
				botC, _ := b.Get(col, botY)
				cell.Rune = '▀'
				cell.FG = topC
				cell.BG = botC
				if topA > botA {
					cell.Alpha = topA
				} else {
					cell.Alpha = botA
				}
			case topOn:
				topC, _ := b.Get(col, topY)
				cell.Rune = '▀'
				cell.FG = topC
				cell.Alpha = topA
			case botOn:
				botC, _ := b.Get(col, botY)
				cell.Rune = '▄'
				cell.FG = botC
				cell.Alpha = botA
			}

			canvas.Set(cx+col, cy+row, cell)
		}
	}
}

// BitmapDrawable wraps a Bitmap to implement the Drawable interface.
type BitmapDrawable struct {
	Bitmap *Bitmap
	Mode   EncodeMode
}

// Draw renders the bitmap onto the canvas at the given offset.
func (bd *BitmapDrawable) Draw(canvas *Canvas, x, y int) {
	if bd.Bitmap == nil {
		return
	}
	switch bd.Mode {
	case EncodeBraille:
		bd.Bitmap.DrawBraille(canvas, x, y)
	case EncodeHalfBlock:
		bd.Bitmap.DrawHalfBlock(canvas, x, y)
	}
}

// Bounds returns the cell-space dimensions of the bitmap.
func (bd *BitmapDrawable) Bounds() (int, int) {
	if bd.Bitmap == nil {
		return 0, 0
	}
	switch bd.Mode {
	case EncodeBraille:
		return (bd.Bitmap.Width + 1) / 2, (bd.Bitmap.Height + 3) / 4
	case EncodeHalfBlock:
		return bd.Bitmap.Width, (bd.Bitmap.Height + 1) / 2
	}
	return 0, 0
}
