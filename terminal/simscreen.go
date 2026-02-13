package terminal

import (
	"flicker/core"
	"strings"
)

// SimScreen is an in-memory Screen for testing. It captures every flushed
// frame as a human-readable text string.
type SimScreen struct {
	width, height int
	frames        []string
}

func NewSimScreen(w, h int) *SimScreen {
	return &SimScreen{width: w, height: h}
}

func (s *SimScreen) Size() (int, int) {
	return s.width, s.height
}

func (s *SimScreen) Flush(canvas *core.Canvas) {
	var b strings.Builder
	for y := 0; y < canvas.Height; y++ {
		if y > 0 {
			b.WriteByte('\n')
		}
		for x := 0; x < canvas.Width; x++ {
			r := canvas.Get(x, y).Rune
			if r == 0 {
				r = ' '
			}
			b.WriteRune(r)
		}
	}
	s.frames = append(s.frames, b.String())
}

func (s *SimScreen) Fini() {}

// Frames returns all captured frames.
func (s *SimScreen) Frames() []string {
	return s.frames
}

// Frame returns the frame at index i, or empty string if out of range.
func (s *SimScreen) Frame(i int) string {
	if i < 0 || i >= len(s.frames) {
		return ""
	}
	return s.frames[i]
}
