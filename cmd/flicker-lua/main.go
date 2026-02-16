package main

import (
	"fmt"
	"os"
	"time"

	"flicker/core"
	flickerlua "flicker/lua"
	"flicker/terminal"
	"github.com/gdamore/tcell/v2"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: flicker-lua <script.lua>\n")
		os.Exit(1)
	}
	scriptPath := os.Args[1]

	screen, err := terminal.NewTcellScreen()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	defer screen.Fini()

	sw, sh := screen.Size()
	canvas := core.NewCanvas(sw, sh)

	// Create Lua engine and load script
	engine := flickerlua.NewEngine()
	defer engine.Close()

	scene, err := engine.Load(scriptPath, sw, sh)
	if err != nil {
		screen.Fini()
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	// Start the scene
	scene.OnEnter(core.SceneContext{Width: sw, Height: sh})
	scene.OnReady()

	// Pump PollEvent in a goroutine so the tick loop never blocks on input
	events := make(chan tcell.Event, 1)
	go func() {
		for {
			events <- screen.PollEvent()
		}
	}()

	var simTime float64
	last := time.Now()

	for {
		// Drain events (non-blocking)
		quit := false
		reload := false
		for draining := true; draining; {
			select {
			case ev := <-events:
				if kev, ok := ev.(*tcell.EventKey); ok {
					switch {
					case kev.Key() == tcell.KeyEscape:
						quit = true
					case kev.Rune() == 'q':
						quit = true
					case kev.Rune() == 'r':
						reload = true
					}
				}
			default:
				draining = false
			}
		}
		if quit {
			scene.OnExit()
			return
		}
		if reload {
			scene.OnExit()
			newScene, err := engine.Reload(sw, sh)
			if err != nil {
				fmt.Fprintf(os.Stderr, "reload error: %v\n", err)
			} else {
				scene = newScene
				scene.OnEnter(core.SceneContext{Width: sw, Height: sh})
				scene.OnReady()
			}
		}

		now := time.Now()
		wallDelta := now.Sub(last).Seconds()
		last = now
		simTime += wallDelta

		t := core.Time{
			Total: simTime,
			Delta: wallDelta,
		}

		scene.OnUpdate(t)

		canvas.Clear()
		canvas.DrawBorder()
		scene.Render(canvas, t)
		screen.Flush(canvas)
	}
}
