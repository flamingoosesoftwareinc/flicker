package particle

import (
	"math"

	"flicker/core"
	"flicker/fmath"
)

// ColorGradient defines a color transition based on speed.
type ColorGradient struct {
	MinSpeed float64
	MaxSpeed float64
	MinColor core.Color // slow
	MaxColor core.Color // fast
}

// VelocityColor changes the foreground color based on velocity magnitude (speed).
// Interpolates between MinColor (slow) and MaxColor (fast) based on speed.
// Returns the original cell if the entity has no Body component.
func VelocityColor(gradient ColorGradient) core.Material {
	return func(f core.Fragment) core.Cell {
		body := f.World.Body(f.Entity)
		if body == nil {
			return f.Cell // nil guard
		}

		speed := math.Sqrt(body.Velocity.X*body.Velocity.X + body.Velocity.Y*body.Velocity.Y)
		t := fmath.Clamp((speed-gradient.MinSpeed)/(gradient.MaxSpeed-gradient.MinSpeed), 0, 1)

		cell := f.Cell
		cell.FG = lerpColor(gradient.MinColor, gradient.MaxColor, t)
		return cell
	}
}

// IdleAndMotion cycles through idle runes when velocity is below threshold,
// switches to directional Braille when moving above threshold.
// Returns the original cell if the entity has no Body component.
func IdleAndMotion(idleRunes []rune, motionThreshold float64) core.Material {
	return func(f core.Fragment) core.Cell {
		body := f.World.Body(f.Entity)
		if body == nil {
			return f.Cell // nil guard
		}

		speed := math.Sqrt(body.Velocity.X*body.Velocity.X + body.Velocity.Y*body.Velocity.Y)

		cell := f.Cell
		if speed < motionThreshold {
			// Idle: cycle through runes based on time
			idx := int(f.Time.Total*4.0) % len(idleRunes)
			cell.Rune = idleRunes[idx]
		} else {
			// Motion: directional Braille
			angle := math.Atan2(body.Velocity.Y, body.Velocity.X)
			cell.Rune = brailleForAngle(angle)
		}

		return cell
	}
}

// BrailleDirectional maps velocity direction to one of 8 Braille patterns
// forming directional lines/arrows. Returns a default dot if the entity has
// no Body component.
func BrailleDirectional() core.Material {
	return func(f core.Fragment) core.Cell {
		body := f.World.Body(f.Entity)
		cell := f.Cell

		if body == nil {
			cell.Rune = '·' // nil guard: default rune
			return cell
		}

		// Map angle to 8 directions
		angle := math.Atan2(body.Velocity.Y, body.Velocity.X)
		cell.Rune = brailleForAngle(angle)
		return cell
	}
}

// SpeedStates changes rune based on multiple speed thresholds.
// Returns the first rune whose threshold is not exceeded, or the last rune
// if all thresholds are exceeded. Returns a space if the entity has no Body component.
func SpeedStates(thresholds []float64, runes []rune) core.Material {
	return func(f core.Fragment) core.Cell {
		body := f.World.Body(f.Entity)
		cell := f.Cell

		if body == nil {
			cell.Rune = ' ' // nil guard
			return cell
		}

		speed := math.Sqrt(body.Velocity.X*body.Velocity.X + body.Velocity.Y*body.Velocity.Y)

		for i, threshold := range thresholds {
			if speed < threshold {
				cell.Rune = runes[i]
				return cell
			}
		}

		cell.Rune = runes[len(runes)-1]
		return cell
	}
}

// AgeBasedSize changes rune based on particle age (particles grow over time).
// Returns the first rune whose age threshold is not exceeded, or the last rune
// if all thresholds are exceeded. Returns the first rune if the entity has no Age component.
func AgeBasedSize(ageThresholds []float64, runes []rune) core.Material {
	return func(f core.Fragment) core.Cell {
		age := f.World.Age(f.Entity)
		cell := f.Cell

		if age == nil {
			cell.Rune = runes[0] // nil guard: default to first rune
			return cell
		}

		for i, threshold := range ageThresholds {
			if age.Age < threshold {
				cell.Rune = runes[i]
				return cell
			}
		}

		cell.Rune = runes[len(runes)-1]
		return cell
	}
}

// RainbowVelocity changes color through HSV spectrum based on speed.
// Cycles hue from red → orange → yellow → green → cyan → blue → magenta → red
// as speed increases from minSpeed to maxSpeed.
func RainbowVelocity(minSpeed, maxSpeed float64) core.Material {
	return func(f core.Fragment) core.Cell {
		body := f.World.Body(f.Entity)
		if body == nil {
			return f.Cell
		}

		speed := math.Sqrt(body.Velocity.X*body.Velocity.X + body.Velocity.Y*body.Velocity.Y)
		t := fmath.Clamp((speed-minSpeed)/(maxSpeed-minSpeed), 0, 1)

		// Map t to hue: 0 = red, 1 = red (full spectrum)
		hue := t * 360.0

		cell := f.Cell
		cell.FG = hsvToRGB(hue, 1.0, 1.0)
		return cell
	}
}

// RainbowTime cycles color through HSV spectrum based on time.
// Each entity gets a different phase offset based on its ID for color variety.
// frequency controls how fast colors cycle (in Hz).
func RainbowTime(frequency float64) core.Material {
	return func(f core.Fragment) core.Cell {
		// Use entity ID as phase offset so each particle has different color
		phase := float64(f.Entity) * 0.1

		// Cycle hue based on time + phase
		hue := math.Mod((f.Time.Total*frequency*360.0)+phase*360.0, 360.0)

		cell := f.Cell
		cell.FG = hsvToRGB(hue, 1.0, 1.0)
		return cell
	}
}

// lerpColor linearly interpolates between two colors.
func lerpColor(a, b core.Color, t float64) core.Color {
	return core.Color{
		R: uint8(fmath.Lerp(float64(a.R), float64(b.R), t)),
		G: uint8(fmath.Lerp(float64(a.G), float64(b.G), t)),
		B: uint8(fmath.Lerp(float64(a.B), float64(b.B), t)),
	}
}

// hsvToRGB converts HSV color space to RGB.
// h: hue in degrees [0, 360], s: saturation [0, 1], v: value/brightness [0, 1]
func hsvToRGB(h, s, v float64) core.Color {
	// Normalize hue to [0, 360)
	for h < 0 {
		h += 360
	}
	for h >= 360 {
		h -= 360
	}

	c := v * s
	x := c * (1 - math.Abs(math.Mod(h/60.0, 2)-1))
	m := v - c

	var r, g, b float64
	switch {
	case h < 60:
		r, g, b = c, x, 0
	case h < 120:
		r, g, b = x, c, 0
	case h < 180:
		r, g, b = 0, c, x
	case h < 240:
		r, g, b = 0, x, c
	case h < 300:
		r, g, b = x, 0, c
	default:
		r, g, b = c, 0, x
	}

	return core.Color{
		R: uint8((r + m) * 255),
		G: uint8((g + m) * 255),
		B: uint8((b + m) * 255),
	}
}

// brailleForAngle maps an angle (in radians) to one of 8 directional Braille patterns.
func brailleForAngle(angle float64) rune {
	// 8 cardinal directions
	patterns := []rune{
		'⠤', // E  (horizontal right)
		'⠡', // SE (diagonal down-right)
		'⡇', // S  (vertical down)
		'⢇', // SW (diagonal down-left)
		'⠒', // W  (horizontal left)
		'⠊', // NW (diagonal up-left)
		'⡀', // N  (vertical up)
		'⠈', // NE (diagonal up-right)
	}

	// Normalize angle to [0, 2π)
	if angle < 0 {
		angle += 2 * math.Pi
	}

	// Map to 8 directions
	idx := int((angle + math.Pi/8) / (math.Pi / 4))
	idx = idx % 8
	return patterns[idx]
}
