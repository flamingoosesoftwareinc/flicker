# Flicker

A terminal rendering engine.

## Philosophy

Iterative, minimal, working software at every step. The terminal is a 2D cell grid. Everything else is abstraction on top of that.

Flat package structure. High cohesion, low coupling. No premature abstraction.

## Terminal Backend: tcell

Flicker uses [tcell](https://github.com/gdamore/tcell) for direct terminal access. Bubble Tea's retained-mode string diffing is the wrong fit for an engine that redraws a full cell buffer every frame. tcell gives us immediate-mode cell writes, input handling, and terminal lifecycle management without overhead we'd fight against.

Bubble Tea remains a good choice for a future editor/preview tool built *on top of* Flicker.

## Iteration 1: A Square on Screen (complete)

Entity/component world, canvas buffer, scene graph traversal, and terminal flush. A rectangle rendered to the terminal via a transform, a geometry component, and a direct cell buffer.

The `Screen` interface decouples rendering from the terminal backend. `TcellScreen` drives a real terminal; `SimScreen` captures frames in-memory for golden tests (using `goldie/v2`).

## What Comes Next

Rough ordering for future iterations, not a commitment:

- **Color**: `FG`/`BG` on `Cell`, style propagation through `Geometry` or a new `Style` component
- **Frame loop**: `tick` → `update` → `render` → `flush` at a target FPS
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
    transform.go   // Transform component (position via Vec2)
    geometry.go    // Geometry component, GeometryKind enum
    canvas.go      // Cell, Canvas (2D cell buffer)
    render.go      // Scene graph traversal, geometry → canvas
  fmath/
    vec2.go        // Vec2 (X, Y float64), add, sub, scale, normalize, lerp
    interpolation.go // Lerp, InverseLerp, Remap, cubic bezier, spring solver
    easing.go      // Easing functions: linear, quad, cubic, elastic, bounce (all func(t float64) float64)
  terminal/
    screen.go      // Screen interface, TcellScreen (tcell backend)
    simscreen.go   // SimScreen (in-memory backend for testing)
  cmd/
    flicker/
      main.go      // Wire everything, run the loop
  golden_test.go   // Integration golden tests
  testdata/        // Golden files
```

`fmath` depends on nothing. `core` depends on `fmath`. `terminal` depends on `core` and `tcell`. `cmd` depends on all three.
