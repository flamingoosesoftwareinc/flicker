package lua

import (
	"flicker/core"
	lua "github.com/epikur-io/gopher-lua"
)

func registerTrailModule(L *lua.LState, mod *lua.LTable) {
	trail := L.NewTable()
	L.SetField(mod, "trail", trail)

	L.SetField(trail, "ghost", L.NewFunction(trailGhost))
	L.SetField(trail, "blur", L.NewFunction(trailBlur))
	L.SetField(trail, "floaty", L.NewFunction(trailFloaty))
	L.SetField(trail, "gravity", L.NewFunction(trailGravity))
	L.SetField(trail, "dissolve", L.NewFunction(trailDissolve))
	L.SetField(trail, "fire", L.NewFunction(trailFire))
}

func trailGhost(L *lua.LState) int {
	decay := float64(L.CheckNumber(1))
	ud := L.NewUserData()
	ud.Value = core.LayerPreProcess(core.GhostTrail(decay))
	L.Push(ud)
	return 1
}

func trailBlur(L *lua.LState) int {
	decay := float64(L.CheckNumber(1))
	blur := float64(L.CheckNumber(2))
	ud := L.NewUserData()
	ud.Value = core.LayerPreProcess(core.BlurTrail(decay, blur))
	L.Push(ud)
	return 1
}

func trailFloaty(L *lua.LState) int {
	decay := float64(L.CheckNumber(1))
	strength := float64(L.CheckNumber(2))
	ud := L.NewUserData()
	ud.Value = core.LayerPreProcess(core.FloatyTrail(decay, strength))
	L.Push(ud)
	return 1
}

func trailGravity(L *lua.LState) int {
	decay := float64(L.CheckNumber(1))
	fallSpeed := float64(L.CheckNumber(2))
	ud := L.NewUserData()
	ud.Value = core.LayerPreProcess(core.GravityTrail(decay, fallSpeed))
	L.Push(ud)
	return 1
}

func trailDissolve(L *lua.LState) int {
	decay := float64(L.CheckNumber(1))
	threshold := float64(L.CheckNumber(2))
	c := checkColor(L, 3)
	ud := L.NewUserData()
	ud.Value = core.LayerPreProcess(core.DissolveTrail(decay, threshold, c))
	L.Push(ud)
	return 1
}

func trailFire(L *lua.LState) int {
	decay := float64(L.CheckNumber(1))
	heat := float64(L.CheckNumber(2))
	ud := L.NewUserData()
	ud.Value = core.LayerPreProcess(core.FireTrail(decay, heat))
	L.Push(ud)
	return 1
}
