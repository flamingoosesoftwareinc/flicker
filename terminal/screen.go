package terminal

import (
	"flicker/core"

	"github.com/gdamore/tcell/v2"
)

type Screen struct {
	tcell tcell.Screen
}

func NewScreen() (*Screen, error) {
	s, err := tcell.NewScreen()
	if err != nil {
		return nil, err
	}
	if err := s.Init(); err != nil {
		return nil, err
	}
	return &Screen{tcell: s}, nil
}

func (s *Screen) Size() (int, int) {
	return s.tcell.Size()
}

func (s *Screen) Flush(canvas *core.Canvas) {
	for y := 0; y < canvas.Height; y++ {
		for x := 0; x < canvas.Width; x++ {
			cell := canvas.Get(x, y)
			r := cell.Rune
			if r == 0 {
				r = ' '
			}
			s.tcell.SetContent(x, y, r, nil, tcell.StyleDefault)
		}
	}
	s.tcell.Show()
}

func (s *Screen) PollEvent() tcell.Event {
	return s.tcell.PollEvent()
}

func (s *Screen) Fini() {
	s.tcell.Fini()
}
