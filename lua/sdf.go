package lua

import (
	"flicker/fmath"
	"flicker/sdf"
	lua "github.com/epikur-io/gopher-lua"
)

// sdfFunc wraps an SDF function for use as Lua userdata.
// It can be called as a function or passed to CSG operations.
type sdfFunc func(fmath.Vec2) float64

func registerSDFModule(L *lua.LState, mod *lua.LTable) {
	// SDF metatable with __call
	mt := L.NewTypeMetatable(typeSDF)
	L.SetField(mt, "__call", L.NewFunction(sdfCall))

	s := L.NewTable()
	L.SetField(mod, "sdf", s)

	// Primitives
	L.SetField(s, "circle", L.NewFunction(sdfCircle))
	L.SetField(s, "box", L.NewFunction(sdfBox))
	L.SetField(s, "rounded_box", L.NewFunction(sdfRoundedBox))
	L.SetField(s, "segment", L.NewFunction(sdfSegment))
	L.SetField(s, "triangle", L.NewFunction(sdfTriangle))
	L.SetField(s, "equilateral_triangle", L.NewFunction(sdfEquilateralTriangle))
	L.SetField(s, "rhombus", L.NewFunction(sdfRhombus))
	L.SetField(s, "pentagon", L.NewFunction(sdfPentagon))
	L.SetField(s, "hexagon", L.NewFunction(sdfHexagon))
	L.SetField(s, "ellipse", L.NewFunction(sdfEllipse))
	L.SetField(s, "arc", L.NewFunction(sdfArc))
	L.SetField(s, "pie", L.NewFunction(sdfPie))

	// CSG Operations
	L.SetField(s, "union", L.NewFunction(sdfUnion))
	L.SetField(s, "subtract", L.NewFunction(sdfSubtract))
	L.SetField(s, "intersect", L.NewFunction(sdfIntersect))
	L.SetField(s, "smooth_union", L.NewFunction(sdfSmoothUnion))
	L.SetField(s, "smooth_subtract", L.NewFunction(sdfSmoothSubtract))
	L.SetField(s, "smooth_intersect", L.NewFunction(sdfSmoothIntersect))
}

func pushSDF(L *lua.LState, fn sdfFunc) {
	ud := L.NewUserData()
	ud.Value = fn
	L.SetMetatable(ud, L.GetTypeMetatable(typeSDF))
	L.Push(ud)
}

func checkSDF(L *lua.LState, n int) sdfFunc {
	ud := L.CheckUserData(n)
	if fn, ok := ud.Value.(sdfFunc); ok {
		return fn
	}
	L.ArgError(n, "sdf expected")
	return nil
}

// sdfCall implements __call so SDFs can be used as sdf_func(vec2).
func sdfCall(L *lua.LState) int {
	fn := checkSDF(L, 1)
	p := checkVec2(L, 2)
	L.Push(lua.LNumber(fn(p)))
	return 1
}

// --- Primitives ---

func sdfCircle(L *lua.LState) int {
	radius := float64(L.CheckNumber(1))
	pushSDF(L, func(p fmath.Vec2) float64 {
		return sdf.Circle(p, radius)
	})
	return 1
}

func sdfBox(L *lua.LState) int {
	w := float64(L.CheckNumber(1))
	h := float64(L.CheckNumber(2))
	size := fmath.Vec2{X: w, Y: h}
	pushSDF(L, func(p fmath.Vec2) float64 {
		return sdf.Box(p, size)
	})
	return 1
}

func sdfRoundedBox(L *lua.LState) int {
	w := float64(L.CheckNumber(1))
	h := float64(L.CheckNumber(2))
	r := float64(L.CheckNumber(3))
	size := fmath.Vec2{X: w, Y: h}
	pushSDF(L, func(p fmath.Vec2) float64 {
		return sdf.RoundedBox(p, size, r)
	})
	return 1
}

func sdfSegment(L *lua.LState) int {
	a := checkVec2(L, 1)
	b := checkVec2(L, 2)
	pushSDF(L, func(p fmath.Vec2) float64 {
		return sdf.Segment(p, a, b)
	})
	return 1
}

func sdfTriangle(L *lua.LState) int {
	p0 := checkVec2(L, 1)
	p1 := checkVec2(L, 2)
	p2 := checkVec2(L, 3)
	pushSDF(L, func(p fmath.Vec2) float64 {
		return sdf.Triangle(p, p0, p1, p2)
	})
	return 1
}

func sdfEquilateralTriangle(L *lua.LState) int {
	radius := float64(L.CheckNumber(1))
	pushSDF(L, func(p fmath.Vec2) float64 {
		return sdf.EquilateralTriangle(p, radius)
	})
	return 1
}

func sdfRhombus(L *lua.LState) int {
	bx := float64(L.CheckNumber(1))
	by := float64(L.CheckNumber(2))
	b := fmath.Vec2{X: bx, Y: by}
	pushSDF(L, func(p fmath.Vec2) float64 {
		return sdf.Rhombus(p, b)
	})
	return 1
}

func sdfPentagon(L *lua.LState) int {
	radius := float64(L.CheckNumber(1))
	pushSDF(L, func(p fmath.Vec2) float64 {
		return sdf.Pentagon(p, radius)
	})
	return 1
}

func sdfHexagon(L *lua.LState) int {
	radius := float64(L.CheckNumber(1))
	pushSDF(L, func(p fmath.Vec2) float64 {
		return sdf.Hexagon(p, radius)
	})
	return 1
}

func sdfEllipse(L *lua.LState) int {
	ax := float64(L.CheckNumber(1))
	ay := float64(L.CheckNumber(2))
	ab := fmath.Vec2{X: ax, Y: ay}
	pushSDF(L, func(p fmath.Vec2) float64 {
		return sdf.Ellipse(p, ab)
	})
	return 1
}

func sdfArc(L *lua.LState) int {
	sc := checkVec2(L, 1)
	ra := float64(L.CheckNumber(2))
	rb := float64(L.CheckNumber(3))
	pushSDF(L, func(p fmath.Vec2) float64 {
		return sdf.Arc(p, sc, ra, rb)
	})
	return 1
}

func sdfPie(L *lua.LState) int {
	c := checkVec2(L, 1)
	r := float64(L.CheckNumber(2))
	pushSDF(L, func(p fmath.Vec2) float64 {
		return sdf.Pie(p, c, r)
	})
	return 1
}

// --- CSG Operations ---

func sdfUnion(L *lua.LState) int {
	a := checkSDF(L, 1)
	b := checkSDF(L, 2)
	pushSDF(L, func(p fmath.Vec2) float64 {
		return sdf.Union(a(p), b(p))
	})
	return 1
}

func sdfSubtract(L *lua.LState) int {
	a := checkSDF(L, 1)
	b := checkSDF(L, 2)
	pushSDF(L, func(p fmath.Vec2) float64 {
		return sdf.Subtract(a(p), b(p))
	})
	return 1
}

func sdfIntersect(L *lua.LState) int {
	a := checkSDF(L, 1)
	b := checkSDF(L, 2)
	pushSDF(L, func(p fmath.Vec2) float64 {
		return sdf.Intersect(a(p), b(p))
	})
	return 1
}

func sdfSmoothUnion(L *lua.LState) int {
	a := checkSDF(L, 1)
	b := checkSDF(L, 2)
	k := float64(L.CheckNumber(3))
	pushSDF(L, func(p fmath.Vec2) float64 {
		return sdf.SmoothUnion(a(p), b(p), k)
	})
	return 1
}

func sdfSmoothSubtract(L *lua.LState) int {
	a := checkSDF(L, 1)
	b := checkSDF(L, 2)
	k := float64(L.CheckNumber(3))
	pushSDF(L, func(p fmath.Vec2) float64 {
		return sdf.SmoothSubtract(a(p), b(p), k)
	})
	return 1
}

func sdfSmoothIntersect(L *lua.LState) int {
	a := checkSDF(L, 1)
	b := checkSDF(L, 2)
	k := float64(L.CheckNumber(3))
	pushSDF(L, func(p fmath.Vec2) float64 {
		return sdf.SmoothIntersect(a(p), b(p), k)
	})
	return 1
}
