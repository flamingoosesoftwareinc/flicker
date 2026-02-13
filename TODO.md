# Flicker

A terminal rendering engine.

## Philosophy

Iterative, minimal, working software at every step. The terminal is a 2D cell grid. Everything else is abstraction on top of that.

Flat package structure. High cohesion, low coupling. No premature abstraction.

## Terminal Backend: tcell

Flicker uses [tcell](https://github.com/gdamore/tcell) for direct terminal access. Bubble Tea's retained-mode string diffing is the wrong fit for an engine that redraws a full cell buffer every frame. tcell gives us immediate-mode cell writes, input handling, and terminal lifecycle management without overhead we'd fight against.

Bubble Tea remains a good choice for a future editor/preview tool built *on top of* Flicker.

## Iteration 1: A Square on Screen (complete)

Entity/component world, canvas buffer, scene graph traversal, and terminal flush. A rectangle rendered to the terminal via a transform, a `Drawable` interface, and a direct cell buffer.

The `Screen` interface decouples rendering from the terminal backend. `TcellScreen` drives a real terminal; `SimScreen` captures frames in-memory for golden tests (using `goldie/v2`).

## Iteration 2: Behavior System, Wave Functions, and Tick Loop (complete)

Tick loop (`update → render → flush` at 60 FPS) with goroutine-pumped non-blocking input. `Behavior` component — a per-entity update function `func(dt, Entity, *World)` — iterated by `UpdateBehaviors`. Wave functions in `fmath` (`Saw`, `Sine`, `Triangle`, `Square`, `Pulse`) with period 1.0, composable with `Remap`. Demo: box seesaws horizontally via `Triangle`. Multi-frame golden test with deterministic fixed `dt`.

## What Comes Next

Rough ordering for future iterations, not a commitment:

- **Color**: `FG`/`BG` on `Cell`, style propagation through `Drawable` or a new `Style` component
- **Animation**: `Tween` system that interpolates component fields over time
- **Materials**: cell transform functions `(x, y, time, cell) → cell`
- **Text rendering**: bitmap font rasterization → braille/block mapping
- **Particles**: `ParticleEmitter` component, particle pool, attractor targets
- **Post-processing**: bloom as a canvas-to-canvas pass
- **Scenes/slides**: ordered scene list, transitions
- **Scripting**: Lua bindings over the Go API
- **Playback/recording**: VHS / asciinema integration

## Package Structure

```
flicker/
  core/
    entity.go      // Entity, World, parent/child relationships
    transform.go   // Transform component (position via Vec3)
    drawable.go    // Drawable interface
    rect.go        // Rect drawable
    behavior.go    // Behavior component + UpdateBehaviors system
    canvas.go      // Cell, Canvas (2D cell buffer)
    render.go      // Scene graph traversal, drawable → canvas
  fmath/
    vec2.go        // Vec2 (X, Y float64), add, sub, scale, normalize, lerp
    vec3.go        // Vec3 (X, Y, Z float64), add, sub, scale, normalize, lerp
    interpolation.go // Lerp, InverseLerp, Remap
    easing.go      // Easing functions: linear, quad, cubic, elastic, bounce (all func(t float64) float64)
    wave.go        // Wave functions: saw, sine, triangle, square, pulse (period 1.0)
  terminal/
    screen.go      // Screen interface, TcellScreen (tcell backend)
    simscreen.go   // SimScreen (in-memory backend for testing)
  cmd/
    flicker/
      main.go      // Wire everything, run the tick loop
  golden_test.go   // Integration golden tests
  testdata/        // Golden files
```

`fmath` depends on nothing. `core` depends on `fmath`. `terminal` depends on `core` and `tcell`. `cmd` depends on all three.
