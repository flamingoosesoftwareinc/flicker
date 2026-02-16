package lua

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"flicker/core"
	"flicker/terminal"
	"github.com/sebdah/goldie/v2"
)

// runLuaGolden is a helper that loads a Lua script, runs the scene for the
// given number of frames, captures each frame via SimScreen, and asserts
// against golden files managed by goldie.
func runLuaGolden(t *testing.T, name, script string, w, h, frames int, dt float64) {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "test.lua")
	if err := os.WriteFile(path, []byte(script), 0o644); err != nil {
		t.Fatal(err)
	}

	engine := NewEngine()
	defer engine.Close()

	scene, err := engine.Load(path, w, h)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	screen := terminal.NewSimScreen(w, h)
	canvas := core.NewCanvas(w, h)

	scene.OnEnter(core.SceneContext{Width: w, Height: h})
	scene.OnReady()

	for i := 0; i < frames; i++ {
		ti := core.Time{
			Total: float64(i+1) * dt,
			Delta: dt,
		}
		scene.OnUpdate(ti)

		canvas.Clear()
		scene.Render(canvas, ti)
		screen.Flush(canvas)
	}

	scene.OnExit()

	var b strings.Builder
	for i, frame := range screen.Frames() {
		fmt.Fprintf(&b, "--- frame %d ---\n", i)
		b.WriteString(frame)
		b.WriteByte('\n')
	}

	g := goldie.New(t)
	g.Assert(t, name, []byte(b.String()))
}

func TestGoldenLuaSDF(t *testing.T) {
	script := `
local f = require("flicker")

local circle_ent

f.on_enter(function(world, ctx)
    -- Create an SDF circle rendered as braille
    local r = 6
    local sz = r * 2 + 1
    local bm = f.bitmap.new(sz, sz)
    local circle = f.sdf.circle(r)
    local green = f.color(0, 255, 128)
    for y = 0, sz - 1 do
        for x = 0, sz - 1 do
            if circle(f.vec2(x - r, y - r)) <= 0 then
                bm:set_dot(x, y, green)
            end
        end
    end

    circle_ent = world:spawn()
    circle_ent:set_transform({
        position = f.vec3(5, 2, 0),
        scale = f.vec3(1, 1, 1),
    })
    circle_ent:set_drawable(f.bitmap.braille(bm))
    world:add_root(circle_ent)
end)

f.on_update(function(world, time)
    -- Move rightward each frame
    local x = 5 + time.total * 10
    circle_ent:set_position(f.vec3(x, 2, 0))
end)
`
	runLuaGolden(t, "lua_sdf_circle", script, 40, 12, 4, 0.5)
}

func TestGoldenLuaTimeline(t *testing.T) {
	script := `
local f = require("flicker")

local box

f.on_enter(function(world, ctx)
    box = world:spawn()
    box:set_transform({
        position = f.vec3(2, 3, 0),
        scale = f.vec3(1, 1, 1),
    })

    -- Simple 5x3 bitmap rectangle
    local bm = f.bitmap.new(5, 3)
    local white = f.color(255, 255, 255)
    for y = 0, 2 do
        for x = 0, 4 do
            bm:set_dot(x, y, white)
        end
    end
    box:set_drawable(f.bitmap.half_block(bm))
    world:add_root(box)

    -- Create timeline that tweens position.x from 2 to 30 over 1.5s
    local tl = f.timeline.new(world)
    local track = tl:add_track()
    track:add(f.timeline.tween(box, "position.x", {
        from = 2,
        to = 30,
        duration = 1.5,
        easing = "in_out_quad",
    }))
    tl:start()
end)
`
	runLuaGolden(t, "lua_timeline", script, 40, 10, 6, 0.5)
}

func TestGoldenLuaPhysics(t *testing.T) {
	script := `
local f = require("flicker")

local particle

f.on_enter(function(world, ctx)
    particle = world:spawn()
    particle:set_transform({
        position = f.vec3(5, 2, 0),
        scale = f.vec3(1, 1, 1),
    })

    local bm = f.bitmap.new(2, 2)
    local cyan = f.color(100, 200, 255)
    for y = 0, 1 do
        for x = 0, 1 do
            bm:set_dot(x, y, cyan)
        end
    end
    particle:set_drawable(f.bitmap.braille(bm))
    particle:set_body({ velocity = f.vec2(15, -5) })
    world:add_root(particle)
end)

f.on_update(function(world, time)
    -- Manual physics: apply gravity and integrate
    local tr = particle:transform()
    local body_data = particle:body()
    if tr and body_data then
        -- Simple manual euler: just move by velocity * delta
        local vx = body_data.velocity.x
        local vy = body_data.velocity.y + 20 * time.delta  -- gravity
        particle:set_position(f.vec3(
            tr.position.x + vx * time.delta,
            tr.position.y + vy * time.delta,
            0
        ))
    end
end)
`
	runLuaGolden(t, "lua_physics", script, 40, 12, 5, 0.2)
}

func TestGoldenLuaMultiShape(t *testing.T) {
	script := `
local f = require("flicker")

local shapes = {}

f.on_enter(function(world, ctx)
    -- Create two shapes side by side
    local colors = {
        f.color(255, 100, 80),
        f.color(80, 100, 255),
    }

    for i = 1, 2 do
        local r = 4
        local sz = r * 2 + 1
        local bm = f.bitmap.new(sz, sz)

        local sdf
        if i == 1 then
            sdf = f.sdf.circle(r)
        else
            sdf = f.sdf.box(r, r)
        end

        for y = 0, sz - 1 do
            for x = 0, sz - 1 do
                if sdf(f.vec2(x - r, y - r)) <= 0 then
                    bm:set_dot(x, y, colors[i])
                end
            end
        end

        local e = world:spawn()
        e:set_transform({
            position = f.vec3(5 + (i - 1) * 18, 3, 0),
            scale = f.vec3(1, 1, 1),
        })
        e:set_drawable(f.bitmap.braille(bm))
        world:add_root(e)
        table.insert(shapes, { entity = e, base_x = 5 + (i - 1) * 18 })
    end
end)

f.on_update(function(world, time)
    for i, s in ipairs(shapes) do
        local phase = (i - 1) * 3.14159
        local x = s.base_x + math.sin(time.total * 2 + phase) * 5
        s.entity:set_position(f.vec3(x, 3, 0))
    end
end)
`
	runLuaGolden(t, "lua_multi_shape", script, 40, 12, 4, 0.5)
}
