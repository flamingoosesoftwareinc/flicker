-- circle.lua: A bouncing SDF circle rendered with braille encoding.
local f = require("flicker")

local hero
local cx, cy

f.on_enter(function(world, ctx)
    cx = ctx.width / 2
    cy = ctx.height / 2

    hero = world:spawn()
    hero:set_transform({
        position = f.vec3(cx, cy, 0),
        scale = f.vec3(1, 1, 1),
    })

    -- Draw a filled circle into a bitmap using the SDF
    local radius = 12
    local size = radius * 2 + 1
    local bm = f.bitmap.new(size, size)
    local circle = f.sdf.circle(radius)
    local green = f.color(0, 255, 128)

    for y = 0, size - 1 do
        for x = 0, size - 1 do
            local d = circle(f.vec2(x - radius, y - radius))
            if d <= 0 then
                bm:set_dot(x, y, green)
            end
        end
    end

    hero:set_drawable(f.bitmap.braille(bm))
    hero:set_material(f.material.solid(f.color(0, 255, 128)))
    world:add_root(hero)
end)

f.on_update(function(world, time)
    local x = cx + math.sin(time.total * 2) * 20
    local y = cy + math.cos(time.total * 1.5) * 8
    hero:set_position(f.vec3(x, y, 0))
end)
