package lua

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"flicker/core"
	lua "github.com/epikur-io/gopher-lua"
	"github.com/fsnotify/fsnotify"
)

// Engine manages the Lua VM and bridges it to the flicker rendering engine.
type Engine struct {
	L          *lua.LState
	scriptPath string
	mu         sync.Mutex

	// Scene built from Lua callbacks
	activeScene      *core.BasicScene
	defaultCallbacks SceneCallbacks

	// Multi-scene support
	sceneManager *core.SceneManager

	// Hot reload
	watcher    *fsnotify.Watcher
	reloadChan chan struct{} // signals that a reload is needed
	stopWatch  chan struct{} // signals the watcher goroutine to stop

	// Error log (prints to stderr)
	errors []string
}

// NewEngine creates a new Lua scripting engine.
func NewEngine() *Engine {
	return &Engine{
		reloadChan: make(chan struct{}, 1),
		stopWatch:  make(chan struct{}),
	}
}

// Load initializes the Lua VM, registers all modules, and executes the script.
// Returns the scene built from the script's callbacks.
func (e *Engine) Load(scriptPath string, width, height int) (*core.BasicScene, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.scriptPath = scriptPath
	e.defaultCallbacks = SceneCallbacks{}
	e.sceneManager = nil
	e.errors = nil

	// Create new Lua state
	L := lua.NewState()
	e.L = L

	// Set package path to script directory
	dir := filepath.Dir(scriptPath)
	pkg := L.GetField(L.Get(lua.EnvironIndex), "package")
	L.SetField(pkg, "path", lua.LString(
		filepath.Join(dir, "?.lua")+";"+filepath.Join(dir, "?", "init.lua"),
	))

	// Register flicker module
	mod := registerAll(L, e)
	L.PreloadModule("flicker", func(L *lua.LState) int {
		L.Push(mod)
		return 1
	})

	// Execute script
	if err := L.DoFile(scriptPath); err != nil {
		return nil, fmt.Errorf("lua script error: %w", err)
	}

	// Build scene from collected callbacks
	scene := buildScene(L, e, &e.defaultCallbacks, width, height)
	return scene, nil
}

// SceneManager returns the scene manager if one was created by the script.
func (e *Engine) SceneManager() *core.SceneManager {
	return e.sceneManager
}

// Reload tears down the current VM and re-executes the script.
func (e *Engine) Reload(width, height int) (*core.BasicScene, error) {
	e.mu.Lock()
	if e.L != nil {
		e.L.Close()
		e.L = nil
	}
	e.activeScene = nil
	e.mu.Unlock()

	return e.Load(e.scriptPath, width, height)
}

// WatchForChanges starts watching the script file and its directory for changes.
// When a change is detected (debounced by 100ms), a signal is sent on the
// channel returned by NeedsReload().
func (e *Engine) WatchForChanges() error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("fsnotify: %w", err)
	}
	e.watcher = watcher

	// Watch the script file's directory (covers new/renamed files too)
	dir := filepath.Dir(e.scriptPath)
	if err := watcher.Add(dir); err != nil {
		_ = watcher.Close()
		return fmt.Errorf("watch %s: %w", dir, err)
	}

	go e.watchLoop()
	return nil
}

// NeedsReload returns a channel that receives a signal when the script
// file has changed and a reload is needed.
func (e *Engine) NeedsReload() <-chan struct{} {
	return e.reloadChan
}

func (e *Engine) watchLoop() {
	var debounce *time.Timer

	for {
		select {
		case event, ok := <-e.watcher.Events:
			if !ok {
				return
			}
			// Only care about .lua files being written or created
			if filepath.Ext(event.Name) != ".lua" {
				continue
			}
			if !event.Has(fsnotify.Write) && !event.Has(fsnotify.Create) {
				continue
			}

			// Debounce: reset timer on each event
			if debounce != nil {
				debounce.Stop()
			}
			debounce = time.AfterFunc(100*time.Millisecond, func() {
				// Non-blocking send
				select {
				case e.reloadChan <- struct{}{}:
				default:
				}
			})

		case err, ok := <-e.watcher.Errors:
			if !ok {
				return
			}
			fmt.Fprintf(os.Stderr, "[flicker-lua] watch error: %v\n", err)

		case <-e.stopWatch:
			return
		}
	}
}

// Close shuts down the Lua VM and file watcher.
func (e *Engine) Close() {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.watcher != nil {
		close(e.stopWatch)
		_ = e.watcher.Close()
		e.watcher = nil
	}
	if e.L != nil {
		e.L.Close()
		e.L = nil
	}
}

func (e *Engine) logError(context string, err error) {
	msg := fmt.Sprintf("[flicker-lua] %s: %v", context, err)
	e.errors = append(e.errors, msg)
	fmt.Fprintln(os.Stderr, msg)
}
