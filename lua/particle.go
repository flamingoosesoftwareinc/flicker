package lua

import (
	"flicker/core"
	"flicker/core/bitmap"
	"flicker/fmath"
	"flicker/particle"
	lua "github.com/epikur-io/gopher-lua"
)

const (
	typePointCloudSeq    = "flicker.point_cloud_sequence"
	typeTrailingEmitter  = "flicker.trailing_emitter"
	typeDistribution     = "flicker.distribution"
	typeTransitionPhase  = "flicker.transition_phase"
	typeEmissionStrategy = "flicker.emission_strategy"
)

func registerParticleModule(L *lua.LState, mod *lua.LTable) {
	// PointCloudSequence metatable
	mt := registerType(L, typePointCloudSeq, map[string]lua.LGFunction{
		"add_target": seqAddTarget,
		"set_loop":   seqSetLoop,
		"particles":  seqParticles,
	})
	L.SetField(mt, "__index", L.GetField(mt, "methods"))

	// TrailingEmitter metatable
	tmt := registerType(L, typeTrailingEmitter, map[string]lua.LGFunction{})
	L.SetField(tmt, "__index", L.GetField(tmt, "methods"))

	// Distribution metatable
	dmt := L.NewTypeMetatable(typeDistribution)
	L.SetField(dmt, "__tostring", L.NewFunction(func(L *lua.LState) int {
		L.Push(lua.LString("distribution"))
		return 1
	}))

	// TransitionPhase metatable
	pmt := L.NewTypeMetatable(typeTransitionPhase)
	L.SetField(pmt, "__tostring", L.NewFunction(func(L *lua.LState) int {
		L.Push(lua.LString("transition_phase"))
		return 1
	}))

	// EmissionStrategy metatable
	emt := L.NewTypeMetatable(typeEmissionStrategy)
	L.SetField(emt, "__tostring", L.NewFunction(func(L *lua.LState) int {
		L.Push(lua.LString("emission_strategy"))
		return 1
	}))

	// particle sub-table
	pt := L.NewTable()
	L.SetField(mod, "particle", pt)

	// particle.bitmap_to_cloud(bitmap) -> table of vec2
	L.SetField(pt, "bitmap_to_cloud", L.NewFunction(func(L *lua.LState) int {
		ud := L.CheckUserData(1)
		bm, ok := ud.Value.(*bitmap.Bitmap)
		if !ok {
			L.ArgError(1, "bitmap expected")
			return 0
		}
		cloud := particle.BitmapToCloud(bm)
		t := L.NewTable()
		for i, pos := range cloud {
			pushVec2(L, pos)
			t.RawSetInt(i+1, L.Get(-1))
			L.Pop(1)
		}
		L.Push(t)
		return 1
	}))

	// particle.cloud_sequence(world, cloud, drawable, material, layer) -> PointCloudSequence
	L.SetField(pt, "cloud_sequence", L.NewFunction(func(L *lua.LState) int {
		w := checkWorld(L, 1)
		cloudTable := L.CheckTable(2)
		drawableUD := L.CheckUserData(3)
		materialUD := L.CheckUserData(4)
		layer := L.CheckInt(5)

		drawable, ok := drawableUD.Value.(core.Drawable)
		if !ok {
			L.ArgError(3, "drawable expected")
			return 0
		}
		material, ok := materialUD.Value.(core.Material)
		if !ok {
			L.ArgError(4, "material expected")
			return 0
		}

		cloud := tableToCloud(L, cloudTable)
		seq := particle.NewPointCloudSequence(w, cloud, drawable, material, layer)
		pushUserData(L, typePointCloudSeq, seq)
		return 1
	}))

	// Distribution strategies
	L.SetField(pt, "linear", L.NewFunction(func(L *lua.LState) int {
		pushUserData(L, typeDistribution, particle.LinearDistribution())
		return 1
	}))
	L.SetField(pt, "round_robin", L.NewFunction(func(L *lua.LState) int {
		pushUserData(L, typeDistribution, particle.RoundRobinDistribution())
		return 1
	}))
	L.SetField(pt, "closest_point", L.NewFunction(func(L *lua.LState) int {
		w := checkWorld(L, 1)
		entitiesTable := L.CheckTable(2)
		targetsTable := L.CheckTable(3)

		entities := tableToEntities(L, entitiesTable)
		targets := tableToCloud(L, targetsTable)

		pushUserData(L, typeDistribution, particle.ClosestPointDistribution(entities, targets, w))
		return 1
	}))

	// Phase constructors
	L.SetField(pt, "burst_phase", L.NewFunction(func(L *lua.LState) int {
		distance := float64(L.CheckNumber(1))
		pushUserData(
			L,
			typeTransitionPhase,
			particle.TransitionPhase(particle.BurstPhase(distance)),
		)
		return 1
	}))
	L.SetField(pt, "seek_phase", L.NewFunction(func(L *lua.LState) int {
		pushUserData(L, typeTransitionPhase, particle.TransitionPhase(particle.SeekPhase()))
		return 1
	}))
	L.SetField(pt, "keyframe_phase", L.NewFunction(func(L *lua.LState) int {
		easingName := L.OptString(1, "in_out_quad")
		easing := resolveParticleEasing(easingName)
		pushUserData(
			L,
			typeTransitionPhase,
			particle.TransitionPhase(&particle.KeyframePhase{Easing: easing}),
		)
		return 1
	}))
	L.SetField(pt, "curve_phase", L.NewFunction(func(L *lua.LState) int {
		arcHeight := float64(L.OptNumber(1, 10))
		pushUserData(
			L,
			typeTransitionPhase,
			particle.TransitionPhase(&particle.CurvePhase{ArcHeight: arcHeight}),
		)
		return 1
	}))
	L.SetField(pt, "turbulence_phase", L.NewFunction(func(L *lua.LState) int {
		scale := float64(L.CheckNumber(1))
		strength := float64(L.CheckNumber(2))
		pushUserData(
			L,
			typeTransitionPhase,
			particle.TransitionPhase(particle.TurbulencePhase(scale, strength)),
		)
		return 1
	}))

	// Emission strategy constants
	pushUserData(L, typeEmissionStrategy, particle.EmissionStrategy(particle.BottomEdge))
	L.SetField(pt, "bottom_edge", L.Get(-1))
	L.Pop(1)
	pushUserData(L, typeEmissionStrategy, particle.EmissionStrategy(particle.TopEdge))
	L.SetField(pt, "top_edge", L.Get(-1))
	L.Pop(1)
	pushUserData(L, typeEmissionStrategy, particle.EmissionStrategy(particle.LeftEdge))
	L.SetField(pt, "left_edge", L.Get(-1))
	L.Pop(1)
	pushUserData(L, typeEmissionStrategy, particle.EmissionStrategy(particle.RightEdge))
	L.SetField(pt, "right_edge", L.Get(-1))
	L.Pop(1)

	// Trailing emitter
	L.SetField(pt, "trailing_emitter", L.NewFunction(func(L *lua.LState) int {
		offsetVec := checkVec2(L, 1)
		emitter := particle.NewTrailingEmitter(offsetVec)

		// Optional config table
		if L.GetTop() >= 2 && L.Get(2).Type() == lua.LTTable {
			opts := L.CheckTable(2)
			if v := L.GetField(opts, "emit_rate"); v != lua.LNil {
				emitter.EmitRate = float64(v.(lua.LNumber))
			}
			if v := L.GetField(opts, "particle_life"); v != lua.LNil {
				emitter.ParticleLife = float64(v.(lua.LNumber))
			}
			if v := L.GetField(opts, "width"); v != lua.LNil {
				emitter.Width = float64(v.(lua.LNumber))
			}
			if v := L.GetField(opts, "color"); v != lua.LNil {
				if ud, ok := v.(*lua.LUserData); ok {
					if c, ok := ud.Value.(core.Color); ok {
						emitter.Color = c
					}
				}
			}
			// Drawable userdata
			if v := L.GetField(opts, "drawable"); v != lua.LNil {
				if ud, ok := v.(*lua.LUserData); ok {
					if d, ok := ud.Value.(core.Drawable); ok {
						emitter.Drawable = d
					}
				}
			}
			// Material: userdata or Lua function
			if v := L.GetField(opts, "material"); v != lua.LNil {
				switch val := v.(type) {
				case *lua.LUserData:
					if m, ok := val.Value.(core.Material); ok {
						emitter.Material = m
					}
				case *lua.LFunction:
					emitter.Material = materialFromLua(L, val)
				}
			}
			// Behaviors: table of BehaviorFunc userdata
			if v := L.GetField(opts, "behaviors"); v != lua.LNil {
				if bt, ok := v.(*lua.LTable); ok {
					var behaviors []core.BehaviorFunc
					bt.ForEach(func(_, val lua.LValue) {
						if ud, ok := val.(*lua.LUserData); ok {
							if fn, ok := ud.Value.(core.BehaviorFunc); ok {
								behaviors = append(behaviors, fn)
							}
						}
					})
					if len(behaviors) > 0 {
						emitter.Behaviors = behaviors
					}
				}
			}
			// Velocity range
			if v := L.GetField(opts, "velocity"); v != lua.LNil {
				if vt, ok := v.(*lua.LTable); ok {
					emitter.VelocityRange = &particle.VelocityRange{
						MinX: getNumberField(L, vt, "min_x", -1.5),
						MaxX: getNumberField(L, vt, "max_x", 1.5),
						MinY: getNumberField(L, vt, "min_y", 0),
						MaxY: getNumberField(L, vt, "max_y", 4),
					}
				}
			}
		}

		ud := L.NewUserData()
		ud.Value = core.Behavior(emitter)
		L.Push(ud)
		return 1
	}))

	// Emission params from bitmap
	L.SetField(pt, "compute_emission", L.NewFunction(func(L *lua.LState) int {
		bmUD := L.CheckUserData(1)
		drawableUD := L.CheckUserData(2)

		bm, ok := bmUD.Value.(*bitmap.Bitmap)
		if !ok {
			L.ArgError(1, "bitmap expected")
			return 0
		}
		drawable, ok := drawableUD.Value.(core.Drawable)
		if !ok {
			L.ArgError(2, "drawable expected")
			return 0
		}

		strategy := resolveEmissionStrategy(L, 3)

		params := particle.ComputeEmissionParams(bm, drawable, strategy)
		result := L.NewTable()
		pushVec2(L, params.Offset)
		L.SetField(result, "offset", L.Get(-1))
		L.Pop(1)
		L.SetField(result, "width", lua.LNumber(params.Width))
		L.Push(result)
		return 1
	}))

	// Particle materials
	L.SetField(pt, "velocity_color", L.NewFunction(particleVelocityColor))
	L.SetField(pt, "idle_and_motion", L.NewFunction(particleIdleAndMotion))
	L.SetField(pt, "braille_directional", L.NewFunction(particleBrailleDirectional))
	L.SetField(pt, "speed_states", L.NewFunction(particleSpeedStates))
	L.SetField(pt, "age_based_size", L.NewFunction(particleAgeBasedSize))
	L.SetField(pt, "rainbow_velocity", L.NewFunction(particleRainbowVelocity))
	L.SetField(pt, "rainbow_time", L.NewFunction(particleRainbowTime))
	L.SetField(pt, "fire_time", L.NewFunction(particleFireTime))
	L.SetField(pt, "cycle_materials", L.NewFunction(particleCycleMaterials))

	// Particle lifecycle
	L.SetField(pt, "age_and_despawn", L.NewFunction(particleAgeAndDespawn))

	// compose_materials(mat1, mat2, ...)
	// Accepts both Go material userdata and Lua functions.
	L.SetField(mod, "compose_materials", L.NewFunction(func(L *lua.LState) int {
		n := L.GetTop()
		materials := make([]core.Material, 0, n)
		for i := 1; i <= n; i++ {
			v := L.Get(i)
			switch val := v.(type) {
			case *lua.LUserData:
				if m, ok := val.Value.(core.Material); ok {
					materials = append(materials, m)
				} else {
					L.ArgError(i, "material expected")
					return 0
				}
			case *lua.LFunction:
				materials = append(materials, materialFromLua(L, val))
			default:
				L.ArgError(i, "material or function expected")
				return 0
			}
		}
		composed := core.ComposeMaterials(materials...)
		ud := L.NewUserData()
		ud.Value = core.Material(composed)
		L.Push(ud)
		return 1
	}))
}

// --- PointCloudSequence methods ---

func seqAddTarget(L *lua.LState) int {
	ud := L.CheckUserData(1)
	seq, ok := ud.Value.(*particle.PointCloudSequence)
	if !ok {
		L.ArgError(1, "point_cloud_sequence expected")
		return 0
	}

	opts := L.CheckTable(2)

	// Cloud (required)
	cloudVal := L.GetField(opts, "cloud")
	if cloudVal == lua.LNil {
		L.ArgError(2, "cloud field required")
		return 0
	}
	cloud := tableToCloud(L, cloudVal.(*lua.LTable))

	// Duration
	duration := getNumberField(L, opts, "duration", 4.0)

	// Strategy
	var strategy particle.DistributionStrategy
	strategyVal := L.GetField(opts, "strategy")
	if strategyVal != lua.LNil {
		if stratUD, ok := strategyVal.(*lua.LUserData); ok {
			if s, ok := stratUD.Value.(particle.DistributionStrategy); ok {
				strategy = s
			}
		}
	}
	if strategy == nil {
		strategy = particle.LinearDistribution()
	}

	// Phases
	var phases []particle.TransitionPhase
	phasesVal := L.GetField(opts, "phases")
	if phasesVal != lua.LNil {
		phasesTable := phasesVal.(*lua.LTable)
		phasesTable.ForEach(func(_, v lua.LValue) {
			if phaseUD, ok := v.(*lua.LUserData); ok {
				if p, ok := phaseUD.Value.(particle.TransitionPhase); ok {
					phases = append(phases, p)
				}
			}
		})
	}
	if len(phases) == 0 {
		phases = []particle.TransitionPhase{particle.SeekPhase()}
	}

	seq.AddTarget(particle.MorphTarget{
		Cloud:    cloud,
		Duration: duration,
		Strategy: strategy,
		Phases:   phases,
	})

	return 0
}

func seqSetLoop(L *lua.LState) int {
	ud := L.CheckUserData(1)
	seq, ok := ud.Value.(*particle.PointCloudSequence)
	if !ok {
		L.ArgError(1, "point_cloud_sequence expected")
		return 0
	}
	loop := L.CheckBool(2)
	seq.SetLoop(loop)
	return 0
}

func seqParticles(L *lua.LState) int {
	seqUD := L.CheckUserData(1)
	seq, ok := seqUD.Value.(*particle.PointCloudSequence)
	if !ok {
		L.ArgError(1, "point_cloud_sequence expected")
		return 0
	}

	// Need to get the world from somewhere - we'll use first particle's world
	particles := seq.Particles()
	t := L.NewTable()
	// We don't have the world easily, so we return entity IDs
	for i, p := range particles {
		t.RawSetInt(i+1, lua.LNumber(p))
	}
	L.Push(t)
	return 1
}

// --- Particle material constructors ---

func particleVelocityColor(L *lua.LState) int {
	opts := L.CheckTable(1)
	gradient := particle.ColorGradient{
		MinSpeed: getNumberField(L, opts, "min_speed", 0),
		MaxSpeed: getNumberField(L, opts, "max_speed", 10),
	}
	if v := L.GetField(opts, "min_color"); v != lua.LNil {
		if ud, ok := v.(*lua.LUserData); ok {
			if c, ok := ud.Value.(core.Color); ok {
				gradient.MinColor = c
			}
		}
	}
	if v := L.GetField(opts, "max_color"); v != lua.LNil {
		if ud, ok := v.(*lua.LUserData); ok {
			if c, ok := ud.Value.(core.Color); ok {
				gradient.MaxColor = c
			}
		}
	}
	mat := particle.VelocityColor(gradient)
	ud := L.NewUserData()
	ud.Value = core.Material(mat)
	L.Push(ud)
	return 1
}

func particleIdleAndMotion(L *lua.LState) int {
	runesStr := L.CheckString(1)
	threshold := float64(L.OptNumber(2, 0.5))
	idleRunes := []rune(runesStr)
	mat := particle.IdleAndMotion(idleRunes, threshold)
	ud := L.NewUserData()
	ud.Value = core.Material(mat)
	L.Push(ud)
	return 1
}

func particleBrailleDirectional(L *lua.LState) int {
	mat := particle.BrailleDirectional()
	ud := L.NewUserData()
	ud.Value = core.Material(mat)
	L.Push(ud)
	return 1
}

func particleSpeedStates(L *lua.LState) int {
	threshTable := L.CheckTable(1)
	runesStr := L.CheckString(2)

	var thresholds []float64
	threshTable.ForEach(func(_, v lua.LValue) {
		if n, ok := v.(lua.LNumber); ok {
			thresholds = append(thresholds, float64(n))
		}
	})

	mat := particle.SpeedStates(thresholds, []rune(runesStr))
	ud := L.NewUserData()
	ud.Value = core.Material(mat)
	L.Push(ud)
	return 1
}

func particleAgeBasedSize(L *lua.LState) int {
	threshTable := L.CheckTable(1)
	runesStr := L.CheckString(2)

	var thresholds []float64
	threshTable.ForEach(func(_, v lua.LValue) {
		if n, ok := v.(lua.LNumber); ok {
			thresholds = append(thresholds, float64(n))
		}
	})

	mat := particle.AgeBasedSize(thresholds, []rune(runesStr))
	ud := L.NewUserData()
	ud.Value = core.Material(mat)
	L.Push(ud)
	return 1
}

func particleRainbowVelocity(L *lua.LState) int {
	minSpeed := float64(L.CheckNumber(1))
	maxSpeed := float64(L.CheckNumber(2))
	mat := particle.RainbowVelocity(minSpeed, maxSpeed)
	ud := L.NewUserData()
	ud.Value = core.Material(mat)
	L.Push(ud)
	return 1
}

func particleRainbowTime(L *lua.LState) int {
	frequency := float64(L.CheckNumber(1))
	mat := particle.RainbowTime(frequency)
	ud := L.NewUserData()
	ud.Value = core.Material(mat)
	L.Push(ud)
	return 1
}

func particleFireTime(L *lua.LState) int {
	frequency := float64(L.CheckNumber(1))
	mat := particle.FireTime(frequency)
	ud := L.NewUserData()
	ud.Value = core.Material(mat)
	L.Push(ud)
	return 1
}

func particleCycleMaterials(L *lua.LState) int {
	period := float64(L.CheckNumber(1))
	fadeRatio := float64(L.CheckNumber(2))

	var materials []core.Material
	for i := 3; i <= L.GetTop(); i++ {
		v := L.Get(i)
		switch val := v.(type) {
		case *lua.LUserData:
			if m, ok := val.Value.(core.Material); ok {
				materials = append(materials, m)
			} else {
				L.ArgError(i, "material expected")
				return 0
			}
		case *lua.LFunction:
			materials = append(materials, materialFromLua(L, val))
		default:
			L.ArgError(i, "material or function expected")
			return 0
		}
	}

	mat := particle.CycleMaterials(period, fadeRatio, materials...)
	ud := L.NewUserData()
	ud.Value = core.Material(mat)
	L.Push(ud)
	return 1
}

func particleAgeAndDespawn(L *lua.LState) int {
	pushBehavior(L, particle.AgeAndDespawn())
	return 1
}

// --- Helpers ---

func tableToCloud(L *lua.LState, t *lua.LTable) []fmath.Vec2 {
	cloud := make([]fmath.Vec2, 0, t.Len())
	t.ForEach(func(_, v lua.LValue) {
		if ud, ok := v.(*lua.LUserData); ok {
			if vec, ok := ud.Value.(fmath.Vec2); ok {
				cloud = append(cloud, vec)
			}
		}
	})
	return cloud
}

func tableToEntities(L *lua.LState, t *lua.LTable) []core.Entity {
	entities := make([]core.Entity, 0, t.Len())
	t.ForEach(func(_, v lua.LValue) {
		if ud, ok := v.(*lua.LUserData); ok {
			if e, ok := ud.Value.(luaEntity); ok {
				entities = append(entities, e.ID)
			}
		}
	})
	return entities
}

// resolveEmissionStrategy accepts a string or EmissionStrategy userdata at stack position n.
func resolveEmissionStrategy(L *lua.LState, n int) particle.EmissionStrategy {
	v := L.Get(n)
	switch val := v.(type) {
	case *lua.LUserData:
		if s, ok := val.Value.(particle.EmissionStrategy); ok {
			return s
		}
	case lua.LString:
		switch string(val) {
		case "bottom":
			return particle.BottomEdge
		case "top":
			return particle.TopEdge
		case "left":
			return particle.LeftEdge
		case "right":
			return particle.RightEdge
		}
	}
	return particle.BottomEdge
}

// resolveParticleEasing maps easing names to particle package easing functions.
func resolveParticleEasing(name string) func(float64) float64 {
	switch name {
	case "linear":
		return particle.EaseLinear
	case "in_quad":
		return particle.EaseInQuad
	case "out_quad":
		return particle.EaseOutQuad
	case "in_out_quad":
		return particle.EaseInOutQuad
	case "in_cubic":
		return particle.EaseInCubic
	case "out_cubic":
		return particle.EaseOutCubic
	default:
		return particle.EaseInOutQuad
	}
}
