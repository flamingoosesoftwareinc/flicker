package lua

import (
	"flicker/asset"
	"flicker/core"
	"flicker/core/bitmap"
	"flicker/fmath"
	lua "github.com/epikur-io/gopher-lua"
)

const (
	typeFont       = "flicker.font"
	typeTextLayout = "flicker.text_layout"
)

func registerAssetModule(L *lua.LState, mod *lua.LTable) {
	// Font metatable (opaque handle)
	L.NewTypeMetatable(typeFont)

	// Mesh metatable (opaque handle)
	meshMT := L.NewTypeMetatable(typeMesh)
	L.SetField(meshMT, "__tostring", L.NewFunction(func(L *lua.LState) int {
		L.Push(lua.LString("mesh"))
		return 1
	}))

	// TextLayout metatable with field access
	mt := registerType(L, typeTextLayout, map[string]lua.LGFunction{
		"split_glyphs": textLayoutSplitGlyphs,
	})
	L.SetField(mt, "__index", L.NewFunction(textLayoutIndex))

	a := L.NewTable()
	L.SetField(mod, "asset", a)

	L.SetField(a, "load_font", L.NewFunction(assetLoadFont))
	L.SetField(a, "load_obj", L.NewFunction(assetLoadOBJ))
	L.SetField(a, "load_image", L.NewFunction(assetLoadImage))
	L.SetField(a, "rasterize_text", L.NewFunction(assetRasterizeText))
	L.SetField(a, "rasterize_wireframe", L.NewFunction(assetRasterizeWireframe))
}

func assetLoadFont(L *lua.LState) int {
	path := L.CheckString(1)
	f, err := asset.LoadFont(path)
	if err != nil {
		L.RaiseError("load_font: %s", err.Error())
		return 0
	}
	pushUserData(L, typeFont, f)
	return 1
}

func assetLoadImage(L *lua.LState) int {
	path := L.CheckString(1)
	maxW := L.OptInt(2, 0)
	maxH := L.OptInt(3, 0)
	bm, err := asset.LoadImage(path, maxW, maxH)
	if err != nil {
		L.RaiseError("load_image: %s", err.Error())
		return 0
	}
	pushUserData(L, typeBitmap, bm)
	return 1
}

func assetRasterizeText(L *lua.LState) int {
	text := L.CheckString(1)
	opts := L.CheckTable(2)

	// Extract font
	fontVal := L.GetField(opts, "font")
	if fontVal == lua.LNil {
		L.ArgError(2, "font field required")
		return 0
	}
	fontUD, ok := fontVal.(*lua.LUserData)
	if !ok {
		L.ArgError(2, "font field must be a font")
		return 0
	}
	f, ok := fontUD.Value.(*asset.Font)
	if !ok {
		L.ArgError(2, "font field must be a font")
		return 0
	}

	size := getNumberField(L, opts, "size", 24)

	// Color (optional, defaults to white)
	color := core.Color{R: 255, G: 255, B: 255}
	if colorVal := L.GetField(opts, "color"); colorVal != lua.LNil {
		if ud, ok := colorVal.(*lua.LUserData); ok {
			if c, ok := ud.Value.(core.Color); ok {
				color = c
			}
		}
	}

	layout := asset.RasterizeText(text, asset.TextOptions{
		Font:  f,
		Size:  size,
		Color: color,
	})

	if layout == nil {
		L.Push(lua.LNil)
		return 1
	}

	pushUserData(L, typeTextLayout, layout)
	return 1
}

func assetLoadOBJ(L *lua.LState) int {
	path := L.CheckString(1)
	m, err := asset.LoadOBJ(path)
	if err != nil {
		L.RaiseError("load_obj: %s", err.Error())
		return 0
	}
	pushUserData(L, typeMesh, m)
	return 1
}

func assetRasterizeWireframe(L *lua.LState) int {
	meshUD := L.CheckUserData(1)
	mesh, ok := meshUD.Value.(*asset.Mesh)
	if !ok {
		L.ArgError(1, "mesh expected")
		return 0
	}
	mvpUD := L.CheckUserData(2)
	mvp, ok := mvpUD.Value.(fmath.Mat4)
	if !ok {
		L.ArgError(2, "mat4 expected")
		return 0
	}

	opts := L.OptTable(3, nil)
	w := 200
	h := 200
	color := core.Color{R: 255, G: 255, B: 255}
	if opts != nil {
		w = int(getNumberField(L, opts, "width", float64(w)))
		h = int(getNumberField(L, opts, "height", float64(h)))
		if colorVal := L.GetField(opts, "color"); colorVal != lua.LNil {
			if ud, ok := colorVal.(*lua.LUserData); ok {
				if c, ok := ud.Value.(core.Color); ok {
					color = c
				}
			}
		}
	}

	bm := bitmap.New(w, h)
	asset.RasterizeWireframe(mesh, mvp, bm, color)
	pushUserData(L, typeBitmap, bm)
	return 1
}

func checkTextLayout(L *lua.LState, n int) *asset.TextLayout {
	ud := L.CheckUserData(n)
	if tl, ok := ud.Value.(*asset.TextLayout); ok {
		return tl
	}
	L.ArgError(n, "text_layout expected")
	return nil
}

func textLayoutIndex(L *lua.LState) int {
	tl := checkTextLayout(L, 1)
	key := L.CheckString(2)
	switch key {
	case "bitmap":
		pushUserData(L, typeBitmap, tl.Bitmap)
	case "width":
		L.Push(lua.LNumber(tl.Bitmap.Width))
	case "height":
		L.Push(lua.LNumber(tl.Bitmap.Height))
	default:
		mt := L.GetTypeMetatable(typeTextLayout)
		methods := L.GetField(mt, "methods")
		L.Push(L.GetField(methods, key))
	}
	return 1
}

func textLayoutSplitGlyphs(L *lua.LState) int {
	tl := checkTextLayout(L, 1)
	bitmaps := tl.SplitGlyphs()
	t := L.NewTable()
	for i, bm := range bitmaps {
		ud := L.NewUserData()
		ud.Value = bm
		L.SetMetatable(ud, L.GetTypeMetatable(typeBitmap))
		t.RawSetInt(i+1, ud)

		// Also store glyph info
		info := L.NewTable()
		L.SetField(info, "bitmap", ud)
		L.SetField(info, "rune", lua.LString(string(tl.Glyphs[i].Rune)))
		L.SetField(info, "x", lua.LNumber(tl.Glyphs[i].X))
		L.SetField(info, "width", lua.LNumber(tl.Glyphs[i].Width))

		// Store as array of tables with bitmap + metadata
		bitmapUD := L.NewUserData()
		bitmapUD.Value = bm
		L.SetMetatable(bitmapUD, L.GetTypeMetatable(typeBitmap))
		t.RawSetInt(i+1, bitmapUD)
	}

	// Return as simple array of bitmaps
	result := L.NewTable()
	for i, bm := range bitmaps {
		ud := L.NewUserData()
		ud.Value = bm
		L.SetMetatable(ud, L.GetTypeMetatable(typeBitmap))
		result.RawSetInt(i+1, ud)
	}
	L.Push(result)
	return 1
}
