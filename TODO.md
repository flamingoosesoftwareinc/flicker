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

High-resolution `Bitmap` pixel buffer in `core/bitmap` with concrete drawable types that each implement `core.Drawable` directly — no enum dispatch. Flat row-major `[]Color` and `[]float64` arrays for cache locality.

**Encoding types** (each a concrete struct in `core/bitmap`):
- `Braille` — 2x4 dots per cell (U+2800-U+28FF), averaged FG color, max alpha. Best for wireframes, particles.
- `HalfBlock` — 1x2 per cell (`▀`/`▄`), two independent colors (FG=top, BG=bottom). Best for color images.
- `FullBlock` — 1:1 pixel-to-cell (`█`), single FG color. Simplest encoding.
- `BGBlock` — 1:1 pixel-to-cell (space with BG color), FGAlpha=0 for compositing transparency.
- `Rect` — convenience type delegating to `HalfBlock`.

All types are nil-safe. `inverseRenderer` is shared by HalfBlock/FullBlock/BGBlock/Braille for gap-free rendering under zoom and rotation.

## Iteration 9: Extended Math (complete)

`Dot` and `Cross` on Vec3. `Mat3` (row-major 3x3) with identity, multiply, transpose, determinant, inverse, apply, and constructors for translate, rotate, scale — the core type for 2D homogeneous transforms. `Mat4` (row-major 4x4) with the same operations plus orthographic and perspective projection matrices. 2D Perlin noise (`Noise2D`) with classic improved permutation table. Quadratic and cubic Bezier curve evaluation. Degree/radian conversion helpers.

## Iteration 10: Transform Rotation & Scale (complete)

`Transform` gains `Rotation` (float64 radians, 2D rotation around Z) and `Scale` (Vec3, where `{0,0,0}` means zero — no magic defaults). `LocalMatrix()` computes the TRS matrix (translate * rotate * scale). Render traversal (`renderEntity`) replaced position accumulation (`ox, oy, oz`) with hierarchical `Mat3` multiplication — parent transforms compose correctly through the scene graph. All call sites updated to set `Scale: {1,1,1}` explicitly. Golden tests regenerated.

### Inverse-Mapped Braille Rendering

`Drawable.Renderer()` returns a `RenderFunc` strategy. Rect and half-block use forward mapping; braille uses inverse mapping that samples 2×4 dot positions through the inverse world matrix for smooth partial-dot edges at rotated angles.

### Independent FG/BG Alpha

`Cell` originally had a single `Alpha` governing both FG and BG. Braille cells need FG-opaque (dots visible) with BG-transparent (show whatever is underneath). A `BGTransparent bool` was tried first but was really just a boolean encoding of "BG alpha is 0." Since Cell already has separate FG and BG colors, separate alphas are the natural model.

`Cell.Alpha` was replaced with `Cell.FGAlpha` and `Cell.BGAlpha`. `BlendCell` blends FG and BG independently — `blend(dst.BG, src.BG, src.BGAlpha=0)` naturally preserves the destination BG with no special cases. Half-block encoding also benefits: top and bottom pixel alphas are now independent (`FGAlpha=topA`, `BGAlpha=botA`) instead of approximated as `max(topA, botA)`.

## Known Limitations

### Braille BG shows destination BG, not destination FG

When a braille entity overlaps another entity, only the destination's BG color shows through between the braille dots. The destination's FG color (carried by the rune glyph) is replaced by the braille rune. This is a fundamental constraint of terminal cells: one rune, one FG, one BG per cell. If the underlying entity's visual identity comes primarily from its FG (e.g. a Rect with `▓` and a vivid FG but dark BG), overlapping braille will appear to "erase" that color. Use saturated BG colors on entities that need to remain visible under braille overlays.

## Roadmap: Foundation

These foundation iterations unblock the feature work below. Order matters — each builds on the last.

- **Iteration 11: Asset loading (complete)** — OBJ mesh loader, PNG/JPEG image loader with downsampling, wireframe rasterizer, resource cache. All in `asset/` package.
- **Iteration 12: Orthographic Camera (complete)** — `Camera` component with `Zoom` (zero-value defaults to 1.0). `ViewMatrix()` computes `Translate(screenCenter) × Scale(zoom) × Rotate(-rotation) × Translate(-pos)`, centering the camera's world position on screen. `World` holds `cameras` map and `activeCamera` entity. `viewMatrix()` helper in `render.go` returns identity when no active camera — fully backward compatible. Both `Render()` and `Compositor.Composite()` use the view matrix as the initial parent transform. Viewport culling deferred — `Canvas.Set` already clips silently, negligible cost for ~10-50 entities. All forward-mapped drawables (HalfBlock, FullBlock, BGBlock) switched to inverse mapping via shared `inverseRenderer` — eliminates scan-line gaps under non-integer zoom. Demo: gentle circular pan + pronounced zoom pulse (0.7×–1.3×).
- **Iteration 13: Text Rendering (complete)** — TTF font loading via `golang.org/x/image/font/sfnt`, glyph rasterization via `golang.org/x/image/vector` with analytical anti-aliasing. Single-line text layout with advance widths. SDF computation from rasterized bitmap using 8SSEDT algorithm. `inverseRenderer` fixed to emit local drawable coordinates (prerequisite for SDF materials). `asset/font.go`: `Font` type wrapping `sfnt.Font`, `LoadFont`, `GetOrLoadFont`, `Metrics`. `asset/text.go`: `TextLayout` struct with per-glyph positions (`Glyphs []Glyph`), `GlyphAt(x,y)`, `SplitGlyphs()` — foundation for per-character effects. `core/bitmap/sdf.go`: `ComputeSDF` (8SSEDT), `At()`, `Gradient()`. Encoding-aware SDF threshold materials (`HalfBlockThreshold`, `BrailleThreshold`) as free functions. `core/entity.go`: `ComposeMaterials` for stacking materials on any entity. `textfx/` package: reusable text effects (typewriter, wave, staggered fade). Demo: "FLICKER" text with SDF threshold materialization, plus typewriter/wave/fade effects. Golden test for static text rendering.

## Roadmap: Features

These are the target features, built on top of the foundations above. Dependencies noted.

- **Text rendering (complete)** — Load Google Fonts TTFs, rasterize glyphs into bitmap buffer, render as braille/block cells. Text effects: typewriter, scramble-reveal, count up/down.
- **Text: multi-line & kerning** — Line wrapping, multi-line layout with line height control. Kerning pair adjustments for professional typography. Variable font axis interpolation (weight animation).
- **Analytical SDF primitives (complete)** — `sdf/` package with 2D signed distance functions from [iquilezles.org/articles/distfunctions2d](https://iquilezles.org/articles/distfunctions2d/). Primitives: Circle, Box, RoundedBox, Segment, Triangle, EquilateralTriangle, Rhombus, Pentagon, Hexagon, Ellipse, Arc, Pie. Operations: Union, Subtract, Intersect, SmoothUnion, SmoothSubtract, SmoothIntersect. Pure functions taking `fmath.Vec2`, returning distance. Separate from bitmap-based SDF in `core/bitmap/sdf.go`.
- **Particle systems (complete)** — `physics/` package: generic physics behaviors (EulerIntegration, VerletIntegration, Attractor, Repulsor, Drag, Gravity, Turbulence, Spring) that work on any entity with Body + Transform. `particle/` package: particle-specific helpers (BitmapToCloud, DistributeTargets, InterpolateToTarget, Emit, AgeAndDespawn). `Body` component (velocity, acceleration) and `Age` component (age, lifetime) in `core/`. Point cloud morphing: bitmap → point cloud → particle interpolation. Demo: "GO" → "FLY" text morph.
- **Particle materials** — Dynamic particle appearance based on component state. Extend `Fragment` with `World` and `Entity` fields. Material helpers: `BrailleDirectional()` (velocity → directional Braille patterns), `VelocityColor()` (speed → color gradient), `IdleAndMotion()` (idle animation + motion), `SpeedStates()`, `AgeBasedSize()`. Pattern: component-based appearance materials read entity state from Fragment and return modified cells. All materials must guard against nil components. Fix background transparency issue (particles render with black BG instead of terminal default). Plan at `docs/plans/particle-materials.md`.
- **Trails** — Post-process fade (`cell.Alpha *= decay`) instead of full clear. Per-layer trail intensity.
- **Physics: springs and effectors** — Spring force `F = -kx - bv`, point effectors (attract/repel), drag. Verlet integration. No collision detection needed initially.
- **SVG rendering** — Parse SVG paths, rasterize bezier curves and fills into bitmap buffer.
- **Scenes/slides** — Ordered scene list, transitions between scenes.
- **Scripting** — Lua bindings over the Go API.
- **Playback/recording** — VHS / asciinema integration.

## Package Structure

```
flicker/
  core/
    entity.go      // Entity, World, parent/child relationships, ComposeMaterials, Body/Age components, Despawn
    camera.go      // Camera component, ViewMatrix (orthographic view transform)
    transform.go   // Transform component (position, rotation, scale) with LocalMatrix()
    drawable.go    // Drawable interface, RenderFunc
    behavior.go    // Behavior component + UpdateBehaviors system
    canvas.go      // Cell, Color, Fragment, Canvas (2D cell buffer)
    blend.go       // BlendMode, ColorBlend, all Photoshop-style blend modes
    render.go      // Scene graph traversal, drawable → canvas
    layer.go       // Compositor, per-layer canvases, back-to-front compositing
  core/bitmap/
    bitmap.go      // Bitmap pixel buffer, New(), Set/Get/SetDot/Clear/Line
    braille.go     // Braille drawable (2x4 dots per cell)
    halfblock.go   // HalfBlock drawable (1x2 per cell)
    fullblock.go   // FullBlock drawable (1:1 pixel-to-cell)
    bgblock.go     // BGBlock drawable (BG-only encoding)
    rect.go        // Rect drawable (delegates to HalfBlock)
    renderer.go    // inverseRenderer shared by all drawable types
    sdf.go         // SDF computation (8SSEDT), At(), Gradient(), threshold materials
  fmath/
    vec2.go        // Vec2 (X, Y float64), add, sub, scale, normalize, lerp
    vec3.go        // Vec3 (X, Y, Z float64), add, sub, scale, normalize, lerp, dot, cross
    angle.go       // DegToRad, RadToDeg
    mat3.go        // Mat3 (row-major 3x3), 2D homogeneous transforms
    mat4.go        // Mat4 (row-major 4x4), 3D transforms, ortho/perspective projection
    noise.go       // Noise2D (2D Perlin noise)
    bezier.go      // BezierQuadratic, BezierCubic
    interpolation.go // Lerp, InverseLerp, Remap, Clamp
    tween.go       // Tween (float64), TweenVec3 (Vec3) — stateful interpolators with easing
    easing.go      // Easing functions: linear, quad, cubic, elastic, bounce (all func(t float64) float64)
    wave.go        // Wave functions: saw, sine, triangle, square, pulse (period 1.0)
  asset/
    obj.go         // OBJ mesh loader
    image.go       // PNG/JPEG image loader → bitmap.Bitmap
    font.go        // Font type, LoadFont, GetOrLoadFont, Metrics
    text.go        // TextLayout, RasterizeText (layout + rasterize) → TextLayout
    rasterize.go   // Wireframe rasterizer (mesh → bitmap)
    cache.go       // Resource cache keyed by path
  textfx/
    encoding.go    // Encoding enum (HalfBlock, Braille, FullBlock), glyphAtForEncoding helper
    typewriter.go  // TypewriterMaterial (left-to-right reveal), TypewriterBehavior
    fade.go        // StaggeredFadeMaterial (per-character alpha fade with delay)
    wave.go        // Wave (multi-entity vertical oscillation with phase offsets)
  sdf/
    sdf.go         // 2D signed distance functions: primitives (Circle, Box, Triangle, polygons, Ellipse, Arc, Pie), operations (Union, Subtract, Intersect, smooth variants)
    sdf_test.go
    example_test.go
  physics/
    integrate.go   // EulerIntegration, VerletIntegration (Verlet maintains state in closure)
    forces.go      // Attractor, Repulsor, Drag, Gravity, Turbulence
    spring.go      // Spring (Hooke's law with damping)
    integrate_test.go
    forces_test.go
    spring_test.go
  particle/
    target.go      // InterpolateToTarget (deterministic motion)
    lifecycle.go   // AgeAndDespawn
    emit.go        // Emit (particle spawning)
    cloud.go       // BitmapToCloud, DistributeTargets (point cloud helpers)
    target_test.go
    lifecycle_test.go
    emit_test.go
    cloud_test.go
  terminal/
    screen.go      // Screen interface, TcellScreen (tcell backend)
    simscreen.go   // SimScreen (in-memory backend for testing)
  cmd/
    flicker/
      main.go      // Wire everything, run the tick loop
  golden_test.go   // Integration golden tests
  testdata/        // Golden files
```

`fmath` depends on nothing. `sdf` depends on `fmath`. `core` depends on `fmath`. `core/bitmap` depends on `core` and `fmath`. `asset` depends on `core/bitmap`, `core`, and `fmath`. `textfx` depends on `core`, `core/bitmap`, `asset`, and `fmath`. `physics` depends on `core` and `fmath`. `particle` depends on `core`, `core/bitmap`, and `fmath`. `terminal` depends on `core` and `tcell`. `cmd` depends on all.
