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
	Font  *Font
	Size  float64    // pixels per em
	Color core.Color // text fill color
}

// RasterizeText rasterizes a single line of text into a bitmap.
// Returns nil if text is empty or the font has no glyphs for any character.
func RasterizeText(text string, opts TextOptions) *bitmap.Bitmap {
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

	// Pass 2: rasterize glyphs into vector rasterizer.
	r := vector.NewRasterizer(bmW, bmH)
	var xOffset float64
	ascent := m.Ascent

	for _, ch := range runes {
		gi, err := sf.GlyphIndex(buf, ch)
		if err != nil || gi == 0 {
			continue
		}

		segs, err := sf.LoadGlyph(buf, gi, ppem, nil)
		if err != nil {
			// Fall back: skip this glyph, just advance.
			adv, _ := sf.GlyphAdvance(buf, gi, ppem, font.HintingNone)
			xOffset += float64(adv) / 64.0
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

		adv, _ := sf.GlyphAdvance(buf, gi, ppem, font.HintingNone)
		xOffset += float64(adv) / 64.0
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
				bm.Set(x, y, opts.Color, float64(a.A)/255.0)
			}
		}
	}

	return bm
}
