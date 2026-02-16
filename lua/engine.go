package lua

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"flicker/core"
	lua "github.com/epikur-io/gopher-lua"
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

	// Error log (prints to stderr)
	errors []string
}

// NewEngine creates a new Lua scripting engine.
func NewEngine() *Engine {
	return &Engine{}
}

// Load initializes the Lua VM, registers all modules, and executes the script.
// Returns the scene built from the script's callbacks.
func (e *Engine) Load(scriptPath string, width, height int) (*core.BasicScene, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.scriptPath = scriptPath
	e.defaultCallbacks = SceneCallbacks{}
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

// Close shuts down the Lua VM.
func (e *Engine) Close() {
	e.mu.Lock()
	defer e.mu.Unlock()
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
