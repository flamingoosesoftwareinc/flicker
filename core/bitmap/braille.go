package bitmap

import (
	"math"

	"flicker/core"
	"flicker/fmath"
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

// Braille wraps a Bitmap and implements core.Drawable using braille encoding.
// Each 2x4 pixel block maps to one braille character (U+2800-U+28FF).
type Braille struct {
	Bitmap *Bitmap
}

// Draw renders the bitmap onto the canvas at the given offset.
func (br *Braille) Draw(canvas *core.Canvas, cx, cy int) {
	if br.Bitmap == nil {
		return
	}
	b := br.Bitmap
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

			fg := core.Color{
				R: uint8(rSum / count),
				G: uint8(gSum / count),
				B: uint8(bSum / count),
			}
			canvas.Set(cx+col, cy+row, core.Cell{
				Rune:    rune(0x2800 | int(bits)),
				FG:      fg,
				FGAlpha: maxAlpha,
			})
		}
	}
}

// CellAt returns the braille-encoded Cell for the cell-grid position (col, row).
func (br *Braille) CellAt(x, y int) core.Cell {
	if br.Bitmap == nil {
		return core.Cell{}
	}
	b := br.Bitmap
	var bits byte
	var rSum, gSum, bSum int
	var count int
	var maxAlpha float64

	for dy := range 4 {
		for dx := range 2 {
			px := x*2 + dx
			py := y*4 + dy
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
		return core.Cell{}
	}

	fg := core.Color{
		R: uint8(rSum / count),
		G: uint8(gSum / count),
		B: uint8(bSum / count),
	}
	return core.Cell{
		Rune:    rune(0x2800 | int(bits)),
		FG:      fg,
		FGAlpha: maxAlpha,
	}
}

// Bounds returns the cell-space dimensions of the bitmap in braille encoding.
func (br *Braille) Bounds() (int, int) {
	if br.Bitmap == nil {
		return 0, 0
	}
	return (br.Bitmap.Width + 1) / 2, (br.Bitmap.Height + 3) / 4
}

// BitmapToScreen converts bitmap pixel coordinates to screen character cell coordinates.
// Braille: 2:1 horizontal (2 pixels = 1 cell), 4:1 vertical (4 pixels = 1 cell).
func (br *Braille) BitmapToScreen(coord fmath.Vec2) fmath.Vec2 {
	return fmath.Vec2{
		X: coord.X / 2.0, // 2:1 horizontal compression
		Y: coord.Y / 4.0, // 4:1 vertical compression
	}
}

// Renderer returns an inverse-mapping RenderFunc for braille mode.
// For each screen cell in the rotated bounding box, it samples 2x4 dot positions
// through the inverse world matrix to determine which source pixels are visible,
// producing partial braille runes at rotated edges.
func (br *Braille) Renderer() core.RenderFunc {
	if br.Bitmap == nil {
		return func(world fmath.Mat3, emit func(dx, dy, sx, sy int, cell core.Cell)) {}
	}
	return func(world fmath.Mat3, emit func(dx, dy, sx, sy int, cell core.Cell)) {
		bw, bh := br.Bounds()
		cx, cy := float64(bw)/2.0, float64(bh)/2.0
		bm := br.Bitmap

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
				var firstLX, firstLY int
				gotFirst := false

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
							if !gotFirst {
								firstLX = int(math.Floor(localX))
								firstLY = int(math.Floor(localY))
								gotFirst = true
							}
						}
					}
				}

				if bits == 0 {
					continue
				}

				fg := core.Color{
					R: uint8(rSum / count),
					G: uint8(gSum / count),
					B: uint8(bSum / count),
				}
				cell := core.Cell{
					Rune:    rune(0x2800 | int(bits)),
					FG:      fg,
					FGAlpha: maxAlpha,
				}
				emit(firstLX, firstLY, sx, sy, cell)
			}
		}
	}
}
