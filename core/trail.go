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
		// Slower time evolution (0.1 vs 0.3) and lower spatial frequency (0.02 vs 0.05)
		// for smoother, less steppy transitions
		noiseX := fmath.Noise2D(float64(f.ScreenX)*0.02, f.Time.Total*0.1)
		noiseY := fmath.Noise2D(float64(f.ScreenY)*0.02+100.0, f.Time.Total*0.1)

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
// Decay controls fade, fallSpeed is no longer used (kept for API compatibility).
func GravityTrail(decay, fallSpeed float64) func(Fragment) Cell {
	return func(f Fragment) Cell {
		// Sample from 1 pixel above to create smooth downward flow
		// Each row shows what was in the row above it on the previous frame
		c := f.Source.Get(f.ScreenX, f.ScreenY-1)
		c.FGAlpha *= decay
		c.BGAlpha *= decay

		// Clear cell when alpha drops too low
		if c.FGAlpha < 0.01 && c.BGAlpha < 0.01 {
			return Cell{}
		}

		return c
	}
}

// DustTrail creates a selective dissolving effect where trails selectively spawn
// dust particles that drift and fade. Uses noise for spawning probability,
// turbulence-like offset for active movement, and gradient bias.
func DustTrail(decay, dustThreshold float64, dustColor Color) func(Fragment) Cell {
	return func(f Fragment) Cell {
		c := f.Cell

		// Skip empty cells immediately
		if c.Rune == 0 || c.Rune == ' ' {
			return Cell{}
		}

		// Turbulence-like noise for dust position offset (creates active/wobbly movement)
		turbX := fmath.Noise2D(float64(f.ScreenX)*0.05, f.Time.Total*0.4)
		turbY := fmath.Noise2D(float64(f.ScreenY)*0.05+50.0, f.Time.Total*0.4)

		// Sample from slightly offset position for turbulent appearance
		offsetX := int(turbX * 2.0)
		offsetY := int(turbY * 1.5)
		sourceCell := f.Source.Get(f.ScreenX+offsetX, f.ScreenY+offsetY)

		// If source cell is empty after turbulence offset, fade current cell fast
		if sourceCell.Rune == 0 || sourceCell.Rune == ' ' {
			c.FGAlpha *= 0.7
			c.BGAlpha *= 0.7
			if c.FGAlpha < 0.05 {
				return Cell{}
			}
			return c
		}

		// Use source cell with turbulence
		c = sourceCell

		// Spawn probability based on noise (selective dust spawning)
		spawnNoise := fmath.Noise2D(float64(f.ScreenX)*0.1, float64(f.ScreenY)*0.1+f.Time.Total*0.2)

		// Gradient bias - more dust toward bottom/trailing edge
		verticalBias := float64(f.ScreenY) / 50.0 // Increases downward
		spawnProbability := (spawnNoise + 1.0) / 2.0 * (0.6 + verticalBias*0.4)

		// Only spawn dust if probability is high enough AND cell is fading
		shouldSpawnDust := c.FGAlpha < dustThreshold && spawnProbability > 0.55

		if shouldSpawnDust {
			// Convert to dust particle
			dustChars := []rune{'·', '∙', '⋅', '•', '⋆'}
			idx := (f.ScreenX + f.ScreenY + int(f.Time.Total*10)) % len(dustChars)
			c.Rune = dustChars[idx]
			c.FG = dustColor
			c.BGAlpha = 0.0

			// Dust particles are visible but fade quickly
			c.FGAlpha = 0.4 + spawnNoise*0.3 // 0.4-0.7 range

			// Fast decay for dust
			c.FGAlpha *= 0.75
		} else if c.FGAlpha < dustThreshold {
			// Fading but didn't spawn dust - just clear quickly
			c.FGAlpha *= 0.6
			c.BGAlpha *= 0.6
		} else {
			// Still solid - apply gentle decay
			c.FGAlpha *= decay
			c.BGAlpha *= decay
		}

		// Clear very faint cells
		if c.FGAlpha < 0.05 {
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
