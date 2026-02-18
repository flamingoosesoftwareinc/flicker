package lua

import (
	"flicker/core"
	lua "github.com/epikur-io/gopher-lua"
)

func registerBlendModule(L *lua.LState, mod *lua.LTable) {
	blend := L.NewTable()
	L.SetField(mod, "blend", blend)

	pushBlend := func(name string, b core.ColorBlend) {
		ud := L.NewUserData()
		ud.Value = b
		L.SetField(blend, name, ud)
	}

	pushBlend("normal", core.NormalColorBlend)
	pushBlend("multiply", core.MultiplyColorBlend)
	pushBlend("screen", core.ScreenColorBlend)
	pushBlend("overlay", core.OverlayColorBlend)
	pushBlend("hard_light", core.HardLightColorBlend)
	pushBlend("soft_light", core.SoftLightColorBlend)
	pushBlend("difference", core.DifferenceColorBlend)
	pushBlend("exclusion", core.ExclusionColorBlend)
	pushBlend("hard_mix", core.HardMixColorBlend)
	pushBlend("darken", core.DarkenColorBlend)
	pushBlend("lighten", core.LightenColorBlend)
	pushBlend("linear_dodge", core.LinearDodgeColorBlend)
	pushBlend("linear_burn", core.LinearBurnColorBlend)
	pushBlend("color_dodge", core.ColorDodgeColorBlend)
	pushBlend("color_burn", core.ColorBurnColorBlend)
}
