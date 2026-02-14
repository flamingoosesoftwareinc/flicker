package asset

import (
	"testing"

	"flicker/core"
	"flicker/fmath"
)

func TestRasterizeWireframe_Simple(t *testing.T) {
	// A single triangle in the XY plane, viewed from Z=3.
	mesh := &Mesh{
		Vertices: []fmath.Vec3{
			{X: 0, Y: 0.5, Z: 0},
			{X: -0.5, Y: -0.5, Z: 0},
			{X: 0.5, Y: -0.5, Z: 0},
		},
		Faces: []Face{
			{V: [3]int{0, 1, 2}, VN: [3]int{-1, -1, -1}, VT: [3]int{-1, -1, -1}},
		},
	}

	proj := fmath.Mat4Perspective(1.0, 1.0, 0.1, 100)
	view := fmath.Mat4Translate(0, 0, -3)
	mvp := proj.Multiply(view)

	bm := core.NewBitmap(20, 20)
	c := core.Color{R: 255, G: 255, B: 255}
	RasterizeWireframe(mesh, mvp, bm, c)

	// At least some pixels should be set.
	count := 0
	for y := range bm.Height {
		for x := range bm.Width {
			_, a := bm.Get(x, y)
			if a > 0 {
				count++
			}
		}
	}
	if count == 0 {
		t.Error("wireframe produced no pixels")
	}
}

func TestRasterizeWireframe_BehindCamera(t *testing.T) {
	// Vertices behind the camera should be clipped.
	mesh := &Mesh{
		Vertices: []fmath.Vec3{
			{X: 0, Y: 0, Z: 5}, // behind camera at z=0 looking -Z
			{X: -1, Y: -1, Z: 5},
			{X: 1, Y: -1, Z: 5},
		},
		Faces: []Face{
			{V: [3]int{0, 1, 2}, VN: [3]int{-1, -1, -1}, VT: [3]int{-1, -1, -1}},
		},
	}

	proj := fmath.Mat4Perspective(1.0, 1.0, 0.1, 100)
	// Camera at origin, looking -Z. Vertices at z=5 are behind.
	mvp := proj

	bm := core.NewBitmap(20, 20)
	RasterizeWireframe(mesh, mvp, bm, core.Color{R: 255})

	count := 0
	for y := range bm.Height {
		for x := range bm.Width {
			_, a := bm.Get(x, y)
			if a > 0 {
				count++
			}
		}
	}
	if count != 0 {
		t.Errorf("behind-camera vertices should produce no pixels, got %d", count)
	}
}

func TestRasterizeWireframe_Suzanne(t *testing.T) {
	mesh, err := LoadOBJ("../suzanne.obj")
	if err != nil {
		t.Fatalf("LoadOBJ: %v", err)
	}

	proj := fmath.Mat4Perspective(1.0, 1.0, 0.1, 100)
	view := fmath.Mat4Translate(0, 0, -3)
	model := fmath.Mat4RotateY(0.5)
	mvp := proj.Multiply(view).Multiply(model)

	bm := core.NewBitmap(60, 60)
	RasterizeWireframe(mesh, mvp, bm, core.Color{R: 0, G: 255, B: 100})

	count := 0
	for y := range bm.Height {
		for x := range bm.Width {
			_, a := bm.Get(x, y)
			if a > 0 {
				count++
			}
		}
	}
	// Suzanne should produce a substantial number of pixels.
	if count < 100 {
		t.Errorf("suzanne wireframe only produced %d pixels, expected >100", count)
	}
}
