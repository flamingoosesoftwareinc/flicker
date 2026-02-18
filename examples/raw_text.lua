-- raw_text.lua: Raw text drawable with keyframed ASCII art animation.
local f = require("flicker")

f.on_enter(function(world, ctx)
    f.set_trail(0, f.trail.ghost(0.88))

    -- Static label
    local label = f.text("[ raw text demo ]", {
        fg = f.color(100, 200, 255),
    })
    local lbl = world:spawn()
    lbl:set_transform({
        position = f.vec3(2, 1, 0),
    })
    lbl:set_drawable(label)
    world:add_root(lbl)

    -- Animated ASCII diagram using text keyframes
    local diagram = f.text("", {
        fg = f.color(255, 220, 100),
    })
    local diag = world:spawn()
    diag:set_transform({
        position = f.vec3(ctx.width / 2 - 12, ctx.height / 2 - 3, 0),
        scale = f.vec3(1, 1, 1),
    })
    diag:set_drawable(diagram)
    world:add_root(diag)

    -- Typewriter status line
    local status = f.text("", {
        fg = f.color(180, 255, 180),
    })
    local stat = world:spawn()
    stat:set_transform({
        position = f.vec3(ctx.width / 2 - 12, ctx.height / 2 + 5, 0),
    })
    stat:set_drawable(status)
    world:add_root(stat)

    -- Timeline: step through diagram states + status text
    local tl = f.timeline.new(world)
    tl:set_loop(true)

    -- Diagram track
    local t1 = tl:add_track()
    t1:add(f.timeline.text_keyframes(diagram, {
        { time = 0.0, value = [[
┌────────────────────┐
│                    │
│                    │
│                    │
└────────────────────┘]] },
        { time = 1.0, value = [[
┌────────────────────┐
│  ┌──────┐          │
│  │ Init │          │
│  └──┬───┘          │
└─────┼──────────────┘]] },
        { time = 2.0, value = [[
┌────────────────────┐
│  ┌──────┐ ┌─────┐ │
│  │ Init ├─► Run │ │
│  └──────┘ └──┬──┘ │
└──────────────┼────┘]] },
        { time = 3.0, value = [[
┌────────────────────┐
│  ┌──────┐ ┌─────┐ │
│  │ Init ├─► Run │ │
│  └──────┘ └──┬──┘ │
└──────────────┼────┘]] },
        { time = 4.0, value = [[
┌────────────────────┐
│  ┌──────┐ ┌─────┐ │
│  │ Init ├─► Run │ │
│  └──────┘ └──┬──┘ │
│         ┌────▼──┐ │
│         │ Done! │ │
│         └───────┘ │
└───────────────────┘]] },
    }, 6.0))

    -- Status text track
    local t2 = tl:add_track()
    t2:add(f.timeline.text_keyframes(status, {
        { time = 0.0, value = "> drawing container..." },
        { time = 1.0, value = "> adding init node..." },
        { time = 2.0, value = "> connecting run node..." },
        { time = 3.0, value = "> processing..." },
        { time = 4.0, value = "> complete!" },
    }, 6.0))

    -- Slide the diagram entity across while animating
    local t3 = tl:add_track()
    t3:add(f.timeline.tween(diag, "position.y", {
        from = ctx.height / 2 - 3,
        to = ctx.height / 2 - 5,
        duration = 4.0,
        easing = "out_quad",
    }))

    tl:start()
end)
