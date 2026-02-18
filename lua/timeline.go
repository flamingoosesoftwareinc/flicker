package lua

import (
	"flicker/core"
	"flicker/fmath"
	lua "github.com/epikur-io/gopher-lua"
)

const (
	typeTimeline = "flicker.timeline"
	typeTrack    = "flicker.track"
)

func registerTimelineModule(L *lua.LState, mod *lua.LTable, engine *Engine) {
	// Timeline metatable
	mt := registerType(L, typeTimeline, map[string]lua.LGFunction{
		"add_track": timelineAddTrack,
		"start":     timelineStart,
		"pause":     timelinePause,
		"resume":    timelineResume,
		"stop":      timelineStop,
		"set_loop":  timelineSetLoop,
		"cleanup":   timelineCleanup,
	})
	L.SetField(mt, "__index", L.GetField(mt, "methods"))

	// Track metatable
	tmt := registerType(L, typeTrack, map[string]lua.LGFunction{
		"add":      trackAdd,
		"at":       trackAt,
		"sequence": trackSequence,
	})
	L.SetField(tmt, "__index", L.GetField(tmt, "methods"))

	// timeline sub-table
	tl := L.NewTable()
	L.SetField(mod, "timeline", tl)

	L.SetField(tl, "new", L.NewFunction(func(L *lua.LState) int {
		w := checkWorld(L, 1)
		timeline := core.NewTimeline(w)
		pushUserData(L, typeTimeline, timeline)
		return 1
	}))

	L.SetField(tl, "tween", L.NewFunction(timelineTween))
	L.SetField(tl, "delay", L.NewFunction(timelineDelay))
	L.SetField(tl, "callback", L.NewFunction(timelineCallback))
	L.SetField(tl, "parallel", L.NewFunction(timelineParallel))
	L.SetField(tl, "sequence_clip", L.NewFunction(timelineSequenceClip))

	// Text keyframe clip (defined in text.go)
	registerTextKeyframesClip(L, tl)
}

func checkTimeline(L *lua.LState, n int) *core.Timeline {
	ud := L.CheckUserData(n)
	if tl, ok := ud.Value.(*core.Timeline); ok {
		return tl
	}
	L.ArgError(n, "timeline expected")
	return nil
}

func checkTrack(L *lua.LState, n int) *core.Track {
	ud := L.CheckUserData(n)
	if t, ok := ud.Value.(*core.Track); ok {
		return t
	}
	L.ArgError(n, "track expected")
	return nil
}

func checkClip(L *lua.LState, n int) core.Clip {
	ud := L.CheckUserData(n)
	if c, ok := ud.Value.(core.Clip); ok {
		return c
	}
	L.ArgError(n, "clip expected")
	return nil
}

// --- Timeline methods ---

func timelineAddTrack(L *lua.LState) int {
	tl := checkTimeline(L, 1)
	track := tl.AddTrack()
	pushUserData(L, typeTrack, track)
	return 1
}

func timelineStart(L *lua.LState) int {
	tl := checkTimeline(L, 1)
	tl.Start()
	return 0
}

func timelinePause(L *lua.LState) int {
	tl := checkTimeline(L, 1)
	tl.Pause()
	return 0
}

func timelineResume(L *lua.LState) int {
	tl := checkTimeline(L, 1)
	tl.Resume()
	return 0
}

func timelineStop(L *lua.LState) int {
	tl := checkTimeline(L, 1)
	tl.Stop()
	return 0
}

func timelineSetLoop(L *lua.LState) int {
	tl := checkTimeline(L, 1)
	loop := L.CheckBool(2)
	tl.SetLoop(loop)
	return 0
}

func timelineCleanup(L *lua.LState) int {
	tl := checkTimeline(L, 1)
	tl.Cleanup()
	return 0
}

// --- Track methods ---

func trackAdd(L *lua.LState) int {
	track := checkTrack(L, 1)
	clip := checkClip(L, 2)
	track.Add(clip)
	// Return track for chaining
	L.Push(L.Get(1))
	return 1
}

func trackAt(L *lua.LState) int {
	track := checkTrack(L, 1)
	time := float64(L.CheckNumber(2))
	clip := checkClip(L, 3)
	track.At(time, clip)
	L.Push(L.Get(1))
	return 1
}

func trackSequence(L *lua.LState) int {
	track := checkTrack(L, 1)
	n := L.GetTop()
	for i := 2; i <= n; i++ {
		clip := checkClip(L, i)
		track.Add(clip)
	}
	L.Push(L.Get(1))
	return 1
}

// --- Clip constructors ---

// resolveEasing maps an easing name string to a Go easing function.
func resolveEasing(name string) func(float64) float64 {
	switch name {
	case "linear":
		return fmath.EaseLinear
	case "in_quad":
		return fmath.EaseInQuad
	case "out_quad":
		return fmath.EaseOutQuad
	case "in_out_quad":
		return fmath.EaseInOutQuad
	case "in_cubic":
		return fmath.EaseInCubic
	case "out_cubic":
		return fmath.EaseOutCubic
	case "in_out_cubic":
		return fmath.EaseInOutCubic
	case "in_elastic":
		return fmath.EaseInElastic
	case "out_elastic":
		return fmath.EaseOutElastic
	case "out_bounce":
		return fmath.EaseOutBounce
	default:
		return fmath.EaseLinear
	}
}

// timeline.tween(entity, property, {from, to, duration, easing})
func timelineTween(L *lua.LState) int {
	e := checkEntity(L, 1)
	property := L.CheckString(2)
	opts := L.CheckTable(3)

	from := getNumberField(L, opts, "from", 0)
	to := getNumberField(L, opts, "to", 0)
	duration := getNumberField(L, opts, "duration", 1)
	easingName := getStringField(L, opts, "easing", "linear")

	clip := core.NewPropertyTweenClip(e.ID, property, from, to, duration)
	clip.WithEasing(resolveEasing(easingName))

	pushUserData(L, "flicker.clip", core.Clip(clip))
	return 1
}

// timeline.delay(duration)
func timelineDelay(L *lua.LState) int {
	duration := float64(L.CheckNumber(1))
	clip := core.NewDelayClip(duration)
	pushUserData(L, "flicker.clip", core.Clip(clip))
	return 1
}

// timeline.callback(fn)
func timelineCallback(L *lua.LState) int {
	fn := L.CheckFunction(1)
	clip := core.NewCallbackClip(func(w *core.World, t core.Time) {
		pushWorld(L, w)
		worldUD := L.Get(-1)
		L.Pop(1)

		timeTable := L.NewTable()
		L.SetField(timeTable, "total", lua.LNumber(t.Total))
		L.SetField(timeTable, "delta", lua.LNumber(t.Delta))

		_ = L.CallByParam(lua.P{
			Fn:      fn,
			NRet:    0,
			Protect: true,
		}, worldUD, timeTable)
	})
	pushUserData(L, "flicker.clip", core.Clip(clip))
	return 1
}

// timeline.parallel(clip1, clip2, ...)
func timelineParallel(L *lua.LState) int {
	n := L.GetTop()
	clips := make([]core.Clip, 0, n)
	for i := 1; i <= n; i++ {
		clips = append(clips, checkClip(L, i))
	}
	clip := core.NewParallelClip(clips...)
	pushUserData(L, "flicker.clip", core.Clip(clip))
	return 1
}

// timeline.sequence_clip(clip1, clip2, ...)
func timelineSequenceClip(L *lua.LState) int {
	n := L.GetTop()
	clips := make([]core.Clip, 0, n)
	for i := 1; i <= n; i++ {
		clips = append(clips, checkClip(L, i))
	}
	clip := core.NewSequenceClip(clips...)
	pushUserData(L, "flicker.clip", core.Clip(clip))
	return 1
}
