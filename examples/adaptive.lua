-- adaptive.lua: Compares adaptive block encoding vs braille for text rendering.
-- Adaptive encoding uses ~130 Unicode characters (sextants, diagonal blocks,
-- triangular blocks) to find the best-fit shape per cell, producing sharper
-- text than braille at small font sizes.
local f = require("flicker")

f.on_enter(function(world, ctx)
    local font = f.asset.load_font("Oxanium/static/Oxanium-Bold.ttf")

    -- Header labels
    local label_size = ctx.height * 0.08
    local label_y = ctx.height * 0.03

    local braille_label = f.asset.rasterize_text("BRAILLE", {
        font = font,
        size = label_size,
        color = f.color(100, 100, 100),
    })
    local bl = world:spawn()
    bl:set_position(f.vec3(ctx.width * 0.25 - braille_label.width / 2, label_y, 0))
    bl:set_drawable(f.bitmap.adaptive(braille_label.bitmap))
    world:add_root(bl)

    local adaptive_label = f.asset.rasterize_text("ADAPTIVE", {
        font = font,
        size = label_size,
        color = f.color(100, 100, 100),
    })
    local al = world:spawn()
    al:set_position(f.vec3(ctx.width * 0.75 - adaptive_label.width / 2, label_y, 0))
    al:set_drawable(f.bitmap.adaptive(adaptive_label.bitmap))
    world:add_root(al)

    -- Divider line down the center
    local div_h = ctx.height * 9  -- bitmap pixels (adaptive = 9 rows per cell)
    local div_bm = f.bitmap.new(1, math.floor(div_h))
    for y = 0, math.floor(div_h) - 1 do
        div_bm:set_dot(0, y, f.color(60, 60, 60))
    end
    local div = world:spawn()
    div:set_position(f.vec3(ctx.width / 2, 0, 0))
    div:set_drawable(f.bitmap.adaptive(div_bm))
    world:add_root(div)

    -- Render text at multiple sizes to show the difference
    local words = { "FLICKER", "Adaptive", "Quality" }
    local sizes = { 0.80, 0.60, 0.45 }
    local colors = {
        f.color(255, 200, 50),
        f.color(100, 200, 255),
        f.color(200, 100, 255),
    }

    local y_cursor = ctx.height * 0.15
    for i, word in ipairs(words) do
        local text_size = ctx.height * sizes[i]

        -- Left side: braille
        local layout_b = f.asset.rasterize_text(word, {
            font = font,
            size = text_size,
            color = colors[i],
        })
        local eb = world:spawn()
        eb:set_position(f.vec3(
            ctx.width * 0.25 - layout_b.width / 4,
            y_cursor,
            0
        ))
        eb:set_drawable(f.bitmap.braille(layout_b.bitmap))
        world:add_root(eb)

        -- Right side: adaptive
        local layout_a = f.asset.rasterize_text(word, {
            font = font,
            size = text_size,
            color = colors[i],
        })
        local ea = world:spawn()
        ea:set_position(f.vec3(
            ctx.width * 0.75 - layout_a.width / 12,
            y_cursor,
            0
        ))
        ea:set_drawable(f.bitmap.adaptive(layout_a.bitmap))
        world:add_root(ea)

        -- Advance y based on the larger of the two encoded heights
        local braille_h = math.ceil(layout_b.height / 4)
        local adaptive_h = math.ceil(layout_a.height / 9)
        y_cursor = y_cursor + math.max(braille_h, adaptive_h) + 2
    end
end)
