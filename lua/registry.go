package lua

import (
	lua "github.com/epikur-io/gopher-lua"
)

// Type name constants for all registered metatables.
const (
	typeVec2   = "flicker.vec2"
	typeVec3   = "flicker.vec3"
	typeMat4   = "flicker.mat4"
	typeColor  = "flicker.color"
	typeEntity = "flicker.entity"
	typeWorld  = "flicker.world"
	typeBitmap = "flicker.bitmap"
	typeSDF    = "flicker.sdf"
	typeMesh   = "flicker.mesh"
)

// registerType creates a metatable for typeName and sets __index to a method table.
func registerType(L *lua.LState, typeName string, methods map[string]lua.LGFunction) *lua.LTable {
	mt := L.NewTypeMetatable(typeName)
	methodTable := L.SetFuncs(L.NewTable(), methods)
	L.SetField(mt, "methods", methodTable)
	return mt
}

// pushUserData wraps value as LUserData with the given metatable type.
func pushUserData(L *lua.LState, typeName string, value interface{}) {
	ud := L.NewUserData()
	ud.Value = value
	L.SetMetatable(ud, L.GetTypeMetatable(typeName))
	L.Push(ud)
}

// optNumber returns the number at stack position n, or defaultVal if absent.
func optNumber(L *lua.LState, n int, defaultVal float64) float64 {
	v := L.Get(n)
	if v == lua.LNil {
		return defaultVal
	}
	if lv, ok := v.(lua.LNumber); ok {
		return float64(lv)
	}
	L.ArgError(n, "number expected")
	return defaultVal
}

// getNumberField reads a numeric field from a Lua table, returning defaultVal if missing.
func getNumberField(L *lua.LState, t *lua.LTable, key string, defaultVal float64) float64 {
	v := L.GetField(t, key)
	if v == lua.LNil {
		return defaultVal
	}
	if lv, ok := v.(lua.LNumber); ok {
		return float64(lv)
	}
	return defaultVal
}

// getStringField reads a string field from a Lua table, returning defaultVal if missing.
func getStringField(L *lua.LState, t *lua.LTable, key string, defaultVal string) string {
	v := L.GetField(t, key)
	if v == lua.LNil {
		return defaultVal
	}
	if lv, ok := v.(lua.LString); ok {
		return string(lv)
	}
	return defaultVal
}
