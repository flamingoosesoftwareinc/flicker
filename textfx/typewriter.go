package textfx

import (
	"math"

	"flicker/asset"
	"flicker/core"
)

// TypewriterMaterial returns a material that reveals characters left-to-right.
// charsRevealed should be a pointer to a float64 that a behavior animates.
func TypewriterMaterial(
	layout *asset.TextLayout,
	encoding Encoding,
	charsRevealed *float64,
) core.Material {
	return func(f core.Fragment) core.Cell {
		glyphIdx := glyphAtForEncoding(layout, encoding, f)
		if glyphIdx < 0 || float64(glyphIdx) >= *charsRevealed {
			return core.Cell{}
		}
		return f.Cell
	}
}

// TypewriterBehavior returns a behavior that animates charsRevealed at charsPerSec.
func TypewriterBehavior(charsRevealed *float64, charsPerSec float64, maxChars int) core.Behavior {
	return core.NewBehavior(func(t core.Time, _ core.Entity, _ *core.World) {
		*charsRevealed = math.Min(float64(maxChars), t.Total*charsPerSec)
	})
}
