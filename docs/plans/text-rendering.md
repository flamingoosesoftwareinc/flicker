# Iteration 13: Text Rendering

## Context

Iterations 1-12 built the full rendering pipeline: entities, transforms, bitmaps, compositing, blend modes, materials, inverse-mapped drawables, and orthographic camera. All visual content is procedural. Text rendering is the most broadly useful feature for motion graphics — titles, credits, data overlays, typographic animation.

Oxanium (Google Fonts, variable weight) is the test font at `/workspace/Oxanium/`.

## Scope

**Include:**
- TTF font loading via `golang.org/x/image/font/sfnt`
- Glyph rasterization via `golang.org/x/image/vector` (analytical anti-aliasing)
- Single-line text layout with advance widths
- SDF computation from rasterized bitmap (8SSEDT algorithm)
- SDF query: point distance and gradient direction
- Fix `inverseRenderer` to emit local drawable coordinates (prerequisite for SDF materials)
- Demo: Oxanium text with SDF threshold materialization effect
- Golden test for static text rendering

**Defer:**
- Multi-line text / line wrapping
- Kerning pair adjustments
- Variable font axis interpolation (weight animation)
- Text effects beyond threshold materialization (typewriter, scramble — future iteration)
- Analytical SDF package (separate roadmap item, see TODO.md note)

## New Dependency

`golang.org/x/image` — provides `font/sfnt` (TTF parsing) and `vector` (2D rasterization). Single module, pure Go, no CGO. Added via `go get golang.org/x/image`.

## Prerequisite Fix: Local Coordinates in inverseRenderer

The `inverseRenderer` in `core/bitmap/renderer.go` currently passes screen coordinates for both `(dx, dy)` and `(sx, sy)` in its emit call:

```go
emit(sx, sy, sx, sy, cell)  // BUG: dx,dy should be local coords, not screen
```

The emit signature is `emit(dx, dy, sx, sy int, cell)` where `(dx, dy)` should be drawable-local coordinates and `(sx, sy)` are screen coordinates. Materials receive `dx, dy` as `Fragment.X, Fragment.Y`, so they currently get screen coords instead of local coords. This breaks any material that needs to index into entity-local data (like an SDF).

**Fix:** Change the sampler signature to return local coordinates:

```go
// Before
sample func(inv [4]float64, tx, ty float64, sx, sy int) (core.Cell, bool)

// After
sample func(inv [4]float64, tx, ty float64, sx, sy int) (lx, ly int, cell core.Cell, ok bool)
```

Each sampler already computes local coords (`px, py` for FullBlock/BGBlock; `topPX, topPY` for HalfBlock). Return them. In `inverseRenderer`:

```go
lx, ly, cell, ok := sample(inv, tx, ty, sx, sy)
emit(lx, ly, sx, sy, cell)
```

For HalfBlock, `ly` is the cell row (not pixel row). The material author knows the encoding and maps accordingly: `sdf.At(lx, ly*2)` for top pixel, `sdf.At(lx, ly*2+1)` for bottom.

## Package Structure

```
asset/
  font.go        // Font type, LoadFont, metrics access
  font_test.go
  text.go        // RasterizeText: string + font + size + color → *bitmap.Bitmap
  text_test.go
core/bitmap/
  sdf.go         // SDF struct, ComputeSDF (8SSEDT), At(), Gradient()
  sdf_test.go
```

`asset/` handles file I/O and processing (loading fonts, rasterizing text — same pattern as `image.go`). `core/bitmap/` hosts the SDF since it's a bitmap companion type used at render time in materials.

## Types / API

### Font (`asset/font.go`)

```go
type Font struct { /* wraps sfnt.Font + reusable sfnt.Buffer */ }

type FontMetrics struct {
    Ascent  float64 // pixels above baseline
    Descent float64 // pixels below baseline (positive value)
    Height  float64 // Ascent + Descent
}

func LoadFont(path string) (*Font, error)
func (c *Cache) GetOrLoadFont(path string) (*Font, error)
func (f *Font) Metrics(size float64) FontMetrics
```

`LoadFont` reads the file, calls `sfnt.Parse`. `size` is pixels-per-em. Wraps a `sfnt.Buffer` for allocation reuse across glyph queries.

### Text Rasterization (`asset/text.go`)

```go
type TextOptions struct {
    Font  *Font
    Size  float64    // pixels per em
    Color core.Color // text fill color
}

func RasterizeText(text string, opts TextOptions) *bitmap.Bitmap
```

Pipeline:
1. Query `FontMetrics` at target size for bitmap height (ascent + descent, rounded up)
2. Walk runes: `GlyphIndex` → `GlyphAdvance` to compute total width
3. Create `vector.Rasterizer` at computed dimensions
4. Walk runes again: `LoadGlyph` → feed `Segments` into rasterizer with accumulated x-offset, Y-flipped (`pixelY = ascent - fontY/64`)
5. `Rasterizer.Draw` → `image.Alpha` mask
6. Convert mask to `bitmap.Bitmap`: each pixel gets `opts.Color` with alpha from mask

Returns a `*bitmap.Bitmap` ready for any encoding (HalfBlock, FullBlock, Braille, BGBlock).

### Coordinate Transform (font → bitmap)

Font coordinates: Y up, origin at baseline-left, units in `fixed.Int26_6` (divide by 64 for pixels).
Bitmap coordinates: Y down, origin at top-left.
Transform: `pixelX = fontX / 64.0 + glyphOffset`, `pixelY = ascent - fontY / 64.0`.

### SDF (`core/bitmap/sdf.go`)

```go
type SDF struct {
    Width, Height int
    Dist          []float64 // row-major; positive = outside, negative = inside
    MaxDist       float64
}

func ComputeSDF(bm *Bitmap, maxDist float64) *SDF
func (s *SDF) At(x, y int) float64        // bounds-checked, OOB returns MaxDist
func (s *SDF) Gradient(x, y int) fmath.Vec2 // central-difference approximation
```

**Convention:** `Dist > 0` = outside glyph, `Dist < 0` = inside glyph, `Dist ≈ 0` = on edge.

**Algorithm (8SSEDT):**

Computes two unsigned distance fields and combines them:
- `dOut[p]` = distance from outside pixel `p` to nearest inside pixel (seeds: all inside pixels)
- `dIn[p]` = distance from inside pixel `p` to nearest outside pixel (seeds: all outside pixels)
- `SDF[p] = dOut[p] - dIn[p]` (positive outside, negative inside)

Each field computed via two-pass sequential sweep:
1. Forward pass (top-left → bottom-right): propagate minimum distance from 4 neighbors (left, top-left, top, top-right)
2. Backward pass (bottom-right → top-left): propagate from 4 neighbors (right, bottom-right, bottom, bottom-left)

O(n) in pixel count. `maxDist` clamps the result — distances beyond maxDist aren't computed, saving work and bounding the output range.

**Gradient** uses central differences: `∇SDF(x,y) = (SDF(x+1,y) - SDF(x-1,y), SDF(x,y+1) - SDF(x,y-1)) / 2`. The gradient direction tells you edge orientation: `gradient.Y < 0` means the nearest edge is above (top edge), etc.

## Implementation Steps

### 1. Add `golang.org/x/image` dependency
`go get golang.org/x/image`

### 2. Fix `inverseRenderer` local coordinates (`core/bitmap/renderer.go`)
- Change sampler signature to return `(lx, ly int, cell core.Cell, ok bool)`
- Update `inverseRenderer` to pass `lx, ly` as `dx, dy` in emit
- Update all four samplers: `fullblock.go`, `halfblock.go`, `bgblock.go`, `braille.go`
- Run `go test ./...` to verify no regressions (existing golden tests validate output)

### 3. Create `asset/font.go` + `asset/font_test.go`
- `Font` struct wrapping `sfnt.Font` + buffer
- `LoadFont`, `GetOrLoadFont`, `Metrics`
- Tests: load Oxanium, verify metrics (ascent > 0, descent > 0), nonexistent path → error, cache roundtrip

### 4. Create `asset/text.go` + `asset/text_test.go`
- `TextOptions` struct, `RasterizeText` function
- Two-pass layout: measure width, then rasterize
- Tests: render "I" at size 32 (verify dimensions, non-zero alpha along stroke), empty string → nil or zero-size bitmap, multi-character string width ≈ sum of advances

### 5. Create `core/bitmap/sdf.go` + `core/bitmap/sdf_test.go`
- `SDF` struct, `ComputeSDF`, `At`, `Gradient`
- 8SSEDT implementation
- Tests: 10×10 bitmap with 6×6 filled rect centered — verify interior negative, exterior positive, edge ≈ 0, gradient directions at edges, OOB returns maxDist

### 6. Add golden test (`golden_test.go`)
- `TestTextRendering`: render "Hi" in Oxanium at fixed size, HalfBlock encoding, compare against golden file

### 7. Update demo (`cmd/flicker/main.go`)
- Load Oxanium Bold via `asset.LoadFont`
- `RasterizeText("FLICKER", ...)` at a size appropriate for the terminal
- `ComputeSDF` on the text bitmap
- Entity with HalfBlock drawable, positioned center-screen
- Material: SDF threshold materialization — threshold sweeps from most-negative → 0 over ~3s, revealing text from skeleton outward
- After reveal completes, text remains fully visible under existing camera movement

### 8. Regenerate golden files
- `go test . -update` for any changed golden output from the inverseRenderer fix

### 9. Update `TODO.md`
- Mark Iteration 13 complete
- Add roadmap note for analytical SDF package (2D primitives from iquilezles.org/articles/distfunctions2d/)

## What NOT to Build

- No `io.Reader` overload for LoadFont — file paths only (matches image.go, obj.go pattern)
- No text alignment / centering — caller positions via Transform
- No multi-line layout or line breaking
- No kerning pair adjustments (add when visually needed)
- No glyph caching within Font — `RasterizeText` is called once, result cached at caller level via `asset.Cache` if needed
- No variable font axis support
- No sub-pixel SDF interpolation (integer pixel queries only)
- No analytical SDF primitives (separate future package)

## Tests

**Font (`asset/font_test.go`):** Load Oxanium Regular, verify metrics at size 32 (ascent > 0, descent > 0, height = ascent + descent). Nonexistent path → error. Cache hit returns same pointer.

**Text (`asset/text_test.go`):** Rasterize "I" at size 48 — verify bitmap width ≈ advance, height = ceil(ascent + descent), pixels along vertical stroke have alpha > 0. Rasterize "AB" — width > single glyph width. Empty string → zero-width bitmap.

**SDF (`core/bitmap/sdf_test.go`):** Create 10×10 bitmap with filled 6×6 rect at (2,2)-(7,7). Verify: center pixel has negative distance, corner pixel (0,0) has positive distance, edge pixel (2,2) ≈ 0. Gradient at top-center edge points upward (Y < 0). At() returns maxDist for OOB. ComputeSDF with maxDist=3 clamps exterior values.

**Golden (`golden_test.go`):** `TestTextRendering` — "Hi" in Oxanium at fixed size on 40×10 SimScreen, HalfBlock encoding.

## Verification

```bash
go test ./asset/...              # font + text tests
go test ./core/bitmap/...        # SDF tests
go test ./...                    # all tests including golden
go build ./cmd/flicker           # demo builds
```

Then: `git add <files>`, `make verify`, `git commit`, `git push`.

## Files

| File | Purpose | New/Modified |
|------|---------|-------------|
| `core/bitmap/renderer.go` | Fix emit to pass local coords | Modified |
| `core/bitmap/fullblock.go` | Return local coords from sampler | Modified |
| `core/bitmap/halfblock.go` | Return local coords from sampler | Modified |
| `core/bitmap/bgblock.go` | Return local coords from sampler | Modified |
| `core/bitmap/braille.go` | Return local coords from sampler | Modified |
| `asset/font.go` | Font type, LoadFont, metrics | New |
| `asset/font_test.go` | Font loading tests | New |
| `asset/text.go` | RasterizeText (layout + rasterize) | New |
| `asset/text_test.go` | Text rasterization tests | New |
| `core/bitmap/sdf.go` | SDF computation (8SSEDT) | New |
| `core/bitmap/sdf_test.go` | SDF tests | New |
| `golden_test.go` | Text golden test | Modified |
| `cmd/flicker/main.go` | Text entity + materialization demo | Modified |
| `TODO.md` | Mark complete, add SDF roadmap note | Modified |
| `go.mod` / `go.sum` | Add golang.org/x/image | Modified |
