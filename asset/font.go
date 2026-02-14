package asset

import (
	"fmt"
	"os"

	"golang.org/x/image/font"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
)

// Font wraps an sfnt.Font with a reusable buffer for glyph queries.
type Font struct {
	font *sfnt.Font
	buf  sfnt.Buffer
}

// FontMetrics holds the key vertical metrics of a font at a given size.
type FontMetrics struct {
	Ascent  float64 // pixels above baseline
	Descent float64 // pixels below baseline (positive value)
	Height  float64 // Ascent + Descent
}

// LoadFont reads a TTF file and returns a Font.
func LoadFont(path string) (*Font, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("load font %s: %w", path, err)
	}
	f, err := sfnt.Parse(data)
	if err != nil {
		return nil, fmt.Errorf("parse font %s: %w", path, err)
	}
	return &Font{font: f}, nil
}

// GetOrLoadFont loads a font from the cache, or loads and caches it.
func (c *Cache) GetOrLoadFont(path string) (*Font, error) {
	if v, ok := c.Get(path); ok {
		return v.(*Font), nil
	}
	f, err := LoadFont(path)
	if err != nil {
		return nil, err
	}
	c.Put(path, f)
	return f, nil
}

// Metrics returns the vertical metrics for the font at the given size
// (pixels per em).
func (f *Font) Metrics(size float64) FontMetrics {
	ppem := fixed26_6(size)
	m, _ := f.font.Metrics(&f.buf, ppem, font.HintingNone)
	ascent := float64(m.Ascent) / 64.0
	descent := float64(m.Descent) / 64.0
	// font.Metrics reports descent as a positive value (distance from
	// baseline to bottom of line). Ensure positive.
	if descent < 0 {
		descent = -descent
	}
	return FontMetrics{
		Ascent:  ascent,
		Descent: descent,
		Height:  ascent + descent,
	}
}

// SFNTFont returns the underlying sfnt.Font for glyph access.
func (f *Font) SFNTFont() *sfnt.Font {
	return f.font
}

// Buffer returns a pointer to the reusable sfnt.Buffer.
func (f *Font) Buffer() *sfnt.Buffer {
	return &f.buf
}

// fixed26_6 converts a float64 to a fixed.Int26_6.
func fixed26_6(v float64) fixed.Int26_6 {
	return fixed.Int26_6(v * 64)
}
