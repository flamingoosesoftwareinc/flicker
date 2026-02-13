package terminal

import (
	"fmt"
	"strings"

	"flicker/core"
	"github.com/charmbracelet/x/vt"
)

// SimScreen is an in-memory Screen for testing. It captures every flushed
// frame through a virtual terminal emulator so golden files include ANSI
// color codes.
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
	var buf strings.Builder
	for y := 0; y < canvas.Height; y++ {
		// Position cursor at start of row.
		fmt.Fprintf(&buf, "\x1b[%d;1H", y+1)
		for x := 0; x < canvas.Width; x++ {
			cell := canvas.Get(x, y)
			r := cell.Rune
			if r == 0 {
				r = ' '
			}
			if cell.Alpha > 0 {
				fmt.Fprintf(&buf, "\x1b[38;2;%d;%d;%dm\x1b[48;2;%d;%d;%dm%c\x1b[0m",
					cell.FG.R, cell.FG.G, cell.FG.B,
					cell.BG.R, cell.BG.G, cell.BG.B,
					r)
			} else {
				buf.WriteRune(r)
			}
		}
	}

	term := vt.NewEmulator(canvas.Width, canvas.Height)
	_, _ = term.WriteString(buf.String())
	s.frames = append(s.frames, term.Render())
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
