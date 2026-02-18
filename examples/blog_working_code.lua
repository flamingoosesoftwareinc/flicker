-- blog_working_code.lua: "Working Code is Not Enough" blog post explainer
-- Press SPACE to advance slides. Press 'q' or ESC to quit.
local f = require("flicker")

local sw, sh

-- Color palette
local BG       = f.color(222, 220, 215) -- #DEDCD7
local BLACK    = f.color(30, 30, 30)
local DARK     = f.color(60, 60, 60)
local MID      = f.color(120, 120, 120)
local ACCENT   = f.color(180, 60, 40)   -- muted red for emphasis
local ACCENT2  = f.color(40, 100, 160)  -- muted blue

-- Background fill material: forces every cell to have the warm gray BG
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

-- Helper: create a full-screen background rect on a given layer
local function make_bg(world, layer)
    local bg = world:spawn()
    bg:set_position(f.vec3(0, 0, 0))
    bg:set_drawable(f.bitmap.rect(sw, sh, BG, BG))
    bg:set_layer(layer or -1)
    world:add_root(bg)
    return bg
end

-- Helper: create a raw text entity
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

-- Helper: center text horizontally
local function cx(text)
    return math.floor(sw / 2 - #text / 2)
end

-- Helper: build typewriter keyframes for a string
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
-- Slide 1: Title Card
-- ============================================================================
local function create_title_slide()
    local timeline

    return f.scene(sw, sh, {
        duration = 6.0,
        transition = { shader = f.transition.cross_fade, duration = 1.5 },
        on_enter = function(world, ctx)
            make_bg(world)
            timeline = f.timeline.new(world)

            -- Title
            local title_str = "Working Code is Not Enough"
            local _, title_txt = make_text(world, "", cx(title_str), math.floor(sh * 0.35), BLACK)

            -- Subtitle
            local sub_str = "When tools change, the principles stay the same"
            local _, sub_txt = make_text(world, "", cx(sub_str), math.floor(sh * 0.35) + 3, MID)

            -- Author
            local author_str = "Ahmed Al-Hulaibi"
            make_text(world, author_str, cx(author_str), math.floor(sh * 0.35) + 6, DARK)

            local title_frames, title_dur = typewriter_frames(title_str, 0.04)
            local sub_frames, sub_dur = typewriter_frames(sub_str, 0.03)

            local t1 = timeline:add_track()
            t1:add(f.timeline.text_keyframes(title_txt, title_frames, title_dur))

            local t2 = timeline:add_track()
            t2:at(title_dur, f.timeline.text_keyframes(sub_txt, sub_frames, sub_dur))

            timeline:start()
        end,
        on_exit = function(world)
            if timeline then timeline:cleanup() end
        end,
    })
end

-- ============================================================================
-- Slide 2: "The Hot Take"
-- ============================================================================
local function create_hot_take_slide()
    local timeline

    return f.scene(sw, sh, {
        duration = 8.0,
        transition = { shader = f.transition.cross_fade, duration = 1.5 },
        on_enter = function(world, ctx)
            make_bg(world)
            timeline = f.timeline.new(world)

            -- Quote
            local quote = {
                '  "Software is just a means to an end.',
                '   If it works, it\'s good code."',
            }
            local quote_y = math.floor(sh * 0.25)

            local quote_txts = {}
            for i, line in ipairs(quote) do
                local _, txt = make_text(world, "", cx(line), quote_y + (i - 1), MID)
                table.insert(quote_txts, txt)
            end

            -- Rebuttal
            local rebuttal_lines = {
                { text = "Working code is not the end of software engineering",  color = BLACK },
                { text = "it's the start of its long-term cost.",               color = ACCENT },
            }
            local reb_y = math.floor(sh * 0.50)

            local reb_txts = {}
            for i, line in ipairs(rebuttal_lines) do
                local _, txt = make_text(world, "", cx(line.text), reb_y + (i - 1) * 2, line.color)
                table.insert(reb_txts, txt)
            end

            -- Timeline: typewriter the quote, then reveal rebuttal
            local t1 = timeline:add_track()
            local offset = 0
            for i, line in ipairs(quote) do
                local frames, dur = typewriter_frames(line, 0.03)
                t1:at(offset, f.timeline.text_keyframes(quote_txts[i], frames, dur))
                offset = offset + dur
            end

            local t2 = timeline:add_track()
            local reb_start = offset + 0.5
            for i, line in ipairs(rebuttal_lines) do
                local frames, dur = typewriter_frames(line.text, 0.02)
                t2:at(reb_start, f.timeline.text_keyframes(reb_txts[i], frames, dur))
                reb_start = reb_start + dur
            end

            timeline:start()
        end,
        on_exit = function(world)
            if timeline then timeline:cleanup() end
        end,
    })
end

-- ============================================================================
-- Slide 3: "The Hard Questions" - progressive reveal
-- ============================================================================
local function create_questions_slide()
    local timeline

    return f.scene(sw, sh, {
        duration = 8.0,
        transition = { shader = f.transition.cross_fade, duration = 1.5 },
        on_enter = function(world, ctx)
            make_bg(world)
            timeline = f.timeline.new(world)

            make_text(world, "Time changes everything in software.",
                cx("Time changes everything in software."), math.floor(sh * 0.15), DARK)

            local questions = {
                "Can it be understood and maintained by others?",
                "Can it be deployed, operated, and debugged?",
                "Can it scale as more people depend on it?",
            }

            local q_y = math.floor(sh * 0.35)
            local q_txts = {}

            for i, q in ipairs(questions) do
                local full = "  " .. tostring(i) .. ". " .. q
                local _, txt = make_text(world, "", cx(full), q_y + (i - 1) * 3, BLACK)
                table.insert(q_txts, { txt = txt, full = full })
            end

            local track = timeline:add_track()
            local t = 0.8
            for _, q in ipairs(q_txts) do
                local frames, dur = typewriter_frames(q.full, 0.025)
                track:at(t, f.timeline.text_keyframes(q.txt, frames, dur))
                t = t + dur + 0.3
            end

            timeline:start()
        end,
        on_exit = function(world)
            if timeline then timeline:cleanup() end
        end,
    })
end

-- ============================================================================
-- Slide 4: Tactical vs Strategic - animated tree diagram
-- ============================================================================
local function create_tree_slide()
    local timeline

    return f.scene(sw, sh, {
        duration = 10.0,
        transition = { shader = f.transition.cross_fade, duration = 1.5 },
        on_enter = function(world, ctx)
            make_bg(world)
            timeline = f.timeline.new(world)

            make_text(world, "Tactical vs Strategic",
                cx("Tactical vs Strategic"), math.floor(sh * 0.08), BLACK)

            local left_x = math.floor(sw * 0.08)
            local right_x = math.floor(sw * 0.55)
            local diagram_y = math.floor(sh * 0.20)

            make_text(world, "Tactical: \"just make it work\"", left_x, diagram_y - 2, ACCENT)
            make_text(world, "Strategic: design for change", right_x, diagram_y - 2, ACCENT2)

            -- Tactical side: tree that grows messy
            local tactical_diagram = f.text("", { fg = DARK })
            local td_ent = world:spawn()
            td_ent:set_position(f.vec3(left_x, diagram_y, 0))
            td_ent:set_drawable(tactical_diagram)
            td_ent:set_material(bg_material())
            world:add_root(td_ent)

            -- Strategic side: clean tree
            local strategic_diagram = f.text("", { fg = DARK })
            local sd_ent = world:spawn()
            sd_ent:set_position(f.vec3(right_x, diagram_y, 0))
            sd_ent:set_drawable(strategic_diagram)
            sd_ent:set_material(bg_material())
            world:add_root(sd_ent)

            local tactical_states = {
                { time = 0.0, value = [[app.go]] },
                { time = 1.0, value = [[
app.go
 └── handler.go]] },
                { time = 2.0, value = [[
app.go
 ├── handler.go
 └── handler2.go (copy-paste)]] },
                { time = 3.0, value = [[
app.go
 ├── handler.go
 ├── handler2.go (copy-paste)
 ├── utils.go
 └── fix_handler.go (hotfix)]] },
                { time = 4.5, value = [[
app.go
 ├── handler.go
 ├── handler2.go (copy-paste)
 ├── utils.go
 ├── fix_handler.go (hotfix)
 ├── handler3.go (copy-paste)
 ├── utils2.go (why?)
 └── tmp_workaround.go]] },
            }

            local strategic_states = {
                { time = 0.0, value = [[app.go]] },
                { time = 1.0, value = [[
app.go
 └── routes/
      └── handler.go]] },
                { time = 2.0, value = [[
app.go
 ├── routes/
 │    └── handler.go
 └── middleware/
      └── auth.go]] },
                { time = 3.0, value = [[
app.go
 ├── routes/
 │    ├── handler.go
 │    └── users.go
 ├── middleware/
 │    └── auth.go
 └── domain/
      └── user.go]] },
                { time = 4.5, value = [[
app.go
 ├── routes/
 │    ├── handler.go
 │    └── users.go
 ├── middleware/
 │    └── auth.go
 ├── domain/
 │    └── user.go
 └── store/
      └── postgres.go]] },
            }

            local t1 = timeline:add_track()
            t1:add(f.timeline.text_keyframes(tactical_diagram, tactical_states, 7.0))

            local t2 = timeline:add_track()
            t2:add(f.timeline.text_keyframes(strategic_diagram, strategic_states, 7.0))

            -- Bottom insight
            local insight_str = "Each shortcut adds complexity. Over time, it compounds."
            local _, insight_txt = make_text(world, "", cx(insight_str), math.floor(sh * 0.85), ACCENT)

            local t3 = timeline:add_track()
            local frames, dur = typewriter_frames(insight_str, 0.03)
            t3:at(5.0, f.timeline.text_keyframes(insight_txt, frames, dur))

            timeline:start()
        end,
        on_exit = function(world)
            if timeline then timeline:cleanup() end
        end,
    })
end

-- ============================================================================
-- Slide 5: The Vibe Loop vs The Real Loop
-- ============================================================================
local function create_loops_slide()
    local timeline

    return f.scene(sw, sh, {
        duration = 10.0,
        transition = { shader = f.transition.cross_fade, duration = 1.5 },
        on_enter = function(world, ctx)
            make_bg(world)
            timeline = f.timeline.new(world)

            make_text(world, "Two Development Loops",
                cx("Two Development Loops"), math.floor(sh * 0.08), BLACK)

            local left_x = math.floor(sw * 0.05)
            local right_x = math.floor(sw * 0.55)
            local loop_y = math.floor(sh * 0.20)

            make_text(world, "Strategic Loop", left_x + 6, loop_y - 2, ACCENT2)
            make_text(world, "Vibe Loop", right_x + 8, loop_y - 2, ACCENT)

            local real_diagram = f.text("", { fg = DARK })
            local rd_ent = world:spawn()
            rd_ent:set_position(f.vec3(left_x, loop_y, 0))
            rd_ent:set_drawable(real_diagram)
            rd_ent:set_material(bg_material())
            world:add_root(rd_ent)

            local vibe_diagram = f.text("", { fg = DARK })
            local vd_ent = world:spawn()
            vd_ent:set_position(f.vec3(right_x, loop_y, 0))
            vd_ent:set_drawable(vibe_diagram)
            vd_ent:set_material(bg_material())
            world:add_root(vd_ent)

            local real_states = {
                { time = 0.0, value = [[
┌─────────────────────────┐
│  Define the problem     │
└────────────┬────────────┘
             │]] },
                { time = 1.2, value = [[
┌─────────────────────────┐
│  Define the problem     │
└────────────┬────────────┘
             │
┌────────────▼────────────┐
│  Design the solution    │
└────────────┬────────────┘
             │]] },
                { time = 2.4, value = [[
┌─────────────────────────┐
│  Define the problem     │
└────────────┬────────────┘
             │
┌────────────▼────────────┐
│  Design the solution    │
└────────────┬────────────┘
             │
┌────────────▼────────────┐
│  Build + Review         │
└────────────┬────────────┘
             │]] },
                { time = 3.6, value = [[
┌─────────────────────────┐
│  Define the problem     │
└────────────┬────────────┘
             │
┌────────────▼────────────┐
│  Design the solution    │
└────────────┬────────────┘
             │
┌────────────▼────────────┐
│  Build + Review         │
└────────────┬────────────┘
             │
┌────────────▼────────────┐
│  Ship + Maintain        │
└─────────────────────────┘]] },
            }

            local vibe_states = {
                { time = 0.0, value = [[
┌─────────────────────────┐
│  Prompt                 │
└────────────┬────────────┘
             │]] },
                { time = 1.2, value = [[
┌─────────────────────────┐
│  Prompt                 │
└────────────┬────────────┘
             │
┌────────────▼────────────┐
│  Smoke test             │
└────────────┬────────────┘
             │]] },
                { time = 2.4, value = [[
┌─────────────────────────┐
│  Prompt                 │
└────────────┬────────────┘
             │
┌────────────▼────────────┐
│  Smoke test             │
└────────────┬────────────┘
             │
┌────────────▼────────────┐
│  Ship it                │
└────────────┬────────────┘
             │]] },
                { time = 3.6, value = [[
┌─────────────────────────┐
│  Prompt                 │
└────────────┬────────────┘
             │
┌────────────▼────────────┐
│  Smoke test             │
└────────────┬────────────┘
             │
┌────────────▼────────────┐
│  Ship it                │
└────────────┬────────────┘
             │
┌────────────▼────────────┐
│  Fix it later (maybe)   │
└─────────────────────────┘]] },
            }

            local t1 = timeline:add_track()
            t1:add(f.timeline.text_keyframes(real_diagram, real_states, 6.0))

            local t2 = timeline:add_track()
            t2:add(f.timeline.text_keyframes(vibe_diagram, vibe_states, 6.0))

            local missing = "What's missing? Intentionality."
            local _, missing_txt = make_text(world, "", cx(missing), math.floor(sh * 0.88), ACCENT)

            local t3 = timeline:add_track()
            local frames, dur = typewriter_frames(missing, 0.04)
            t3:at(4.5, f.timeline.text_keyframes(missing_txt, frames, dur))

            timeline:start()
        end,
        on_exit = function(world)
            if timeline then timeline:cleanup() end
        end,
    })
end

-- ============================================================================
-- Slide 6: Takeaways
-- ============================================================================
local function create_takeaways_slide()
    local timeline

    return f.scene(sw, sh, {
        on_enter = function(world, ctx)
            make_bg(world)
            timeline = f.timeline.new(world)

            make_text(world, "Takeaways", cx("Takeaways"), math.floor(sh * 0.10), BLACK)

            local points = {
                "Software success is measured over time, not at first run",
                "Tactical programming trades future clarity for present speed",
                "LLMs lower the cost of choosing the tactical path",
                "Lower cost does not mean lower consequence",
                "Design judgment determines whether speed compounds or collapses",
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
                t = t + dur + 0.4
            end

            local closing = "Ask which decisions you're accelerating and which you're skipping."
            local _, closing_txt = make_text(world, "", cx(closing), math.floor(sh * 0.82), ACCENT)

            local frames, dur = typewriter_frames(closing, 0.03)
            track:at(t + 0.5, f.timeline.text_keyframes(closing_txt, frames, dur))

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
    sm:add(create_hot_take_slide())
    sm:add(create_questions_slide())
    sm:add(create_tree_slide())
    sm:add(create_loops_slide())
    sm:add(create_takeaways_slide())

    sm:start()
end)
