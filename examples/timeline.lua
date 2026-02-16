-- timeline.lua: Animated boxes using the timeline system.
local f = require("flicker")

f.on_enter(function(world, ctx)
    f.set_trail(0, f.trail.ghost(0.85))

    -- Create a box
    local r = 4
    local sz = r * 2 + 1
    local bm = f.bitmap.new(sz, sz)
    local orange = f.color(255, 180, 50)
    local box_sdf = f.sdf.rounded_box(r, r, 2)
    for y = 0, sz - 1 do
        for x = 0, sz - 1 do
            if box_sdf(f.vec2(x - r, y - r)) <= 0 then
                bm:set_dot(x, y, orange)
            end
        end
    end

    local box = world:spawn()
    box:set_transform({
        position = f.vec3(2, ctx.height / 2, 0),
        scale = f.vec3(1, 1, 1),
    })
    box:set_drawable(f.bitmap.braille(bm))
    world:add_root(box)

    -- Timeline: slide right, then slide down, then slide left
    local tl = f.timeline.new(world)
    tl:set_loop(true)

    local track = tl:add_track()
    track:add(f.timeline.tween(box, "position.x", {
        from = 2, to = ctx.width - 10,
        duration = 2.0,
        easing = "in_out_quad",
    }))
    track:add(f.timeline.tween(box, "position.y", {
        from = ctx.height / 2, to = ctx.height - 5,
        duration = 1.0,
        easing = "out_bounce",
    }))
    track:add(f.timeline.tween(box, "position.x", {
        from = ctx.width - 10, to = 2,
        duration = 2.0,
        easing = "in_out_cubic",
    }))
    track:add(f.timeline.tween(box, "position.y", {
        from = ctx.height - 5, to = ctx.height / 2,
        duration = 1.0,
        easing = "out_elastic",
    }))

    tl:start()
end)
