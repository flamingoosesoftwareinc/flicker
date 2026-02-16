package lua

import (
	"flicker/core"
	"flicker/fmath"
	lua "github.com/epikur-io/gopher-lua"
)

// luaEntity wraps an entity ID and its world for method dispatch.
type luaEntity struct {
	ID    core.Entity
	World *core.World
}

func registerWorldType(L *lua.LState) {
	// World metatable
	mt := registerType(L, typeWorld, map[string]lua.LGFunction{
		"spawn":    worldSpawn,
		"despawn":  worldDespawn,
		"add_root": worldAddRoot,
		"roots":    worldRoots,
		"attach":   worldAttach,
		"children": worldChildren,
	})
	L.SetField(mt, "__index", L.GetField(mt, "methods"))

	// Entity metatable
	emt := registerType(L, typeEntity, map[string]lua.LGFunction{
		"set_transform": entitySetTransform,
		"transform":     entityTransform,
		"set_position":  entitySetPosition,
		"set_rotation":  entitySetRotation,
		"set_scale":     entitySetScale,
		"set_drawable":  entitySetDrawable,
		"set_material":  entitySetMaterial,
		"set_behavior":  entitySetBehavior,
		"set_layer":     entitySetLayer,
		"id":            entityID,
	})
	L.SetField(emt, "__index", L.GetField(emt, "methods"))
}

func pushWorld(L *lua.LState, w *core.World) {
	pushUserData(L, typeWorld, w)
}

func checkWorld(L *lua.LState, n int) *core.World {
	ud := L.CheckUserData(n)
	if w, ok := ud.Value.(*core.World); ok {
		return w
	}
	L.ArgError(n, "world expected")
	return nil
}

func pushEntity(L *lua.LState, e luaEntity) {
	pushUserData(L, typeEntity, e)
}

func checkEntity(L *lua.LState, n int) luaEntity {
	ud := L.CheckUserData(n)
	if e, ok := ud.Value.(luaEntity); ok {
		return e
	}
	L.ArgError(n, "entity expected")
	return luaEntity{}
}

// --- World methods ---

func worldSpawn(L *lua.LState) int {
	w := checkWorld(L, 1)
	id := w.Spawn()
	pushEntity(L, luaEntity{ID: id, World: w})
	return 1
}

func worldDespawn(L *lua.LState) int {
	w := checkWorld(L, 1)
	e := checkEntity(L, 2)
	w.Despawn(e.ID)
	return 0
}

func worldAddRoot(L *lua.LState) int {
	w := checkWorld(L, 1)
	e := checkEntity(L, 2)
	w.AddRoot(e.ID)
	return 0
}

func worldRoots(L *lua.LState) int {
	w := checkWorld(L, 1)
	roots := w.Roots()
	t := L.NewTable()
	for i, r := range roots {
		ud := L.NewUserData()
		ud.Value = luaEntity{ID: r, World: w}
		L.SetMetatable(ud, L.GetTypeMetatable(typeEntity))
		t.RawSetInt(i+1, ud)
	}
	L.Push(t)
	return 1
}

func worldAttach(L *lua.LState) int {
	w := checkWorld(L, 1)
	child := checkEntity(L, 2)
	parent := checkEntity(L, 3)
	w.Attach(child.ID, parent.ID)
	return 0
}

func worldChildren(L *lua.LState) int {
	w := checkWorld(L, 1)
	e := checkEntity(L, 2)
	children := w.Children(e.ID)
	t := L.NewTable()
	for i, c := range children {
		ud := L.NewUserData()
		ud.Value = luaEntity{ID: c, World: w}
		L.SetMetatable(ud, L.GetTypeMetatable(typeEntity))
		t.RawSetInt(i+1, ud)
	}
	L.Push(t)
	return 1
}

// --- Entity methods ---

func entitySetTransform(L *lua.LState) int {
	e := checkEntity(L, 1)
	t := L.CheckTable(2)

	transform := &core.Transform{
		Scale: fmath.Vec3{X: 1, Y: 1, Z: 1},
	}

	// Position
	if pos := L.GetField(t, "position"); pos != lua.LNil {
		if ud, ok := pos.(*lua.LUserData); ok {
			if v, ok := ud.Value.(fmath.Vec3); ok {
				transform.Position = v
			}
		}
	}

	// Rotation
	transform.Rotation = getNumberField(L, t, "rotation", 0)

	// Scale
	if sc := L.GetField(t, "scale"); sc != lua.LNil {
		if ud, ok := sc.(*lua.LUserData); ok {
			if v, ok := ud.Value.(fmath.Vec3); ok {
				transform.Scale = v
			}
		}
	}

	e.World.AddTransform(e.ID, transform)
	return 0
}

func entityTransform(L *lua.LState) int {
	e := checkEntity(L, 1)
	tr := e.World.Transform(e.ID)
	if tr == nil {
		L.Push(lua.LNil)
		return 1
	}
	t := L.NewTable()
	pushVec3(L, tr.Position)
	L.SetField(t, "position", L.Get(-1))
	L.Pop(1)
	L.SetField(t, "rotation", lua.LNumber(tr.Rotation))
	pushVec3(L, tr.Scale)
	L.SetField(t, "scale", L.Get(-1))
	L.Pop(1)
	L.Push(t)
	return 1
}

func entitySetPosition(L *lua.LState) int {
	e := checkEntity(L, 1)
	pos := checkVec3(L, 2)
	tr := e.World.Transform(e.ID)
	if tr == nil {
		tr = &core.Transform{Scale: fmath.Vec3{X: 1, Y: 1, Z: 1}}
		e.World.AddTransform(e.ID, tr)
	}
	tr.Position = pos
	return 0
}

func entitySetRotation(L *lua.LState) int {
	e := checkEntity(L, 1)
	rot := float64(L.CheckNumber(2))
	tr := e.World.Transform(e.ID)
	if tr == nil {
		tr = &core.Transform{Scale: fmath.Vec3{X: 1, Y: 1, Z: 1}}
		e.World.AddTransform(e.ID, tr)
	}
	tr.Rotation = rot
	return 0
}

func entitySetScale(L *lua.LState) int {
	e := checkEntity(L, 1)
	sc := checkVec3(L, 2)
	tr := e.World.Transform(e.ID)
	if tr == nil {
		tr = &core.Transform{Scale: fmath.Vec3{X: 1, Y: 1, Z: 1}}
		e.World.AddTransform(e.ID, tr)
	}
	tr.Scale = sc
	return 0
}

func entitySetDrawable(L *lua.LState) int {
	e := checkEntity(L, 1)
	ud := L.CheckUserData(2)
	if d, ok := ud.Value.(core.Drawable); ok {
		e.World.AddDrawable(e.ID, d)
	} else {
		L.ArgError(2, "drawable expected")
	}
	return 0
}

func entitySetMaterial(L *lua.LState) int {
	e := checkEntity(L, 1)
	v := L.Get(2)
	switch val := v.(type) {
	case *lua.LUserData:
		// Go-native material (from flicker.material.solid etc.)
		if m, ok := val.Value.(core.Material); ok {
			e.World.AddMaterial(e.ID, m)
		} else {
			L.ArgError(2, "material expected")
		}
	case *lua.LFunction:
		// Lua function material (custom shader)
		e.World.AddMaterial(e.ID, materialFromLua(L, val))
	default:
		L.ArgError(2, "material or function expected")
	}
	return 0
}

func entitySetBehavior(L *lua.LState) int {
	e := checkEntity(L, 1)
	v := L.Get(2)

	switch val := v.(type) {
	case *lua.LFunction:
		// Lua function behavior
		behavior := core.NewBehavior(func(t core.Time, eid core.Entity, w *core.World) {
			timeTable := L.NewTable()
			L.SetField(timeTable, "total", lua.LNumber(t.Total))
			L.SetField(timeTable, "delta", lua.LNumber(t.Delta))

			pushEntity(L, luaEntity{ID: eid, World: w})
			entityUD := L.Get(-1)
			L.Pop(1)

			pushWorld(L, w)
			worldUD := L.Get(-1)
			L.Pop(1)

			_ = L.CallByParam(lua.P{
				Fn:      val,
				NRet:    0,
				Protect: true,
			}, entityUD, worldUD, timeTable)
		})
		e.World.AddBehavior(e.ID, behavior)
	case *lua.LUserData:
		// Go-native behavior (physics forces, integration, core.Behavior)
		switch beh := val.Value.(type) {
		case core.BehaviorFunc:
			e.World.AddBehavior(e.ID, core.NewBehavior(beh))
		case core.Behavior:
			e.World.AddBehavior(e.ID, beh)
		default:
			L.ArgError(2, "behavior function or userdata expected")
		}
	default:
		L.ArgError(2, "behavior function or userdata expected")
	}
	return 0
}

func entitySetLayer(L *lua.LState) int {
	e := checkEntity(L, 1)
	layer := L.CheckInt(2)
	e.World.AddLayer(e.ID, layer)
	return 0
}

func entityID(L *lua.LState) int {
	e := checkEntity(L, 1)
	L.Push(lua.LNumber(e.ID))
	return 1
}
