package core

import "flicker/fmath"

// TransitionFunc renders a transition between two scenes.
// progress is in [0, 1] where 0 = fully from, 1 = fully to.
type TransitionFunc func(from, to *Canvas, dst *Canvas, progress float64)

// Transition manages state for a scene-to-scene transition.
type Transition struct {
	From     Scene
	To       Scene
	Duration float64
	Elapsed  float64
	Func     TransitionFunc

	// Scratch canvases for rendering from/to scenes
	fromCanvas *Canvas
	toCanvas   *Canvas
}

// NewTransition creates a transition between two scenes.
func NewTransition(from, to Scene, duration float64, fn TransitionFunc) *Transition {
	return &Transition{
		From:     from,
		To:       to,
		Duration: duration,
		Func:     fn,
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

	// Apply transition function
	t.Func(t.fromCanvas, t.toCanvas, dst, t.Progress())
}

// CrossFade is a simple alpha blend transition.
func CrossFade(from, to *Canvas, dst *Canvas, progress float64) {
	for y := range dst.Height {
		for x := range dst.Width {
			fromCell := from.Get(x, y)
			toCell := to.Get(x, y)

			// Fade from's alpha down, to's alpha up
			fromCell.FGAlpha *= (1.0 - progress)
			fromCell.BGAlpha *= (1.0 - progress)
			toCell.FGAlpha *= progress
			toCell.BGAlpha *= progress

			// Composite to over from with normal blending
			dst.Set(x, y, BlendCell(fromCell, toCell, NormalColorBlend))
		}
	}
}

// WipeLeft is a left-to-right wipe transition.
func WipeLeft(from, to *Canvas, dst *Canvas, progress float64) {
	threshold := int(float64(dst.Width) * progress)
	for y := range dst.Height {
		for x := range dst.Width {
			if x < threshold {
				dst.Set(x, y, to.Get(x, y))
			} else {
				dst.Set(x, y, from.Get(x, y))
			}
		}
	}
}

// WipeRight is a right-to-left wipe transition.
func WipeRight(from, to *Canvas, dst *Canvas, progress float64) {
	threshold := int(float64(dst.Width) * (1.0 - progress))
	for y := range dst.Height {
		for x := range dst.Width {
			if x >= threshold {
				dst.Set(x, y, to.Get(x, y))
			} else {
				dst.Set(x, y, from.Get(x, y))
			}
		}
	}
}

// WipeUp is a bottom-to-top wipe transition.
func WipeUp(from, to *Canvas, dst *Canvas, progress float64) {
	threshold := int(float64(dst.Height) * (1.0 - progress))
	for y := range dst.Height {
		for x := range dst.Width {
			if y >= threshold {
				dst.Set(x, y, to.Get(x, y))
			} else {
				dst.Set(x, y, from.Get(x, y))
			}
		}
	}
}

// WipeDown is a top-to-bottom wipe transition.
func WipeDown(from, to *Canvas, dst *Canvas, progress float64) {
	threshold := int(float64(dst.Height) * progress)
	for y := range dst.Height {
		for x := range dst.Width {
			if y < threshold {
				dst.Set(x, y, to.Get(x, y))
			} else {
				dst.Set(x, y, from.Get(x, y))
			}
		}
	}
}

// PushLeft slides the new scene in from the right, pushing the old scene left.
func PushLeft(from, to *Canvas, dst *Canvas, progress float64) {
	offset := int(float64(dst.Width) * progress)
	for y := range dst.Height {
		for x := range dst.Width {
			fromX := x + offset
			toX := x + offset - dst.Width

			if toX >= 0 && toX < dst.Width {
				dst.Set(x, y, to.Get(toX, y))
			} else if fromX >= 0 && fromX < dst.Width {
				dst.Set(x, y, from.Get(fromX, y))
			}
		}
	}
}

// PushRight slides the new scene in from the left, pushing the old scene right.
func PushRight(from, to *Canvas, dst *Canvas, progress float64) {
	offset := int(float64(dst.Width) * progress)
	for y := range dst.Height {
		for x := range dst.Width {
			fromX := x - offset
			toX := x - offset + dst.Width

			if toX >= 0 && toX < dst.Width {
				dst.Set(x, y, to.Get(toX, y))
			} else if fromX >= 0 && fromX < dst.Width {
				dst.Set(x, y, from.Get(fromX, y))
			}
		}
	}
}
