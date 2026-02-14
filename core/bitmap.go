package core

import (
	"math"

	"flicker/fmath"
)

// EncodeMode selects how a Bitmap is mapped to terminal cells.
type EncodeMode int

const (
	// EncodeBraille maps 2x4 pixel blocks to braille characters (U+2800-U+28FF).
	// One FG color per cell. Best for wireframes, particles, monochrome text.
	EncodeBraille EncodeMode = iota
	// EncodeHalfBlock maps 1x2 pixel pairs to half-block characters (▀/▄).
	// Two independent colors per cell (FG=top, BG=bottom). Best for color images.
	EncodeHalfBlock
	// EncodeFullBlock maps 1x1 pixels to full-block characters (█).
	// One color per cell. Simplest encoding, 1:1 pixel-to-cell mapping.
	EncodeFullBlock
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

// Line draws a line from (x0,y0) to (x1,y1) using Bresenham's algorithm.
func (b *Bitmap) Line(x0, y0, x1, y1 int, c Color) {
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
				Rune:    rune(0x2800 | int(bits)),
				FG:      fg,
				FGAlpha: maxAlpha,
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
				cell.FGAlpha = topA
				cell.BGAlpha = botA
			case topOn:
				topC, _ := b.Get(col, topY)
				cell.Rune = '▀'
				cell.FG = topC
				cell.FGAlpha = topA
			case botOn:
				botC, _ := b.Get(col, botY)
				cell.Rune = '▄'
				cell.FG = botC
				cell.FGAlpha = botA
			}

			canvas.Set(cx+col, cy+row, cell)
		}
	}
}

// BrailleCellAt returns the braille-encoded Cell for the cell-grid position (col, row).
func (b *Bitmap) BrailleCellAt(col, row int) Cell {
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
		return Cell{}
	}

	fg := Color{
		R: uint8(rSum / count),
		G: uint8(gSum / count),
		B: uint8(bSum / count),
	}
	return Cell{
		Rune:    rune(0x2800 | int(bits)),
		FG:      fg,
		FGAlpha: maxAlpha,
	}
}

// HalfBlockCellAt returns the half-block-encoded Cell for the cell-grid position (col, row).
func (b *Bitmap) HalfBlockCellAt(col, row int) Cell {
	topY := row * 2
	botY := row*2 + 1

	_, topA := b.Get(col, topY)
	_, botA := b.Get(col, botY)
	topOn := topA > 0
	botOn := botA > 0

	if !topOn && !botOn {
		return Cell{}
	}

	var cell Cell

	switch {
	case topOn && botOn:
		topC, _ := b.Get(col, topY)
		botC, _ := b.Get(col, botY)
		cell.Rune = '▀'
		cell.FG = topC
		cell.BG = botC
		cell.FGAlpha = topA
		cell.BGAlpha = botA
	case topOn:
		topC, _ := b.Get(col, topY)
		cell.Rune = '▀'
		cell.FG = topC
		cell.FGAlpha = topA
	case botOn:
		botC, _ := b.Get(col, botY)
		cell.Rune = '▄'
		cell.FG = botC
		cell.FGAlpha = botA
	}

	return cell
}

// DrawFullBlock encodes the bitmap into full-block characters on the canvas.
// Each pixel becomes one cell at canvas position (cx+x, cy+y).
func (b *Bitmap) DrawFullBlock(canvas *Canvas, cx, cy int) {
	for y := range b.Height {
		for x := range b.Width {
			c, a := b.Get(x, y)
			if a == 0 {
				continue
			}
			canvas.Set(cx+x, cy+y, Cell{
				Rune:    '█',
				FG:      c,
				FGAlpha: a,
			})
		}
	}
}

// FullBlockCellAt returns the full-block-encoded Cell for the pixel at (col, row).
func (b *Bitmap) FullBlockCellAt(col, row int) Cell {
	c, a := b.Get(col, row)
	if a == 0 {
		return Cell{}
	}
	return Cell{
		Rune:    '█',
		FG:      c,
		FGAlpha: a,
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
	case EncodeFullBlock:
		bd.Bitmap.DrawFullBlock(canvas, x, y)
	}
}

// CellAt returns the encoded cell at position (x, y) in cell-grid space.
func (bd *BitmapDrawable) CellAt(x, y int) Cell {
	if bd.Bitmap == nil {
		return Cell{}
	}
	switch bd.Mode {
	case EncodeBraille:
		return bd.Bitmap.BrailleCellAt(x, y)
	case EncodeHalfBlock:
		return bd.Bitmap.HalfBlockCellAt(x, y)
	case EncodeFullBlock:
		return bd.Bitmap.FullBlockCellAt(x, y)
	}
	return Cell{}
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
	case EncodeFullBlock:
		return bd.Bitmap.Width, bd.Bitmap.Height
	}
	return 0, 0
}

// Renderer returns a mode-dependent RenderFunc strategy.
func (bd *BitmapDrawable) Renderer() RenderFunc {
	if bd.Bitmap == nil {
		return func(world fmath.Mat3, emit func(dx, dy, sx, sy int, cell Cell)) {}
	}
	switch bd.Mode {
	case EncodeBraille:
		return bd.brailleRenderer()
	case EncodeHalfBlock, EncodeFullBlock:
		return bd.forwardRenderer()
	}
	return func(world fmath.Mat3, emit func(dx, dy, sx, sy int, cell Cell)) {}
}

// forwardRenderer returns a forward-mapping RenderFunc (used for half-block mode).
func (bd *BitmapDrawable) forwardRenderer() RenderFunc {
	return func(world fmath.Mat3, emit func(dx, dy, sx, sy int, cell Cell)) {
		bw, bh := bd.Bounds()
		cx, cy := float64(bw)/2.0, float64(bh)/2.0

		for dy := range bh {
			for dx := range bw {
				cell := bd.CellAt(dx, dy)
				if cell.FGAlpha == 0 && cell.BGAlpha == 0 {
					continue
				}
				relX := float64(dx) - cx
				relY := float64(dy) - cy
				sx := int(world[0]*relX + world[1]*relY + world[2] + cx)
				sy := int(world[3]*relX + world[4]*relY + world[5] + cy)
				emit(dx, dy, sx, sy, cell)
			}
		}
	}
}

// brailleRenderer returns an inverse-mapping RenderFunc for braille mode.
// For each screen cell in the rotated bounding box, it samples 2x4 dot positions
// through the inverse world matrix to determine which source pixels are visible,
// producing partial braille runes at rotated edges.
func (bd *BitmapDrawable) brailleRenderer() RenderFunc {
	return func(world fmath.Mat3, emit func(dx, dy, sx, sy int, cell Cell)) {
		bw, bh := bd.Bounds()
		cx, cy := float64(bw)/2.0, float64(bh)/2.0
		bm := bd.Bitmap

		det := world[0]*world[4] - world[1]*world[3]
		if det == 0 {
			return
		}
		invDet := 1.0 / det

		// Transform 4 corners of the drawable to find the screen bounding box.
		corners := [4][2]float64{
			{-cx, -cy},
			{float64(bw) - cx, -cy},
			{-cx, float64(bh) - cy},
			{float64(bw) - cx, float64(bh) - cy},
		}

		minSX := math.Inf(1)
		minSY := math.Inf(1)
		maxSX := math.Inf(-1)
		maxSY := math.Inf(-1)
		for _, cr := range corners {
			scrX := world[0]*cr[0] + world[1]*cr[1] + world[2] + cx
			scrY := world[3]*cr[0] + world[4]*cr[1] + world[5] + cy
			if scrX < minSX {
				minSX = scrX
			}
			if scrX > maxSX {
				maxSX = scrX
			}
			if scrY < minSY {
				minSY = scrY
			}
			if scrY > maxSY {
				maxSY = scrY
			}
		}

		startX := int(math.Floor(minSX)) - 1
		startY := int(math.Floor(minSY)) - 1
		endX := int(math.Ceil(maxSX)) + 1
		endY := int(math.Ceil(maxSY)) + 1

		// For each screen cell, sample 2x4 dot positions through inverse transform.
		for sy := startY; sy <= endY; sy++ {
			for sx := startX; sx <= endX; sx++ {
				var bits byte
				var rSum, gSum, bSum int
				var count int
				var maxAlpha float64

				for ddy := range 4 {
					for ddx := range 2 {
						// Screen-space position of this dot's center.
						dotSX := float64(sx) + (float64(ddx)+0.5)/2.0
						dotSY := float64(sy) + (float64(ddy)+0.5)/4.0

						// Inverse transform to local cell-space.
						P := dotSX - world[2] - cx
						Q := dotSY - world[5] - cy
						localX := (world[4]*P-world[1]*Q)*invDet + cx
						localY := (-world[3]*P+world[0]*Q)*invDet + cy

						// Convert to bitmap pixel coordinates.
						px := int(math.Floor(localX * 2))
						py := int(math.Floor(localY * 4))

						if px < 0 || px >= bm.Width || py < 0 || py >= bm.Height {
							continue
						}
						_, a := bm.Get(px, py)
						if a > 0 {
							bits |= brailleBits[ddx][ddy]
							clr := bm.Pix[py*bm.Width+px]
							rSum += int(clr.R)
							gSum += int(clr.G)
							bSum += int(clr.B)
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
				cell := Cell{
					Rune:    rune(0x2800 | int(bits)),
					FG:      fg,
					FGAlpha: maxAlpha,
				}
				emit(sx, sy, sx, sy, cell)
			}
		}
	}
}
