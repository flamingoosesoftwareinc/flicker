package lua

import (
	"flicker/core"
	lua "github.com/epikur-io/gopher-lua"
)

// SceneCallbacks holds the Lua callbacks for scene lifecycle.
type SceneCallbacks struct {
	OnEnter  *lua.LFunction
	OnReady  *lua.LFunction
	OnUpdate *lua.LFunction
	OnExit   *lua.LFunction
}

// registerSceneAPI registers flicker.scene() and simple-mode flicker.on_enter/on_update/on_exit.
func registerSceneAPI(L *lua.LState, mod *lua.LTable, engine *Engine) {
	// Simple mode: global callbacks
	L.SetField(mod, "on_enter", L.NewFunction(func(L *lua.LState) int {
		engine.defaultCallbacks.OnEnter = L.CheckFunction(1)
		return 0
	}))
	L.SetField(mod, "on_ready", L.NewFunction(func(L *lua.LState) int {
		engine.defaultCallbacks.OnReady = L.CheckFunction(1)
		return 0
	}))
	L.SetField(mod, "on_update", L.NewFunction(func(L *lua.LState) int {
		engine.defaultCallbacks.OnUpdate = L.CheckFunction(1)
		return 0
	}))
	L.SetField(mod, "on_exit", L.NewFunction(func(L *lua.LState) int {
		engine.defaultCallbacks.OnExit = L.CheckFunction(1)
		return 0
	}))

	// Trail effects sub-table
	registerTrailModule(L, mod)
}

// buildScene creates a core.BasicScene wired to the Lua callbacks.
func buildScene(
	L *lua.LState,
	engine *Engine,
	cb *SceneCallbacks,
	width, height int,
) *core.BasicScene {
	scene := core.NewBasicScene(width, height)

	scene.SetEnter(func(w *core.World, ctx core.SceneContext) {
		// Update active scene so f.set_blend/set_trail/set_post_process
		// target the correct compositor in multi-scene mode.
		engine.activeScene = scene

		if cb.OnEnter == nil {
			return
		}
		pushWorld(L, w)
		worldUD := L.Get(-1)
		L.Pop(1)

		ctxTable := L.NewTable()
		L.SetField(ctxTable, "width", lua.LNumber(ctx.Width))
		L.SetField(ctxTable, "height", lua.LNumber(ctx.Height))

		if err := L.CallByParam(lua.P{
			Fn:      cb.OnEnter,
			NRet:    0,
			Protect: true,
		}, worldUD, ctxTable); err != nil {
			engine.logError("on_enter", err)
		}
	})

	scene.SetReady(func(w *core.World) {
		if cb.OnReady == nil {
			return
		}
		pushWorld(L, w)
		worldUD := L.Get(-1)
		L.Pop(1)

		if err := L.CallByParam(lua.P{
			Fn:      cb.OnReady,
			NRet:    0,
			Protect: true,
		}, worldUD); err != nil {
			engine.logError("on_ready", err)
		}
	})

	scene.SetUpdate(func(w *core.World, t core.Time) {
		if cb.OnUpdate == nil {
			return
		}
		pushWorld(L, w)
		worldUD := L.Get(-1)
		L.Pop(1)

		timeTable := L.NewTable()
		L.SetField(timeTable, "total", lua.LNumber(t.Total))
		L.SetField(timeTable, "delta", lua.LNumber(t.Delta))

		if err := L.CallByParam(lua.P{
			Fn:      cb.OnUpdate,
			NRet:    0,
			Protect: true,
		}, worldUD, timeTable); err != nil {
			engine.logError("on_update", err)
		}
	})

	scene.SetExit(func(w *core.World) {
		if cb.OnExit == nil {
			return
		}
		pushWorld(L, w)
		worldUD := L.Get(-1)
		L.Pop(1)

		if err := L.CallByParam(lua.P{
			Fn:      cb.OnExit,
			NRet:    0,
			Protect: true,
		}, worldUD); err != nil {
			engine.logError("on_exit", err)
		}
	})

	return scene
}
