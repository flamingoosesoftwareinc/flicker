# Flicker

A terminal rendering engine.

## Philosophy

Iterative, minimal, working software at every step. The terminal is a 2D cell grid. Everything else is abstraction on top of that.

Flat package structure. High cohesion, low coupling. No premature abstraction.

## Terminal Backend: tcell

Flicker uses [tcell](https://github.com/gdamore/tcell) for direct terminal access. Bubble Tea's retained-mode string diffing is the wrong fit for an engine that redraws a full cell buffer every frame. tcell gives us immediate-mode cell writes, input handling, and terminal lifecycle management without overhead we'd fight against.

Bubble Tea remains a good choice for a future editor/preview tool built *on top of* Flicker.

## Iteration 1: A Square on Screen

The goal is the simplest possible proof of life: a rectangle rendered to the terminal via a scene graph, a transform, a geometry component, and a direct cell buffer. No materials, no effects, no scripting.

### Core Types

```go
// Entity is just an ID.
type Entity uint64

// World holds all entities and their components.
type World struct {
    next       Entity
    transforms map[Entity]*Transform
    geometries map[Entity]*Geometry
    children   map[Entity][]Entity
    roots      []Entity
}

func (w *World) Spawn() Entity
func (w *World) Attach(e Entity, parent Entity)
func (w *World) AddTransform(e Entity, t *Transform)
func (w *World) AddGeometry(e Entity, g *Geometry)
```

### Transform

Position in cell-space. Float64 internally for sub-cell precision.

```go
type Transform struct {
    Position fmath.Vec2
}
```

Kept deliberately minimal. Scale, rotation, and anchor points come later when there's a reason for them.

### Geometry

Describes what to draw. For iteration 1, just a filled rectangle.

```go
type GeometryKind int

const (
    GeoRect GeometryKind = iota
)

type Geometry struct {
    Kind   GeometryKind
    Width  int
    Height int
    Rune   rune  // fill character
}
```

### Canvas

The cell buffer. This is the core rendering target. Everything draws to this.

```go
type Cell struct {
    Rune rune
    FG   Color
    BG   Color
}

type Canvas struct {
    Width, Height int
    Cells         [][]Cell
}

func NewCanvas(w, h int) *Canvas
func (c *Canvas) Set(x, y int, cell Cell)
func (c *Canvas) Get(x, y int) Cell
func (c *Canvas) Clear()
```

### Render System

Walks the scene graph, resolves transforms, writes to the canvas.

```go
func Render(world *World, canvas *Canvas) {
    for _, root := range world.roots {
        renderEntity(world, canvas, root, 0, 0)
    }
}

func renderEntity(w *World, c *Canvas, e Entity, ox, oy float64) {
    t := w.transforms[e]
    if t == nil {
        return
    }

    ax, ay := ox+t.X, oy+t.Y

    if g := w.geometries[e]; g != nil {
        drawGeometry(c, g, int(ax), int(ay))
    }

    for _, child := range w.children[e] {
        renderEntity(w, c, child, ax, ay)
    }
}

func drawGeometry(c *Canvas, g *Geometry, x, y int) {
    for dy := 0; dy < g.Height; dy++ {
        for dx := 0; dx < g.Width; dx++ {
            c.Set(x+dx, y+dy, Cell{Rune: g.Rune})
        }
    }
}
```

### Terminal Output

Flush the canvas to the terminal. For iteration 1, this can be raw ANSI via tcell or even just `fmt.Print`.

```go
func Flush(canvas *Canvas, screen tcell.Screen) {
    for y := 0; y < canvas.Height; y++ {
        for x := 0; x < canvas.Width; x++ {
            cell := canvas.Get(x, y)
            screen.SetContent(x, y, cell.Rune, nil, tcell.StyleDefault)
        }
    }
    screen.Show()
}
```

### Main

```go
func main() {
    screen, _ := tcell.NewScreen()
    screen.Init()
    defer screen.Fini()

    w, h := screen.Size()
    canvas := NewCanvas(w, h)
    world := &World{ /* init maps */ }

    box := world.Spawn()
    world.AddTransform(box, &Transform{X: 10, Y: 5})
    world.AddGeometry(box, &Geometry{
        Kind:   GeoRect,
        Width:  20,
        Height: 10,
        Rune:   '█',
    })
    world.roots = append(world.roots, box)

    // render once
    canvas.Clear()
    Render(world, canvas)
    Flush(canvas, screen)

    // wait for quit
    for {
        ev := screen.PollEvent()
        if _, ok := ev.(*tcell.EventKey); ok {
            return
        }
    }
}
```

## What This Gives You

A running program that puts a white rectangle on screen. More importantly, the foundational abstractions: entity/component world, canvas buffer, scene graph traversal, and terminal flush. Everything that comes next (color, materials, animation, particles) layers onto these primitives without replacing them.

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
    screen.go      // tcell init, teardown, canvas flush, input polling
  cmd/
    flicker/
      main.go      // Wire everything, run the loop
```

`fmath` depends on nothing. `core` depends on `fmath`. `terminal` depends on `core` and `tcell`. `cmd` depends on all three.
