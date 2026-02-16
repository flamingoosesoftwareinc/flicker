package lua

import (
	"flicker/core"
	lua "github.com/epikur-io/gopher-lua"
)

func registerColorType(L *lua.LState, mod *lua.LTable) {
	mt := registerType(L, typeColor, map[string]lua.LGFunction{})
	L.SetField(mt, "__index", L.NewFunction(colorIndex))
	L.SetField(mt, "__newindex", L.NewFunction(colorNewIndex))
	L.SetField(mt, "__tostring", L.NewFunction(colorToString))

	// Constructor: flicker.color(r, g, b)
	L.SetField(mod, "color", L.NewFunction(newColor))

	// Built-in materials sub-table
	registerMaterialModule(L, mod)
}

func newColor(L *lua.LState) int {
	r := L.CheckInt(1)
	g := L.CheckInt(2)
	b := L.CheckInt(3)
	pushColor(L, core.Color{R: uint8(r), G: uint8(g), B: uint8(b)})
	return 1
}

func pushColor(L *lua.LState, c core.Color) {
	ud := L.NewUserData()
	ud.Value = c
	L.SetMetatable(ud, L.GetTypeMetatable(typeColor))
	L.Push(ud)
}

func checkColor(L *lua.LState, n int) core.Color {
	ud := L.CheckUserData(n)
	if c, ok := ud.Value.(core.Color); ok {
		return c
	}
	L.ArgError(n, "color expected")
	return core.Color{}
}

func colorIndex(L *lua.LState) int {
	c := checkColor(L, 1)
	key := L.CheckString(2)
	switch key {
	case "r":
		L.Push(lua.LNumber(c.R))
	case "g":
		L.Push(lua.LNumber(c.G))
	case "b":
		L.Push(lua.LNumber(c.B))
	default:
		L.Push(lua.LNil)
	}
	return 1
}

func colorNewIndex(L *lua.LState) int {
	ud := L.CheckUserData(1)
	key := L.CheckString(2)
	val := uint8(L.CheckInt(3))
	c := ud.Value.(core.Color)
	switch key {
	case "r":
		c.R = val
	case "g":
		c.G = val
	case "b":
		c.B = val
	default:
		L.ArgError(2, "unknown field: "+key)
	}
	ud.Value = c
	return 0
}

func colorToString(L *lua.LState) int {
	c := checkColor(L, 1)
	L.Push(
		lua.LString(
			"color(" + itoa(int(c.R)) + ", " + itoa(int(c.G)) + ", " + itoa(int(c.B)) + ")",
		),
	)
	return 1
}

// --- Built-in Materials ---

func registerMaterialModule(L *lua.LState, mod *lua.LTable) {
	mat := L.NewTable()
	L.SetField(mod, "material", mat)

	L.SetField(mat, "solid", L.NewFunction(materialSolid))
}

// materialSolid creates a Go-native solid color material (fast path).
func materialSolid(L *lua.LState) int {
	c := checkColor(L, 1)
	material := func(f core.Fragment) core.Cell {
		cell := f.Cell
		cell.FG = c
		return cell
	}
	ud := L.NewUserData()
	ud.Value = core.Material(material)
	L.Push(ud)
	return 1
}

// materialFromLua wraps a Lua function as a core.Material.
// Uses a pooled fragment table to reduce allocations.
func materialFromLua(L *lua.LState, fn *lua.LFunction) core.Material {
	fragTable := L.NewTable()
	return func(f core.Fragment) core.Cell {
		L.SetField(fragTable, "x", lua.LNumber(f.X))
		L.SetField(fragTable, "y", lua.LNumber(f.Y))
		L.SetField(fragTable, "screen_x", lua.LNumber(f.ScreenX))
		L.SetField(fragTable, "screen_y", lua.LNumber(f.ScreenY))
		L.SetField(fragTable, "time", lua.LNumber(f.Time.Total))
		L.SetField(fragTable, "delta", lua.LNumber(f.Time.Delta))
		L.SetField(fragTable, "rune", lua.LString(string(f.Cell.Rune)))
		L.SetField(fragTable, "fg_r", lua.LNumber(f.Cell.FG.R))
		L.SetField(fragTable, "fg_g", lua.LNumber(f.Cell.FG.G))
		L.SetField(fragTable, "fg_b", lua.LNumber(f.Cell.FG.B))
		L.SetField(fragTable, "fg_alpha", lua.LNumber(f.Cell.FGAlpha))
		L.SetField(fragTable, "bg_r", lua.LNumber(f.Cell.BG.R))
		L.SetField(fragTable, "bg_g", lua.LNumber(f.Cell.BG.G))
		L.SetField(fragTable, "bg_b", lua.LNumber(f.Cell.BG.B))
		L.SetField(fragTable, "bg_alpha", lua.LNumber(f.Cell.BGAlpha))

		if err := L.CallByParam(lua.P{
			Fn:      fn,
			NRet:    1,
			Protect: true,
		}, fragTable); err != nil {
			return f.Cell
		}

		ret := L.Get(-1)
		L.Pop(1)
		return luaToCell(L, ret)
	}
}

// luaToCell converts a Lua table to a core.Cell.
func luaToCell(L *lua.LState, v lua.LValue) core.Cell {
	t, ok := v.(*lua.LTable)
	if !ok {
		return core.Cell{}
	}
	cell := core.Cell{
		FGAlpha: getNumberField(L, t, "fg_alpha", 1.0),
		BGAlpha: getNumberField(L, t, "bg_alpha", 0.0),
	}

	// Rune
	if rv := L.GetField(t, "rune"); rv != lua.LNil {
		if s, ok := rv.(lua.LString); ok && len(string(s)) > 0 {
			cell.Rune = []rune(string(s))[0]
		}
	}

	// FG color - can be a color userdata or inline r,g,b
	if fg := L.GetField(t, "fg"); fg != lua.LNil {
		if ud, ok := fg.(*lua.LUserData); ok {
			if c, ok := ud.Value.(core.Color); ok {
				cell.FG = c
			}
		}
	} else {
		cell.FG = core.Color{
			R: uint8(getNumberField(L, t, "fg_r", 0)),
			G: uint8(getNumberField(L, t, "fg_g", 0)),
			B: uint8(getNumberField(L, t, "fg_b", 0)),
		}
	}

	// BG color
	if bg := L.GetField(t, "bg"); bg != lua.LNil {
		if ud, ok := bg.(*lua.LUserData); ok {
			if c, ok := ud.Value.(core.Color); ok {
				cell.BG = c
			}
		}
	} else {
		cell.BG = core.Color{
			R: uint8(getNumberField(L, t, "bg_r", 0)),
			G: uint8(getNumberField(L, t, "bg_g", 0)),
			B: uint8(getNumberField(L, t, "bg_b", 0)),
		}
	}

	return cell
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	s := ""
	neg := false
	if n < 0 {
		neg = true
		n = -n
	}
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	if neg {
		s = "-" + s
	}
	return s
}
