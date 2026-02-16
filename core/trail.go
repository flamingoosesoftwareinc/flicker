package core

import "flicker/fmath"

// Trail shader factory functions that return LayerPreProcess shaders.
// Each factory captures parameters via closure and returns a shader
// that conforms to the Fragment shader interface.

// GhostTrail creates a simple alpha fade trail effect.
// Decay should be < 1.0 (e.g., 0.95 = 5% fade per frame).
func GhostTrail(decay float64) func(Fragment) Cell {
	return func(f Fragment) Cell {
		c := f.Cell
		c.FGAlpha *= decay
		c.BGAlpha *= decay

		// Clear cell when alpha drops too low to avoid black artifacts
		if c.FGAlpha < 0.01 && c.BGAlpha < 0.01 {
			return Cell{} // Return empty cell (true transparency)
		}

		return c
	}
}

// BlurTrail creates a blur/diffusion trail effect by averaging with neighbors.
// Decay controls fade rate, blurAmount controls how much to blend with neighbors.
func BlurTrail(decay, blurAmount float64) func(Fragment) Cell {
	return func(f Fragment) Cell {
		// Sample neighbors
		center := f.Cell
		left := f.Source.Get(f.ScreenX-1, f.ScreenY)
		right := f.Source.Get(f.ScreenX+1, f.ScreenY)
		up := f.Source.Get(f.ScreenX, f.ScreenY-1)
		down := f.Source.Get(f.ScreenX, f.ScreenY+1)

		// Average colors (simple box blur)
		avgR := (uint16(center.FG.R) + uint16(left.FG.R) + uint16(right.FG.R) + uint16(up.FG.R) + uint16(down.FG.R)) / 5
		avgG := (uint16(center.FG.G) + uint16(left.FG.G) + uint16(right.FG.G) + uint16(up.FG.G) + uint16(down.FG.G)) / 5
		avgB := (uint16(center.FG.B) + uint16(left.FG.B) + uint16(right.FG.B) + uint16(up.FG.B) + uint16(down.FG.B)) / 5

		blurred := center
		blurred.FG.R = uint8(avgR)
		blurred.FG.G = uint8(avgG)
		blurred.FG.B = uint8(avgB)

		// Lerp between original and blurred based on blurAmount
		c := center
		c.FG.R = uint8(fmath.Lerp(float64(center.FG.R), float64(blurred.FG.R), blurAmount))
		c.FG.G = uint8(fmath.Lerp(float64(center.FG.G), float64(blurred.FG.G), blurAmount))
		c.FG.B = uint8(fmath.Lerp(float64(center.FG.B), float64(blurred.FG.B), blurAmount))

		c.FGAlpha *= decay
		c.BGAlpha *= decay

		// Clear cell when alpha drops too low
		if c.FGAlpha < 0.01 && c.BGAlpha < 0.01 {
			return Cell{}
		}

		return c
	}
}

// FloatyTrail creates a noise-distorted trail effect where trails drift/warp.
// Decay controls fade, strength controls distortion amount.
func FloatyTrail(decay, strength float64) func(Fragment) Cell {
	return func(f Fragment) Cell {
		// Use Perlin noise to create drifting offset
		noiseX := fmath.Noise2D(float64(f.ScreenX)*0.05, f.Time.Total*0.3)
		noiseY := fmath.Noise2D(float64(f.ScreenY)*0.05+100.0, f.Time.Total*0.3)

		offsetX := int(noiseX * strength)
		offsetY := int(noiseY * strength)

		// Sample from offset position
		c := f.Source.Get(f.ScreenX+offsetX, f.ScreenY+offsetY)
		c.FGAlpha *= decay
		c.BGAlpha *= decay

		// Clear cell when alpha drops too low
		if c.FGAlpha < 0.01 && c.BGAlpha < 0.01 {
			return Cell{}
		}

		return c
	}
}

// GravityTrail creates a downward-drifting trail effect.
// Decay controls fade, fallSpeed controls how fast trails fall (pixels per second).
func GravityTrail(decay, fallSpeed float64) func(Fragment) Cell {
	return func(f Fragment) Cell {
		// Calculate vertical offset based on time
		// Use modulo to create repeating pattern
		offset := int(f.Time.Total*fallSpeed) % 3

		// Sample from above (creates downward drift)
		c := f.Source.Get(f.ScreenX, f.ScreenY-offset)
		c.FGAlpha *= decay
		c.BGAlpha *= decay

		// Clear cell when alpha drops too low
		if c.FGAlpha < 0.01 && c.BGAlpha < 0.01 {
			return Cell{}
		}

		return c
	}
}

// DustTrail creates a dissolving effect where trails turn into dust particles.
// Decay controls fade, dustThreshold controls when transformation occurs.
func DustTrail(decay, dustThreshold float64, dustColor Color) func(Fragment) Cell {
	return func(f Fragment) Cell {
		c := f.Cell

		// As cell fades, convert to dust
		if c.FGAlpha < dustThreshold && c.Rune != ' ' {
			// Use various dust characters based on position for variety
			dustChars := []rune{'·', '∙', '⋅', '•'}
			idx := (f.ScreenX + f.ScreenY) % len(dustChars)
			c.Rune = dustChars[idx]
			c.FG = dustColor
		}

		c.FGAlpha *= decay
		c.BGAlpha *= decay * 0.5 // BG fades faster

		// Clear cell when alpha drops too low
		if c.FGAlpha < 0.01 && c.BGAlpha < 0.01 {
			return Cell{}
		}

		return c
	}
}

// FireTrail creates a fire-like trail that shifts from original color to orange/red.
// Decay controls fade, heatShift controls color transformation speed.
func FireTrail(decay, heatShift float64) func(Fragment) Cell {
	return func(f Fragment) Cell {
		c := f.Cell

		// Shift colors toward orange/red as alpha decreases
		if c.FGAlpha < 0.8 && c.Rune != ' ' {
			// Progress from original → orange → red
			fadeProgress := 1.0 - c.FGAlpha
			c.FG.R = 255
			c.FG.G = uint8(
				fmath.Clamp(
					float64(c.FG.G)*(1.0-fadeProgress*heatShift)+200.0*fadeProgress*heatShift,
					0,
					255,
				),
			)
			c.FG.B = uint8(fmath.Clamp(float64(c.FG.B)*(1.0-fadeProgress*heatShift), 0, 255))
		}

		c.FGAlpha *= decay
		c.BGAlpha *= decay

		// Clear cell when alpha drops too low
		if c.FGAlpha < 0.01 && c.BGAlpha < 0.01 {
			return Cell{}
		}

		return c
	}
}
