package core

import (
	"strings"
	"unicode/utf8"

	"flicker/fmath"
)

// Text is a drawable that renders raw terminal text characters directly to
// cells. Unlike bitmap-based drawables, it bypasses encoding/sampling entirely
// — each rune occupies one cell with no aliasing artifacts.
type Text struct {
	lines   []string
	FG      Color
	FGAlpha float64
}

// NewText creates a Text drawable from a string. Lines are split on '\n'.
func NewText(text string, fg Color, fgAlpha float64) *Text {
	return &Text{
		lines:   strings.Split(text, "\n"),
		FG:      fg,
		FGAlpha: fgAlpha,
	}
}

// SetText updates the text content, re-splitting lines.
func (t *Text) SetText(text string) {
	t.lines = strings.Split(text, "\n")
}

// Bounds returns (maxLineWidth, lineCount).
func (t *Text) Bounds() (int, int) {
	maxW := 0
	for _, line := range t.lines {
		w := utf8.RuneCountInString(line)
		if w > maxW {
			maxW = w
		}
	}
	return maxW, len(t.lines)
}

// CellAt returns the cell at local position (x, y).
// Returns a zero Cell for out-of-bounds or space/zero-rune positions.
func (t *Text) CellAt(x, y int) Cell {
	if y < 0 || y >= len(t.lines) {
		return Cell{}
	}
	runes := []rune(t.lines[y])
	if x < 0 || x >= len(runes) {
		return Cell{}
	}
	r := runes[x]
	if r == ' ' || r == 0 {
		return Cell{}
	}
	return Cell{
		Rune:    r,
		FG:      t.FG,
		FGAlpha: t.FGAlpha,
	}
}

// Draw writes each rune to the canvas at (x+col, y+row), skipping spaces.
func (t *Text) Draw(canvas *Canvas, x, y int) {
	for row, line := range t.lines {
		for col, r := range line {
			if r == ' ' || r == 0 {
				continue
			}
			canvas.Set(x+col, y+row, Cell{
				Rune:    r,
				FG:      t.FG,
				FGAlpha: t.FGAlpha,
			})
		}
	}
}

// Renderer returns a forward-mapping RenderFunc. For each character at local
// (col, row), it transforms through the world matrix to a screen position.
// Characters move to transformed positions while remaining upright.
func (t *Text) Renderer() RenderFunc {
	return func(world fmath.Mat3, emit func(dx, dy, sx, sy int, cell Cell)) {
		bw, bh := t.Bounds()
		cx := float64(bw) / 2.0
		cy := float64(bh) / 2.0

		for row, line := range t.lines {
			for col, r := range line {
				if r == ' ' || r == 0 {
					continue
				}
				// Local position centered on drawable origin
				lx := float64(col) - cx + 0.5
				ly := float64(row) - cy + 0.5

				// Transform to screen position
				sx := int(world[0]*lx + world[1]*ly + world[2] + cx)
				sy := int(world[3]*lx + world[4]*ly + world[5] + cy)

				emit(col, row, sx, sy, Cell{
					Rune:    r,
					FG:      t.FG,
					FGAlpha: t.FGAlpha,
				})
			}
		}
	}
}
