package lua

import (
	"flicker/core"
	"flicker/core/bitmap"
	lua "github.com/epikur-io/gopher-lua"
)

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

	bm := L.NewTable()
	L.SetField(mod, "bitmap", bm)

	// Constructor
	L.SetField(bm, "new", L.NewFunction(newBitmap))

	// Encoding constructors (return Drawables)
	L.SetField(bm, "half_block", L.NewFunction(newHalfBlock))
	L.SetField(bm, "braille", L.NewFunction(newBraille))
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
