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

## Iteration 3: Color, Alpha, and Materials (complete)

Color plumbing end-to-end (FG/BG through to tcell). Cell-level `Alpha float64` for transparency. Per-entity `Material` component `func(x, y int, t float64, cell Cell) Cell` — a fragment shader applied during render. Time passed into `Render`. Uncap framerate. VT emulator golden tests capture ANSI color sequences. Overlapping animated objects test validates painter's algorithm with distinct colored entities.

## Iteration 4: Layers and Compositing (complete)

Ordered canvas layers with per-entity layer assignment (`AddLayer`/`Layer` component). Pluggable `BlendMode func(d, s uint8, alpha float64) uint8` for per-channel color blending; `BlendNormal` (linear interpolation) is the default. `BlendColor` applies a `BlendMode` per-channel; `BlendCell` implements the "over" operator for terminal cells (alpha blending FG/BG, rune precedence: real src rune wins, empty src preserves dst text). `Canvas.Composite` applies cell-by-cell blending with a given `BlendMode`. `Compositor` owns per-layer canvases, renders each root's entity tree into its layer, sorts layers back-to-front, applies optional `LayerPostProcess` passes, then composites onto the destination canvas. Children inherit their root's layer. Unassigned entities default to layer 0. `core.Render` remains available for simple code that doesn't need layers.

## Iteration 5: Photoshop-Style Blend Modes (complete)

Full set of Photoshop-style per-channel blend modes in `core/blend.go`, all following the same pattern: normalize to [0,1], compute raw blend, lerp with alpha, convert back to uint8. Internal `blendLerp` helper eliminates boilerplate. Each `BlendMode` has a corresponding `ColorBlend` wrapper for direct use with `Compositor.SetBlend`.

**Blend modes added:** Multiply, Screen, Overlay, HardLight, SoftLight, Difference, Exclusion, HardMix, Darken, Lighten, LinearDodge, LinearBurn, ColorDodge, ColorBurn. Categorized as Darken (Multiply, Darken, LinearBurn, ColorBurn), Lighten (Screen, Lighten, LinearDodge, ColorDodge), Contrast (Overlay, HardLight, SoftLight), Inversion (Difference, Exclusion), and Posterize (HardMix).

Demo expanded to 5 layers: Red/Normal base, Green/Multiply, Blue/Screen, Yellow/Overlay, Cyan/Difference — each with distinct animations and `Alpha=0.7` so blending is visible in overlap regions. Golden test `TestBlendModes` verifies blended ANSI output from 3 overlapping layers (Normal, Multiply, Screen). Demo has pause/step controls: Space toggles pause, Right/`.` steps one frame (1/60s), `q`/Esc quits. Simulation time is decoupled from wall clock.

## Iteration 6: Tween System (complete)

`Tween` and `TweenVec3` — stateful interpolators in `fmath` that track elapsed time, apply optional easing, and return interpolated values. Pure math, no engine dependency. Used from behavior closures to replace manual wave + `Remap` patterns.

`Tween` interpolates `float64` values; `TweenVec3` interpolates `Vec3` values. Both support `Update(dt)` → current value, `Done()` → completion check, `Reset()` → replay. Easing is pluggable via `func(float64) float64` (nil defaults to linear). `Clamp` helper added to `interpolation.go`.

Demo updated: Box A uses `Tween` with `EaseInOutCubic` for smooth horizontal ping-pong. Box D uses `TweenVec3` with `EaseInOutQuad` for diagonal ping-pong. Golden test `TestTween` verifies tween-driven animation with easing over 6 frames.

Canvas background color: `Canvas.Background` field — `Clear()` fills with it instead of `Cell{}`. Default is zero-value (transparent), fully backward compatible. Demo sets opaque black background so non-Normal blend modes (Multiply, Difference, etc.) composite correctly over empty regions.

## Iteration 7: Fragment Shader System (complete)

Unified `Material` and `LayerPostProcess` under a single `Fragment` struct. Both are now `func(f Fragment) Cell` — per-cell fragment shaders with consistent signatures, extensible for future Lua scripting.

`Fragment` carries local coords (`X`, `Y`), absolute canvas position (`ScreenX`, `ScreenY`), `Time`, the current `Cell`, and a `Source` canvas for neighbor reads. Materials receive the layer canvas as `Source`; post-process passes receive a snapshot (double-buffered via `Canvas.Clone`/`CopyInto`).

`Canvas.Clone()` returns a deep copy. `Canvas.CopyInto(dst)` copies cells into an existing canvas without allocating. The `Compositor` reuses a single `scratch` buffer across frames for post-process snapshot double-buffering, eliminating per-frame allocations after warmup.

## Iteration 8: Sub-cell Bitmap Buffer (complete)

High-resolution `Bitmap` pixel buffer with two encoding modes that map back to terminal cells. Flat row-major `[]Color` and `[]float64` arrays for cache locality, matching the `Cell` alpha pattern.

**Braille encoding** (2x4 dots per cell, U+2800-U+28FF): Each 2x4 pixel block maps to one braille character. Bit mapping follows the Unicode standard. FG color is the average RGB of all lit pixels; cell alpha is the max alpha in the block. Empty blocks are skipped. Resolution: 2x horizontal, 4x vertical (160x96 in 80x24). Best for wireframes, particles, monochrome text.

**Half-block encoding** (1x2 per cell, `▀`/`▄`): Each 1x2 pixel pair maps to one cell with two independent colors (FG=top, BG=bottom). Both-on uses `▀` with FG+BG; top-only uses `▀`; bottom-only uses `▄`; both-off is transparent. Resolution: 1x horizontal, 2x vertical (80x48 in 80x24). Best for color images, gradients.

`BitmapDrawable` implements `Drawable`, wrapping a `Bitmap` with an `EncodeMode` selector. Nil-safe (zero bounds, no-op draw). Bounds returns cell-space dimensions computed from pixel dimensions and encoding ratios.

## Roadmap: Foundation

These foundation iterations unblock the feature work below. Order matters — each builds on the last.

- **Iteration 9: Extended math** — `Dot`, `Cross` on Vec3. `Mat3`/`Mat4` with multiply, inverse, transpose. Perspective/orthographic projection matrices. Perlin/simplex noise. Bezier curves (quadratic, cubic). Degrees/radians helpers.
- **Iteration 10: Transform rotation and scale** — Add `Rotation` (float64 radians, 2D for now) and `Scale` (Vec3) to Transform. Hierarchical composition via matrices. Update render traversal to apply full transform chain.
- **Iteration 11: Asset loading** — Loader pattern for fonts (TTF via `golang.org/x/image/font`), images (PNG/JPEG via `image/png`, `image/jpeg`), 3D models (OBJ), and SVG. Resource cache keyed by path.
- **Iteration 12: Camera and projection** — Camera entity with position/target. View matrix. Orthographic and perspective projection. Viewport bounds and culling. World-to-screen coordinate pipeline.

## Roadmap: Features

These are the target features, built on top of the foundations above. Dependencies noted.

- **Text rendering** (needs: 8, 11) — Load Google Fonts TTFs, rasterize glyphs into bitmap buffer, render as braille/block cells. Text effects: typewriter, scramble-reveal, count up/down.
- **Particle systems** (needs: 8) — Point emitters, velocity/acceleration integration, attractor/repulsor effectors, particle pooling. Sub-cell rendering for pixel-precise particles. Text materialization from particles.
- **Trails** (needs: nothing, achievable now) — Post-process fade (`cell.Alpha *= decay`) instead of full clear. Per-layer trail intensity.
- **Physics: springs and effectors** (needs: 9) — Spring force `F = -kx - bv`, point effectors (attract/repel), drag. Verlet integration. No collision detection needed initially.
- **OBJ rendering** (needs: 8, 9, 10, 11, 12) — Load OBJ meshes, project vertices, rasterize wireframe or filled triangles into bitmap buffer. Basic vertex lighting (Phong). ASCII luminance mapping for shading.
- **SVG rendering** (needs: 8, 9, 11) — Parse SVG paths, rasterize bezier curves and fills into bitmap buffer.
- **PNG/image rendering** (needs: 8, 11) — Decode images, downsample to bitmap resolution, map to braille/half-block cells with color.
- **Scenes/slides** — Ordered scene list, transitions between scenes.
- **Scripting** — Lua bindings over the Go API.
- **Playback/recording** — VHS / asciinema integration.

## Package Structure

```
flicker/
  core/
    entity.go      // Entity, World, parent/child relationships
    transform.go   // Transform component (position via Vec3)
    drawable.go    // Drawable interface
    rect.go        // Rect drawable
    bitmap.go      // Bitmap pixel buffer, braille/half-block encoding, BitmapDrawable
    behavior.go    // Behavior component + UpdateBehaviors system
    canvas.go      // Cell, Canvas (2D cell buffer)
    blend.go       // BlendMode, ColorBlend, all Photoshop-style blend modes
    render.go      // Scene graph traversal, drawable → canvas
    layer.go       // Compositor, per-layer canvases, back-to-front compositing
  fmath/
    vec2.go        // Vec2 (X, Y float64), add, sub, scale, normalize, lerp
    vec3.go        // Vec3 (X, Y, Z float64), add, sub, scale, normalize, lerp
    interpolation.go // Lerp, InverseLerp, Remap, Clamp
    tween.go       // Tween (float64), TweenVec3 (Vec3) — stateful interpolators with easing
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
