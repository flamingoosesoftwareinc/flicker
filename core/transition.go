package core

import (
	"math"

	"flicker/fmath"
)

// TransitionFragment provides context for a transition fragment shader.
type TransitionFragment struct {
	X, Y       int     // Screen coordinates
	FromCanvas *Canvas // Old scene canvas (for sampling)
	ToCanvas   *Canvas // New scene canvas (for sampling)
	Progress   float64 // Transition progress [0, 1]
	Time       Time    // Current time
}

// TransitionShader is a per-pixel shader that composites two scenes.
// Returns the final cell color for the given fragment.
type TransitionShader func(f TransitionFragment) Cell

// Transition manages state for a scene-to-scene transition.
type Transition struct {
	From     Scene
	To       Scene
	Duration float64
	Elapsed  float64
	Shader   TransitionShader

	// Scratch canvases for rendering from/to scenes
	fromCanvas *Canvas
	toCanvas   *Canvas
}

// NewTransition creates a transition between two scenes.
func NewTransition(from, to Scene, duration float64, shader TransitionShader) *Transition {
	return &Transition{
		From:     from,
		To:       to,
		Duration: duration,
		Shader:   shader,
	}
}

// Update advances the transition and returns true when complete.
func (t *Transition) Update(dt float64) bool {
	t.Elapsed += dt
	return t.Elapsed >= t.Duration
}

// Progress returns the transition progress in [0, 1].
func (t *Transition) Progress() float64 {
	if t.Duration == 0 {
		return 1.0
	}
	return fmath.Clamp(t.Elapsed/t.Duration, 0, 1)
}

// Render renders the transition to dst canvas.
func (t *Transition) Render(dst *Canvas, time Time) {
	// Lazy init scratch canvases
	if t.fromCanvas == nil {
		t.fromCanvas = NewCanvas(dst.Width, dst.Height)
		t.toCanvas = NewCanvas(dst.Width, dst.Height)
	}

	// Render both scenes to scratch canvases
	t.fromCanvas.Clear()
	t.From.Render(t.fromCanvas, time)

	t.toCanvas.Clear()
	t.To.Render(t.toCanvas, time)

	// Apply transition shader per-pixel
	progress := t.Progress()
	for y := range dst.Height {
		for x := range dst.Width {
			frag := TransitionFragment{
				X:          x,
				Y:          y,
				FromCanvas: t.fromCanvas,
				ToCanvas:   t.toCanvas,
				Progress:   progress,
				Time:       time,
			}
			dst.Set(x, y, t.Shader(frag))
		}
	}
}

// CrossFade is a simple alpha blend transition shader.
// Directly interpolates colors and alphas for a true cross-fade effect.
func CrossFade(f TransitionFragment) Cell {
	fromCell := f.FromCanvas.Get(f.X, f.Y)
	toCell := f.ToCanvas.Get(f.X, f.Y)

	// Lerp colors
	fg := lerpColor(fromCell.FG, toCell.FG, f.Progress)
	bg := lerpColor(fromCell.BG, toCell.BG, f.Progress)

	// Lerp alphas
	fgAlpha := fromCell.FGAlpha*(1-f.Progress) + toCell.FGAlpha*f.Progress
	bgAlpha := fromCell.BGAlpha*(1-f.Progress) + toCell.BGAlpha*f.Progress

	// Choose rune based on progress (sharp transition at 50%)
	rune := fromCell.Rune
	if f.Progress > 0.5 {
		rune = toCell.Rune
	}

	return Cell{
		FG:      fg,
		BG:      bg,
		Rune:    rune,
		FGAlpha: fgAlpha,
		BGAlpha: bgAlpha,
	}
}

// lerpColor linearly interpolates between two colors.
func lerpColor(a, b Color, t float64) Color {
	return Color{
		R: uint8(float64(a.R)*(1-t) + float64(b.R)*t),
		G: uint8(float64(a.G)*(1-t) + float64(b.G)*t),
		B: uint8(float64(a.B)*(1-t) + float64(b.B)*t),
	}
}

// WipeLeft is a left-to-right wipe transition shader.
func WipeLeft(f TransitionFragment) Cell {
	threshold := int(float64(f.FromCanvas.Width) * f.Progress)
	if f.X < threshold {
		return f.ToCanvas.Get(f.X, f.Y)
	}
	return f.FromCanvas.Get(f.X, f.Y)
}

// WipeRight is a right-to-left wipe transition shader.
func WipeRight(f TransitionFragment) Cell {
	threshold := int(float64(f.FromCanvas.Width) * (1.0 - f.Progress))
	if f.X >= threshold {
		return f.ToCanvas.Get(f.X, f.Y)
	}
	return f.FromCanvas.Get(f.X, f.Y)
}

// WipeUp is a bottom-to-top wipe transition shader.
func WipeUp(f TransitionFragment) Cell {
	threshold := int(float64(f.FromCanvas.Height) * (1.0 - f.Progress))
	if f.Y >= threshold {
		return f.ToCanvas.Get(f.X, f.Y)
	}
	return f.FromCanvas.Get(f.X, f.Y)
}

// WipeDown is a top-to-bottom wipe transition shader.
func WipeDown(f TransitionFragment) Cell {
	threshold := int(float64(f.FromCanvas.Height) * f.Progress)
	if f.Y < threshold {
		return f.ToCanvas.Get(f.X, f.Y)
	}
	return f.FromCanvas.Get(f.X, f.Y)
}

// PushLeft slides the new scene in from the right, pushing the old scene left.
func PushLeft(f TransitionFragment) Cell {
	offset := int(float64(f.FromCanvas.Width) * f.Progress)
	fromX := f.X + offset
	toX := f.X + offset - f.FromCanvas.Width

	if toX >= 0 && toX < f.ToCanvas.Width {
		return f.ToCanvas.Get(toX, f.Y)
	} else if fromX >= 0 && fromX < f.FromCanvas.Width {
		return f.FromCanvas.Get(fromX, f.Y)
	}
	return Cell{} // Empty cell if out of bounds
}

// PushRight slides the new scene in from the left, pushing the old scene right.
func PushRight(f TransitionFragment) Cell {
	offset := int(float64(f.FromCanvas.Width) * f.Progress)
	fromX := f.X - offset
	toX := f.X - offset + f.FromCanvas.Width

	if toX >= 0 && toX < f.ToCanvas.Width {
		return f.ToCanvas.Get(toX, f.Y)
	} else if fromX >= 0 && fromX < f.FromCanvas.Width {
		return f.FromCanvas.Get(fromX, f.Y)
	}
	return Cell{} // Empty cell if out of bounds
}

// RadialWipe reveals the new scene in a circle expanding from the center.
func RadialWipe(f TransitionFragment) Cell {
	centerX := float64(f.FromCanvas.Width) / 2.0
	centerY := float64(f.FromCanvas.Height) / 2.0

	dx := float64(f.X) - centerX
	dy := float64(f.Y) - centerY
	distance := math.Sqrt(dx*dx + dy*dy)

	maxDistance := math.Sqrt(centerX*centerX + centerY*centerY)
	threshold := maxDistance * f.Progress

	if distance < threshold {
		return f.ToCanvas.Get(f.X, f.Y)
	}
	return f.FromCanvas.Get(f.X, f.Y)
}

// Pixelate creates a pixelated dissolve effect.
func Pixelate(f TransitionFragment) Cell {
	// Pixelate in increasing block sizes, then reveal
	blockSize := int(1.0 + (1.0-f.Progress)*8.0)
	if blockSize < 1 {
		blockSize = 1
	}

	blockX := (f.X / blockSize) * blockSize
	blockY := (f.Y / blockSize) * blockSize

	// Sample from block corner
	if f.Progress > 0.7 {
		// Final 30%: reveal new scene
		return f.ToCanvas.Get(f.X, f.Y)
	} else if f.Progress > 0.3 {
		// Middle: pixelated new scene
		return f.ToCanvas.Get(blockX, blockY)
	}
	// Start: pixelated old scene
	return f.FromCanvas.Get(blockX, blockY)
}
