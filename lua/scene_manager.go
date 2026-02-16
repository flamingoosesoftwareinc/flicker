package lua

import (
	"flicker/core"
	lua "github.com/epikur-io/gopher-lua"
)

const typeSceneManager = "flicker.scene_manager"

func registerSceneManagerModule(L *lua.LState, mod *lua.LTable, engine *Engine) {
	// SceneManager metatable
	mt := registerType(L, typeSceneManager, map[string]lua.LGFunction{
		"add":              smAdd,
		"start":            smStart,
		"next":             smNext,
		"previous":         smPrevious,
		"goto":             smGoTo,
		"current":          smCurrent,
		"count":            smCount,
		"is_transitioning": smIsTransitioning,
	})
	L.SetField(mt, "__index", L.GetField(mt, "methods"))

	// Transition shaders sub-table
	tr := L.NewTable()
	L.SetField(mod, "transition", tr)

	L.SetField(tr, "cross_fade", pushTransitionShader(L, core.CrossFade))
	L.SetField(tr, "radial_wipe", pushTransitionShader(L, core.RadialWipe))
	L.SetField(tr, "pixelate", pushTransitionShader(L, core.Pixelate))
	L.SetField(tr, "wipe_left", pushTransitionShader(L, core.WipeLeft))
	L.SetField(tr, "wipe_right", pushTransitionShader(L, core.WipeRight))
	L.SetField(tr, "wipe_up", pushTransitionShader(L, core.WipeUp))
	L.SetField(tr, "wipe_down", pushTransitionShader(L, core.WipeDown))
	L.SetField(tr, "push_left", pushTransitionShader(L, core.PushLeft))
	L.SetField(tr, "push_right", pushTransitionShader(L, core.PushRight))

	// scene_manager constructor: flicker.scene_manager(width, height)
	L.SetField(mod, "scene_manager", L.NewFunction(func(L *lua.LState) int {
		width := L.CheckInt(1)
		height := L.CheckInt(2)
		sm := core.NewSceneManager(width, height)
		engine.sceneManager = sm
		pushUserData(L, typeSceneManager, sm)
		return 1
	}))

	// scene constructor: flicker.scene(width, height, callbacks_table)
	L.SetField(mod, "scene", L.NewFunction(func(L *lua.LState) int {
		width := L.CheckInt(1)
		height := L.CheckInt(2)
		opts := L.CheckTable(3)

		cb := &SceneCallbacks{}
		if fn := L.GetField(opts, "on_enter"); fn != lua.LNil {
			if f, ok := fn.(*lua.LFunction); ok {
				cb.OnEnter = f
			}
		}
		if fn := L.GetField(opts, "on_ready"); fn != lua.LNil {
			if f, ok := fn.(*lua.LFunction); ok {
				cb.OnReady = f
			}
		}
		if fn := L.GetField(opts, "on_update"); fn != lua.LNil {
			if f, ok := fn.(*lua.LFunction); ok {
				cb.OnUpdate = f
			}
		}
		if fn := L.GetField(opts, "on_exit"); fn != lua.LNil {
			if f, ok := fn.(*lua.LFunction); ok {
				cb.OnExit = f
			}
		}

		scene := buildScene(L, engine, cb, width, height)

		// Check for trail setting
		if trail := L.GetField(opts, "trail"); trail != lua.LNil {
			if trailTable, ok := trail.(*lua.LTable); ok {
				layer := int(getNumberField(L, trailTable, "layer", 0))
				if effectVal := L.GetField(trailTable, "effect"); effectVal != lua.LNil {
					if ud, ok := effectVal.(*lua.LUserData); ok {
						if pp, ok := ud.Value.(core.LayerPreProcess); ok {
							scene.Compositor().SetPreProcess(layer, pp)
						}
					}
				}
			}
		}

		ud := L.NewUserData()
		ud.Value = core.Scene(scene)
		L.Push(ud)
		return 1
	}))
}

func pushTransitionShader(L *lua.LState, shader core.TransitionShader) *lua.LUserData {
	ud := L.NewUserData()
	ud.Value = shader
	return ud
}

func checkTransitionShader(L *lua.LState, n int) core.TransitionShader {
	ud := L.CheckUserData(n)
	if s, ok := ud.Value.(core.TransitionShader); ok {
		return s
	}
	L.ArgError(n, "transition shader expected")
	return nil
}

func checkSceneManager(L *lua.LState, n int) *core.SceneManager {
	ud := L.CheckUserData(n)
	if sm, ok := ud.Value.(*core.SceneManager); ok {
		return sm
	}
	L.ArgError(n, "scene_manager expected")
	return nil
}

// --- SceneManager methods ---

func smAdd(L *lua.LState) int {
	sm := checkSceneManager(L, 1)
	ud := L.CheckUserData(2)
	if s, ok := ud.Value.(core.Scene); ok {
		sm.Add(s)
	} else {
		L.ArgError(2, "scene expected")
	}
	return 0
}

func smStart(L *lua.LState) int {
	sm := checkSceneManager(L, 1)
	sm.Start()
	return 0
}

func smNext(L *lua.LState) int {
	sm := checkSceneManager(L, 1)
	shader := checkTransitionShader(L, 2)
	duration := float64(L.CheckNumber(3))
	sm.Next(shader, duration)
	return 0
}

func smPrevious(L *lua.LState) int {
	sm := checkSceneManager(L, 1)
	shader := checkTransitionShader(L, 2)
	duration := float64(L.CheckNumber(3))
	sm.Previous(shader, duration)
	return 0
}

func smGoTo(L *lua.LState) int {
	sm := checkSceneManager(L, 1)
	index := L.CheckInt(2)
	sm.GoTo(index)
	return 0
}

func smCurrent(L *lua.LState) int {
	sm := checkSceneManager(L, 1)
	L.Push(lua.LNumber(sm.Current()))
	return 1
}

func smCount(L *lua.LState) int {
	sm := checkSceneManager(L, 1)
	L.Push(lua.LNumber(sm.Count()))
	return 1
}

func smIsTransitioning(L *lua.LState) int {
	sm := checkSceneManager(L, 1)
	L.Push(lua.LBool(sm.IsTransitioning()))
	return 1
}
