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

// DissolveTrail creates a dissolving trail effect where cells randomly break into
// sparse dust particles with subtle drift. Creates an organic dissolve appearance.
func DissolveTrail(decay, dustThreshold float64, dustColor Color) func(Fragment) Cell {
	return func(f Fragment) Cell {
		c := f.Cell

		// Skip empty cells
		if c.Rune == 0 || c.Rune == ' ' {
			return Cell{}
		}

		// Hash-based randomness for sparse, non-clustered dust spawning
		// Using prime numbers for good distribution
		hash := (f.ScreenX*73856093 ^ f.ScreenY*19349663 ^ int(f.Time.Total*100)*83492791) & 0xFFFF
		hashNorm := float64(hash) / 65535.0 // 0.0 to 1.0

		// Very sparse spawning - only ~10-15% of cells become dust
		spawnThreshold := 0.88 + (float64(f.ScreenY)/100.0)*0.05 // Slightly more dust lower down

		// Check if this cell should become dust
		isDust := c.FGAlpha < dustThreshold && hashNorm > spawnThreshold

		if isDust {
			// Subtle turbulent drift (much smaller than before)
			driftX := fmath.Noise2D(float64(f.ScreenX)*0.08, f.Time.Total*0.3) * 1.2
			driftY := fmath.Noise2D(float64(f.ScreenY)*0.08+100.0, f.Time.Total*0.25) * 0.8

			// Sample from slightly offset position for gentle drift
			offsetX := int(driftX)
			offsetY := int(driftY)
			driftedCell := f.Source.Get(f.ScreenX+offsetX, f.ScreenY+offsetY)

			// Only show dust if drifted position had content
			if driftedCell.Rune != 0 && driftedCell.Rune != ' ' {
				// Small dust particle
				c.Rune = '·' // Use smallest dust character only
				c.FG = dustColor
				c.BGAlpha = 0.0

				// Faint and fading fast
				c.FGAlpha = 0.25 + hashNorm*0.15 // 0.25-0.4 range (fainter)
				c.FGAlpha *= 0.7                 // Fast decay
			} else {
				// Drifted into empty space - disappear
				return Cell{}
			}
		} else {
			// Not dust - apply normal decay (slower so trail is visible)
			c.FGAlpha *= decay
			c.BGAlpha *= decay
		}

		// Clear faint cells
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
