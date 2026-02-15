package asset

import (
	"image"
	"math"

	"flicker/core"
	"flicker/core/bitmap"
	"golang.org/x/image/font"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/vector"
)

// TextOptions configures text rasterization.
type TextOptions struct {
	Font      *Font
	Size      float64    // pixels per em
	Color     core.Color // text fill color
	AntiAlias bool       // enable anti-aliasing (default: sharp edges)
}

// Glyph holds layout information for a single character in rendered text.
type Glyph struct {
	Rune   rune
	Index  int // position in the original string
	X, Y   int // top-left corner in bitmap pixels
	Width  int // advance width in pixels
	Height int // bounding box height in pixels
}

// TextLayout holds a rasterized text bitmap and per-glyph layout information.
type TextLayout struct {
	Bitmap *bitmap.Bitmap
	Glyphs []Glyph
}

// RasterizeText rasterizes a single line of text into a bitmap with glyph layout.
// Returns nil if text is empty or the font has no glyphs for any character.
func RasterizeText(text string, opts TextOptions) *TextLayout {
	if len(text) == 0 || opts.Font == nil {
		return nil
	}

	sf := opts.Font.SFNTFont()
	buf := opts.Font.Buffer()
	ppem := fixed26_6(opts.Size)
	m := opts.Font.Metrics(opts.Size)

	// Pass 1: measure total advance width.
	var totalAdvance float64
	runes := []rune(text)
	for _, r := range runes {
		gi, err := sf.GlyphIndex(buf, r)
		if err != nil || gi == 0 {
			continue
		}
		adv, err := sf.GlyphAdvance(buf, gi, ppem, font.HintingNone)
		if err != nil {
			continue
		}
		totalAdvance += float64(adv) / 64.0
	}

	bmW := int(math.Ceil(totalAdvance))
	bmH := int(math.Ceil(m.Height))
	if bmW <= 0 || bmH <= 0 {
		return nil
	}

	// Pass 2: rasterize glyphs into vector rasterizer and track layout.
	r := vector.NewRasterizer(bmW, bmH)
	var xOffset float64
	ascent := m.Ascent
	var glyphs []Glyph

	for idx, ch := range runes {
		gi, err := sf.GlyphIndex(buf, ch)
		if err != nil || gi == 0 {
			continue
		}

		adv, err := sf.GlyphAdvance(buf, gi, ppem, font.HintingNone)
		if err != nil {
			continue
		}
		glyphWidth := float64(adv) / 64.0

		// Record glyph layout.
		glyphs = append(glyphs, Glyph{
			Rune:   ch,
			Index:  idx,
			X:      int(math.Floor(xOffset)),
			Y:      0,
			Width:  int(math.Ceil(glyphWidth)),
			Height: bmH,
		})

		segs, err := sf.LoadGlyph(buf, gi, ppem, nil)
		if err != nil {
			// Skip rasterization but still advance.
			xOffset += glyphWidth
			continue
		}

		for _, seg := range segs {
			switch seg.Op {
			case sfnt.SegmentOpMoveTo:
				px := float64(seg.Args[0].X)/64.0 + xOffset
				py := float64(seg.Args[0].Y)/64.0 + ascent
				r.MoveTo(float32(px), float32(py))
			case sfnt.SegmentOpLineTo:
				px := float64(seg.Args[0].X)/64.0 + xOffset
				py := float64(seg.Args[0].Y)/64.0 + ascent
				r.LineTo(float32(px), float32(py))
			case sfnt.SegmentOpQuadTo:
				bx := float64(seg.Args[0].X)/64.0 + xOffset
				by := float64(seg.Args[0].Y)/64.0 + ascent
				cx := float64(seg.Args[1].X)/64.0 + xOffset
				cy := float64(seg.Args[1].Y)/64.0 + ascent
				r.QuadTo(float32(bx), float32(by), float32(cx), float32(cy))
			case sfnt.SegmentOpCubeTo:
				bx := float64(seg.Args[0].X)/64.0 + xOffset
				by := float64(seg.Args[0].Y)/64.0 + ascent
				cx := float64(seg.Args[1].X)/64.0 + xOffset
				cy := float64(seg.Args[1].Y)/64.0 + ascent
				dx := float64(seg.Args[2].X)/64.0 + xOffset
				dy := float64(seg.Args[2].Y)/64.0 + ascent
				r.CubeTo(
					float32(bx),
					float32(by),
					float32(cx),
					float32(cy),
					float32(dx),
					float32(dy),
				)
			}
		}

		xOffset += glyphWidth
	}

	// Draw the rasterizer into an alpha mask.
	mask := image.NewAlpha(image.Rect(0, 0, bmW, bmH))
	r.Draw(mask, mask.Bounds(), image.Opaque, image.Point{})

	// Convert alpha mask to bitmap.
	bm := bitmap.New(bmW, bmH)
	for y := range bmH {
		for x := range bmW {
			a := mask.AlphaAt(x, y)
			if a.A > 0 {
				alpha := float64(a.A) / 255.0
				if !opts.AntiAlias && alpha > 0 {
					alpha = 1.0
				}
				bm.Set(x, y, opts.Color, alpha)
			}
		}
	}

	return &TextLayout{
		Bitmap: bm,
		Glyphs: glyphs,
	}
}

// GlyphAt returns the index of the glyph at pixel coordinates (x, y).
// Returns -1 if (x, y) is not within any glyph's bounding box.
func (tl *TextLayout) GlyphAt(x, y int) int {
	for i, g := range tl.Glyphs {
		if x >= g.X && x < g.X+g.Width && y >= g.Y && y < g.Y+g.Height {
			return i
		}
	}
	return -1
}

// SplitGlyphs crops the text bitmap into individual per-glyph bitmaps.
// Returns a slice where each element corresponds to tl.Glyphs[i].
// Useful for creating per-character entities with independent transforms.
func (tl *TextLayout) SplitGlyphs() []*bitmap.Bitmap {
	result := make([]*bitmap.Bitmap, len(tl.Glyphs))
	for i, g := range tl.Glyphs {
		glyph := bitmap.New(g.Width, g.Height)
		for dy := range g.Height {
			for dx := range g.Width {
				srcX := g.X + dx
				srcY := g.Y + dy
				if srcX < tl.Bitmap.Width && srcY < tl.Bitmap.Height {
					c, a := tl.Bitmap.Get(srcX, srcY)
					glyph.Set(dx, dy, c, a)
				}
			}
		}
		result[i] = glyph
	}
	return result
}
