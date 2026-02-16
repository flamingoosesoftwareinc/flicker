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

	// Start hot reload file watcher
	if watchErr := engine.WatchForChanges(); watchErr != nil {
		fmt.Fprintf(os.Stderr, "warning: hot reload disabled: %v\n", watchErr)
	}

	// Check if the script created a scene manager during Load (top-level)
	sm := engine.SceneManager()
	multiScene := sm != nil

	if !multiScene {
		// Single scene mode - call OnEnter which may create a scene manager
		scene.OnEnter(core.SceneContext{Width: sw, Height: sh})

		// Re-check: on_enter may have created a scene manager
		sm = engine.SceneManager()
		multiScene = sm != nil

		if !multiScene {
			scene.OnReady()
		}
	}

	// Pump PollEvent in a goroutine so the tick loop never blocks on input
	events := make(chan tcell.Event, 1)
	go func() {
		for {
			events <- screen.PollEvent()
		}
	}()

	// doReload performs a full teardown and reload of the script.
	doReload := func() {
		if !multiScene {
			scene.OnExit()
		}
		newScene, reloadErr := engine.Reload(sw, sh)
		if reloadErr != nil {
			fmt.Fprintf(os.Stderr, "reload error: %v\n", reloadErr)
			return
		}

		sm = engine.SceneManager()
		multiScene = sm != nil
		if !multiScene {
			scene = newScene
			scene.OnEnter(core.SceneContext{Width: sw, Height: sh})

			sm = engine.SceneManager()
			multiScene = sm != nil

			if !multiScene {
				scene.OnReady()
			}
		}
	}

	var simTime float64
	last := time.Now()

	for {
		// Drain events (non-blocking)
		quit := false
		reload := false
		nextSlide := false
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
					case kev.Rune() == ' ':
						nextSlide = true
					}
				}
			default:
				draining = false
			}
		}

		// Check for file-watcher-triggered reload (non-blocking)
		select {
		case <-engine.NeedsReload():
			reload = true
		default:
		}

		if quit {
			if multiScene {
				// SceneManager handles cleanup
			} else {
				scene.OnExit()
			}
			return
		}
		if reload {
			doReload()
		}
		if nextSlide && multiScene && !sm.IsTransitioning() {
			sm.Next(core.CrossFade, 1.5)
		}

		now := time.Now()
		wallDelta := now.Sub(last).Seconds()
		last = now
		simTime += wallDelta

		t := core.Time{
			Total: simTime,
			Delta: wallDelta,
		}

		canvas.Clear()
		canvas.DrawBorder()

		if multiScene {
			sm.Update(t)
			sm.Render(canvas, t)
		} else {
			scene.OnUpdate(t)
			scene.Render(canvas, t)
		}

		screen.Flush(canvas)
	}
}
