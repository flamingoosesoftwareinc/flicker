# Iteration 11: Asset Loading

## Context

Iterations 1-10 built the rendering pipeline: entities, transforms, canvas, bitmaps, compositing, blend modes, tweens, materials, sub-cell encoding, and rotation/scale. All assets are currently constructed procedurally in Go code. There's no file I/O anywhere in the codebase.

The roadmap lists four downstream features that need asset loading: text rendering (fonts), OBJ rendering (3D models), SVG rendering (vector paths), and PNG/image rendering. This iteration builds the loading foundation they all depend on.

`suzanne.obj` (500 vertices, 968 triangulated faces) already sits in the repo waiting to be parsed.

## Scope

**Include:**
- Image loading (PNG/JPEG → `*core.Bitmap`) — stdlib only, immediately useful
- OBJ loading (Wavefront OBJ → `*Mesh` struct) — `suzanne.obj` is ready
- Resource cache — simple path-keyed map

**Defer:**
- Font loading (TTF) — needs `golang.org/x/image/font`, glyph rasterization is complex; better as part of the text rendering iteration
- SVG loading — most complex parser, separate roadmap feature

## Package: `asset/`

New package at the same level as `core/`, `fmath/`, `terminal/`. Keeps file I/O concerns out of the rendering pipeline. Imports `core` (for `Bitmap`, `Color`) and `fmath` (for `Vec3`, `Vec2`). No new external dependencies — all stdlib.

```
asset/
  cache.go       // Cache type: path→asset map
  cache_test.go
  image.go       // LoadImage: PNG/JPEG → *core.Bitmap
  image_test.go
  obj.go         // LoadOBJ: Wavefront OBJ → *Mesh
  obj_test.go
```

## Types

### Cache (`asset/cache.go`)

```go
type Cache struct {
    mu      sync.Mutex
    entries map[string]any
}

func NewCache() *Cache
func (c *Cache) Get(path string) (any, bool)
func (c *Cache) Put(path string, asset any)
```

Uses `any` — two loaders don't justify a generic. Thread-safe via mutex. No TTL or eviction (terminal apps are short-lived). Typed convenience methods on Cache for each loader (below).

### Image Loader (`asset/image.go`)

```go
func LoadImage(path string, maxWidth, maxHeight int) (*core.Bitmap, error)
func (c *Cache) GetOrLoadImage(path string, maxW, maxH int) (*core.Bitmap, error)
```

- Registers `image/png` and `image/jpeg` decoders via blank imports
- `image.Decode` auto-detects format
- Converts pixels via `color.NRGBAModel.Convert()` → `core.Color` + alpha
- Nearest-neighbor downsampling when maxWidth/maxHeight > 0 (preserves aspect ratio)
- Cache key includes dimensions: `path@WxH`

### OBJ Loader (`asset/obj.go`)

```go
type Mesh struct {
    Vertices []fmath.Vec3
    Normals  []fmath.Vec3
    UVs      []fmath.Vec2
    Faces    []Face
}

type Face struct {
    V  [3]int // vertex indices (0-based)
    VN [3]int // normal indices (-1 if absent)
    VT [3]int // UV indices (-1 if absent)
}

func LoadOBJ(path string) (*Mesh, error)
func (c *Cache) GetOrLoadOBJ(path string) (*Mesh, error)
```

- Line-by-line `bufio.Scanner`, tokenize with `strings.Fields`
- Handles face formats: `v`, `v/vt`, `v/vt/vn`, `v//vn`
- Converts 1-based OBJ indices to 0-based
- Fan-triangulates polygons with >3 vertices
- Ignores `#`, `s`, `g`, `o`, `usemtl`, `mtllib`
- No material/texture support (deferred)

## Implementation Order

1. `cache.go` + `cache_test.go` — no dependencies
2. `obj.go` + `obj_test.go` — test against `suzanne.obj` (511 vertices, 968 faces)
3. `image.go` + `image_test.go` — tests create PNG/JPEG fixtures via `testing.TempDir()`

## What NOT to build

- No `Loader` interface — two concrete functions, not a pattern yet
- No `io.Reader` overloads — all use cases are file paths
- No bilinear/bicubic resampling — terminal resolution is too coarse to matter
- No OBJ material (`.mtl`) parsing
- No concurrent/async loading
- No negative OBJ index support (document limitation)

## Tests

**Cache:** miss returns nil/false, put+get roundtrip, overwrite, mixed types.

**OBJ:** load `suzanne.obj` and verify counts (511 vertices, 507 normals, 590 UVs, 968 faces), spot-check first vertex `{0.4375, 0.164063, 0.765625}`, all indices in bounds; minimal OBJ with vertices-only faces (VN/VT = -1); quad fan-triangulation; comments/blanks skipped; nonexistent path errors.

**Image:** programmatically create 4x4 PNG with known pixels, load and verify roundtrip; JPEG approximate values; downsampling 10x10→5x5; aspect ratio preservation; alpha channel transfer; nonexistent path error; invalid format error; cache hit returns same pointer.

## Verification

1. `go build ./...` compiles
2. `go test ./asset/...` — all new tests pass
3. `go test ./...` — no regressions
4. `go build ./cmd/flicker/...` — demo builds
5. `make verify` passes

## Files to create

| File | Purpose | Est. lines |
|------|---------|-----------|
| `asset/cache.go` | Path-keyed asset cache | ~35 |
| `asset/cache_test.go` | Cache unit tests | ~50 |
| `asset/obj.go` | OBJ parser, Mesh/Face types | ~120 |
| `asset/obj_test.go` | OBJ tests against suzanne.obj | ~120 |
| `asset/image.go` | PNG/JPEG loader with downsampling | ~80 |
| `asset/image_test.go` | Image loader tests | ~120 |

No existing files modified. No new external dependencies.
