package textfx

import (
	"flicker/asset"
	"flicker/core"
	"flicker/fmath"
)

// StaggeredFadeMaterial returns a material that fades in each character with a delay.
// delayPerChar is the time offset between characters (e.g., 0.15 for 150ms).
// fadeDuration is how long each character takes to fade from 0 to 1 alpha.
func StaggeredFadeMaterial(
	layout *asset.TextLayout,
	encoding Encoding,
	delayPerChar, fadeDuration float64,
) core.Material {
	return func(f core.Fragment) core.Cell {
		glyphIdx := glyphAtForEncoding(layout, encoding, f)
		if glyphIdx < 0 {
			return f.Cell
		}

		delay := float64(glyphIdx) * delayPerChar
		elapsed := f.Time.Total - delay
		alpha := fmath.Clamp(elapsed/fadeDuration, 0, 1)

		cell := f.Cell
		cell.FGAlpha *= alpha
		cell.BGAlpha *= alpha
		return cell
	}
}
