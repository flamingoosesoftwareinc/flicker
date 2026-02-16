-- shapes.lua: Multiple SDF shapes with ghost trail effect.
local f = require("flicker")

local shapes = {}

f.on_enter(function(world, ctx)
    -- Enable ghost trail for motion blur
    f.set_trail(0, f.trail.ghost(0.92))

    local green = f.color(0, 255, 128)
    local blue = f.color(100, 150, 255)
    local red = f.color(255, 100, 80)

    -- Shape 1: Circle
    local r1 = 8
    local sz1 = r1 * 2 + 1
    local bm1 = f.bitmap.new(sz1, sz1)
    local circle = f.sdf.circle(r1)
    for y = 0, sz1 - 1 do
        for x = 0, sz1 - 1 do
            if circle(f.vec2(x - r1, y - r1)) <= 0 then
                bm1:set_dot(x, y, green)
            end
        end
    end

    local e1 = world:spawn()
    e1:set_transform({ position = f.vec3(ctx.width * 0.25, ctx.height / 2, 0), scale = f.vec3(1, 1, 1) })
    e1:set_drawable(f.bitmap.braille(bm1))
    world:add_root(e1)
    table.insert(shapes, { entity = e1, cx = ctx.width * 0.25, cy = ctx.height / 2 })

    -- Shape 2: Hexagon
    local r2 = 8
    local sz2 = r2 * 2 + 1
    local bm2 = f.bitmap.new(sz2, sz2)
    local hex = f.sdf.hexagon(r2)
    for y = 0, sz2 - 1 do
        for x = 0, sz2 - 1 do
            if hex(f.vec2(x - r2, y - r2)) <= 0 then
                bm2:set_dot(x, y, blue)
            end
        end
    end

    local e2 = world:spawn()
    e2:set_transform({ position = f.vec3(ctx.width * 0.5, ctx.height / 2, 0), scale = f.vec3(1, 1, 1) })
    e2:set_drawable(f.bitmap.braille(bm2))
    world:add_root(e2)
    table.insert(shapes, { entity = e2, cx = ctx.width * 0.5, cy = ctx.height / 2 })

    -- Shape 3: Smooth union of circle + box (blob)
    local r3 = 10
    local sz3 = r3 * 2 + 5
    local bm3 = f.bitmap.new(sz3, sz3)
    local blob = f.sdf.smooth_union(
        f.sdf.circle(6),
        f.sdf.box(4, 4),
        3.0
    )
    local half3 = sz3 / 2
    for y = 0, sz3 - 1 do
        for x = 0, sz3 - 1 do
            if blob(f.vec2(x - half3, y - half3)) <= 0 then
                bm3:set_dot(x, y, red)
            end
        end
    end

    local e3 = world:spawn()
    e3:set_transform({ position = f.vec3(ctx.width * 0.75, ctx.height / 2, 0), scale = f.vec3(1, 1, 1) })
    e3:set_drawable(f.bitmap.braille(bm3))
    world:add_root(e3)
    table.insert(shapes, { entity = e3, cx = ctx.width * 0.75, cy = ctx.height / 2 })
end)

f.on_update(function(world, time)
    for i, s in ipairs(shapes) do
        local phase = (i - 1) * 2.094 -- 120 degrees apart
        local x = s.cx + math.sin(time.total * 1.5 + phase) * 10
        local y = s.cy + math.cos(time.total * 2.0 + phase) * 4
        s.entity:set_position(f.vec3(x, y, 0))
    end
end)
