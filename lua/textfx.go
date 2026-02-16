package lua

import (
	"flicker/core"
	"flicker/fmath"
	"flicker/textfx"
	lua "github.com/epikur-io/gopher-lua"
)

const typeEncoding = "flicker.encoding"

func registerTextFXModule(L *lua.LState, mod *lua.LTable) {
	// Encoding metatable
	emt := L.NewTypeMetatable(typeEncoding)
	L.SetField(emt, "__tostring", L.NewFunction(func(L *lua.LState) int {
		ud := L.CheckUserData(1)
		if enc, ok := ud.Value.(textfx.Encoding); ok {
			switch enc {
			case textfx.Braille:
				L.Push(lua.LString("encoding(braille)"))
			case textfx.FullBlock:
				L.Push(lua.LString("encoding(full_block)"))
			default:
				L.Push(lua.LString("encoding(half_block)"))
			}
		} else {
			L.Push(lua.LString("encoding(?)"))
		}
		return 1
	}))

	tfx := L.NewTable()
	L.SetField(mod, "textfx", tfx)

	// Encoding constants sub-table
	enc := L.NewTable()
	L.SetField(tfx, "encoding", enc)

	pushUserData(L, typeEncoding, textfx.Braille)
	L.SetField(enc, "braille", L.Get(-1))
	L.Pop(1)
	pushUserData(L, typeEncoding, textfx.HalfBlock)
	L.SetField(enc, "half_block", L.Get(-1))
	L.Pop(1)
	pushUserData(L, typeEncoding, textfx.FullBlock)
	L.SetField(enc, "full_block", L.Get(-1))
	L.Pop(1)

	L.SetField(tfx, "wave", L.NewFunction(textfxWave))
	L.SetField(tfx, "typewriter", L.NewFunction(textfxTypewriter))
	L.SetField(tfx, "staggered_fade", L.NewFunction(textfxStaggeredFade))
}

// resolveEncoding accepts both encoding userdata and string, returning default if absent.
func resolveEncoding(L *lua.LState, v lua.LValue, def textfx.Encoding) textfx.Encoding {
	if v == nil || v == lua.LNil {
		return def
	}
	switch val := v.(type) {
	case *lua.LUserData:
		if enc, ok := val.Value.(textfx.Encoding); ok {
			return enc
		}
	case lua.LString:
		return parseEncoding(string(val))
	}
	return def
}

// textfx.wave(world, layout, {base_position, encoding, layer, amplitude, frequency, phase_per_char})
func textfxWave(L *lua.LState) int {
	w := checkWorld(L, 1)
	tl := checkTextLayout(L, 2)
	opts := L.CheckTable(3)

	waveOpts := textfx.WaveOptions{
		Encoding:     textfx.HalfBlock,
		Amplitude:    2.0,
		Frequency:    1.0,
		PhasePerChar: 0.3,
	}

	if bp := L.GetField(opts, "base_position"); bp != lua.LNil {
		if ud, ok := bp.(*lua.LUserData); ok {
			if v, ok := ud.Value.(fmath.Vec3); ok {
				waveOpts.BasePosition = v
			}
		}
	}

	waveOpts.Encoding = resolveEncoding(L, L.GetField(opts, "encoding"), waveOpts.Encoding)

	waveOpts.Layer = int(getNumberField(L, opts, "layer", 0))
	waveOpts.Amplitude = getNumberField(L, opts, "amplitude", waveOpts.Amplitude)
	waveOpts.Frequency = getNumberField(L, opts, "frequency", waveOpts.Frequency)
	waveOpts.PhasePerChar = getNumberField(L, opts, "phase_per_char", waveOpts.PhasePerChar)

	entities := textfx.Wave(w, tl, waveOpts)

	// Return array of entities
	result := L.NewTable()
	for i, eid := range entities {
		pushEntity(L, luaEntity{ID: eid, World: w})
		entUD := L.Get(-1)
		L.Pop(1)
		result.RawSetInt(i+1, entUD)
	}
	L.Push(result)
	return 1
}

// textfx.typewriter(layout, {encoding, chars_per_sec}) → {material=ud, behavior=ud}
func textfxTypewriter(L *lua.LState) int {
	tl := checkTextLayout(L, 1)
	opts := L.CheckTable(2)

	encoding := resolveEncoding(L, L.GetField(opts, "encoding"), textfx.HalfBlock)

	charsPerSec := getNumberField(L, opts, "chars_per_sec", 10)
	maxChars := len(tl.Glyphs)

	charsRevealed := new(float64)
	mat := textfx.TypewriterMaterial(tl, encoding, charsRevealed)
	beh := textfx.TypewriterBehavior(charsRevealed, charsPerSec, maxChars)

	result := L.NewTable()

	// Material as userdata
	matUD := L.NewUserData()
	matUD.Value = core.Material(mat)
	L.SetField(result, "material", matUD)

	// Behavior as userdata
	behUD := L.NewUserData()
	behUD.Value = beh
	L.SetField(result, "behavior", behUD)

	L.Push(result)
	return 1
}

// textfx.staggered_fade(layout, {encoding, delay_per_char, fade_duration}) → material userdata
func textfxStaggeredFade(L *lua.LState) int {
	tl := checkTextLayout(L, 1)
	opts := L.CheckTable(2)

	encoding := resolveEncoding(L, L.GetField(opts, "encoding"), textfx.HalfBlock)

	delayPerChar := getNumberField(L, opts, "delay_per_char", 0.15)
	fadeDuration := getNumberField(L, opts, "fade_duration", 0.5)

	mat := textfx.StaggeredFadeMaterial(tl, encoding, delayPerChar, fadeDuration)

	ud := L.NewUserData()
	ud.Value = core.Material(mat)
	L.Push(ud)
	return 1
}

func parseEncoding(s string) textfx.Encoding {
	switch s {
	case "braille":
		return textfx.Braille
	case "full_block":
		return textfx.FullBlock
	default:
		return textfx.HalfBlock
	}
}
