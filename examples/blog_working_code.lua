-- blog_working_code.lua: "Working Code is Not Enough" — visual-first redesign
-- Auto-advances through slides. Press 'q' or ESC to quit.
local f = require("flicker")

local sw, sh

-- Color palette
local BG      = f.color(222, 220, 215) -- warm gray
local BLACK   = f.color(30, 30, 30)
local DARK    = f.color(60, 60, 60)
local MID     = f.color(120, 120, 120)
local ACCENT  = f.color(180, 60, 40)   -- tactical / warning red
local ACCENT2 = f.color(40, 100, 160)  -- strategic / calm blue

-- ── Helpers ──────────────────────────────────────────────────────────────

local function bg_material()
    return function(frag)
        return {
            rune = frag.rune,
            fg_r = frag.fg_r, fg_g = frag.fg_g, fg_b = frag.fg_b,
            fg_alpha = frag.fg_alpha,
            bg_r = BG.r, bg_g = BG.g, bg_b = BG.b,
            bg_alpha = 1.0,
        }
    end
end

local function make_bg(world, layer)
    local bg = world:spawn()
    bg:set_position(f.vec3(0, 0, 0))
    bg:set_drawable(f.bitmap.rect(sw, sh, BG, BG))
    bg:set_layer(layer or -1)
    world:add_root(bg)
    return bg
end

local function make_text(world, text, x, y, color, layer)
    local txt = f.text(text, { fg = color or BLACK })
    local ent = world:spawn()
    ent:set_position(f.vec3(x, y, 0))
    ent:set_drawable(txt)
    ent:set_material(bg_material())
    if layer then ent:set_layer(layer) end
    world:add_root(ent)
    return ent, txt
end

local function cx(text)
    return math.floor(sw / 2 - #text / 2)
end

local function typewriter_frames(str, char_delay)
    char_delay = char_delay or 0.03
    local frames = {}
    for i = 1, #str do
        table.insert(frames, {
            time = (i - 1) * char_delay,
            value = string.sub(str, 1, i),
        })
    end
    return frames, #str * char_delay + 0.3
end

-- ============================================================================
-- Slide 1: Title Card — particle cloud formation (5s, auto-advance)
-- ============================================================================
local function create_title_slide()
    return f.scene(sw, sh, {
        duration = 5.0,
        transition = { shader = f.transition.cross_fade, duration = 1.5 },
        on_enter = function(world, ctx)
            make_bg(world, -1)

            -- "Working code is" as regular text above the particle title
            local prefix = "Working code is"
            make_text(world, prefix, cx(prefix), math.floor(sh * 0.22), MID, 1)

            -- Rasterize "NOT ENOUGH" for particle cloud
            local font = f.asset.load_font("Oxanium/static/Oxanium-Bold.ttf")
            local layout = f.asset.rasterize_text("NOT ENOUGH", {
                font = font,
                size = sh * 0.35,
                color = BLACK,
            })

            -- Convert to particle cloud
            local cloud = f.particle.bitmap_to_cloud(layout.bitmap)
            local offset_x = sw / 2 - layout.width / 2
            local offset_y = sh * 0.32

            -- Target positions: centered text
            local text_cloud = {}
            for i, pos in ipairs(cloud) do
                text_cloud[i] = f.vec2(pos.x + offset_x, pos.y + offset_y)
            end

            -- Initial positions: random scatter from center
            local initial_cloud = {}
            for i = 1, #cloud do
                local angle = math.random() * 2 * math.pi
                local radius = math.random() * 8 + 1
                initial_cloud[i] = f.vec2(
                    sw / 2 + math.cos(angle) * radius,
                    sh / 2 + math.sin(angle) * radius
                )
            end

            -- Single pixel drawable for particles
            local pixel = f.bitmap.new(1, 1)
            pixel:set_dot(0, 0, BLACK)

            -- Particle material
            local material = f.compose_materials(function(frag)
                return {
                    rune = frag.rune,
                    fg_r = BLACK.r, fg_g = BLACK.g, fg_b = BLACK.b,
                    fg_alpha = frag.fg_alpha,
                    bg_r = BG.r, bg_g = BG.g, bg_b = BG.b,
                    bg_alpha = 1.0,
                }
            end)

            -- Create cloud sequence: random positions → text formation
            local seq = f.particle.cloud_sequence(
                world, initial_cloud, f.bitmap.braille(pixel), material, 0
            )
            seq:add_target({
                cloud = text_cloud,
                duration = 3.0,
                strategy = f.particle.linear(),
                phases = {
                    f.particle.burst_phase(sh * 0.3),
                    f.particle.seek_phase(),
                },
            })

            -- Subtitle: staggered word reveal
            local timeline = f.timeline.new(world)
            local sub_str = "When tools change, the principles stay the same"
            local _, sub_txt = make_text(world, "", cx(sub_str), math.floor(sh * 0.72), MID, 1)
            local sub_words = { "When", "tools", "change,", "the", "principles", "stay", "the", "same" }

            local track = timeline:add_track()
            local word_frames = {}
            local built = ""
            local t = 2.5
            for i, word in ipairs(sub_words) do
                if i > 1 then built = built .. " " end
                built = built .. word
                table.insert(word_frames, { time = t, value = built })
                t = t + 0.18
            end
            track:add(f.timeline.text_keyframes(sub_txt, word_frames, 5.0))
            timeline:start()
        end,
    })
end

-- ============================================================================
-- Slide 2: The Cost of "Just Working" (8s, auto-advance)
-- ============================================================================
local function create_cost_slide()
    local timeline
    local fade_start = 0
    local fading = false

    return f.scene(sw, sh, {
        duration = 8.0,
        transition = { shader = f.transition.cross_fade, duration = 1.5 },
        trail = { layer = 0, effect = f.trail.ghost(0.92) },
        on_enter = function(world, ctx)
            make_bg(world, -1)
            timeline = f.timeline.new(world)

            -- Quote on layer 0 (ghost trail creates dissolve smear when fading)
            local quote_str = '"If it works, it\'s good code."'
            local _, quote_txt = make_text(world, "", cx(quote_str), math.floor(sh * 0.25), MID, 0)

            -- Rebuttal on layer 1 (no trail, clean reveal)
            local reb_str = "Working code is the start of its long-term cost."
            local _, reb_txt = make_text(world, "", cx(reb_str), math.floor(sh * 0.55), ACCENT, 1)

            -- Typewrite the quote
            local track = timeline:add_track()
            local frames, dur = typewriter_frames(quote_str, 0.03)
            track:add(f.timeline.text_keyframes(quote_txt, frames, dur))

            -- After quote finishes + pause, start dissolving
            track:at(dur + 1.5, f.timeline.callback(function(w, t)
                fading = true
                fade_start = t.total
            end))

            -- Reveal rebuttal after dissolve starts
            local track2 = timeline:add_track()
            local reb_frames, reb_dur = typewriter_frames(reb_str, 0.025)
            track2:at(dur + 2.5, f.timeline.text_keyframes(reb_txt, reb_frames, reb_dur))

            timeline:start()
        end,
        on_update = function(world, time)
            -- Fade the quote layer via post-process shader
            if fading then
                local elapsed = time.total - fade_start
                local alpha = math.max(0, 1.0 - elapsed * 0.5)
                f.set_post_process(0, function(frag)
                    return {
                        rune = frag.rune,
                        fg_r = frag.fg_r, fg_g = frag.fg_g, fg_b = frag.fg_b,
                        fg_alpha = frag.fg_alpha * alpha,
                        bg_r = BG.r, bg_g = BG.g, bg_b = BG.b,
                        bg_alpha = frag.bg_alpha,
                    }
                end)
            end
        end,
        on_exit = function(world)
            if timeline then timeline:cleanup() end
        end,
    })
end

-- ============================================================================
-- Slide 3: Change Amplification — Domino Effect (10s, auto-advance)
-- ============================================================================
local function create_domino_slide()
    local timeline

    return f.scene(sw, sh, {
        duration = 10.0,
        transition = { shader = f.transition.cross_fade, duration = 1.5 },
        trail = { layer = 1, effect = f.trail.dissolve(0.90, 0.5, MID) },
        on_enter = function(world, ctx)
            make_bg(world, -1)
            timeline = f.timeline.new(world)

            -- Title
            make_text(world, "Change Amplification",
                cx("Change Amplification"), math.floor(sh * 0.05), DARK, 2)

            -- ASCII domino boxes — center of screen
            local box_x = math.floor(sw / 2 - 24)
            local box_y = math.floor(sh * 0.20)
            local _, box_txt = make_text(world, "", box_x, box_y, DARK, 0)

            local domino_states = {
                { time = 0.0, value =
                    "                  [A]" },
                { time = 1.8, value =
                    "            [A]       [B]" },
                { time = 3.2, value =
                    "        [A]   [B]   [C]   [D]" },
                { time = 4.8, value =
                    "    [A] [B] [C] [D]\n" ..
                    "      [E] [F] [G] [H]" },
                { time = 6.2, value =
                    "[A][B][C][D][E][F][G][H]\n" ..
                    "  [I][J][K][L][M][N][O][P]\n" ..
                    "[Q][R][S][T][U][V][W][X][Y]" },
            }

            local track = timeline:add_track()
            track:add(f.timeline.text_keyframes(box_txt, domino_states, 8.5))

            -- Particle burst pixel
            local pixel = f.bitmap.new(2, 2)
            for py = 0, 1 do
                for px = 0, 1 do
                    pixel:set_dot(px, py, ACCENT)
                end
            end

            local function spawn_burst(w, bx, by, count)
                for _ = 1, count do
                    local p = w:spawn()
                    local angle = math.random() * 2 * math.pi
                    local speed = 3 + math.random() * 12
                    p:set_position(f.vec3(bx, by, 0))
                    p:set_drawable(f.bitmap.braille(pixel))
                    p:set_layer(1)
                    p:set_body({
                        velocity = f.vec2(
                            math.cos(angle) * speed,
                            math.sin(angle) * speed
                        ),
                    })
                    p:set_age({ lifetime = 1.5 + math.random() * 1.5 })
                    p:set_behavior(f.physics.gravity(f.vec2(0, 5)))
                    p:set_behavior(f.physics.turbulence(0.3, 4.0))
                    p:set_behavior(f.physics.drag(0.3))
                    p:set_behavior(f.physics.euler())
                    p:set_behavior(f.particle.age_and_despawn())
                    p:set_material(function(frag)
                        local life_frac = 0
                        if frag.lifetime > 0 then
                            life_frac = frag.age / frag.lifetime
                        end
                        return {
                            rune = frag.rune,
                            fg_r = ACCENT.r, fg_g = ACCENT.g, fg_b = ACCENT.b,
                            fg_alpha = 1.0 - life_frac,
                            bg_alpha = 0,
                        }
                    end)
                    w:add_root(p)
                end
            end

            -- Bursts at each split moment
            local center_x = sw / 2
            local center_y = box_y + 1
            local burst_track = timeline:add_track()
            burst_track:at(1.8, f.timeline.callback(function(w, t)
                spawn_burst(w, center_x, center_y, 15)
            end))
            burst_track:at(3.2, f.timeline.callback(function(w, t)
                spawn_burst(w, center_x - 8, center_y, 12)
                spawn_burst(w, center_x + 8, center_y, 12)
            end))
            burst_track:at(4.8, f.timeline.callback(function(w, t)
                for dx = -15, 15, 10 do
                    spawn_burst(w, center_x + dx, center_y, 8)
                end
            end))
            burst_track:at(6.2, f.timeline.callback(function(w, t)
                for dx = -20, 20, 5 do
                    spawn_burst(w, center_x + dx, center_y + 1, 6)
                end
            end))

            -- Bottom text after cascade
            local insight = "Each shortcut compounds into complexity"
            local _, insight_txt = make_text(world, "", cx(insight), math.floor(sh * 0.85), ACCENT, 2)
            local ins_track = timeline:add_track()
            local ins_frames, ins_dur = typewriter_frames(insight, 0.03)
            ins_track:at(7.0, f.timeline.text_keyframes(insight_txt, ins_frames, ins_dur))

            timeline:start()
        end,
        on_exit = function(world)
            if timeline then timeline:cleanup() end
        end,
    })
end

-- ============================================================================
-- Slide 4: Tactical vs Strategic (12s, auto-advance)
-- ============================================================================
local function create_tactical_slide()
    local timeline
    local darken_start = nil

    return f.scene(sw, sh, {
        duration = 12.0,
        transition = { shader = f.transition.cross_fade, duration = 1.5 },
        trail = { layer = 0, effect = f.trail.dissolve(0.88, 0.4, MID) },
        on_enter = function(world, ctx)
            make_bg(world, -1)
            timeline = f.timeline.new(world)

            local mid_x = math.floor(sw / 2)
            local left_x = math.floor(sw * 0.05)
            local right_x = math.floor(sw * 0.55)
            local diagram_y = math.floor(sh * 0.22)

            -- Divider line on layer 2
            local divider = ""
            for _ = 1, sh do
                divider = divider .. "\xe2\x94\x82\n" -- │
            end
            make_text(world, divider, mid_x, 0, MID, 2)

            -- Labels on layer 2
            make_text(world, "Tactical", left_x + 8, diagram_y - 4, ACCENT, 2)
            make_text(world, "Strategic", right_x + 6, diagram_y - 4, ACCENT2, 2)

            -- Tactical tree on layer 0 (dissolve trail + chaos particles)
            local _, tac_txt = make_text(world, "", left_x, diagram_y, DARK, 0)

            -- Strategic tree on layer 1 (clean, no trail)
            local _, str_txt = make_text(world, "", right_x, diagram_y, DARK, 1)

            local tactical_states = {
                { time = 0.0, value = "app.go" },
                { time = 1.5, value =
                    "app.go\n" ..
                    " \xe2\x94\x94\xe2\x94\x80\xe2\x94\x80 handler.go" },
                { time = 3.0, value =
                    "app.go\n" ..
                    " \xe2\x94\x9c\xe2\x94\x80\xe2\x94\x80 handler.go\n" ..
                    " \xe2\x94\x94\xe2\x94\x80\xe2\x94\x80 handler2.go (copy)" },
                { time = 5.0, value =
                    "app.go\n" ..
                    " \xe2\x94\x9c\xe2\x94\x80\xe2\x94\x80 handler.go\n" ..
                    " \xe2\x94\x9c\xe2\x94\x80\xe2\x94\x80 handler2.go (copy)\n" ..
                    " \xe2\x94\x9c\xe2\x94\x80\xe2\x94\x80 utils.go\n" ..
                    " \xe2\x94\x94\xe2\x94\x80\xe2\x94\x80 fix_handler.go (hotfix)" },
                { time = 7.0, value =
                    "app.go\n" ..
                    " \xe2\x94\x9c\xe2\x94\x80\xe2\x94\x80 handler.go\n" ..
                    " \xe2\x94\x9c\xe2\x94\x80\xe2\x94\x80 handler2.go (copy)\n" ..
                    " \xe2\x94\x9c\xe2\x94\x80\xe2\x94\x80 utils.go\n" ..
                    " \xe2\x94\x9c\xe2\x94\x80\xe2\x94\x80 fix_handler.go (hotfix)\n" ..
                    " \xe2\x94\x9c\xe2\x94\x80\xe2\x94\x80 handler3.go (why?)\n" ..
                    " \xe2\x94\x9c\xe2\x94\x80\xe2\x94\x80 utils2.go\n" ..
                    " \xe2\x94\x94\xe2\x94\x80\xe2\x94\x80 tmp_workaround.go" },
            }

            local strategic_states = {
                { time = 0.0, value = "app.go" },
                { time = 2.0, value =
                    "app.go\n" ..
                    " \xe2\x94\x94\xe2\x94\x80\xe2\x94\x80 routes/\n" ..
                    "      \xe2\x94\x94\xe2\x94\x80\xe2\x94\x80 handler.go" },
                { time = 4.0, value =
                    "app.go\n" ..
                    " \xe2\x94\x9c\xe2\x94\x80\xe2\x94\x80 routes/\n" ..
                    " \xe2\x94\x82    \xe2\x94\x94\xe2\x94\x80\xe2\x94\x80 handler.go\n" ..
                    " \xe2\x94\x94\xe2\x94\x80\xe2\x94\x80 middleware/\n" ..
                    "      \xe2\x94\x94\xe2\x94\x80\xe2\x94\x80 auth.go" },
                { time = 6.0, value =
                    "app.go\n" ..
                    " \xe2\x94\x9c\xe2\x94\x80\xe2\x94\x80 routes/\n" ..
                    " \xe2\x94\x82    \xe2\x94\x9c\xe2\x94\x80\xe2\x94\x80 handler.go\n" ..
                    " \xe2\x94\x82    \xe2\x94\x94\xe2\x94\x80\xe2\x94\x80 users.go\n" ..
                    " \xe2\x94\x9c\xe2\x94\x80\xe2\x94\x80 middleware/\n" ..
                    " \xe2\x94\x82    \xe2\x94\x94\xe2\x94\x80\xe2\x94\x80 auth.go\n" ..
                    " \xe2\x94\x94\xe2\x94\x80\xe2\x94\x80 domain/\n" ..
                    "      \xe2\x94\x94\xe2\x94\x80\xe2\x94\x80 user.go" },
                { time = 8.0, value =
                    "app.go\n" ..
                    " \xe2\x94\x9c\xe2\x94\x80\xe2\x94\x80 routes/\n" ..
                    " \xe2\x94\x82    \xe2\x94\x9c\xe2\x94\x80\xe2\x94\x80 handler.go\n" ..
                    " \xe2\x94\x82    \xe2\x94\x94\xe2\x94\x80\xe2\x94\x80 users.go\n" ..
                    " \xe2\x94\x9c\xe2\x94\x80\xe2\x94\x80 middleware/\n" ..
                    " \xe2\x94\x82    \xe2\x94\x94\xe2\x94\x80\xe2\x94\x80 auth.go\n" ..
                    " \xe2\x94\x9c\xe2\x94\x80\xe2\x94\x80 domain/\n" ..
                    " \xe2\x94\x82    \xe2\x94\x94\xe2\x94\x80\xe2\x94\x80 user.go\n" ..
                    " \xe2\x94\x94\xe2\x94\x80\xe2\x94\x80 store/\n" ..
                    "      \xe2\x94\x94\xe2\x94\x80\xe2\x94\x80 postgres.go" },
            }

            local t1 = timeline:add_track()
            t1:add(f.timeline.text_keyframes(tac_txt, tactical_states, 10.0))

            local t2 = timeline:add_track()
            t2:add(f.timeline.text_keyframes(str_txt, strategic_states, 10.0))

            -- Start darkening the tactical side after 5s
            local t3 = timeline:add_track()
            t3:at(5.0, f.timeline.callback(function(w, t)
                darken_start = t.total
            end))

            -- Chaos particles on the left side (layer 0)
            local pixel = f.bitmap.new(2, 2)
            for py = 0, 1 do
                for px = 0, 1 do
                    pixel:set_dot(px, py, ACCENT)
                end
            end

            local spawner = world:spawn()
            spawner:set_position(f.vec3(0, 0, 0))
            world:add_root(spawner)
            local spawn_timer = 0
            local spawn_elapsed = 0
            spawner:set_behavior(function(e, w, t)
                spawn_elapsed = spawn_elapsed + t.delta
                if spawn_elapsed < 3.0 then return end
                spawn_timer = spawn_timer + t.delta
                if spawn_timer < 0.3 then return end
                spawn_timer = 0

                local p = w:spawn()
                local px_pos = left_x + math.random() * (mid_x - left_x - 4)
                local py_pos = diagram_y + math.random() * (sh * 0.5)
                p:set_position(f.vec3(px_pos, py_pos, 0))
                p:set_drawable(f.bitmap.braille(pixel))
                p:set_layer(0)
                p:set_body({
                    velocity = f.vec2(
                        (math.random() - 0.5) * 6,
                        (math.random() - 0.5) * 4
                    ),
                })
                p:set_age({ lifetime = 2.0 + math.random() * 2.0 })
                p:set_behavior(f.physics.turbulence(0.5, 3.0))
                p:set_behavior(f.physics.drag(0.4))
                p:set_behavior(f.physics.euler())
                p:set_behavior(f.particle.age_and_despawn())
                p:set_material(function(frag)
                    local life_frac = 0
                    if frag.lifetime > 0 then
                        life_frac = frag.age / frag.lifetime
                    end
                    return {
                        rune = frag.rune,
                        fg_r = ACCENT.r, fg_g = ACCENT.g, fg_b = ACCENT.b,
                        fg_alpha = (1.0 - life_frac) * 0.4,
                        bg_alpha = 0,
                    }
                end)
                w:add_root(p)
            end)

            -- Bottom insight on layer 2
            local insight = "Complexity compounds. Structure compounds too."
            local _, insight_txt = make_text(world, "", cx(insight), math.floor(sh * 0.88), BLACK, 2)
            local t4 = timeline:add_track()
            local ins_frames, ins_dur = typewriter_frames(insight, 0.03)
            t4:at(8.5, f.timeline.text_keyframes(insight_txt, ins_frames, ins_dur))

            timeline:start()
        end,
        on_update = function(world, time)
            -- Gradually darken the tactical (left) side
            if darken_start then
                local elapsed = time.total - darken_start
                local darken = math.min(0.6, elapsed * 0.1)
                f.set_post_process(0, function(frag)
                    local factor = 1.0 - darken
                    return {
                        rune = frag.rune,
                        fg_r = frag.fg_r * factor,
                        fg_g = frag.fg_g * factor,
                        fg_b = frag.fg_b * factor,
                        fg_alpha = frag.fg_alpha,
                        bg_r = frag.bg_r, bg_g = frag.bg_g, bg_b = frag.bg_b,
                        bg_alpha = frag.bg_alpha,
                    }
                end)
            end
        end,
        on_exit = function(world)
            if timeline then timeline:cleanup() end
        end,
    })
end

-- ============================================================================
-- Slide 5: Tetris — AI vs Human (12s, auto-advance)
-- ============================================================================
local function create_tetris_slide()
    local timeline

    return f.scene(sw, sh, {
        duration = 12.0,
        transition = { shader = f.transition.cross_fade, duration = 1.5 },
        trail = { layer = 0, effect = f.trail.ghost(0.88) },
        on_enter = function(world, ctx)
            make_bg(world, -1)
            timeline = f.timeline.new(world)

            -- Title
            make_text(world, "Speed vs Intentionality",
                cx("Speed vs Intentionality"), math.floor(sh * 0.05), BLACK, 2)

            local left_x = math.floor(sw * 0.08)
            local right_x = math.floor(sw * 0.55)
            local board_y = math.floor(sh * 0.15)

            -- Labels
            make_text(world, "Agent", left_x + 5, board_y - 2, ACCENT, 2)
            make_text(world, "Engineer", right_x + 3, board_y - 2, ACCENT2, 2)

            -- Left Tetris on layer 0 (ghost trail for speed blur)
            local _, left_txt = make_text(world, "", left_x, board_y, DARK, 0)

            -- Right Tetris on layer 1 (no trail)
            local _, right_txt = make_text(world, "", right_x, board_y, DARK, 1)

            -- Board border chars
            local TL = "\xe2\x94\x8c" -- ┌
            local TR = "\xe2\x94\x90" -- ┐
            local BL = "\xe2\x94\x94" -- └
            local BR = "\xe2\x94\x98" -- ┘
            local HZ = "\xe2\x94\x80" -- ─
            local VT = "\xe2\x94\x82" -- │
            local BK = "\xe2\x96\x88\xe2\x96\x88" -- ██
            local SP = "  " -- 2 spaces
            local W = 10 -- board width in blocks

            local function board_top()
                return TL .. string.rep(HZ, W * 2) .. TR
            end
            local function board_bot()
                return BL .. string.rep(HZ, W * 2) .. BR
            end
            local function row(blocks)
                -- blocks is a string of W chars: '#' = filled, '.' = empty
                local s = VT
                for i = 1, #blocks do
                    local c = blocks:sub(i, i)
                    if c == "#" then s = s .. BK
                    else s = s .. SP end
                end
                return s .. VT
            end

            -- Messy board states (fast cycling — agent dumps blocks)
            local messy = {
                { time = 0.0, value =
                    board_top() .. "\n" ..
                    row("..........") .. "\n" ..
                    row("..........") .. "\n" ..
                    row("..........") .. "\n" ..
                    row("..........") .. "\n" ..
                    row("..........") .. "\n" ..
                    row("..........") .. "\n" ..
                    row("..........") .. "\n" ..
                    row("..........") .. "\n" ..
                    row("..........") .. "\n" ..
                    row("..........") .. "\n" ..
                    board_bot() },
                { time = 1.0, value =
                    board_top() .. "\n" ..
                    row("..........") .. "\n" ..
                    row("..........") .. "\n" ..
                    row("..........") .. "\n" ..
                    row("..........") .. "\n" ..
                    row("..........") .. "\n" ..
                    row("..........") .. "\n" ..
                    row("..........") .. "\n" ..
                    row("..........") .. "\n" ..
                    row("....##....") .. "\n" ..
                    row(".##.......") .. "\n" ..
                    board_bot() },
                { time = 1.8, value =
                    board_top() .. "\n" ..
                    row("..........") .. "\n" ..
                    row("..........") .. "\n" ..
                    row("..........") .. "\n" ..
                    row("..........") .. "\n" ..
                    row("..........") .. "\n" ..
                    row("..........") .. "\n" ..
                    row("..........") .. "\n" ..
                    row("...##.....") .. "\n" ..
                    row("..####....") .. "\n" ..
                    row(".##..#....") .. "\n" ..
                    board_bot() },
                { time = 2.6, value =
                    board_top() .. "\n" ..
                    row("..........") .. "\n" ..
                    row("..........") .. "\n" ..
                    row("..........") .. "\n" ..
                    row("..........") .. "\n" ..
                    row("..........") .. "\n" ..
                    row("..........") .. "\n" ..
                    row("..#.......") .. "\n" ..
                    row("..##......") .. "\n" ..
                    row("..####.##.") .. "\n" ..
                    row(".##..#.##.") .. "\n" ..
                    board_bot() },
                { time = 3.4, value =
                    board_top() .. "\n" ..
                    row("..........") .. "\n" ..
                    row("..........") .. "\n" ..
                    row("..........") .. "\n" ..
                    row("..........") .. "\n" ..
                    row("..........") .. "\n" ..
                    row("#.........") .. "\n" ..
                    row("##........") .. "\n" ..
                    row("####......") .. "\n" ..
                    row("..####.##.") .. "\n" ..
                    row(".##..#.##.") .. "\n" ..
                    board_bot() },
                { time = 4.2, value =
                    board_top() .. "\n" ..
                    row("..........") .. "\n" ..
                    row("..........") .. "\n" ..
                    row("..........") .. "\n" ..
                    row(".....#....") .. "\n" ..
                    row("....##....") .. "\n" ..
                    row("#...#.....") .. "\n" ..
                    row("##..##....") .. "\n" ..
                    row("####.##...") .. "\n" ..
                    row("..####.##.") .. "\n" ..
                    row(".##..#.##.") .. "\n" ..
                    board_bot() },
                { time = 5.0, value =
                    board_top() .. "\n" ..
                    row("..........") .. "\n" ..
                    row("..........") .. "\n" ..
                    row("....##....") .. "\n" ..
                    row(".#...#..#.") .. "\n" ..
                    row("##..##..#.") .. "\n" ..
                    row("#...#..##.") .. "\n" ..
                    row("##..##..#.") .. "\n" ..
                    row("####.##...") .. "\n" ..
                    row("..####.##.") .. "\n" ..
                    row(".##..#.##.") .. "\n" ..
                    board_bot() },
                { time = 6.0, value =
                    board_top() .. "\n" ..
                    row("..........") .. "\n" ..
                    row("..##......") .. "\n" ..
                    row("..####..#.") .. "\n" ..
                    row(".#.#.#..#.") .. "\n" ..
                    row("##..##.##.") .. "\n" ..
                    row("#.#.#..##.") .. "\n" ..
                    row("##.###..#.") .. "\n" ..
                    row("####.##.#.") .. "\n" ..
                    row("..####.##.") .. "\n" ..
                    row(".##..#.##.") .. "\n" ..
                    board_bot() },
                { time = 7.0, value =
                    board_top() .. "\n" ..
                    row(".##.......") .. "\n" ..
                    row(".####.#.#.") .. "\n" ..
                    row("..####..#.") .. "\n" ..
                    row(".#.#.#.##.") .. "\n" ..
                    row("##.###.##.") .. "\n" ..
                    row("#.#.#..##.") .. "\n" ..
                    row("##.###.##.") .. "\n" ..
                    row("####.####.") .. "\n" ..
                    row("#.####.##.") .. "\n" ..
                    row(".##.##.##.") .. "\n" ..
                    board_bot() .. "\n" ..
                    "     GAME OVER" },
            }

            -- Clean board states (slow, methodical — engineer thinks)
            local clean = {
                { time = 0.0, value =
                    board_top() .. "\n" ..
                    row("..........") .. "\n" ..
                    row("..........") .. "\n" ..
                    row("..........") .. "\n" ..
                    row("..........") .. "\n" ..
                    row("..........") .. "\n" ..
                    row("..........") .. "\n" ..
                    row("..........") .. "\n" ..
                    row("..........") .. "\n" ..
                    row("..........") .. "\n" ..
                    row("..........") .. "\n" ..
                    board_bot() },
                { time = 2.0, value =
                    board_top() .. "\n" ..
                    row("..........") .. "\n" ..
                    row("..........") .. "\n" ..
                    row("..........") .. "\n" ..
                    row("..........") .. "\n" ..
                    row("..........") .. "\n" ..
                    row("..........") .. "\n" ..
                    row("..........") .. "\n" ..
                    row("..........") .. "\n" ..
                    row("..........") .. "\n" ..
                    row("...####...") .. "\n" ..
                    board_bot() },
                { time = 3.5, value =
                    board_top() .. "\n" ..
                    row("..........") .. "\n" ..
                    row("..........") .. "\n" ..
                    row("..........") .. "\n" ..
                    row("..........") .. "\n" ..
                    row("..........") .. "\n" ..
                    row("..........") .. "\n" ..
                    row("..........") .. "\n" ..
                    row("..........") .. "\n" ..
                    row("......##..") .. "\n" ..
                    row("...####...") .. "\n" ..
                    board_bot() },
                { time = 5.0, value =
                    board_top() .. "\n" ..
                    row("..........") .. "\n" ..
                    row("..........") .. "\n" ..
                    row("..........") .. "\n" ..
                    row("..........") .. "\n" ..
                    row("..........") .. "\n" ..
                    row("..........") .. "\n" ..
                    row("..........") .. "\n" ..
                    row("..........") .. "\n" ..
                    row("..######..") .. "\n" ..
                    row("..########") .. "\n" ..
                    board_bot() },
                { time = 6.5, value =
                    board_top() .. "\n" ..
                    row("..........") .. "\n" ..
                    row("..........") .. "\n" ..
                    row("..........") .. "\n" ..
                    row("..........") .. "\n" ..
                    row("..........") .. "\n" ..
                    row("..........") .. "\n" ..
                    row("..........") .. "\n" ..
                    row("..........") .. "\n" ..
                    row("##########") .. "\n" ..
                    row("##########") .. "\n" ..
                    board_bot() .. "\n" ..
                    "   \xe2\x94\x80\xe2\x94\x80 rows cleared \xe2\x94\x80\xe2\x94\x80" },
                { time = 8.0, value =
                    board_top() .. "\n" ..
                    row("..........") .. "\n" ..
                    row("..........") .. "\n" ..
                    row("..........") .. "\n" ..
                    row("..........") .. "\n" ..
                    row("..........") .. "\n" ..
                    row("..........") .. "\n" ..
                    row("..........") .. "\n" ..
                    row("..........") .. "\n" ..
                    row("..........") .. "\n" ..
                    row("....##....") .. "\n" ..
                    board_bot() .. "\n" ..
                    "   \xe2\x94\x80\xe2\x94\x80 rows cleared \xe2\x94\x80\xe2\x94\x80" },
                { time = 9.5, value =
                    board_top() .. "\n" ..
                    row("..........") .. "\n" ..
                    row("..........") .. "\n" ..
                    row("..........") .. "\n" ..
                    row("..........") .. "\n" ..
                    row("..........") .. "\n" ..
                    row("..........") .. "\n" ..
                    row("..........") .. "\n" ..
                    row("..........") .. "\n" ..
                    row("..##..##..") .. "\n" ..
                    row("..########") .. "\n" ..
                    board_bot() .. "\n" ..
                    "   \xe2\x94\x80\xe2\x94\x80 rows cleared \xe2\x94\x80\xe2\x94\x80" },
            }

            local t1 = timeline:add_track()
            t1:add(f.timeline.text_keyframes(left_txt, messy, 10.0))

            local t2 = timeline:add_track()
            t2:add(f.timeline.text_keyframes(right_txt, clean, 10.0))

            -- Bottom text
            local bottom = "Same blocks. Different judgment."
            local _, bottom_txt = make_text(world, "", cx(bottom), math.floor(sh * 0.90), BLACK, 2)
            local t3 = timeline:add_track()
            local frames, dur = typewriter_frames(bottom, 0.04)
            t3:at(8.0, f.timeline.text_keyframes(bottom_txt, frames, dur))

            timeline:start()
        end,
        on_exit = function(world)
            if timeline then timeline:cleanup() end
        end,
    })
end

-- ============================================================================
-- Slide 6: Takeaways (no auto-advance, final slide)
-- ============================================================================
local function create_takeaways_slide()
    local timeline

    return f.scene(sw, sh, {
        on_enter = function(world, ctx)
            make_bg(world)
            timeline = f.timeline.new(world)

            make_text(world, "Takeaways", cx("Takeaways"), math.floor(sh * 0.10), BLACK)

            local points = {
                "Working code is the beginning, not the finish line",
                "Every shortcut is a bet against your future self",
                "AI accelerates output \xe2\x94\x80 judgment determines direction",
            }

            local start_y = math.floor(sh * 0.28)
            local spacing = 4
            local point_txts = {}

            for i, pt in ipairs(points) do
                local bullet = "  > " .. pt
                local _, txt = make_text(world, "", cx(bullet), start_y + (i - 1) * spacing, BLACK)
                table.insert(point_txts, { txt = txt, full = bullet })
            end

            local track = timeline:add_track()
            local t = 0.3
            for _, pt in ipairs(point_txts) do
                local frames, dur = typewriter_frames(pt.full, 0.02)
                track:at(t, f.timeline.text_keyframes(pt.txt, frames, dur))
                t = t + dur + 0.5
            end

            -- Closing phrase
            local closing = "Ask which decisions you're accelerating and which you're skipping."
            local _, closing_txt = make_text(world, "", cx(closing), math.floor(sh * 0.78), DARK)
            local closing_frames, closing_dur = typewriter_frames(closing, 0.03)
            track:at(t + 0.8, f.timeline.text_keyframes(closing_txt, closing_frames, closing_dur))

            -- Final emphasis
            local emphasis = "Be intentional."
            local _, emphasis_txt = make_text(world, "", cx(emphasis), math.floor(sh * 0.85), ACCENT)
            local emp_frames, _ = typewriter_frames(emphasis, 0.06)
            track:at(t + closing_dur + 1.5, f.timeline.text_keyframes(emphasis_txt, emp_frames, 2.0))

            timeline:start()
        end,
        on_exit = function(world)
            if timeline then timeline:cleanup() end
        end,
    })
end

-- ============================================================================
-- Main
-- ============================================================================
f.on_enter(function(world, ctx)
    sw = ctx.width
    sh = ctx.height

    local sm = f.scene_manager(sw, sh)

    sm:add(create_title_slide())
    sm:add(create_cost_slide())
    sm:add(create_domino_slide())
    sm:add(create_tactical_slide())
    sm:add(create_tetris_slide())
    sm:add(create_takeaways_slide())

    sm:start()
end)
