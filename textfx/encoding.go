package textfx

import (
	"flicker/asset"
	"flicker/core"
)

// Encoding specifies how bitmap pixels map to terminal cells.
type Encoding int

const (
	// HalfBlock encodes 1×2 pixels per cell (top/bottom halves).
	HalfBlock Encoding = iota
	// Braille encodes 2×4 pixels per cell (8 dots).
	Braille
	// FullBlock encodes 1:1 pixels per cell.
	FullBlock
	// Adaptive encodes 6×9 pixels per cell using best-fit Unicode blocks.
	Adaptive
)

// glyphAtForEncoding maps fragment coordinates to glyph index based on encoding.
func glyphAtForEncoding(layout *asset.TextLayout, encoding Encoding, f core.Fragment) int {
	switch encoding {
	case HalfBlock:
		// HalfBlock: cell row Y maps to 2 pixel rows (Y*2, Y*2+1).
		return layout.GlyphAt(f.X, f.Y*2)
	case Braille:
		// Braille: 2×4 dots per cell, sample at center.
		return layout.GlyphAt(f.X*2+1, f.Y*4+2)
	case FullBlock:
		// FullBlock: 1:1 pixel-to-cell mapping.
		return layout.GlyphAt(f.X, f.Y)
	case Adaptive:
		// Adaptive: 6×9 pixels per cell, sample at center.
		return layout.GlyphAt(f.X*6+3, f.Y*9+4)
	default:
		return -1
	}
}
