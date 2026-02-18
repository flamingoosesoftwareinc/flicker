-- new_apis.lua: Demonstrates engine APIs added in recent commits.
-- Press SPACE to advance scenes. Press 'q' or ESC to quit.
--
-- Features demonstrated:
--   1. Drawable encodings: full_block, bg_block, rect (vs half_block, braille)
--   2. Bitmap SDF + animated threshold reveal
--   3. Blend modes on compositor layers
--   4. Layer post-process shader
--   5. Camera zoom and panning
--   6. Age component + particle age_and_despawn lifecycle
local f = require("flicker")

local sw, sh

-- HSV helper (reused for coloring)
local function hsv_to_rgb(h, s, v)
    h = h % 360
    local c = v * s
    local x = c * (1 - math.abs((h / 60) % 2 - 1))
    local m = v - c
    local r, g, b
    if h < 60 then      r, g, b = c, x, 0
    elseif h < 120 then r, g, b = x, c, 0
    elseif h < 180 then r, g, b = 0, c, x
    elseif h < 240 then r, g, b = 0, x, c
    elseif h < 300 then r, g, b = x, 0, c
    else                r, g, b = c, 0, x
    end
    return (r + m) * 255, (g + m) * 255, (b + m) * 255
end

-- Helper: rasterize centered title text
local function make_title(world, text, size_frac, y_frac, color)
    local font = f.asset.load_font("Oxanium/static/Oxanium-ExtraLight.ttf")
    local layout = f.asset.rasterize_text(text, {
        font = font,
        size = sh * size_frac,
        color = color or f.color(255, 255, 255),
    })
    local entity = world:spawn()
    entity:set_position(f.vec3(sw / 2 - layout.width / 6, sh * y_frac, 0))
    entity:set_drawable(f.bitmap.adaptive(layout.bitmap))
    world:add_root(entity)
    return entity, layout
end

-- Helper: rasterize a label below a position
local function make_label(world, font, text, cx, y)
    local layout = f.asset.rasterize_text(text, {
        font = font,
        size = sh * 0.12,
        color = f.color(200, 200, 200),
    })
    local lbl = world:spawn()
    lbl:set_position(f.vec3(cx - layout.width / 6, y, 0))
    lbl:set_drawable(f.bitmap.adaptive(layout.bitmap))
    world:add_root(lbl)
    return lbl, layout
end

-- ============================================================================
-- Scene 1: Drawable encodings comparison
-- ============================================================================
local function create_drawables_scene()
    return f.scene(sw, sh, {
        on_enter = function(world, ctx)
            make_title(world, "DRAWABLES", 0.2, 0.03)

            -- Create a gradient bitmap large enough to see encoding differences
            local size = math.floor(math.min(sw, sh) * 0.25)
            local bm = f.bitmap.new(size, size)
            for y = 0, size - 1 do
                for x = 0, size - 1 do
                    local hue = (x / size) * 360
                    local val = 1.0 - (y / size) * 0.5
                    local r, g, b = hsv_to_rgb(hue, 1.0, val)
                    bm:set_dot(x, y, f.color(r, g, b))
                end
            end

            local encodings = {
                { name = "HALF",    make = function() return f.bitmap.half_block(bm) end },
                { name = "BRAILLE", make = function() return f.bitmap.braille(bm) end },
                { name = "FULL",    make = function() return f.bitmap.full_block(bm) end },
                { name = "BG",      make = function() return f.bitmap.bg_block(bm) end },
            }

            local font = f.asset.load_font("Oxanium/static/Oxanium-ExtraLight.ttf")
            local spacing = sw / (#encodings + 1)
            local top_y = sh * 0.28

            for i, enc in ipairs(encodings) do
                local cx = spacing * i
                local x = cx - size / 2

                local entity = world:spawn()
                entity:set_position(f.vec3(x, top_y, 0))
                entity:set_drawable(enc.make())
                world:add_root(entity)

                make_label(world, font, enc.name, cx, top_y + size + 2)
            end

            -- Rect drawable below
            local rect_w = math.floor(sw * 0.15)
            local rect_h = math.floor(sh * 0.12)
            local rect_entity = world:spawn()
            rect_entity:set_position(f.vec3(sw / 2 - rect_w / 2, sh * 0.78, 0))
            rect_entity:set_drawable(f.bitmap.rect(rect_w, rect_h,
                f.color(255, 100, 50),
                f.color(50, 100, 255)
            ))
            world:add_root(rect_entity)

            make_label(world, font, "RECT", sw / 2, sh * 0.78 + rect_h + 2)
        end,
    })
end

-- ============================================================================
-- Scene 2: SDF threshold reveal animation
-- ============================================================================
local function create_sdf_scene()
    local set_threshold
    local elapsed = 0

    return f.scene(sw, sh, {
        on_enter = function(world, ctx)
            make_title(world, "SDF REVEAL", 0.2, 0.03)

            local font = f.asset.load_font("Oxanium/static/Oxanium-ExtraLight.ttf")
            local layout = f.asset.rasterize_text("FLICKER", {
                font = font,
                size = sh * 0.5,
                color = f.color(0, 255, 180),
            })

            -- Compute SDF from the text bitmap
            local sdf = f.bitmap.compute_sdf(layout.bitmap, 40)

            -- Create the threshold material (returns material + setter)
            local mat, setter = f.bitmap.half_block_threshold(sdf, -40)
            set_threshold = setter

            local entity = world:spawn()
            entity:set_position(f.vec3(
                sw / 2 - layout.width / 2,
                sh * 0.35,
                0
            ))
            entity:set_drawable(f.bitmap.half_block(layout.bitmap))
            entity:set_material(mat)
            world:add_root(entity)

            elapsed = 0
        end,
        on_update = function(world, time)
            elapsed = elapsed + time.delta
            -- Animate threshold from -40 (hidden) to +40 (fully revealed) over 3 seconds
            -- Then hold, then reverse
            local cycle = elapsed % 6
            local threshold
            if cycle < 3 then
                threshold = -40 + (cycle / 3) * 80
            else
                threshold = 40 - ((cycle - 3) / 3) * 80
            end
            if set_threshold then
                set_threshold(threshold)
            end
        end,
    })
end

-- ============================================================================
-- Scene 3: Braille SDF threshold reveal
-- ============================================================================
local function create_braille_sdf_scene()
    local set_threshold
    local elapsed = 0

    return f.scene(sw, sh, {
        on_enter = function(world, ctx)
            make_title(world, "BRAILLE SDF", 0.2, 0.03)

            -- Create a circle bitmap
            local radius = math.min(sw, sh) * 0.25
            local size = math.floor(radius * 2 + 1)
            local bm = f.bitmap.new(size, size)
            local circle = f.sdf.circle(radius)
            for y = 0, size - 1 do
                for x = 0, size - 1 do
                    if circle(f.vec2(x - radius, y - radius)) <= 0 then
                        local hue = math.atan2(y - radius, x - radius) / (2 * math.pi) * 360 + 180
                        local r, g, b = hsv_to_rgb(hue, 1.0, 1.0)
                        bm:set_dot(x, y, f.color(r, g, b))
                    end
                end
            end

            local sdf = f.bitmap.compute_sdf(bm, radius)
            local mat, setter = f.bitmap.braille_threshold(sdf, -radius)
            set_threshold = setter

            local entity = world:spawn()
            entity:set_position(f.vec3(sw / 2 - size / 4, sh * 0.3, 0))
            entity:set_drawable(f.bitmap.braille(bm))
            entity:set_material(mat)
            world:add_root(entity)

            elapsed = 0
        end,
        on_update = function(world, time)
            elapsed = elapsed + time.delta
            local cycle = elapsed % 4
            local threshold
            local radius = math.min(sw, sh) * 0.25
            if cycle < 2 then
                threshold = -radius + (cycle / 2) * radius * 2
            else
                threshold = radius - ((cycle - 2) / 2) * radius * 2
            end
            if set_threshold then
                set_threshold(threshold)
            end
        end,
    })
end

-- ============================================================================
-- Scene 4: Blend modes
-- ============================================================================
local function create_blend_scene()
    local blend_modes = {
        { name = "screen",       mode = f.blend.screen },
        { name = "multiply",     mode = f.blend.multiply },
        { name = "overlay",      mode = f.blend.overlay },
        { name = "difference",   mode = f.blend.difference },
        { name = "hard_light",   mode = f.blend.hard_light },
        { name = "soft_light",   mode = f.blend.soft_light },
        { name = "linear_dodge", mode = f.blend.linear_dodge },
        { name = "color_burn",   mode = f.blend.color_burn },
    }
    local current_idx = 1
    local elapsed = 0
    local label_entity

    return f.scene(sw, sh, {
        on_enter = function(world, ctx)
            make_title(world, "BLEND MODES", 0.2, 0.03)

            -- Layer 0: base content (red-ish gradient)
            local size = math.floor(math.min(sw, sh) * 0.35)
            local bm0 = f.bitmap.new(size, size)
            for y = 0, size - 1 do
                for x = 0, size - 1 do
                    local r = 200 - (y / size) * 150
                    bm0:set_dot(x, y, f.color(r, 50, 50))
                end
            end
            local e0 = world:spawn()
            e0:set_position(f.vec3(sw / 2 - size / 2 - size * 0.15, sh * 0.3, 0))
            e0:set_drawable(f.bitmap.full_block(bm0))
            e0:set_layer(0)
            world:add_root(e0)

            -- Layer 1: overlapping content (blue-ish gradient)
            local bm1 = f.bitmap.new(size, size)
            for y = 0, size - 1 do
                for x = 0, size - 1 do
                    local b = 200 - (x / size) * 150
                    bm1:set_dot(x, y, f.color(50, 50, b))
                end
            end
            local e1 = world:spawn()
            e1:set_position(f.vec3(sw / 2 - size / 2 + size * 0.15, sh * 0.3, 0))
            e1:set_drawable(f.bitmap.full_block(bm1))
            e1:set_layer(1)
            world:add_root(e1)

            -- Set initial blend mode on layer 1
            f.set_blend(1, blend_modes[1].mode)

            -- Label showing current mode
            local font = f.asset.load_font("Oxanium/static/Oxanium-ExtraLight.ttf")
            label_entity = world:spawn()
            label_entity:set_position(f.vec3(0, sh * 0.85, 0))
            world:add_root(label_entity)

            local function update_label(name)
                local layout = f.asset.rasterize_text(name, {
                    font = font,
                    size = sh * 0.18,
                    color = f.color(255, 255, 100),
                })
                label_entity:set_position(f.vec3(sw / 2 - layout.width / 6, sh * 0.85, 0))
                label_entity:set_drawable(f.bitmap.adaptive(layout.bitmap))
            end
            update_label(blend_modes[1].name)

            elapsed = 0
        end,
        on_update = function(world, time)
            elapsed = elapsed + time.delta
            -- Cycle blend mode every 2 seconds
            local new_idx = math.floor(elapsed / 2) % #blend_modes + 1
            if new_idx ~= current_idx then
                current_idx = new_idx
                f.set_blend(1, blend_modes[current_idx].mode)

                -- Update label
                local font = f.asset.load_font("Oxanium/static/Oxanium-ExtraLight.ttf")
                local layout = f.asset.rasterize_text(blend_modes[current_idx].name, {
                    font = font,
                    size = sh * 0.18,
                    color = f.color(255, 255, 100),
                })
                label_entity:set_position(f.vec3(sw / 2 - layout.width / 6, sh * 0.85, 0))
                label_entity:set_drawable(f.bitmap.adaptive(layout.bitmap))
            end
        end,
    })
end

-- ============================================================================
-- Scene 5: Post-process shader
-- ============================================================================
local function create_post_process_scene()
    return f.scene(sw, sh, {
        on_enter = function(world, ctx)
            make_title(world, "POST-PROCESS", 0.2, 0.03)

            local font = f.asset.load_font("Oxanium/static/Oxanium-ExtraLight.ttf")
            local layout = f.asset.rasterize_text("SHADER", {
                font = font,
                size = sh * 0.5,
                color = f.color(255, 255, 255),
            })
            local entity = world:spawn()
            entity:set_position(f.vec3(sw / 2 - layout.width / 2, sh * 0.3, 0))
            entity:set_drawable(f.bitmap.half_block(layout.bitmap))
            world:add_root(entity)

            -- Apply a rainbow tint post-process to layer 0
            f.set_post_process(0, function(frag)
                local hue = (frag.time * 60 + frag.screen_x * 3 + frag.screen_y * 2) % 360
                local r, g, b = hsv_to_rgb(hue, 0.6, 1.0)
                -- Blend the rainbow tint with the original color
                local blend = 0.5
                return {
                    rune = frag.rune,
                    fg_r = frag.fg_r * (1 - blend) + r * blend,
                    fg_g = frag.fg_g * (1 - blend) + g * blend,
                    fg_b = frag.fg_b * (1 - blend) + b * blend,
                    fg_alpha = frag.fg_alpha,
                    bg_r = frag.bg_r,
                    bg_g = frag.bg_g,
                    bg_b = frag.bg_b,
                    bg_alpha = frag.bg_alpha,
                }
            end)
        end,
    })
end

-- ============================================================================
-- Scene 6: Camera zoom
-- ============================================================================
local function create_camera_scene()
    local cam_entity
    local elapsed = 0

    return f.scene(sw, sh, {
        on_enter = function(world, ctx)
            -- Create a camera entity
            cam_entity = world:spawn()
            cam_entity:set_camera({ zoom = 1.0 })
            cam_entity:set_position(f.vec3(sw / 2, sh / 2, 0))
            world:add_root(cam_entity)
            world:set_active_camera(cam_entity)

            -- Scene title
            local font = f.asset.load_font("Oxanium/static/Oxanium-ExtraLight.ttf")
            local layout = f.asset.rasterize_text("CAMERA ZOOM", {
                font = font,
                size = sh * 0.2,
                color = f.color(255, 255, 255),
            })
            local title_e = world:spawn()
            title_e:set_position(f.vec3(sw / 2 - layout.width / 2, sh * 0.1, 0))
            title_e:set_drawable(f.bitmap.half_block(layout.bitmap))
            world:add_root(title_e)

            -- Create scattered shapes to see camera movement
            local colors = {
                f.color(255, 100, 100),
                f.color(100, 255, 100),
                f.color(100, 100, 255),
                f.color(255, 255, 100),
                f.color(255, 100, 255),
                f.color(100, 255, 255),
            }

            for i = 1, 6 do
                local angle = (i - 1) * math.pi * 2 / 6
                local radius = math.min(sw, sh) * 0.3
                local px = sw / 2 + math.cos(angle) * radius
                local py = sh / 2 + math.sin(angle) * radius

                local size = 10
                local bm = f.bitmap.new(size, size)
                local shape_sdf = f.sdf.circle(size / 2)
                for y = 0, size - 1 do
                    for x = 0, size - 1 do
                        if shape_sdf(f.vec2(x - size / 2, y - size / 2)) <= 0 then
                            bm:set_dot(x, y, colors[i])
                        end
                    end
                end

                local entity = world:spawn()
                entity:set_position(f.vec3(px, py, 0))
                entity:set_drawable(f.bitmap.braille(bm))
                world:add_root(entity)
            end

            elapsed = 0
        end,
        on_update = function(world, time)
            elapsed = elapsed + time.delta
            -- Pulse zoom between 1.0 and 2.0
            local zoom = 1.0 + 0.5 * (1 + math.sin(elapsed * 1.5))
            cam_entity:set_camera({ zoom = zoom })

            -- Gentle camera rotation
            cam_entity:set_rotation(math.sin(elapsed * 0.5) * 0.15)
        end,
    })
end

-- ============================================================================
-- Scene 7: Age component + particle age_and_despawn
-- ============================================================================
local function create_age_scene()
    return f.scene(sw, sh, {
        trail = { layer = 0, effect = f.trail.ghost(0.92) },
        on_enter = function(world, ctx)
            make_title(world, "AGE + DESPAWN", 0.2, 0.03)

            -- Spawner entity that creates particles with age/lifetime
            local spawner = world:spawn()
            spawner:set_position(f.vec3(sw / 2, sh * 0.5, 0))
            world:add_root(spawner)

            local pixel = f.bitmap.new(2, 2)
            for y = 0, 1 do
                for x = 0, 1 do
                    pixel:set_dot(x, y, f.color(255, 255, 255))
                end
            end

            local spawn_timer = 0
            spawner:set_behavior(function(e, w, t)
                spawn_timer = spawn_timer + t.delta
                if spawn_timer < 0.1 then return end
                spawn_timer = 0

                -- Spawn a particle with age tracking
                local p = w:spawn()
                local angle = math.random() * 2 * math.pi
                local speed = 5 + math.random() * 15
                local px = sw / 2
                local py = sh * 0.5

                p:set_position(f.vec3(px, py, 0))
                p:set_drawable(f.bitmap.braille(pixel))
                p:set_body({
                    velocity = f.vec2(
                        math.cos(angle) * speed,
                        math.sin(angle) * speed - 5
                    ),
                })
                p:set_age({ lifetime = 2.0 + math.random() * 2.0 })
                w:add_root(p)

                -- Age-based color fade material
                p:set_material(function(frag)
                    local life_frac = 0
                    if frag.lifetime > 0 then
                        life_frac = frag.age / frag.lifetime
                    end
                    -- Fade from bright cyan to dim red over lifetime
                    local r = 100 + life_frac * 155
                    local g = 255 * (1 - life_frac)
                    local b = 255 * (1 - life_frac * 0.5)
                    local alpha = 1.0 - life_frac * 0.8
                    return {
                        rune = frag.rune,
                        fg_r = r, fg_g = g, fg_b = b,
                        fg_alpha = alpha,
                        bg_alpha = 0,
                    }
                end)

                -- Add physics + lifecycle behaviors
                p:set_behavior(f.physics.gravity(f.vec2(0, 8)))
                p:set_behavior(f.physics.drag(0.5))
                p:set_behavior(f.physics.euler())
                p:set_behavior(f.particle.age_and_despawn())
            end)
        end,
    })
end

-- ============================================================================
-- Main: scene manager with all demo scenes
-- ============================================================================
f.on_enter(function(world, ctx)
    sw = ctx.width
    sh = ctx.height

    local sm = f.scene_manager(sw, sh)

    sm:add(create_drawables_scene())
    sm:add(create_sdf_scene())
    sm:add(create_braille_sdf_scene())
    sm:add(create_blend_scene())
    sm:add(create_post_process_scene())
    sm:add(create_camera_scene())
    sm:add(create_age_scene())

    sm:start()
end)
