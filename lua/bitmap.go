package lua

import (
	"flicker/core"
	"flicker/core/bitmap"
	lua "github.com/epikur-io/gopher-lua"
)

const typeBitmapSDF = "flicker.bitmap_sdf"

func registerBitmapModule(L *lua.LState, mod *lua.LTable) {
	mt := registerType(L, typeBitmap, map[string]lua.LGFunction{
		"set":     bitmapSet,
		"get":     bitmapGet,
		"set_dot": bitmapSetDot,
		"line":    bitmapLine,
		"clear":   bitmapClear,
		"width":   bitmapWidth,
		"height":  bitmapHeight,
	})
	L.SetField(mt, "__index", L.GetField(mt, "methods"))

	// Bitmap SDF metatable
	sdfMT := registerType(L, typeBitmapSDF, map[string]lua.LGFunction{
		"at":       bitmapSDFAt,
		"gradient": bitmapSDFGradient,
		"bounds":   bitmapSDFBounds,
	})
	L.SetField(sdfMT, "__index", L.GetField(sdfMT, "methods"))

	bm := L.NewTable()
	L.SetField(mod, "bitmap", bm)

	// Constructor
	L.SetField(bm, "new", L.NewFunction(newBitmap))

	// Encoding constructors (return Drawables)
	L.SetField(bm, "half_block", L.NewFunction(newHalfBlock))
	L.SetField(bm, "braille", L.NewFunction(newBraille))
	L.SetField(bm, "adaptive", L.NewFunction(newAdaptive))
	L.SetField(bm, "full_block", L.NewFunction(newFullBlock))
	L.SetField(bm, "bg_block", L.NewFunction(newBGBlock))
	L.SetField(bm, "rect", L.NewFunction(newRect))

	// SDF computation
	L.SetField(bm, "compute_sdf", L.NewFunction(bitmapComputeSDF))
	L.SetField(bm, "half_block_threshold", L.NewFunction(bitmapHalfBlockThreshold))
	L.SetField(bm, "braille_threshold", L.NewFunction(bitmapBrailleThreshold))
	L.SetField(bm, "adaptive_threshold", L.NewFunction(bitmapAdaptiveThreshold))
}

func newBitmap(L *lua.LState) int {
	w := L.CheckInt(1)
	h := L.CheckInt(2)
	bm := bitmap.New(w, h)
	pushUserData(L, typeBitmap, bm)
	return 1
}

func checkBitmap(L *lua.LState, n int) *bitmap.Bitmap {
	ud := L.CheckUserData(n)
	if bm, ok := ud.Value.(*bitmap.Bitmap); ok {
		return bm
	}
	L.ArgError(n, "bitmap expected")
	return nil
}

func bitmapSet(L *lua.LState) int {
	bm := checkBitmap(L, 1)
	x := L.CheckInt(2)
	y := L.CheckInt(3)
	c := checkColor(L, 4)
	alpha := optNumber(L, 5, 1.0)
	bm.Set(x, y, c, alpha)
	return 0
}

func bitmapGet(L *lua.LState) int {
	bm := checkBitmap(L, 1)
	x := L.CheckInt(2)
	y := L.CheckInt(3)
	c, a := bm.Get(x, y)
	pushColor(L, c)
	L.Push(lua.LNumber(a))
	return 2
}

func bitmapSetDot(L *lua.LState) int {
	bm := checkBitmap(L, 1)
	x := L.CheckInt(2)
	y := L.CheckInt(3)
	c := checkColor(L, 4)
	bm.SetDot(x, y, c)
	return 0
}

func bitmapLine(L *lua.LState) int {
	bm := checkBitmap(L, 1)
	x0 := L.CheckInt(2)
	y0 := L.CheckInt(3)
	x1 := L.CheckInt(4)
	y1 := L.CheckInt(5)
	c := checkColor(L, 6)
	bm.Line(x0, y0, x1, y1, c)
	return 0
}

func bitmapClear(L *lua.LState) int {
	bm := checkBitmap(L, 1)
	bm.Clear()
	return 0
}

func bitmapWidth(L *lua.LState) int {
	bm := checkBitmap(L, 1)
	L.Push(lua.LNumber(bm.Width))
	return 1
}

func bitmapHeight(L *lua.LState) int {
	bm := checkBitmap(L, 1)
	L.Push(lua.LNumber(bm.Height))
	return 1
}

// --- Encoding constructors ---

func newHalfBlock(L *lua.LState) int {
	bm := checkBitmap(L, 1)
	drawable := &bitmap.HalfBlock{Bitmap: bm}
	ud := L.NewUserData()
	ud.Value = core.Drawable(drawable)
	L.Push(ud)
	return 1
}

func newBraille(L *lua.LState) int {
	bm := checkBitmap(L, 1)
	drawable := &bitmap.Braille{Bitmap: bm}
	ud := L.NewUserData()
	ud.Value = core.Drawable(drawable)
	L.Push(ud)
	return 1
}

func newAdaptive(L *lua.LState) int {
	bm := checkBitmap(L, 1)
	ad := &bitmap.Adaptive{Bitmap: bm}
	if opts := L.OptTable(2, nil); opts != nil {
		ad.AlphaThreshold = getNumberField(L, opts, "threshold", 0)
	}
	ud := L.NewUserData()
	ud.Value = core.Drawable(ad)
	L.Push(ud)
	return 1
}

func newFullBlock(L *lua.LState) int {
	bm := checkBitmap(L, 1)
	drawable := &bitmap.FullBlock{Bitmap: bm}
	ud := L.NewUserData()
	ud.Value = core.Drawable(drawable)
	L.Push(ud)
	return 1
}

func newBGBlock(L *lua.LState) int {
	bm := checkBitmap(L, 1)
	drawable := &bitmap.BGBlock{Bitmap: bm}
	ud := L.NewUserData()
	ud.Value = core.Drawable(drawable)
	L.Push(ud)
	return 1
}

func newRect(L *lua.LState) int {
	w := L.CheckInt(1)
	h := L.CheckInt(2)
	fg := checkColor(L, 3)
	bg := core.Color{}
	if L.GetTop() >= 4 {
		bg = checkColor(L, 4)
	}
	drawable := &bitmap.Rect{
		Width:  w,
		Height: h,
		FG:     fg,
		BG:     bg,
	}
	ud := L.NewUserData()
	ud.Value = core.Drawable(drawable)
	L.Push(ud)
	return 1
}

// --- Bitmap SDF ---

func bitmapComputeSDF(L *lua.LState) int {
	bm := checkBitmap(L, 1)
	maxDist := float64(L.OptNumber(2, 50))
	s := bitmap.ComputeSDF(bm, maxDist)
	pushUserData(L, typeBitmapSDF, s)
	return 1
}

func checkBitmapSDF(L *lua.LState, n int) *bitmap.SDF {
	ud := L.CheckUserData(n)
	if s, ok := ud.Value.(*bitmap.SDF); ok {
		return s
	}
	L.ArgError(n, "bitmap_sdf expected")
	return nil
}

func bitmapSDFAt(L *lua.LState) int {
	s := checkBitmapSDF(L, 1)
	x := L.CheckInt(2)
	y := L.CheckInt(3)
	L.Push(lua.LNumber(s.At(x, y)))
	return 1
}

func bitmapSDFGradient(L *lua.LState) int {
	s := checkBitmapSDF(L, 1)
	x := L.CheckInt(2)
	y := L.CheckInt(3)
	g := s.Gradient(x, y)
	pushVec2(L, g)
	return 1
}

func bitmapSDFBounds(L *lua.LState) int {
	s := checkBitmapSDF(L, 1)
	b := s.Bounds()
	t := L.NewTable()
	L.SetField(t, "empty", lua.LBool(b.Empty))
	L.SetField(t, "min_x", lua.LNumber(b.MinX))
	L.SetField(t, "min_y", lua.LNumber(b.MinY))
	L.SetField(t, "max_x", lua.LNumber(b.MaxX))
	L.SetField(t, "max_y", lua.LNumber(b.MaxY))
	L.Push(t)
	return 1
}

// bitmapHalfBlockThreshold returns a material that reveals half-block content via SDF threshold.
// The threshold is stored in a userdata so Lua can animate it.
func bitmapHalfBlockThreshold(L *lua.LState) int {
	s := checkBitmapSDF(L, 1)
	threshold := float64(L.OptNumber(2, 0))

	// Store threshold as a pointer so it can be mutated by the returned setter.
	t := &threshold
	mat := bitmap.HalfBlockThreshold(s, t)

	ud := L.NewUserData()
	ud.Value = core.Material(mat)
	L.Push(ud)

	// Return a setter function as the second return value for animating the threshold.
	setter := L.NewFunction(func(L *lua.LState) int {
		*t = float64(L.CheckNumber(1))
		return 0
	})
	L.Push(setter)

	return 2
}

// bitmapBrailleThreshold returns a material that reveals braille content via SDF threshold.
func bitmapBrailleThreshold(L *lua.LState) int {
	s := checkBitmapSDF(L, 1)
	threshold := float64(L.OptNumber(2, 0))

	t := &threshold
	mat := bitmap.BrailleThreshold(s, t)

	ud := L.NewUserData()
	ud.Value = core.Material(mat)
	L.Push(ud)

	setter := L.NewFunction(func(L *lua.LState) int {
		*t = float64(L.CheckNumber(1))
		return 0
	})
	L.Push(setter)

	return 2
}

// bitmapAdaptiveThreshold returns a material that reveals adaptive content via SDF threshold.
func bitmapAdaptiveThreshold(L *lua.LState) int {
	s := checkBitmapSDF(L, 1)
	threshold := float64(L.OptNumber(2, 0))

	t := &threshold
	mat := bitmap.AdaptiveThreshold(s, t)

	ud := L.NewUserData()
	ud.Value = core.Material(mat)
	L.Push(ud)

	setter := L.NewFunction(func(L *lua.LState) int {
		*t = float64(L.CheckNumber(1))
		return 0
	})
	L.Push(setter)

	return 2
}
