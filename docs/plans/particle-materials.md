# Particle Materials: Dynamic Appearance

## Context

Particles in Flicker currently render as static blocks/braille cells. For dynamic, expressive particle effects, particles should **change appearance based on their state** — velocity, acceleration, age, position, or any combination.

Examples:
- **Idle particles**: Randomly flickering dots to show "alive" state
- **Moving particles**: Directional patterns (Braille dots forming lines/arrows in velocity direction)
- **Fast particles**: Motion blur / streaks
- **Young particles**: Small dots that grow over time
- **Proximity-based**: Change appearance near attractors

The current demo shows point cloud morphing ("GO" → "FLY"), but particles look like "shifting blocks" rather than dynamic particles with motion trails.

## Philosophy

**Materials are fragment shaders.** They receive a `Fragment` and return a `Cell`. To enable particle appearance effects, materials need access to entity state (velocity, age, position, etc.).

**Functional interface, maximum flexibility.** Instead of hardcoded "modes" or enums, use injected functions (`RuneEncoder`) that map fragment data to appearance. Users can compose, wrap, and define custom encoders.

**Fragment provides everything.** The `Fragment` struct should provide access to all data an encoder might need: time, position, world, entity. The encoder decides what to use.

## Core Design

### Everything is a Material

**No separate abstraction needed.** Materials are `func(Fragment) Cell` — they have full access to entity state via `Fragment.World` and `Fragment.Entity`.

**Pattern: Component-Based Appearance Materials**

A common pattern for particle effects is reading component state and computing appearance:

```go
func SomeMaterial() core.Material {
    return func(f core.Fragment) core.Cell {
        // Read component state
        body := f.World.Body(f.Entity)
        age := f.World.Age(f.Entity)
        transform := f.World.Transform(f.Entity)

        // Compute appearance based on state
        // ...

        cell := f.Cell
        cell.Rune = computedRune
        cell.FG = computedColor
        return cell
    }
}
```

**No assumptions about what data is needed.** Materials read from `Fragment` whatever they require:
- `f.Time` — for animation
- `f.World.Body(f.Entity)` — for velocity/acceleration
- `f.World.Age(f.Entity)` — for age/lifetime
- `f.World.Transform(f.Entity)` — for position
- Or any combination

Factory functions close over **configuration**, not data. Data is read fresh each frame.

### Critical Requirement: Nil Guards

**Materials MUST handle missing components gracefully.** Not every entity has Body, Age, Transform, etc.

**Bad (crashes):**
```go
func BadMaterial() core.Material {
    return func(f core.Fragment) core.Cell {
        body := f.World.Body(f.Entity)
        speed := body.Velocity.Length()  // CRASH if body is nil!
        // ...
    }
}
```

**Good (safe):**
```go
func GoodMaterial() core.Material {
    return func(f core.Fragment) core.Cell {
        body := f.World.Body(f.Entity)
        if body == nil {
            return f.Cell  // or return default appearance
        }

        speed := body.Velocity.Length()  // safe
        // ...
    }
}
```

**Pattern:** Check component, return early if nil, then proceed safely.

## Design Rationale: Why No RuneEncoder Abstraction?

**Early consideration:** Create a `RuneEncoder` type (`func(Fragment) rune`) as an intermediate abstraction between Fragment and Material.

**Why we rejected it:** Materials already have the signature `func(Fragment) Cell` with full access to entity state. Adding `RuneEncoder` would create two abstractions doing similar things:
- `RuneEncoder`: `func(Fragment) rune` — decides which rune
- `Material`: `func(Fragment) Cell` — decides full cell appearance

Since materials can access everything via `Fragment.World` and `Fragment.Entity`, there's no need for a separate abstraction. Users can write materials that:
- Only modify runes (keeping colors/alpha)
- Only modify colors (keeping runes/alpha)
- Modify everything based on complex logic

**The pattern we document instead:** "Component-based appearance materials" — materials that read component state and compute appearance. Factory functions return `Material` directly. Users compose them with `ComposeMaterials`.

**Benefits of this approach:**
- ✅ One abstraction (Material) instead of two
- ✅ No unnecessary wrapping (no `VelocityMaterial(encoder)`, just `BrailleDirectional()`)
- ✅ Materials have full power — can modify rune, color, alpha together
- ✅ Simpler mental model

**Key insight:** The existence of Fragment with World/Entity access eliminates the need for intermediate abstractions. Everything is just a Material.

### Fragment Extension

Current `Fragment` doesn't have `World` or `Entity`. Add them:

```go
type Fragment struct {
    X, Y         int      // local drawable coords
    ScreenX, ScreenY int  // screen coords
    Time         Time     // current time
    Cell         Cell     // current cell state
    Source       *Canvas  // source canvas for neighbor reads

    // NEW: enable access to any component
    World        *World
    Entity       Entity
}
```

**Impact:** All materials now have access to entity components. This is a breaking change but unlocks powerful shader-like capabilities.

## Materials

All particle materials are just `func(Fragment) Cell`. They read component state and return modified cells.

### VelocityColor

Changes the **color** based on velocity magnitude (speed → color gradient).

```go
type ColorGradient struct {
    MinSpeed float64
    MaxSpeed float64
    MinColor core.Color  // slow
    MaxColor core.Color  // fast
}

func VelocityColor(gradient ColorGradient) core.Material {
    return func(f core.Fragment) core.Cell {
        body := f.World.Body(f.Entity)
        if body == nil {
            return f.Cell  // nil guard!
        }

        speed := math.Sqrt(body.Velocity.X*body.Velocity.X + body.Velocity.Y*body.Velocity.Y)
        t := fmath.Clamp((speed - gradient.MinSpeed) / (gradient.MaxSpeed - gradient.MinSpeed), 0, 1)

        cell := f.Cell
        cell.FG = lerpColor(gradient.MinColor, gradient.MaxColor, t)
        return cell
    }
}
```

### Composition

Stack materials to combine effects:

```go
world.AddMaterial(particle, core.ComposeMaterials(
    particle.BrailleDirectional(),  // Material that sets rune
    particle.VelocityColor(particle.ColorGradient{
        MinSpeed: 0.0,
        MaxSpeed: 20.0,
        MinColor: core.Color{R: 100, G: 150, B: 255},  // blue = slow
        MaxColor: core.Color{R: 255, G: 100, B: 100},  // red = fast
    }),
))
```

## Material Helpers

Provide factory functions for common patterns. Users can define custom materials.

**All helpers return `core.Material` directly.** No intermediate abstractions.

### IdleAndMotion

Cycles through runes when idle (low velocity), switches to directional Braille when moving.

```go
func IdleAndMotion(idleRunes []rune, motionThreshold float64) core.Material {
    return func(f core.Fragment) core.Cell {
        body := f.World.Body(f.Entity)
        if body == nil {
            return f.Cell  // nil guard
        }

        speed := math.Sqrt(body.Velocity.X*body.Velocity.X + body.Velocity.Y*body.Velocity.Y)

        cell := f.Cell
        if speed < motionThreshold {
            // Idle: cycle through runes based on time
            idx := int(f.Time.Total * 4.0) % len(idleRunes)
            cell.Rune = idleRunes[idx]
        } else {
            // Motion: directional Braille
            angle := math.Atan2(body.Velocity.Y, body.Velocity.X)
            cell.Rune = brailleForAngle(angle)
        }

        return cell
    }
}
```

### BrailleDirectional

Maps velocity direction to one of 8 Braille patterns forming directional lines/arrows.

```go
func BrailleDirectional() core.Material {
    // 8 cardinal directions
    patterns := []rune{
        '⠤', // E  (horizontal right)
        '⠡', // SE (diagonal down-right)
        '⡇', // S  (vertical down)
        '⢇', // SW (diagonal down-left)
        '⠒', // W  (horizontal left)
        '⠊', // NW (diagonal up-left)
        '⡀', // N  (vertical up)
        '⠈', // NE (diagonal up-right)
    }

    return func(f core.Fragment) core.Cell {
        body := f.World.Body(f.Entity)
        cell := f.Cell

        if body == nil {
            cell.Rune = '·'  // nil guard: default rune
            return cell
        }

        // Map angle to 8 directions
        angle := math.Atan2(body.Velocity.Y, body.Velocity.X)
        if angle < 0 {
            angle += 2 * math.Pi
        }

        idx := int((angle + math.Pi/8) / (math.Pi / 4)) % 8
        cell.Rune = patterns[idx]
        return cell
    }
}
```

### SpeedStates

Multiple speed thresholds with different runes.

```go
func SpeedStates(thresholds []float64, runes []rune) core.Material {
    return func(f core.Fragment) core.Cell {
        body := f.World.Body(f.Entity)
        cell := f.Cell

        if body == nil {
            cell.Rune = ' '  // nil guard
            return cell
        }

        speed := math.Sqrt(body.Velocity.X*body.Velocity.X + body.Velocity.Y*body.Velocity.Y)

        for i, threshold := range thresholds {
            if speed < threshold {
                cell.Rune = runes[i]
                return cell
            }
        }

        cell.Rune = runes[len(runes)-1]
        return cell
    }
}
```

**Usage:**
```go
world.AddMaterial(particle, particle.SpeedStates(
    []float64{0.5, 2.0, 10.0},
    []rune{'·', '○', '◎', '●'},
))
```

### AgeBasedSize

Changes rune based on particle age (particles grow over time).

```go
func AgeBasedSize(ageThresholds []float64, runes []rune) core.Material {
    return func(f core.Fragment) core.Cell {
        age := f.World.Age(f.Entity)
        cell := f.Cell

        if age == nil {
            cell.Rune = runes[0]  // nil guard: default to first rune
            return cell
        }

        for i, threshold := range ageThresholds {
            if age.Age < threshold {
                cell.Rune = runes[i]
                return cell
            }
        }

        cell.Rune = runes[len(runes)-1]
        return cell
    }
}
```

### Custom Example: Velocity + Age

Show how users can combine multiple factors:

```go
func YoungAndFast() core.Material {
    return func(f core.Fragment) core.Cell {
        body := f.World.Body(f.Entity)
        age := f.World.Age(f.Entity)
        cell := f.Cell

        // Young particles (< 1 second) are always small
        if age != nil && age.Age < 1.0 {
            cell.Rune = '·'
            return cell
        }

        // Mature particles show velocity-based appearance
        if body != nil {
            speed := math.Sqrt(body.Velocity.X*body.Velocity.X + body.Velocity.Y*body.Velocity.Y)
            if speed > 10.0 {
                cell.Rune = '●'  // fast
                return cell
            }
            if speed > 1.0 {
                cell.Rune = '○'  // medium
                return cell
            }
        }

        cell.Rune = '∘'  // slow/idle
        return cell
    }
}
```

## Implementation Steps

### 1. Extend `Fragment` struct in `core/canvas.go`
- Add `World *World` field
- Add `Entity Entity` field
- Update all call sites that construct `Fragment` to pass world and entity

### 2. Update render pipeline to pass world and entity
- In `core/render.go`: `renderDrawable` should pass world and entity when constructing Fragment
- In `core/layer.go`: `Compositor.Composite` should pass world and entity when applying materials

### 3. Create `particle/materials.go`
- `VelocityColor(gradient ColorGradient) core.Material`
- `IdleAndMotion(idleRunes []rune, threshold float64) core.Material`
- `BrailleDirectional() core.Material`
- `SpeedStates(thresholds []float64, runes []rune) core.Material`
- `AgeBasedSize(ageThresholds []float64, runes []rune) core.Material`
- `ColorGradient` struct
- Helper: `lerpColor(a, b core.Color, t float64) core.Color`
- Helper: `brailleForAngle(angle float64) rune` (8-directional mapping)

### 4. Create `particle/materials_test.go`
- `TestVelocityColor`: verify color interpolation based on speed
- `TestIdleAndMotion`: verify idle/motion state switching
- `TestBrailleDirectional`: verify angle → rune mapping
- `TestSpeedStates`: verify multi-threshold behavior
- `TestAgeBasedSize`: verify age → rune progression
- All tests must verify nil guard behavior

### 5. Update demo (`cmd/flicker/main.go`)
- Replace static Braille drawable with composed materials:
  - `BrailleDirectional()` (rune based on velocity)
  - `VelocityColor()` with blue→red gradient (color based on speed)
- Add physics integration to particles (they should have actual motion, not just interpolation)
- Show dynamic appearance changes as particles morph from "GO" to "FLY"

### 6. Fix background transparency issue
- Investigate why Braille cells render with black BG instead of transparent
- Likely in `core/bitmap/braille.go` or rendering pipeline
- Cells with sparse content should have transparent BG (terminal default shows through)

### 7. Add golden test
- `TestParticleAppearance`: particles with different velocities, verify runes change correctly

### 8. Update `TODO.md`
- Document particle materials feature

## Background Transparency Issue

**Current problem:** Particles render with black background instead of terminal's actual background color.

**Expected:** Cells with sparse content (few Braille dots, partial blocks) should have **transparent background** so terminal's configured background shows through.

**Investigation needed:**
1. Check `core/bitmap/braille.go` — how does it set cell BG color?
2. Check rendering pipeline — is BG being explicitly set to black?
3. Solution: Cells should use `BGAlpha = 0` when background is empty/transparent

## API Examples

### Basic: Directional particles with color gradient

```go
for _, p := range particles {
    world.AddMaterial(p, core.ComposeMaterials(
        particle.BrailleDirectional(),  // rune based on velocity direction
        particle.VelocityColor(particle.ColorGradient{
            MinSpeed: 0.0,
            MaxSpeed: 20.0,
            MinColor: core.Color{R: 100, G: 150, B: 255},  // blue = slow
            MaxColor: core.Color{R: 255, G: 100, B: 100},  // red = fast
        }),
    ))
}
```

### Idle animation + motion

```go
world.AddMaterial(particle, particle.IdleAndMotion(
    []rune{'·', '•', '○', '●'},  // cycle through these when idle
    1.0,                          // motion threshold
))
```

### Speed-based appearance

```go
world.AddMaterial(particle, particle.SpeedStates(
    []float64{0.5, 2.0, 10.0},
    []rune{'·', '○', '◎', '●'},
))
```

### Custom: Age + velocity

```go
customMaterial := func(f core.Fragment) core.Cell {
    body := f.World.Body(f.Entity)
    age := f.World.Age(f.Entity)
    cell := f.Cell

    if age != nil && age.Age < 0.5 {
        cell.Rune = '·'  // newborn
        return cell
    }

    if body != nil && body.Velocity.Length() > 15.0 {
        cell.Rune = '●'  // fast
        return cell
    }

    cell.Rune = '○'  // default
    return cell
}

world.AddMaterial(particle, customMaterial)
```

## Tests

**All tests must verify nil guard behavior** — materials should handle missing components gracefully.

**Materials (`particle/materials_test.go`):**
- `TestVelocityColor`: Verify color interpolation at min/max/mid speeds
- `TestVelocityColorNoBody`: Fragment without Body, verify returns original cell (nil guard)
- `TestIdleAndMotion`: Mock fragments with different velocities, verify idle vs motion behavior
- `TestIdleAndMotionNoBody`: Fragment without Body, verify nil guard
- `TestBrailleDirectional`: Test all 8 cardinal directions, verify correct Braille runes
- `TestBrailleDirectionalNoBody`: Fragment without Body, verify nil guard
- `TestSpeedStates`: Test threshold boundaries, verify correct rune selection
- `TestSpeedStatesNoBody`: Fragment without Body, verify nil guard
- `TestAgeBasedSize`: Test age thresholds, verify rune progression
- `TestAgeBasedSizeNoAge`: Fragment without Age, verify nil guard

**Golden (`golden_test.go`):**
- `TestParticleAppearance`: Spawn particles with different velocities, apply materials, capture output

## What NOT to Build

- No hardcoded "modes" or enums (use injected functions instead)
- No separate "idle config" and "motion config" (encoder handles all states)
- No assumptions about what data is needed (encoder reads from Fragment)
- No complex state machines (just pure functions)

## Verification

```bash
go test ./particle/...   # material and encoder tests
go test ./core/...       # Fragment extension tests
go test ./...            # all tests including golden
go build ./cmd/flicker   # demo builds and runs
```

Then: `git add <files>`, `make verify`, `git commit`, `git push`.

## Files

| File | Purpose | New/Modified |
|------|---------|-------------|
| `core/canvas.go` | Add World and Entity to Fragment | Modified |
| `core/render.go` | Pass world and entity when constructing Fragment | Modified |
| `core/layer.go` | Pass world and entity in Compositor | Modified |
| `particle/materials.go` | Material helpers (VelocityColor, BrailleDirectional, IdleAndMotion, SpeedStates, AgeBasedSize) | New |
| `particle/materials_test.go` | Material tests with nil guard verification | New |
| `cmd/flicker/main.go` | Demo with dynamic particle appearance | Modified |
| `golden_test.go` | Particle appearance golden test | Modified |
| `TODO.md` | Mark feature complete | Modified |
