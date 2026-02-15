# Particle Systems and Physics

## Context

Flicker is a terminal motion graphics engine. Particles enable a wide range of motion graphics effects: text materialization, point cloud interpolation, trails, attractor fields, and organic movement. The existing ECS (Entity-Component-System) architecture with `Behavior` closures provides the foundation for a purely functional particle/physics system.

## Philosophy

**Physics behaviors are generic and reusable.** They operate on any entity with the right components — not just particles. A "particle" is simply an entity with `Body` + `Age` components, but physics can apply to text, shapes, camera, or anything else.

**Separation of concerns:**
- Visual representation: existing `Drawable` + `Transform` + `Material` components
- Physics state: `Body` component (velocity, acceleration)
- Lifecycle: `Age` component (age, lifetime)
- Physics behaviors: `physics/` package (integration, forces, springs)
- Particle patterns: `particle/` package (point clouds, emission)

**Functional interface:** All behaviors are pure functions that return `core.Behavior`. This enables:
- Composition (multiple behaviors per entity)
- Scripting (Lua can call behavior factories or define custom behaviors)
- Pluggable simulation (Euler vs Verlet, custom integrators)
- Testability (each behavior in isolation)
- Reusability (physics works on any entity, not just particles)

## Scope

**Include:**
- `physics/` package with integration and force behaviors
- `particle/` package with particle-specific helpers
- `Body` component (velocity, acceleration) in `core/`
- `Age` component (age, lifetime) in `core/`
- Integration behaviors: `EulerIntegration()`, `VerletIntegration()`
- Force behaviors: `Attractor()`, `Repulsor()`, `Drag()`, `Gravity()`, `Turbulence()`
- Target behaviors: `InterpolateToTarget()`, `Spring()`
- Lifecycle behavior: `AgeAndDespawn()`
- Emission behavior: `Emit()`
- Point cloud helpers: `BitmapToCloud()`, `DistributeTargets()`
- Unit tests for each behavior
- Demo: point cloud interpolation (text A → text B)
- Golden test for deterministic particle motion

**Defer:**
- Particle pooling (spawn/despawn optimization)
- Collision detection
- Spatial partitioning (quadtree/grid for large particle counts)
- Particle sorting (depth-based render order)
- Advanced effectors (vortex, wind fields, vector field following)
- Particle trails as separate entities (can be done with materials for now)

## Package Structure

```
physics/
  integrate.go       // EulerIntegration, VerletIntegration
  forces.go          // Attractor, Repulsor, Drag, Gravity, Turbulence
  spring.go          // Spring
  integrate_test.go
  forces_test.go
  spring_test.go

particle/
  target.go          // InterpolateToTarget
  lifecycle.go       // AgeAndDespawn
  emit.go            // Emit behavior
  cloud.go           // BitmapToCloud, DistributeTargets helpers
  target_test.go
  lifecycle_test.go
  emit_test.go
  cloud_test.go
```

## Component Design

### Body (`core/entity.go`)

```go
type Body struct {
    Velocity     fmath.Vec2
    Acceleration fmath.Vec2
}
```

Pure physics state. Used by integration and force behaviors. Works on **any entity** — particles, text, shapes, camera, etc.

**Component accessors in `core/entity.go`:**
```go
func (w *World) AddBody(e Entity, b *Body)
func (w *World) Body(e Entity) *Body
```

### Age (`core/entity.go`)

```go
type Age struct {
    Age      float64  // seconds since spawn
    Lifetime float64  // 0 = infinite
}
```

Lifecycle state. Used by `AgeAndDespawn()` behavior. Optional — entities without `Age` live forever.

**Component accessors in `core/entity.go`:**
```go
func (w *World) AddAge(e Entity, a *Age)
func (w *World) Age(e Entity) *Age
```

### Despawn (`core/entity.go`)

```go
func (w *World) Despawn(e Entity)
```

Removes entity from all component maps and parent/child relationships. Required by `AgeAndDespawn()`.

## Physics Behaviors (`physics/`)

All physics behaviors operate on entities with `Body` + `Transform` components. They work on **any entity**, not just particles.

### Integration (`physics/integrate.go`)

```go
func EulerIntegration() core.Behavior
```
**Algorithm:**
```
pos += vel * dt
vel += acc * dt
acc = 0  // reset for next frame
```

```go
func VerletIntegration() core.Behavior
```
**Algorithm (maintains PrevPosition in closure):**
```
prevPositions := make(map[Entity]Vec2)  // closed over
newPos = 2*pos - prevPos + acc*dt²
prevPos = pos
pos = newPos
vel = (pos - prevPos) / dt  // compute velocity for other behaviors
acc = 0
```

**Note:** Verlet maintains previous positions in its own state (closure). No component pollution.

### Forces (`physics/forces.go`)

```go
func Attractor(center fmath.Vec2, strength float64) core.Behavior
```
Applies force towards `center` with inverse-square falloff: `F = strength / dist²`.

```go
func Repulsor(center fmath.Vec2, strength float64) core.Behavior
```
Applies force away from `center` with inverse-square falloff: `F = -strength / dist²`.

```go
func Drag(coefficient float64) core.Behavior
```
Applies drag force opposing velocity: `vel *= (1 - coefficient * dt)`.

```go
func Gravity(force fmath.Vec2) core.Behavior
```
Applies constant acceleration: `acc += force`.

```go
func Turbulence(scale, strength float64) core.Behavior
```
Applies Perlin noise-based force at entity position. `scale` controls noise frequency, `strength` controls force magnitude. Uses `fmath.Noise2D`.

### Spring (`physics/spring.go`)

```go
func Spring(anchor fmath.Vec2, k, damping float64) core.Behavior
```
Applies spring force: `F = -k*(pos - anchor) - damping*vel`. Classic Hooke's law with damping. Works well with Verlet integration.

## Particle Behaviors (`particle/`)

Particle-specific helpers for common particle patterns. These are **not** physics — they're higher-level patterns built on top of physics.

### Target (`particle/target.go`)

```go
func InterpolateToTarget(target fmath.Vec2, speed float64) core.Behavior
```
Moves entity towards `target` at `speed` units/second. No physics — directly modifies `Transform.Position`. Stops when within epsilon distance. Use for deterministic motion (point cloud morphing).

### Lifecycle (`particle/lifecycle.go`)

```go
func AgeAndDespawn() core.Behavior
```
Increments `Age.Age` by `dt`. If `Lifetime > 0` and `Age >= Lifetime`, despawns the entity via `world.Despawn(e)`. Requires `Age` component.

### Emission (`particle/emit.go`)

```go
func Emit(rate float64, spawnFunc func(*core.World) core.Entity) core.Behavior
```

Spawns entities at `rate` entities/second. `spawnFunc` defines what to spawn (particles, text, shapes, etc.). Attached to an emitter entity (doesn't need physics itself). Maintains accumulated time in closure.

**Example:**
```go
emitter := world.Spawn()
world.AddBehavior(emitter, particle.Emit(10.0, func(w *core.World) core.Entity {
    p := w.Spawn()
    w.AddTransform(p, &core.Transform{Position: emitterPos})
    w.AddBody(p, &core.Body{})
    w.AddAge(p, &core.Age{Lifetime: 5.0})
    w.AddDrawable(p, &bitmap.FullBlock{Bitmap: singlePixel})
    w.AddBehavior(p, physics.EulerIntegration())
    w.AddBehavior(p, physics.Gravity(fmath.Vec2{X: 0, Y: 9.8}))
    w.AddBehavior(p, particle.AgeAndDespawn())
    return p
}))
```

### Point Cloud Helpers (`particle/cloud.go`)

```go
func BitmapToCloud(bm *bitmap.Bitmap) []fmath.Vec2
```
Samples all non-transparent pixels (alpha > 0) from bitmap. Returns their positions as `[]fmath.Vec2`.

```go
func DistributeTargets(entities []core.Entity, cloud []fmath.Vec2, speed float64, world *core.World)
```
Assigns target positions from `cloud` to `entities`. Simple strategy: round-robin assignment (`entities[i]` gets `cloud[i % len(cloud)]`). Adds `InterpolateToTarget` behavior to each entity.

**Note:** For more advanced distribution (closest point, random, etc.), user can implement custom logic.

## Implementation Steps

### 1. Add `Body` and `Age` components to `core/entity.go`
- Add `bodies map[Entity]*Body` to `World`
- Add `ages map[Entity]*Age` to `World`
- Add `AddBody(e, b)`, `Body(e)`, `AddAge(e, a)`, `Age(e)` methods
- Initialize maps in `NewWorld()`

### 2. Add `Despawn(e)` method to `core/entity.go`
- Remove entity from all component maps (transforms, drawables, bodies, ages, etc.)
- Remove from parent/child relationships
- Mark entity ID as available for reuse (or just increment spawn counter)

### 3. Create `physics/integrate.go`
- `EulerIntegration()` — simple forward integration
- `VerletIntegration()` — maintains `prevPositions map[Entity]Vec2` in closure
- Unit tests: verify position/velocity after N steps with known acceleration

### 4. Create `physics/forces.go`
- `Attractor()`, `Repulsor()`, `Drag()`, `Gravity()`, `Turbulence()`
- Unit tests: verify acceleration/velocity changes

### 5. Create `physics/spring.go`
- `Spring()`
- Unit tests: verify spring force applied, damping works

### 6. Create `particle/target.go`
- `InterpolateToTarget()`
- Unit tests: verify convergence to target

### 7. Create `particle/lifecycle.go`
- `AgeAndDespawn()`
- Unit tests: verify age increments, despawn at lifetime

### 8. Create `particle/emit.go`
- `Emit()`
- Unit tests: verify spawn rate (deterministic time)

### 9. Create `particle/cloud.go`
- `BitmapToCloud()`, `DistributeTargets()`
- Unit tests: verify cloud extraction from bitmap, target assignment

### 10. Add golden test (`golden_test.go`)
- `TestParticleMotion`: spawn particles, apply attractor, verify positions over time

### 11. Update demo (`cmd/flicker/main.go`)
- Load two text bitmaps ("A" and "B")
- Convert to point clouds
- Spawn particles at cloud A positions
- Distribute targets from cloud B
- Add `physics.EulerIntegration`, `particle.InterpolateToTarget`, `particle.AgeAndDespawn` behaviors
- Particles morph from A to B over ~3 seconds

### 12. Update `TODO.md`
- Mark particle systems complete
- Note deferred features (pooling, collision, spatial partitioning)

## API Examples

### Simple Gravity Fountain

```go
emitter := world.Spawn()
world.AddTransform(emitter, &core.Transform{Position: fmath.Vec3{X: 40, Y: 5}})
world.AddBehavior(emitter, particle.Emit(20.0, func(w *core.World) core.Entity {
    p := w.Spawn()
    angle := rand.Float64() * math.Pi * 2
    speed := 5.0 + rand.Float64()*5.0
    w.AddTransform(p, &core.Transform{Position: fmath.Vec3{X: 40, Y: 5}})
    w.AddBody(p, &core.Body{
        Velocity: fmath.Vec2{X: math.Cos(angle) * speed, Y: math.Sin(angle) * speed},
    })
    w.AddAge(p, &core.Age{Lifetime: 3.0})
    w.AddDrawable(p, &bitmap.FullBlock{Bitmap: singlePixel})
    w.AddBehavior(p, physics.EulerIntegration())
    w.AddBehavior(p, physics.Gravity(fmath.Vec2{X: 0, Y: 20}))
    w.AddBehavior(p, particle.AgeAndDespawn())
    return p
}))
```

### Point Cloud Morph

```go
cloudA := particle.BitmapToCloud(textA.Bitmap)
cloudB := particle.BitmapToCloud(textB.Bitmap)

particles := make([]core.Entity, len(cloudA))
for i, pos := range cloudA {
    p := world.Spawn()
    particles[i] = p
    world.AddTransform(p, &core.Transform{Position: fmath.Vec3{X: pos.X, Y: pos.Y}})
    world.AddBody(p, &core.Body{})
    world.AddDrawable(p, &bitmap.FullBlock{Bitmap: singlePixel})
    world.AddBehavior(p, physics.EulerIntegration())
}

particle.DistributeTargets(particles, cloudB, 2.0, world)
```

### Attractor Field with Turbulence (works on any entity!)

```go
attractor := fmath.Vec2{X: 40, Y: 12}
for i := 0; i < 100; i++ {
    p := world.Spawn()
    world.AddTransform(p, &core.Transform{Position: randomPos()})
    world.AddBody(p, &core.Body{})
    world.AddDrawable(p, &bitmap.Braille{Bitmap: dot})
    world.AddBehavior(p, physics.EulerIntegration())
    world.AddBehavior(p, physics.Attractor(attractor, 500.0))
    world.AddBehavior(p, physics.Drag(0.1))
    world.AddBehavior(p, physics.Turbulence(0.1, 5.0))
}
```

### Physics on Text (not a particle!)

```go
// Make text entity fall with gravity
textEntity := world.Spawn()
world.AddTransform(textEntity, &core.Transform{Position: fmath.Vec3{X: 40, Y: 5}})
world.AddDrawable(textEntity, &bitmap.HalfBlock{Bitmap: textBitmap})
world.AddBody(textEntity, &core.Body{})
world.AddBehavior(textEntity, physics.EulerIntegration())
world.AddBehavior(textEntity, physics.Gravity(fmath.Vec2{X: 0, Y: 9.8}))
world.AddBehavior(textEntity, physics.Drag(0.02))
```

## Tests

**Integration (`physics/integrate_test.go`):**
- `TestEulerIntegration`: Entity with constant acceleration, verify position after N steps
- `TestVerletIntegration`: Same, verify Verlet produces different (more stable) trajectory
- `TestVerletStateSeparation`: Multiple entities with Verlet, verify each maintains separate prev position

**Forces (`physics/forces_test.go`):**
- `TestAttractor`: Entity near attractor, verify acceleration points towards center
- `TestRepulsor`: Entity near repulsor, verify acceleration points away
- `TestDrag`: Entity with velocity, verify velocity decays exponentially
- `TestGravity`: Verify constant acceleration applied each frame
- `TestTurbulence`: Verify acceleration changes based on position (deterministic noise)

**Spring (`physics/spring_test.go`):**
- `TestSpring`: Entity offset from anchor, verify spring force applied
- `TestSpringDamping`: Verify damping reduces oscillation

**Target (`particle/target_test.go`):**
- `TestInterpolateToTarget`: Entity far from target, verify it moves towards target and stops

**Lifecycle (`particle/lifecycle_test.go`):**
- `TestAgeAndDespawn`: Entity with Age component, verify age increments each frame
- `TestAgeAndDespawnLifetime`: Entity with lifetime, verify despawn when age exceeds lifetime

**Emit (`particle/emit_test.go`):**
- `TestEmit`: Emitter with rate 10/sec, run for 1 second, verify ~10 entities spawned

**Cloud (`particle/cloud_test.go`):**
- `TestBitmapToCloud`: 10×10 bitmap with 5 non-transparent pixels, verify cloud has 5 positions
- `TestDistributeTargets`: 10 entities, 3 cloud positions, verify round-robin assignment and behaviors added

**Golden (`golden_test.go`):**
- `TestParticleAttractor`: Spawn 5 particles in circle, central attractor, verify convergence over 60 frames

## What NOT to Build

- No particle pooling (spawn/despawn is fine for initial version)
- No collision detection between entities
- No spatial partitioning (acceptable performance for ~1000 entities without it)
- No entity sorting by depth (layer system handles draw order)
- No built-in trails (can be done with post-process fade material)
- No soft-body physics or constraints (springs are enough for initial version)
- No integration with external physics engines
- No `PrevPosition` in `Body` component (Verlet maintains its own state)

## Verification

```bash
go test ./physics/...           # physics behavior tests
go test ./particle/...          # particle helper tests
go test ./core/...              # Body/Age component tests
go test ./...                   # all tests including golden
go build ./cmd/flicker          # demo builds
```

Then: `git add <files>`, `make verify`, `git commit`, `git push`.

## Files

| File | Purpose | New/Modified |
|------|---------|-------------|
| `core/entity.go` | Body component, Age component, Despawn method | Modified |
| `physics/integrate.go` | EulerIntegration, VerletIntegration | New |
| `physics/integrate_test.go` | Integration tests | New |
| `physics/forces.go` | Attractor, Repulsor, Drag, Gravity, Turbulence | New |
| `physics/forces_test.go` | Force behavior tests | New |
| `physics/spring.go` | Spring | New |
| `physics/spring_test.go` | Spring tests | New |
| `particle/target.go` | InterpolateToTarget | New |
| `particle/target_test.go` | Target tests | New |
| `particle/lifecycle.go` | AgeAndDespawn | New |
| `particle/lifecycle_test.go` | Lifecycle tests | New |
| `particle/emit.go` | Emit | New |
| `particle/emit_test.go` | Emission tests | New |
| `particle/cloud.go` | BitmapToCloud, DistributeTargets | New |
| `particle/cloud_test.go` | Cloud helper tests | New |
| `golden_test.go` | Particle motion golden test | Modified |
| `cmd/flicker/main.go` | Point cloud morph demo | Modified |
| `TODO.md` | Mark complete | Modified |
