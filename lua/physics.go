package lua

import (
	"flicker/core"
	"flicker/fmath"
	"flicker/physics"
	lua "github.com/epikur-io/gopher-lua"
)

func registerPhysicsModule(L *lua.LState, mod *lua.LTable) {
	phys := L.NewTable()
	L.SetField(mod, "physics", phys)

	// Forces
	L.SetField(phys, "attractor", L.NewFunction(physicsAttractor))
	L.SetField(phys, "repulsor", L.NewFunction(physicsRepulsor))
	L.SetField(phys, "drag", L.NewFunction(physicsDrag))
	L.SetField(phys, "gravity", L.NewFunction(physicsGravity))
	L.SetField(phys, "turbulence", L.NewFunction(physicsTurbulence))
	L.SetField(phys, "spring", L.NewFunction(physicsSpring))

	// Integration
	L.SetField(phys, "euler", L.NewFunction(physicsEuler))
	L.SetField(phys, "verlet", L.NewFunction(physicsVerlet))
}

// Entity method: e:set_body({velocity = vec2, acceleration = vec2})
func registerBodyMethod(L *lua.LState) {
	// Get entity metatable and add set_body method
	mt := L.GetTypeMetatable(typeEntity)
	methods := L.GetField(mt, "methods")
	if methodsT, ok := methods.(*lua.LTable); ok {
		L.SetField(methodsT, "set_body", L.NewFunction(entitySetBody))
		L.SetField(methodsT, "body", L.NewFunction(entityGetBody))
	}
}

func entitySetBody(L *lua.LState) int {
	e := checkEntity(L, 1)
	body := &core.Body{}

	if L.GetTop() >= 2 {
		opts := L.CheckTable(2)
		if vel := L.GetField(opts, "velocity"); vel != lua.LNil {
			if ud, ok := vel.(*lua.LUserData); ok {
				if v, ok := ud.Value.(fmath.Vec2); ok {
					body.Velocity = v
				}
			}
		}
		if acc := L.GetField(opts, "acceleration"); acc != lua.LNil {
			if ud, ok := acc.(*lua.LUserData); ok {
				if v, ok := ud.Value.(fmath.Vec2); ok {
					body.Acceleration = v
				}
			}
		}
	}

	e.World.AddBody(e.ID, body)
	return 0
}

func entityGetBody(L *lua.LState) int {
	e := checkEntity(L, 1)
	body := e.World.Body(e.ID)
	if body == nil {
		L.Push(lua.LNil)
		return 1
	}
	t := L.NewTable()
	pushVec2(L, body.Velocity)
	L.SetField(t, "velocity", L.Get(-1))
	L.Pop(1)
	pushVec2(L, body.Acceleration)
	L.SetField(t, "acceleration", L.Get(-1))
	L.Pop(1)
	L.Push(t)
	return 1
}

func pushBehavior(L *lua.LState, fn core.BehaviorFunc) {
	ud := L.NewUserData()
	ud.Value = fn
	L.Push(ud)
}

// physics.attractor(center_vec2, strength)
func physicsAttractor(L *lua.LState) int {
	center := checkVec2(L, 1)
	strength := float64(L.CheckNumber(2))
	pushBehavior(L, physics.Attractor(center, strength))
	return 1
}

// physics.repulsor(center_vec2, strength)
func physicsRepulsor(L *lua.LState) int {
	center := checkVec2(L, 1)
	strength := float64(L.CheckNumber(2))
	pushBehavior(L, physics.Repulsor(center, strength))
	return 1
}

// physics.drag(coefficient)
func physicsDrag(L *lua.LState) int {
	coeff := float64(L.CheckNumber(1))
	pushBehavior(L, physics.Drag(coeff))
	return 1
}

// physics.gravity(vec2)
func physicsGravity(L *lua.LState) int {
	force := checkVec2(L, 1)
	pushBehavior(L, physics.Gravity(force))
	return 1
}

// physics.turbulence(scale, strength)
func physicsTurbulence(L *lua.LState) int {
	scale := float64(L.CheckNumber(1))
	strength := float64(L.CheckNumber(2))
	pushBehavior(L, physics.Turbulence(scale, strength))
	return 1
}

// physics.spring(anchor_vec2, k, damping)
func physicsSpring(L *lua.LState) int {
	anchor := checkVec2(L, 1)
	k := float64(L.CheckNumber(2))
	damping := float64(L.CheckNumber(3))
	pushBehavior(L, physics.Spring(anchor, k, damping))
	return 1
}

// physics.euler() → integration behavior
func physicsEuler(L *lua.LState) int {
	pushBehavior(L, physics.EulerIntegration())
	return 1
}

// physics.verlet() → integration behavior
func physicsVerlet(L *lua.LState) int {
	pushBehavior(L, physics.VerletIntegration())
	return 1
}
