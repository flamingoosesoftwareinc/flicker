package lua

import (
	"flicker/core"
	lua "github.com/epikur-io/gopher-lua"
)

// registerAll registers all metatables and returns the flicker module table.
func registerAll(L *lua.LState, engine *Engine) *lua.LTable {
	mod := L.NewTable()

	// Register types (order matters: dependencies first)
	registerMathTypes(L, mod)
	registerColorType(L, mod)
	registerWorldType(L)
	registerBitmapModule(L, mod)
	registerSDFModule(L, mod)
	registerSceneAPI(L, mod, engine)
	registerSceneManagerModule(L, mod, engine)
	registerAssetModule(L, mod)
	registerTimelineModule(L, mod, engine)
	registerPhysicsModule(L, mod)
	registerBodyMethod(L)
	registerTextModule(L, mod)
	registerTextFXModule(L, mod)
	registerParticleModule(L, mod)

	// Clip metatable (used by timeline clip constructors)
	L.NewTypeMetatable("flicker.clip")

	// Set trail on compositor: flicker.set_trail(layer, trail_effect)
	L.SetField(mod, "set_trail", L.NewFunction(func(L *lua.LState) int {
		layer := L.CheckInt(1)
		ud := L.CheckUserData(2)
		if pp, ok := ud.Value.(core.LayerPreProcess); ok {
			if engine.activeScene != nil {
				engine.activeScene.Compositor().SetPreProcess(layer, pp)
			}
		} else {
			L.ArgError(2, "trail effect expected")
		}
		return 0
	}))

	// Set blend mode on compositor layer: flicker.set_blend(layer, blend)
	L.SetField(mod, "set_blend", L.NewFunction(func(L *lua.LState) int {
		layer := L.CheckInt(1)
		ud := L.CheckUserData(2)
		if b, ok := ud.Value.(core.ColorBlend); ok {
			if engine.activeScene != nil {
				engine.activeScene.Compositor().SetBlend(layer, b)
			}
		} else {
			L.ArgError(2, "blend mode expected")
		}
		return 0
	}))

	// Set post-process on compositor layer: flicker.set_post_process(layer, shader_fn)
	L.SetField(mod, "set_post_process", L.NewFunction(func(L *lua.LState) int {
		layer := L.CheckInt(1)
		fn := L.CheckFunction(2)
		pp := core.LayerPostProcess(materialFromLua(L, fn))
		if engine.activeScene != nil {
			engine.activeScene.Compositor().SetPostProcess(layer, pp)
		}
		return 0
	}))

	// Blend modes sub-table
	registerBlendModule(L, mod)

	return mod
}
