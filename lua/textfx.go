package lua

import (
	"flicker/core"
	"flicker/fmath"
	"flicker/textfx"
	lua "github.com/epikur-io/gopher-lua"
)

func registerTextFXModule(L *lua.LState, mod *lua.LTable) {
	tfx := L.NewTable()
	L.SetField(mod, "textfx", tfx)

	L.SetField(tfx, "wave", L.NewFunction(textfxWave))
	L.SetField(tfx, "typewriter", L.NewFunction(textfxTypewriter))
	L.SetField(tfx, "staggered_fade", L.NewFunction(textfxStaggeredFade))
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

	if enc := L.GetField(opts, "encoding"); enc != lua.LNil {
		if s, ok := enc.(lua.LString); ok {
			waveOpts.Encoding = parseEncoding(string(s))
		}
	}

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

	encoding := textfx.HalfBlock
	if enc := L.GetField(opts, "encoding"); enc != lua.LNil {
		if s, ok := enc.(lua.LString); ok {
			encoding = parseEncoding(string(s))
		}
	}

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

	encoding := textfx.HalfBlock
	if enc := L.GetField(opts, "encoding"); enc != lua.LNil {
		if s, ok := enc.(lua.LString); ok {
			encoding = parseEncoding(string(s))
		}
	}

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
