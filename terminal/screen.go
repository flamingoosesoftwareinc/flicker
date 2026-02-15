package terminal

import (
	"flicker/core"
	"github.com/gdamore/tcell/v2"
)

// Screen is the minimal interface a renderer needs to present frames.
type Screen interface {
	Size() (int, int)
	Flush(canvas *core.Canvas)
	Fini()
}

// TcellScreen is a Screen backed by a real tcell terminal.
type TcellScreen struct {
	tcell tcell.Screen
}

func NewTcellScreen() (*TcellScreen, error) {
	s, err := tcell.NewScreen()
	if err != nil {
		return nil, err
	}
	if err := s.Init(); err != nil {
		return nil, err
	}
	return &TcellScreen{tcell: s}, nil
}

func (s *TcellScreen) Size() (int, int) {
	return s.tcell.Size()
}

func (s *TcellScreen) Flush(canvas *core.Canvas) {
	for y := 0; y < canvas.Height; y++ {
		for x := 0; x < canvas.Width; x++ {
			cell := canvas.Get(x, y)
			r := cell.Rune
			if r == 0 {
				r = ' '
			}
			style := tcell.StyleDefault
			if cell.FGAlpha > 0 {
				style = style.Foreground(
					tcell.NewRGBColor(int32(cell.FG.R), int32(cell.FG.G), int32(cell.FG.B)),
				)
			}
			if cell.BGAlpha > 0 {
				style = style.Background(
					tcell.NewRGBColor(int32(cell.BG.R), int32(cell.BG.G), int32(cell.BG.B)),
				)
			}
			s.tcell.SetContent(x, y, r, nil, style)
		}
	}
	s.tcell.Show()
}

func (s *TcellScreen) PollEvent() tcell.Event {
	return s.tcell.PollEvent()
}

func (s *TcellScreen) Fini() {
	s.tcell.Fini()
}
