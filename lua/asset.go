package lua

import (
	"flicker/asset"
	"flicker/core"
	lua "github.com/epikur-io/gopher-lua"
)

const (
	typeFont       = "flicker.font"
	typeTextLayout = "flicker.text_layout"
)

func registerAssetModule(L *lua.LState, mod *lua.LTable) {
	// Font metatable (opaque handle)
	L.NewTypeMetatable(typeFont)

	// TextLayout metatable with field access
	mt := registerType(L, typeTextLayout, map[string]lua.LGFunction{
		"split_glyphs": textLayoutSplitGlyphs,
	})
	L.SetField(mt, "__index", L.NewFunction(textLayoutIndex))

	a := L.NewTable()
	L.SetField(mod, "asset", a)

	L.SetField(a, "load_font", L.NewFunction(assetLoadFont))
	L.SetField(a, "rasterize_text", L.NewFunction(assetRasterizeText))
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
