-- demo.lua: Full demo reproducing the Go cmd/flicker/main.go demo.
-- Press SPACE to advance scenes. Press 'q' or ESC to quit.
local f = require("flicker")

-- Screen dimensions are passed via the default scene's on_enter context.
-- We capture them on first scene entry and use them throughout.
local sw, sh

-- Helper: create title text entity centered horizontally at given y fraction.
local function make_title(world, text, size_frac, y_frac, color)
    local font = f.asset.load_font("Oxanium/static/Oxanium-Bold.ttf")
    local layout = f.asset.rasterize_text(text, {
        font = font,
        size = sh * size_frac,
        color = color or f.color(255, 255, 255),
    })
    local entity = world:spawn()
    entity:set_position(f.vec3(
        sw / 2 - layout.width / 2,
        sh * y_frac,
        0
    ))
    entity:set_drawable(f.bitmap.half_block(layout.bitmap))
    world:add_root(entity)
    return entity, layout
end

-- Helper: create moving text with tween oscillation.
local function make_oscillating_text(world, text, size_frac, color, start_x_frac, end_x_frac, duration, easing)
    local font = f.asset.load_font("Oxanium/static/Oxanium-Bold.ttf")
    local layout = f.asset.rasterize_text(text, {
        font = font,
        size = sh * size_frac,
        color = color,
    })
    local entity = world:spawn()
    local start_x = sw * start_x_frac
    local center_y = sh / 2 - layout.height / 2
    entity:set_position(f.vec3(start_x, center_y, 0))
    entity:set_drawable(f.bitmap.half_block(layout.bitmap))
    world:add_root(entity)

    local tween = f.tween_vec3({
        from = f.vec3(start_x, center_y, 0),
        to = f.vec3(sw * end_x_frac, center_y, 0),
        duration = duration,
        easing = easing or "in_out_quad",
    })

    entity:set_behavior(function(e, w, t)
        local pos = tween:update(t.delta)
        if tween:done() then
            tween:reset()
        end
        pos = tween:update(0)
        e:set_position(pos)
    end)

    return entity, layout
end

-- ============================================================================
-- Custom materials (pure Lua, using fragment.entity_id and fragment.time)
-- ============================================================================

-- HSV to RGB conversion (hue in degrees, s/v in 0..1)
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

-- Fire gradient: dark ember → red → orange → yellow → hot white
local fire_stops = {
    { 40,   0,   0},   -- dark ember
    {200,  30,   0},   -- deep red
    {255, 100,   0},   -- orange
    {255, 200,  50},   -- yellow-orange
    {255, 255, 200},   -- hot white
}

local function sample_fire(t)
    t = t % 1.0
    local pos = t * (#fire_stops - 1)
    local idx = math.floor(pos)
    local frac = pos - idx
    if idx >= #fire_stops - 1 then idx = #fire_stops - 2; frac = 1.0 end
    local a = fire_stops[idx + 1]
    local b = fire_stops[idx + 2]
    return a[1] + (b[1] - a[1]) * frac,
           a[2] + (b[2] - a[2]) * frac,
           a[3] + (b[3] - a[3]) * frac
end

-- Cycling material: smoothly crossfades between rainbow and fire every 4 seconds.
-- Uses entity_id for per-particle phase offset.
function fire_rainbow_material()
    return function(frag)
        local phase = frag.entity_id * 0.1
        local t = frag.time

        -- Cycle: 4s rainbow, 4s fire, with 30% crossfade
        local cycle_pos = (t % 8.0) / 8.0  -- 0..1 over 8 seconds
        local rainbow_t = (t * 2.0 + phase) % 1.0
        local fire_t = (t * 1.5 + phase) % 1.0

        local rr, rg, rb = hsv_to_rgb(rainbow_t * 360, 1.0, 1.0)
        local fr, fg, fb = sample_fire(fire_t)

        -- Blend factor: 0 = full rainbow, 1 = full fire
        -- First half (0..0.5) = rainbow, second half (0.5..1) = fire
        -- With smooth transitions at the boundaries
        local blend
        if cycle_pos < 0.35 then
            blend = 0  -- pure rainbow
        elseif cycle_pos < 0.5 then
            blend = (cycle_pos - 0.35) / 0.15  -- fade to fire
        elseif cycle_pos < 0.85 then
            blend = 1  -- pure fire
        else
            blend = 1 - (cycle_pos - 0.85) / 0.15  -- fade to rainbow
        end

        return {
            rune = frag.rune,
            fg_r = rr + (fr - rr) * blend,
            fg_g = rg + (fg - rg) * blend,
            fg_b = rb + (fb - rb) * blend,
            fg_alpha = frag.fg_alpha,
            bg_alpha = 0,
        }
    end
end

-- ============================================================================
-- Scene definitions
-- ============================================================================

local function create_no_trail_scene()
    return f.scene(sw, sh, {
        on_enter = function(world, ctx)
            make_title(world, "NO TRAILS", 0.2, 0.15)
            make_oscillating_text(world, "MOVING", 0.3, f.color(100, 200, 255),
                0.1, 0.6, 2.0, "in_out_quad")
        end,
    })
end

local function create_ghost_trail_scene()
    return f.scene(sw, sh, {
        trail = { layer = 0, effect = f.trail.ghost(0.95) },
        on_enter = function(world, ctx)
            make_title(world, "GHOST TRAIL", 0.2, 0.15)
            make_oscillating_text(world, "FADE", 0.4, f.color(100, 255, 100),
                0.1, 0.6, 2.5, "in_out_quad")
        end,
    })
end

local function create_blur_trail_scene()
    return f.scene(sw, sh, {
        trail = { layer = 0, effect = f.trail.blur(0.94, 0.3) },
        on_enter = function(world, ctx)
            make_title(world, "BLUR TRAIL", 0.2, 0.15)

            -- Create moving circle with circular motion
            local pixel = f.bitmap.new(1, 1)
            pixel:set_dot(0, 0, f.color(255, 100, 255))

            local entity = world:spawn()
            local cx = sw / 2
            local cy = sh / 2
            entity:set_transform({
                position = f.vec3(cx, cy, 0),
                scale = f.vec3(15, 15, 1),
            })
            entity:set_drawable(f.bitmap.braille(pixel))
            world:add_root(entity)

            local angle = 0
            entity:set_behavior(function(e, w, t)
                angle = angle + t.delta * 1.5
                local radius = sw * 0.25
                e:set_position(f.vec3(
                    cx + math.cos(angle) * radius,
                    cy + math.sin(angle) * radius * 0.5,
                    0
                ))
            end)
        end,
    })
end

local function create_floaty_trail_scene()
    return f.scene(sw, sh, {
        trail = { layer = 0, effect = f.trail.floaty(0.96, 3.0) },
        on_enter = function(world, ctx)
            make_title(world, "FLOATY TRAIL", 0.2, 0.15)
            make_oscillating_text(world, "DRIFT", 0.4, f.color(255, 200, 100),
                0.1, 0.6, 3.0, "in_out_cubic")
        end,
    })
end

local function create_gravity_trail_scene()
    return f.scene(sw, sh, {
        trail = { layer = 0, effect = f.trail.gravity(0.96, 5.0) },
        on_enter = function(world, ctx)
            make_title(world, "GRAVITY TRAIL", 0.2, 0.15)

            local font = f.asset.load_font("Oxanium/static/Oxanium-Bold.ttf")
            local layout = f.asset.rasterize_text("FALL", {
                font = font,
                size = sh * 0.4,
                color = f.color(255, 100, 100),
            })
            local entity = world:spawn()
            local start_x = sw * 0.1
            local start_y = sh * 0.35
            entity:set_position(f.vec3(start_x, start_y, 0))
            entity:set_drawable(f.bitmap.half_block(layout.bitmap))
            world:add_root(entity)

            local tween = f.tween_vec3({
                from = f.vec3(start_x, start_y, 0),
                to = f.vec3(sw * 0.6, start_y, 0),
                duration = 2.5,
                easing = "in_out_quad",
            })
            entity:set_behavior(function(e, w, t)
                local pos = tween:update(t.delta)
                if tween:done() then tween:reset() end
                pos = tween:update(0)
                e:set_position(pos)
            end)
        end,
    })
end

local function create_dissolve_trail_scene()
    return f.scene(sw, sh, {
        trail = { layer = 0, effect = f.trail.dissolve(0.93, 0.6, f.color(120, 120, 120)) },
        on_enter = function(world, ctx)
            make_title(world, "DISSOLVE TRAIL", 0.2, 0.15)
            make_oscillating_text(world, "DISSOLVE", 0.4, f.color(200, 150, 255),
                0.1, 0.6, 2.0, "in_out_quad")
        end,
    })
end

local function create_fire_trail_scene()
    return f.scene(sw, sh, {
        trail = { layer = 0, effect = f.trail.fire(0.94, 2.0) },
        on_enter = function(world, ctx)
            make_title(world, "FIRE TRAIL", 0.2, 0.15)
            make_oscillating_text(world, "BURN", 0.4, f.color(255, 255, 100),
                0.1, 0.6, 2.5, "in_out_quad")
        end,
    })
end

local function create_trailing_scene()
    return f.scene(sw, sh, {
        on_enter = function(world, ctx)
            make_title(world, "TRAILING PARTICLES", 0.18, 0.12)

            local font = f.asset.load_font("Oxanium/static/Oxanium-Bold.ttf")
            local layout = f.asset.rasterize_text("DUST", {
                font = font,
                size = sh * 0.4,
                color = f.color(255, 200, 100),
            })

            local entity = world:spawn()
            local cx = sw / 2
            local cy = sh / 2
            entity:set_position(f.vec3(cx, cy, 0))
            local drawable = f.bitmap.half_block(layout.bitmap)
            entity:set_drawable(drawable)
            world:add_root(entity)

            -- Circular motion
            local angle = 0
            local radius = sh * 0.3
            entity:set_behavior(function(e, w, t)
                angle = angle + t.delta * 3.0
                e:set_position(f.vec3(
                    cx + math.cos(angle) * radius - layout.width / 2,
                    cy + math.sin(angle) * radius - layout.height / 2,
                    0
                ))
            end)

            -- Trailing emitter
            local params = f.particle.compute_emission(layout.bitmap, drawable, "bottom")
            local emitter = f.particle.trailing_emitter(params.offset, {
                width = params.width,
                emit_rate = 5.0,
                particle_life = 3.0,
            })
            entity:set_behavior(emitter)
        end,
    })
end

local function create_intro_scene()
    local timeline
    local text_entity
    local center_x

    return f.scene(sw, sh, {
        on_enter = function(world, ctx)
            local font = f.asset.load_font("Oxanium/static/Oxanium-Bold.ttf")
            local layout = f.asset.rasterize_text("INTRO", {
                font = font,
                size = sh * 0.6,
                color = f.color(100, 200, 255),
            })

            center_x = sw / 2 - layout.width / 2
            local center_y = sh / 2 - layout.height / 2

            text_entity = world:spawn()
            text_entity:set_position(f.vec3(-layout.width, center_y, 0))
            text_entity:set_drawable(f.bitmap.half_block(layout.bitmap))
            world:add_root(text_entity)

            timeline = f.timeline.new(world)
        end,
        on_ready = function(world)
            local track = timeline:add_track()
            local tween = f.timeline.tween(text_entity, "position.x", {
                from = -100,
                to = center_x,
                duration = 1.5,
                easing = "out_cubic",
            })
            track:add(tween)
            timeline:start()
        end,
        on_exit = function(world)
            if timeline then timeline:cleanup() end
        end,
    })
end

local function create_timeline_scene()
    local timeline
    local text1, text2
    local target_y1, layout1_height

    return f.scene(sw, sh, {
        on_enter = function(world, ctx)
            timeline = f.timeline.new(world)
            local font = f.asset.load_font("Oxanium/static/Oxanium-Bold.ttf")

            -- Word 1: "TIMELINE"
            local layout1 = f.asset.rasterize_text("TIMELINE", {
                font = font,
                size = sh * 0.3,
                color = f.color(255, 100, 100),
            })
            local cx1 = sw / 2 - layout1.width / 2
            target_y1 = sh * 0.25
            layout1_height = layout1.height

            text1 = world:spawn()
            text1:set_position(f.vec3(cx1, -layout1_height, 0))
            text1:set_drawable(f.bitmap.half_block(layout1.bitmap))
            world:add_root(text1)

            -- Word 2: "DEMO"
            local layout2 = f.asset.rasterize_text("DEMO", {
                font = font,
                size = sh * 0.4,
                color = f.color(100, 255, 100),
            })
            local cx2 = sw / 2 - layout2.width / 2
            local cy2 = sh * 0.55

            text2 = world:spawn()
            text2:set_transform({
                position = f.vec3(cx2, cy2, 0),
                scale = f.vec3(0.1, 0.1, 1),
            })
            text2:set_drawable(f.bitmap.half_block(layout2.bitmap))
            world:add_root(text2)
        end,
        on_ready = function(world)
            -- Track 1: Word 1 slides down
            local track1 = timeline:add_track()
            track1:add(f.timeline.tween(text1, "position.y", {
                from = -layout1_height,
                to = target_y1,
                duration = 1.2,
                easing = "out_bounce",
            }))

            -- Track 2: Word 2 scales up
            local track2 = timeline:add_track()
            track2:at(0.8, f.timeline.parallel(
                f.timeline.tween(text2, "scale.x", {
                    from = 0.1, to = 1.0, duration = 1.0, easing = "out_elastic",
                }),
                f.timeline.tween(text2, "scale.y", {
                    from = 0.1, to = 1.0, duration = 1.0, easing = "out_elastic",
                })
            ))

            timeline:start()
        end,
        on_exit = function(world)
            if timeline then timeline:cleanup() end
        end,
    })
end

local function create_particle_scene()
    return f.scene(sw, sh, {
        on_enter = function(world, ctx)
            local font = f.asset.load_font("Oxanium/static/Oxanium-Bold.ttf")
            local text_size = sh * 0.8

            local words = { "GO", "BURST", "END" }
            local layouts = {}
            local clouds = {}

            for i, word in ipairs(words) do
                layouts[i] = f.asset.rasterize_text(word, {
                    font = font,
                    size = text_size,
                    color = f.color(255, 255, 255),
                })
                clouds[i] = f.particle.bitmap_to_cloud(layouts[i].bitmap)
            end

            -- Single pixel for particles
            local pixel = f.bitmap.new(1, 1)
            pixel:set_dot(0, 0, f.color(255, 255, 255))

            -- Center initial cloud
            local offset_x = sw / 2 - layouts[1].width / 2
            local offset_y = sh / 2 - layouts[1].height / 2

            local initial_cloud = {}
            for i, pos in ipairs(clouds[1]) do
                initial_cloud[i] = f.vec2(pos.x + offset_x, pos.y + offset_y)
            end

            -- Material: braille directional runes + cycling rainbow/fire color
            local material = f.compose_materials(
                f.particle.braille_directional(),
                fire_rainbow_material()
            )

            -- Create point cloud sequence
            local seq = f.particle.cloud_sequence(
                world, initial_cloud,
                f.bitmap.braille(pixel),
                material,
                0
            )

            -- Add morph targets
            for i = 2, #words do
                local target_offset_x = sw / 2 - layouts[i].width / 2
                local offset_cloud = {}
                for j, pos in ipairs(clouds[i]) do
                    offset_cloud[j] = f.vec2(pos.x + target_offset_x, pos.y + offset_y)
                end

                local phases
                if i == 2 then
                    phases = {
                        f.particle.burst_phase(sh * 0.4),
                        f.particle.seek_phase(),
                    }
                else
                    phases = {
                        f.particle.keyframe_phase("in_out_quad"),
                    }
                end

                seq:add_target({
                    cloud = offset_cloud,
                    duration = 4.0,
                    strategy = f.particle.linear(),
                    phases = phases,
                })
            end
        end,
    })
end

local function create_thanks_scene()
    local timeline
    local text_entity

    return f.scene(sw, sh, {
        on_enter = function(world, ctx)
            local font = f.asset.load_font("Oxanium/static/Oxanium-Bold.ttf")
            local layout = f.asset.rasterize_text("THANKS", {
                font = font,
                size = sh * 0.5,
                color = f.color(255, 255, 255),
            })

            local cx = sw / 2 - layout.width / 2
            local cy = sh / 2 - layout.height / 2

            text_entity = world:spawn()
            text_entity:set_transform({
                position = f.vec3(cx, cy, 0),
                scale = f.vec3(0.3, 0.3, 1),
                rotation = 0,
            })
            text_entity:set_drawable(f.bitmap.half_block(layout.bitmap))
            text_entity:set_material(f.particle.rainbow_time(3.0))
            world:add_root(text_entity)

            timeline = f.timeline.new(world)
        end,
        on_ready = function(world)
            local track = timeline:add_track()
            track:add(f.timeline.parallel(
                f.timeline.tween(text_entity, "scale.x", {
                    from = 0.3, to = 1.0, duration = 1.5, easing = "out_elastic",
                }),
                f.timeline.tween(text_entity, "scale.y", {
                    from = 0.3, to = 1.0, duration = 1.5, easing = "out_elastic",
                }),
                f.timeline.tween(text_entity, "rotation", {
                    from = -0.5, to = 0.0, duration = 1.5, easing = "out_cubic",
                })
            ))
            timeline:start()
        end,
        on_exit = function(world)
            if timeline then timeline:cleanup() end
        end,
    })
end

-- ============================================================================
-- Main: set up scene manager and add all scenes
-- ============================================================================

-- We use the simple-mode on_enter to capture dimensions and build everything
f.on_enter(function(world, ctx)
    sw = ctx.width
    sh = ctx.height

    -- Create scene manager
    local sm = f.scene_manager(sw, sh)

    -- Trail demo scenes
    sm:add(create_no_trail_scene())
    sm:add(create_ghost_trail_scene())
    sm:add(create_blur_trail_scene())
    sm:add(create_floaty_trail_scene())
    sm:add(create_gravity_trail_scene())
    sm:add(create_dissolve_trail_scene())
    sm:add(create_fire_trail_scene())
    sm:add(create_trailing_scene())

    -- Timeline & particle scenes
    sm:add(create_intro_scene())
    sm:add(create_timeline_scene())
    sm:add(create_particle_scene())
    sm:add(create_thanks_scene())

    -- Start with first scene
    sm:start()
end)
