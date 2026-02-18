package lua

import (
	"flicker/core"
	lua "github.com/epikur-io/gopher-lua"
)

const typeText = "flicker.text"

func registerTextModule(L *lua.LState, mod *lua.LTable) {
	// Text metatable with set_value method
	mt := registerType(L, typeText, map[string]lua.LGFunction{
		"set_value": textSetValue,
	})
	L.SetField(mt, "__index", L.GetField(mt, "methods"))

	// Top-level constructor: f.text("content", { fg = ... })
	L.SetField(mod, "text", L.NewFunction(newText))
}

// newText creates a Text drawable from Lua.
//
//	f.text("Hello\nWorld")
//	f.text("Hello", { fg = f.color(255, 255, 255), alpha = 0.8 })
func newText(L *lua.LState) int {
	text := L.CheckString(1)

	fg := core.Color{R: 255, G: 255, B: 255}
	fgAlpha := 1.0

	// Optional options table
	if opts := L.Get(2); opts != lua.LNil {
		if tbl, ok := opts.(*lua.LTable); ok {
			if v := L.GetField(tbl, "fg"); v != lua.LNil {
				if ud, ok := v.(*lua.LUserData); ok {
					if c, ok := ud.Value.(core.Color); ok {
						fg = c
					}
				}
			}
			fgAlpha = getNumberField(L, tbl, "alpha", 1.0)
		}
	}

	t := core.NewText(text, fg, fgAlpha)
	pushUserData(L, typeText, t)
	return 1
}

func checkText(L *lua.LState, n int) *core.Text {
	ud := L.CheckUserData(n)
	if t, ok := ud.Value.(*core.Text); ok {
		return t
	}
	L.ArgError(n, "text expected")
	return nil
}

// textSetValue updates the text content: txt:set_value("new text")
func textSetValue(L *lua.LState) int {
	t := checkText(L, 1)
	text := L.CheckString(2)
	t.SetText(text)
	return 0
}

// registerTextKeyframesClip registers f.timeline.text_keyframes on the timeline sub-table.
func registerTextKeyframesClip(L *lua.LState, tl *lua.LTable) {
	L.SetField(tl, "text_keyframes", L.NewFunction(newTextKeyframesClip))
}

// f.timeline.text_keyframes(text, { {time=0, value="A"}, {time=1, value="B"} }, duration)
func newTextKeyframesClip(L *lua.LState) int {
	t := checkText(L, 1)
	tbl := L.CheckTable(2)
	duration := float64(L.CheckNumber(3))

	var keyframes []core.TextKeyframe
	tbl.ForEach(func(_, v lua.LValue) {
		entry, ok := v.(*lua.LTable)
		if !ok {
			return
		}
		time := getNumberField(L, entry, "time", 0)
		value := getStringField(L, entry, "value", "")
		keyframes = append(keyframes, core.TextKeyframe{
			Time:  time,
			Value: value,
		})
	})

	clip := core.NewTextKeyframeClip(keyframes, duration, t.SetText)
	pushUserData(L, "flicker.clip", core.Clip(clip))
	return 1
}
