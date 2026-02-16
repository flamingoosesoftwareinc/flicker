-- physics.lua: Particles with gravity, attractor, and drag.
local f = require("flicker")

local particles = {}
local center

f.on_enter(function(world, ctx)
    f.set_trail(0, f.trail.ghost(0.9))

    center = f.vec2(ctx.width / 2, ctx.height / 2)

    -- Spawn particles in a circle around center
    local count = 8
    local radius = 12
    local pixel = f.bitmap.new(2, 2)
    local cyan = f.color(100, 200, 255)
    for y = 0, 1 do
        for x = 0, 1 do
            pixel:set_dot(x, y, cyan)
        end
    end

    for i = 0, count - 1 do
        local angle = i * 2 * math.pi / count
        local px = center.x + radius * math.cos(angle)
        local py = center.y + radius * math.sin(angle)

        local p = world:spawn()
        p:set_transform({
            position = f.vec3(px, py, 0),
            scale = f.vec3(1, 1, 1),
        })
        p:set_drawable(f.bitmap.braille(pixel))
        p:set_body({
            velocity = f.vec2(
                -math.sin(angle) * 10,
                math.cos(angle) * 10
            ),
        })
        world:add_root(p)
        table.insert(particles, p)
    end
end)

f.on_update(function(world, time)
    for _, p in ipairs(particles) do
        local tr = p:transform()
        local body = p:body()
        if tr and body then
            local pos = f.vec2(tr.position.x, tr.position.y)
            local delta = f.vec2(center.x - pos.x, center.y - pos.y)
            local dist_sq = delta.x * delta.x + delta.y * delta.y
            if dist_sq < 0.01 then dist_sq = 0.01 end

            -- Attractor force
            local force_mag = 200 / dist_sq
            local dist = math.sqrt(dist_sq)
            local dir = f.vec2(delta.x / dist, delta.y / dist)

            local vx = body.velocity.x + dir.x * force_mag * time.delta
            local vy = body.velocity.y + dir.y * force_mag * time.delta

            -- Drag
            vx = vx * (1 - 0.5 * time.delta)
            vy = vy * (1 - 0.5 * time.delta)

            p:set_position(f.vec3(
                tr.position.x + vx * time.delta,
                tr.position.y + vy * time.delta,
                0
            ))
        end
    end
end)
