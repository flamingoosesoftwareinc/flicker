-- text.lua: Animated text with fire trail effect.
local f = require("flicker")

local title, subtitle
local title_w, sub_w

f.on_enter(function(world, ctx)
    -- Enable fire trail for dramatic effect
    f.set_trail(0, f.trail.fire(0.94, 2.0))

    local font = f.asset.load_font("Oxanium/static/Oxanium-Bold.ttf")

    -- Title text
    local title_layout = f.asset.rasterize_text("FLICKER", {
        font = font,
        size = ctx.height * 0.35,
        color = f.color(255, 200, 50),
    })
    title_w = title_layout.width

    title = world:spawn()
    title:set_transform({
        position = f.vec3(ctx.width / 2 - title_w / 2, ctx.height * 0.2, 0),
        scale = f.vec3(1, 1, 1),
    })
    title:set_drawable(f.bitmap.half_block(title_layout.bitmap))
    world:add_root(title)

    -- Subtitle
    local sub_layout = f.asset.rasterize_text("LUA ENGINE", {
        font = font,
        size = ctx.height * 0.2,
        color = f.color(100, 200, 255),
    })
    sub_w = sub_layout.width

    subtitle = world:spawn()
    subtitle:set_transform({
        position = f.vec3(ctx.width / 2 - sub_w / 2, ctx.height * 0.6, 0),
        scale = f.vec3(1, 1, 1),
    })
    subtitle:set_drawable(f.bitmap.half_block(sub_layout.bitmap))
    world:add_root(subtitle)
end)

f.on_update(function(world, time)
    -- Gentle horizontal oscillation
    local drift = math.sin(time.total * 1.2) * 8
    local tr = title:transform()
    title:set_position(f.vec3(tr.position.x + drift * time.delta, tr.position.y, 0))

    -- Subtitle bobs vertically
    local bob = math.sin(time.total * 2.5) * 3
    local str = subtitle:transform()
    subtitle:set_position(f.vec3(str.position.x, str.position.y + bob * time.delta, 0))
end)
