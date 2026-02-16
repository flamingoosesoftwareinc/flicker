package lua

import (
	"flicker/fmath"
	lua "github.com/epikur-io/gopher-lua"
)

// registerMathTypes registers Vec2, Vec3 metatables and the flicker.math sub-table.
func registerMathTypes(L *lua.LState, mod *lua.LTable) {
	registerVec2(L, mod)
	registerVec3(L, mod)
	registerTweenVec3(L, mod)
	registerMathModule(L, mod)
}

// --- Vec2 ---

func registerVec2(L *lua.LState, mod *lua.LTable) {
	mt := registerType(L, typeVec2, map[string]lua.LGFunction{
		"length":    vec2Length,
		"normalize": vec2Normalize,
		"lerp":      vec2Lerp,
		"scale":     vec2Scale,
		"dot":       vec2Dot,
	})
	L.SetField(mt, "__index", L.NewFunction(vec2Index))
	L.SetField(mt, "__newindex", L.NewFunction(vec2NewIndex))
	L.SetField(mt, "__add", L.NewFunction(vec2Add))
	L.SetField(mt, "__sub", L.NewFunction(vec2Sub))
	L.SetField(mt, "__mul", L.NewFunction(vec2Mul))
	L.SetField(mt, "__len", L.NewFunction(vec2Len))
	L.SetField(mt, "__tostring", L.NewFunction(vec2ToString))

	// Constructor: flicker.vec2(x, y)
	L.SetField(mod, "vec2", L.NewFunction(newVec2))
}

func newVec2(L *lua.LState) int {
	x := float64(L.CheckNumber(1))
	y := float64(L.CheckNumber(2))
	pushVec2(L, fmath.Vec2{X: x, Y: y})
	return 1
}

func pushVec2(L *lua.LState, v fmath.Vec2) {
	ud := L.NewUserData()
	ud.Value = v
	L.SetMetatable(ud, L.GetTypeMetatable(typeVec2))
	L.Push(ud)
}

func checkVec2(L *lua.LState, n int) fmath.Vec2 {
	ud := L.CheckUserData(n)
	if v, ok := ud.Value.(fmath.Vec2); ok {
		return v
	}
	L.ArgError(n, "vec2 expected")
	return fmath.Vec2{}
}

func vec2Index(L *lua.LState) int {
	v := checkVec2(L, 1)
	key := L.CheckString(2)
	switch key {
	case "x":
		L.Push(lua.LNumber(v.X))
	case "y":
		L.Push(lua.LNumber(v.Y))
	default:
		mt := L.GetTypeMetatable(typeVec2)
		methods := L.GetField(mt, "methods")
		L.Push(L.GetField(methods, key))
	}
	return 1
}

func vec2NewIndex(L *lua.LState) int {
	ud := L.CheckUserData(1)
	key := L.CheckString(2)
	val := float64(L.CheckNumber(3))
	v := ud.Value.(fmath.Vec2)
	switch key {
	case "x":
		v.X = val
	case "y":
		v.Y = val
	default:
		L.ArgError(2, "unknown field: "+key)
	}
	ud.Value = v
	return 0
}

func vec2Add(L *lua.LState) int {
	a := checkVec2(L, 1)
	b := checkVec2(L, 2)
	pushVec2(L, a.Add(b))
	return 1
}

func vec2Sub(L *lua.LState) int {
	a := checkVec2(L, 1)
	b := checkVec2(L, 2)
	pushVec2(L, a.Sub(b))
	return 1
}

func vec2Mul(L *lua.LState) int {
	// Support vec2 * number and number * vec2
	v1 := L.Get(1)
	v2 := L.Get(2)
	if ud, ok := v1.(*lua.LUserData); ok {
		vec := ud.Value.(fmath.Vec2)
		s := float64(L.CheckNumber(2))
		pushVec2(L, vec.Scale(s))
	} else if ud, ok := v2.(*lua.LUserData); ok {
		s := float64(v1.(lua.LNumber))
		vec := ud.Value.(fmath.Vec2)
		pushVec2(L, vec.Scale(s))
	} else {
		L.ArgError(1, "vec2 or number expected")
	}
	return 1
}

func vec2Len(L *lua.LState) int {
	v := checkVec2(L, 1)
	L.Push(lua.LNumber(v.Length()))
	return 1
}

func vec2ToString(L *lua.LState) int {
	v := checkVec2(L, 1)
	L.Push(lua.LString("vec2(" + floatStr(v.X) + ", " + floatStr(v.Y) + ")"))
	return 1
}

func vec2Length(L *lua.LState) int {
	v := checkVec2(L, 1)
	L.Push(lua.LNumber(v.Length()))
	return 1
}

func vec2Normalize(L *lua.LState) int {
	v := checkVec2(L, 1)
	pushVec2(L, v.Normalize())
	return 1
}

func vec2Lerp(L *lua.LState) int {
	v := checkVec2(L, 1)
	to := checkVec2(L, 2)
	t := float64(L.CheckNumber(3))
	pushVec2(L, v.Lerp(to, t))
	return 1
}

func vec2Scale(L *lua.LState) int {
	v := checkVec2(L, 1)
	s := float64(L.CheckNumber(2))
	pushVec2(L, v.Scale(s))
	return 1
}

func vec2Dot(L *lua.LState) int {
	a := checkVec2(L, 1)
	b := checkVec2(L, 2)
	L.Push(lua.LNumber(a.X*b.X + a.Y*b.Y))
	return 1
}

// --- Vec3 ---

func registerVec3(L *lua.LState, mod *lua.LTable) {
	mt := registerType(L, typeVec3, map[string]lua.LGFunction{
		"length":    vec3Length,
		"normalize": vec3Normalize,
		"lerp":      vec3Lerp,
		"scale":     vec3Scale,
		"dot":       vec3Dot,
		"cross":     vec3Cross,
	})
	L.SetField(mt, "__index", L.NewFunction(vec3Index))
	L.SetField(mt, "__newindex", L.NewFunction(vec3NewIndex))
	L.SetField(mt, "__add", L.NewFunction(vec3Add))
	L.SetField(mt, "__sub", L.NewFunction(vec3Sub))
	L.SetField(mt, "__mul", L.NewFunction(vec3Mul))
	L.SetField(mt, "__len", L.NewFunction(vec3Len))
	L.SetField(mt, "__tostring", L.NewFunction(vec3ToString))

	// Constructor: flicker.vec3(x, y, z)
	L.SetField(mod, "vec3", L.NewFunction(newVec3))
}

func newVec3(L *lua.LState) int {
	x := float64(L.CheckNumber(1))
	y := float64(L.CheckNumber(2))
	z := optNumber(L, 3, 0)
	pushVec3(L, fmath.Vec3{X: x, Y: y, Z: z})
	return 1
}

func pushVec3(L *lua.LState, v fmath.Vec3) {
	ud := L.NewUserData()
	ud.Value = v
	L.SetMetatable(ud, L.GetTypeMetatable(typeVec3))
	L.Push(ud)
}

func checkVec3(L *lua.LState, n int) fmath.Vec3 {
	ud := L.CheckUserData(n)
	if v, ok := ud.Value.(fmath.Vec3); ok {
		return v
	}
	L.ArgError(n, "vec3 expected")
	return fmath.Vec3{}
}

func vec3Index(L *lua.LState) int {
	v := checkVec3(L, 1)
	key := L.CheckString(2)
	switch key {
	case "x":
		L.Push(lua.LNumber(v.X))
	case "y":
		L.Push(lua.LNumber(v.Y))
	case "z":
		L.Push(lua.LNumber(v.Z))
	default:
		mt := L.GetTypeMetatable(typeVec3)
		methods := L.GetField(mt, "methods")
		L.Push(L.GetField(methods, key))
	}
	return 1
}

func vec3NewIndex(L *lua.LState) int {
	ud := L.CheckUserData(1)
	key := L.CheckString(2)
	val := float64(L.CheckNumber(3))
	v := ud.Value.(fmath.Vec3)
	switch key {
	case "x":
		v.X = val
	case "y":
		v.Y = val
	case "z":
		v.Z = val
	default:
		L.ArgError(2, "unknown field: "+key)
	}
	ud.Value = v
	return 0
}

func vec3Add(L *lua.LState) int {
	a := checkVec3(L, 1)
	b := checkVec3(L, 2)
	pushVec3(L, a.Add(b))
	return 1
}

func vec3Sub(L *lua.LState) int {
	a := checkVec3(L, 1)
	b := checkVec3(L, 2)
	pushVec3(L, a.Sub(b))
	return 1
}

func vec3Mul(L *lua.LState) int {
	v1 := L.Get(1)
	v2 := L.Get(2)
	if ud, ok := v1.(*lua.LUserData); ok {
		vec := ud.Value.(fmath.Vec3)
		s := float64(L.CheckNumber(2))
		pushVec3(L, vec.Scale(s))
	} else if ud, ok := v2.(*lua.LUserData); ok {
		s := float64(v1.(lua.LNumber))
		vec := ud.Value.(fmath.Vec3)
		pushVec3(L, vec.Scale(s))
	} else {
		L.ArgError(1, "vec3 or number expected")
	}
	return 1
}

func vec3Len(L *lua.LState) int {
	v := checkVec3(L, 1)
	L.Push(lua.LNumber(v.Length()))
	return 1
}

func vec3ToString(L *lua.LState) int {
	v := checkVec3(L, 1)
	L.Push(lua.LString("vec3(" + floatStr(v.X) + ", " + floatStr(v.Y) + ", " + floatStr(v.Z) + ")"))
	return 1
}

func vec3Length(L *lua.LState) int {
	v := checkVec3(L, 1)
	L.Push(lua.LNumber(v.Length()))
	return 1
}

func vec3Normalize(L *lua.LState) int {
	v := checkVec3(L, 1)
	pushVec3(L, v.Normalize())
	return 1
}

func vec3Lerp(L *lua.LState) int {
	v := checkVec3(L, 1)
	to := checkVec3(L, 2)
	t := float64(L.CheckNumber(3))
	pushVec3(L, v.Lerp(to, t))
	return 1
}

func vec3Scale(L *lua.LState) int {
	v := checkVec3(L, 1)
	s := float64(L.CheckNumber(2))
	pushVec3(L, v.Scale(s))
	return 1
}

func vec3Dot(L *lua.LState) int {
	a := checkVec3(L, 1)
	b := checkVec3(L, 2)
	L.Push(lua.LNumber(a.Dot(b)))
	return 1
}

func vec3Cross(L *lua.LState) int {
	a := checkVec3(L, 1)
	b := checkVec3(L, 2)
	pushVec3(L, a.Cross(b))
	return 1
}

// --- TweenVec3 ---

const typeTweenVec3 = "flicker.tween_vec3"

func registerTweenVec3(L *lua.LState, mod *lua.LTable) {
	mt := registerType(L, typeTweenVec3, map[string]lua.LGFunction{
		"update": tweenVec3Update,
		"done":   tweenVec3Done,
		"reset":  tweenVec3Reset,
	})
	L.SetField(mt, "__index", L.GetField(mt, "methods"))

	// Constructor: flicker.tween_vec3({from, to, duration, easing})
	L.SetField(mod, "tween_vec3", L.NewFunction(func(L *lua.LState) int {
		opts := L.CheckTable(1)

		tw := &fmath.TweenVec3{
			Duration: getNumberField(L, opts, "duration", 1.0),
		}

		if from := L.GetField(opts, "from"); from != lua.LNil {
			if ud, ok := from.(*lua.LUserData); ok {
				if v, ok := ud.Value.(fmath.Vec3); ok {
					tw.From = v
				}
			}
		}
		if to := L.GetField(opts, "to"); to != lua.LNil {
			if ud, ok := to.(*lua.LUserData); ok {
				if v, ok := ud.Value.(fmath.Vec3); ok {
					tw.To = v
				}
			}
		}

		easingName := getStringField(L, opts, "easing", "linear")
		tw.Easing = resolveEasing(easingName)

		pushUserData(L, typeTweenVec3, tw)
		return 1
	}))
}

func checkTweenVec3(L *lua.LState, n int) *fmath.TweenVec3 {
	ud := L.CheckUserData(n)
	if tw, ok := ud.Value.(*fmath.TweenVec3); ok {
		return tw
	}
	L.ArgError(n, "tween_vec3 expected")
	return nil
}

func tweenVec3Update(L *lua.LState) int {
	tw := checkTweenVec3(L, 1)
	dt := float64(L.CheckNumber(2))
	result := tw.Update(dt)
	pushVec3(L, result)
	return 1
}

func tweenVec3Done(L *lua.LState) int {
	tw := checkTweenVec3(L, 1)
	L.Push(lua.LBool(tw.Done()))
	return 1
}

func tweenVec3Reset(L *lua.LState) int {
	tw := checkTweenVec3(L, 1)
	tw.Reset()
	return 0
}

// --- Math module ---

func registerMathModule(L *lua.LState, mod *lua.LTable) {
	math := L.NewTable()
	L.SetField(mod, "math", math)

	// Interpolation
	L.SetField(math, "lerp", L.NewFunction(mathLerp))
	L.SetField(math, "inverse_lerp", L.NewFunction(mathInverseLerp))
	L.SetField(math, "remap", L.NewFunction(mathRemap))
	L.SetField(math, "clamp", L.NewFunction(mathClamp))
	L.SetField(math, "deg_to_rad", L.NewFunction(mathDegToRad))
	L.SetField(math, "rad_to_deg", L.NewFunction(mathRadToDeg))
	L.SetField(math, "noise2d", L.NewFunction(mathNoise2D))

	// Easing functions
	L.SetField(math, "ease_linear", L.NewFunction(wrapEasing(fmath.EaseLinear)))
	L.SetField(math, "ease_in_quad", L.NewFunction(wrapEasing(fmath.EaseInQuad)))
	L.SetField(math, "ease_out_quad", L.NewFunction(wrapEasing(fmath.EaseOutQuad)))
	L.SetField(math, "ease_in_out_quad", L.NewFunction(wrapEasing(fmath.EaseInOutQuad)))
	L.SetField(math, "ease_in_cubic", L.NewFunction(wrapEasing(fmath.EaseInCubic)))
	L.SetField(math, "ease_out_cubic", L.NewFunction(wrapEasing(fmath.EaseOutCubic)))
	L.SetField(math, "ease_in_out_cubic", L.NewFunction(wrapEasing(fmath.EaseInOutCubic)))
	L.SetField(math, "ease_in_elastic", L.NewFunction(wrapEasing(fmath.EaseInElastic)))
	L.SetField(math, "ease_out_elastic", L.NewFunction(wrapEasing(fmath.EaseOutElastic)))
	L.SetField(math, "ease_out_bounce", L.NewFunction(wrapEasing(fmath.EaseOutBounce)))

	// Wave functions
	L.SetField(math, "saw", L.NewFunction(wrapWave(fmath.Saw)))
	L.SetField(math, "sine", L.NewFunction(wrapWave(fmath.Sine)))
	L.SetField(math, "triangle", L.NewFunction(wrapWave(fmath.Triangle)))
	L.SetField(math, "square", L.NewFunction(wrapWave(fmath.Square)))
	L.SetField(math, "pulse", L.NewFunction(mathPulse))

	// Bezier
	L.SetField(math, "bezier_quadratic", L.NewFunction(mathBezierQuadratic))
	L.SetField(math, "bezier_cubic", L.NewFunction(mathBezierCubic))
}

func wrapEasing(fn func(float64) float64) lua.LGFunction {
	return func(L *lua.LState) int {
		t := float64(L.CheckNumber(1))
		L.Push(lua.LNumber(fn(t)))
		return 1
	}
}

func wrapWave(fn func(float64) float64) lua.LGFunction {
	return func(L *lua.LState) int {
		t := float64(L.CheckNumber(1))
		L.Push(lua.LNumber(fn(t)))
		return 1
	}
}

func mathLerp(L *lua.LState) int {
	a := float64(L.CheckNumber(1))
	b := float64(L.CheckNumber(2))
	t := float64(L.CheckNumber(3))
	L.Push(lua.LNumber(fmath.Lerp(a, b, t)))
	return 1
}

func mathInverseLerp(L *lua.LState) int {
	a := float64(L.CheckNumber(1))
	b := float64(L.CheckNumber(2))
	v := float64(L.CheckNumber(3))
	L.Push(lua.LNumber(fmath.InverseLerp(a, b, v)))
	return 1
}

func mathRemap(L *lua.LState) int {
	v := float64(L.CheckNumber(1))
	inMin := float64(L.CheckNumber(2))
	inMax := float64(L.CheckNumber(3))
	outMin := float64(L.CheckNumber(4))
	outMax := float64(L.CheckNumber(5))
	L.Push(lua.LNumber(fmath.Remap(inMin, inMax, outMin, outMax, v)))
	return 1
}

func mathClamp(L *lua.LState) int {
	v := float64(L.CheckNumber(1))
	lo := float64(L.CheckNumber(2))
	hi := float64(L.CheckNumber(3))
	L.Push(lua.LNumber(fmath.Clamp(v, lo, hi)))
	return 1
}

func mathDegToRad(L *lua.LState) int {
	d := float64(L.CheckNumber(1))
	L.Push(lua.LNumber(fmath.DegToRad(d)))
	return 1
}

func mathRadToDeg(L *lua.LState) int {
	r := float64(L.CheckNumber(1))
	L.Push(lua.LNumber(fmath.RadToDeg(r)))
	return 1
}

func mathNoise2D(L *lua.LState) int {
	x := float64(L.CheckNumber(1))
	y := float64(L.CheckNumber(2))
	L.Push(lua.LNumber(fmath.Noise2D(x, y)))
	return 1
}

func mathPulse(L *lua.LState) int {
	t := float64(L.CheckNumber(1))
	duty := float64(L.CheckNumber(2))
	L.Push(lua.LNumber(fmath.Pulse(t, duty)))
	return 1
}

func mathBezierQuadratic(L *lua.LState) int {
	p0 := checkVec2(L, 1)
	p1 := checkVec2(L, 2)
	p2 := checkVec2(L, 3)
	t := float64(L.CheckNumber(4))
	pushVec2(L, fmath.BezierQuadratic(p0, p1, p2, t))
	return 1
}

func mathBezierCubic(L *lua.LState) int {
	p0 := checkVec2(L, 1)
	p1 := checkVec2(L, 2)
	p2 := checkVec2(L, 3)
	p3 := checkVec2(L, 4)
	t := float64(L.CheckNumber(5))
	pushVec2(L, fmath.BezierCubic(p0, p1, p2, p3, t))
	return 1
}

// floatStr formats a float64 as a compact string.
func floatStr(f float64) string {
	return lua.LNumber(f).String()
}
