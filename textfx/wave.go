package textfx

import (
	"math"

	"flicker/asset"
	"flicker/core"
	"flicker/core/bitmap"
	"flicker/fmath"
)

// WaveOptions configures the Wave effect.
type WaveOptions struct {
	BasePosition fmath.Vec3 // position of the first character
	Encoding     Encoding   // HalfBlock, Braille, or FullBlock
	Layer        int        // render layer
	Amplitude    float64    // vertical oscillation range in cells
	Frequency    float64    // oscillation speed in Hz
	PhasePerChar float64    // phase offset per character in radians
}

// Wave creates per-character entities that oscillate vertically with phase offsets.
// Returns the created entities for further manipulation if needed.
func Wave(world *core.World, layout *asset.TextLayout, opts WaveOptions) []core.Entity {
	glyphBitmaps := layout.SplitGlyphs()
	entities := make([]core.Entity, len(layout.Glyphs))

	for i, glyph := range layout.Glyphs {
		charEnt := world.Spawn()

		baseX := opts.BasePosition.X + float64(glyph.X)
		baseY := opts.BasePosition.Y

		world.AddTransform(charEnt, &core.Transform{
			Position: fmath.Vec3{X: baseX, Y: baseY},
			Scale:    fmath.Vec3{X: 1, Y: 1, Z: 1},
		})

		// Choose drawable based on encoding.
		var drawable core.Drawable
		switch opts.Encoding {
		case HalfBlock:
			drawable = &bitmap.HalfBlock{Bitmap: glyphBitmaps[i]}
		case Braille:
			drawable = &bitmap.Braille{Bitmap: glyphBitmaps[i]}
		case FullBlock:
			drawable = &bitmap.FullBlock{Bitmap: glyphBitmaps[i]}
		}
		world.AddDrawable(charEnt, drawable)
		world.AddLayer(charEnt, opts.Layer)
		world.AddRoot(charEnt)

		// Wave behavior with phase offset.
		phase := float64(i) * opts.PhasePerChar
		world.AddBehavior(
			charEnt,
			core.NewBehavior(func(t core.Time, e core.Entity, w *core.World) {
				offset := opts.Amplitude * math.Sin(t.Total*opts.Frequency*2*math.Pi+phase)
				w.Transform(e).Position.Y = baseY + offset
			}),
		)

		entities[i] = charEnt
	}

	return entities
}
