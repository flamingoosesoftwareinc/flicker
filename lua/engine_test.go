package lua

import (
	"os"
	"path/filepath"
	"testing"

	"flicker/core"
	glua "github.com/epikur-io/gopher-lua"
)

func TestLoadAndRunScene(t *testing.T) {
	// Write a test script to a temp file
	script := `
local f = require("flicker")

local hero

f.on_enter(function(world, ctx)
    hero = world:spawn()
    hero:set_transform({
        position = f.vec3(10, 20, 0),
        scale = f.vec3(1, 1, 1),
    })

    local bm = f.bitmap.new(5, 5)
    local green = f.color(0, 255, 0)
    bm:set_dot(2, 2, green)
    hero:set_drawable(f.bitmap.braille(bm))
    hero:set_material(f.material.solid(f.color(255, 0, 0)))
    world:add_root(hero)
end)

f.on_update(function(world, time)
    hero:set_position(f.vec3(time.total * 10, 20, 0))
end)
`
	dir := t.TempDir()
	path := filepath.Join(dir, "test.lua")
	if err := os.WriteFile(path, []byte(script), 0o644); err != nil {
		t.Fatal(err)
	}

	engine := NewEngine()
	defer engine.Close()

	scene, err := engine.Load(path, 80, 24)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Run lifecycle
	scene.OnEnter(core.SceneContext{Width: 80, Height: 24})
	scene.OnReady()

	// Verify entity was created
	roots := scene.World().Roots()
	if len(roots) != 1 {
		t.Fatalf("expected 1 root, got %d", len(roots))
	}

	tr := scene.World().Transform(roots[0])
	if tr == nil {
		t.Fatal("expected transform on root entity")
	}
	if tr.Position.X != 10 || tr.Position.Y != 20 {
		t.Fatalf("expected position (10, 20), got (%v, %v)", tr.Position.X, tr.Position.Y)
	}

	// Run a few update frames
	for i := 0; i < 10; i++ {
		scene.OnUpdate(core.Time{Total: float64(i) * 0.016, Delta: 0.016})
	}

	// Position should have changed
	tr = scene.World().Transform(roots[0])
	if tr.Position.X == 10 {
		t.Fatal("expected position to change after updates")
	}

	// Render to a canvas
	canvas := core.NewCanvas(80, 24)
	scene.Render(canvas, core.Time{Total: 0.16, Delta: 0.016})

	scene.OnExit()
}

func TestVec2Arithmetic(t *testing.T) {
	script := `
local f = require("flicker")
local a = f.vec2(3, 4)
local b = f.vec2(1, 2)
local c = a + b
result_x = c.x
result_y = c.y
result_len = #a
`
	dir := t.TempDir()
	path := filepath.Join(dir, "test.lua")
	if err := os.WriteFile(path, []byte(script), 0o644); err != nil {
		t.Fatal(err)
	}

	engine := NewEngine()
	defer engine.Close()

	_, err := engine.Load(path, 80, 24)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	L := engine.L
	if x := float64(L.GetGlobal("result_x").(glua.LNumber)); x != 4 {
		t.Fatalf("expected x=4, got %v", x)
	}
	if y := float64(L.GetGlobal("result_y").(glua.LNumber)); y != 6 {
		t.Fatalf("expected y=6, got %v", y)
	}
	if l := float64(L.GetGlobal("result_len").(glua.LNumber)); l != 5 {
		t.Fatalf("expected len=5, got %v", l)
	}
}

func TestSDFPrimitives(t *testing.T) {
	script := `
local f = require("flicker")
local circle = f.sdf.circle(10)
-- Point at origin should be inside (negative distance)
d_inside = circle(f.vec2(0, 0))
-- Point at (15, 0) should be outside (positive distance)
d_outside = circle(f.vec2(15, 0))
-- Point at (10, 0) should be on boundary (zero distance)
d_boundary = circle(f.vec2(10, 0))
`
	dir := t.TempDir()
	path := filepath.Join(dir, "test.lua")
	if err := os.WriteFile(path, []byte(script), 0o644); err != nil {
		t.Fatal(err)
	}

	engine := NewEngine()
	defer engine.Close()

	_, err := engine.Load(path, 80, 24)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	L := engine.L
	dInside := float64(L.GetGlobal("d_inside").(glua.LNumber))
	dOutside := float64(L.GetGlobal("d_outside").(glua.LNumber))
	dBoundary := float64(L.GetGlobal("d_boundary").(glua.LNumber))

	if dInside >= 0 {
		t.Fatalf("expected d_inside < 0, got %v", dInside)
	}
	if dOutside <= 0 {
		t.Fatalf("expected d_outside > 0, got %v", dOutside)
	}
	if dBoundary != 0 {
		t.Fatalf("expected d_boundary == 0, got %v", dBoundary)
	}
}

func TestAssetTextRasterization(t *testing.T) {
	script := `
local f = require("flicker")
local font = f.asset.load_font("Oxanium/static/Oxanium-Bold.ttf")
local layout = f.asset.rasterize_text("HI", {
    font = font,
    size = 20,
    color = f.color(255, 255, 255),
})
text_w = layout.width
text_h = layout.height
has_bitmap = layout.bitmap ~= nil
`
	// Run from workspace root so font path resolves
	dir := t.TempDir()
	path := filepath.Join(dir, "test.lua")
	if err := os.WriteFile(path, []byte(script), 0o644); err != nil {
		t.Fatal(err)
	}

	// Change to repo root so font path resolves
	oldDir, _ := os.Getwd()
	if err := os.Chdir(".."); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(oldDir) }()

	engine := NewEngine()
	defer engine.Close()

	_, err := engine.Load(path, 80, 24)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	L := engine.L
	w := float64(L.GetGlobal("text_w").(glua.LNumber))
	h := float64(L.GetGlobal("text_h").(glua.LNumber))
	hasBitmap := bool(L.GetGlobal("has_bitmap").(glua.LBool))

	if w <= 0 {
		t.Fatalf("expected text_w > 0, got %v", w)
	}
	if h <= 0 {
		t.Fatalf("expected text_h > 0, got %v", h)
	}
	if !hasBitmap {
		t.Fatal("expected layout to have a bitmap")
	}
}
