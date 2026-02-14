package asset

import (
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"testing"

	"flicker/core"
)

func writePNG(t *testing.T, dir, name string, img image.Image) string {
	t.Helper()
	path := filepath.Join(dir, name)
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
	return path
}

func writeJPEG(t *testing.T, dir, name string, img image.Image) string {
	t.Helper()
	path := filepath.Join(dir, name)
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	if err := jpeg.Encode(f, img, &jpeg.Options{Quality: 100}); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadImage_PNG_Roundtrip(t *testing.T) {
	dir := t.TempDir()
	img := image.NewNRGBA(image.Rect(0, 0, 4, 4))
	img.SetNRGBA(0, 0, color.NRGBA{R: 255, G: 0, B: 0, A: 255})
	img.SetNRGBA(1, 0, color.NRGBA{R: 0, G: 255, B: 0, A: 255})
	img.SetNRGBA(2, 0, color.NRGBA{R: 0, G: 0, B: 255, A: 255})
	img.SetNRGBA(3, 0, color.NRGBA{R: 100, G: 150, B: 200, A: 128})

	path := writePNG(t, dir, "test.png", img)

	bm, err := LoadImage(path, 0, 0)
	if err != nil {
		t.Fatalf("LoadImage: %v", err)
	}

	if bm.Width != 4 || bm.Height != 4 {
		t.Fatalf("size = %dx%d, want 4x4", bm.Width, bm.Height)
	}

	c, a := bm.Get(0, 0)
	if c != (core.Color{R: 255}) || a != 1.0 {
		t.Errorf("(0,0) = %v %v, want {255 0 0} 1.0", c, a)
	}

	c, a = bm.Get(1, 0)
	if c != (core.Color{G: 255}) || a != 1.0 {
		t.Errorf("(1,0) = %v %v, want {0 255 0} 1.0", c, a)
	}

	c, a = bm.Get(2, 0)
	if c != (core.Color{B: 255}) || a != 1.0 {
		t.Errorf("(2,0) = %v %v, want {0 0 255} 1.0", c, a)
	}

	// Semi-transparent pixel.
	c, a = bm.Get(3, 0)
	if c != (core.Color{R: 100, G: 150, B: 200}) {
		t.Errorf("(3,0) color = %v, want {100 150 200}", c)
	}
	wantA := float64(128) / 255.0
	if math.Abs(a-wantA) > 0.01 {
		t.Errorf("(3,0) alpha = %v, want ~%v", a, wantA)
	}
}

func TestLoadImage_JPEG(t *testing.T) {
	dir := t.TempDir()
	img := image.NewNRGBA(image.Rect(0, 0, 4, 4))
	for y := range 4 {
		for x := range 4 {
			img.SetNRGBA(x, y, color.NRGBA{R: 200, G: 100, B: 50, A: 255})
		}
	}

	path := writeJPEG(t, dir, "test.jpg", img)

	bm, err := LoadImage(path, 0, 0)
	if err != nil {
		t.Fatalf("LoadImage: %v", err)
	}

	// JPEG is lossy — check approximate values.
	c, a := bm.Get(2, 2)
	if a != 1.0 {
		t.Errorf("alpha = %v, want 1.0", a)
	}
	if absDiff(c.R, 200) > 10 || absDiff(c.G, 100) > 10 || absDiff(c.B, 50) > 10 {
		t.Errorf("color = %v, want approximately {200 100 50}", c)
	}
}

func TestLoadImage_Downsample(t *testing.T) {
	dir := t.TempDir()
	img := image.NewNRGBA(image.Rect(0, 0, 10, 10))
	for y := range 10 {
		for x := range 10 {
			img.SetNRGBA(x, y, color.NRGBA{R: uint8(x * 25), G: uint8(y * 25), A: 255})
		}
	}

	path := writePNG(t, dir, "big.png", img)

	bm, err := LoadImage(path, 5, 5)
	if err != nil {
		t.Fatalf("LoadImage: %v", err)
	}

	if bm.Width != 5 || bm.Height != 5 {
		t.Errorf("size = %dx%d, want 5x5", bm.Width, bm.Height)
	}
}

func TestLoadImage_AspectRatio(t *testing.T) {
	dir := t.TempDir()
	img := image.NewNRGBA(image.Rect(0, 0, 20, 10))
	path := writePNG(t, dir, "wide.png", img)

	bm, err := LoadImage(path, 10, 10)
	if err != nil {
		t.Fatalf("LoadImage: %v", err)
	}

	// 20x10 scaled to fit 10x10 → scale=0.5 → 10x5
	if bm.Width != 10 || bm.Height != 5 {
		t.Errorf("size = %dx%d, want 10x5", bm.Width, bm.Height)
	}
}

func TestLoadImage_Alpha(t *testing.T) {
	dir := t.TempDir()
	img := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	img.SetNRGBA(0, 0, color.NRGBA{R: 255, A: 255})
	img.SetNRGBA(1, 0, color.NRGBA{R: 255, A: 0}) // fully transparent

	path := writePNG(t, dir, "alpha.png", img)

	bm, err := LoadImage(path, 0, 0)
	if err != nil {
		t.Fatalf("LoadImage: %v", err)
	}

	_, a := bm.Get(0, 0)
	if a != 1.0 {
		t.Errorf("opaque alpha = %v, want 1.0", a)
	}

	_, a = bm.Get(1, 0)
	if a != 0 {
		t.Errorf("transparent alpha = %v, want 0", a)
	}
}

func TestLoadImage_Nonexistent(t *testing.T) {
	_, err := LoadImage("/nonexistent/path.png", 0, 0)
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestLoadImage_InvalidFormat(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.png")
	if err := os.WriteFile(path, []byte("not an image"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadImage(path, 0, 0)
	if err == nil {
		t.Error("expected error for invalid image data")
	}
}

func TestLoadImage_CacheHit(t *testing.T) {
	dir := t.TempDir()
	img := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	path := writePNG(t, dir, "cached.png", img)

	ca := NewCache()
	bm1, err := ca.GetOrLoadImage(path, 0, 0)
	if err != nil {
		t.Fatalf("first load: %v", err)
	}
	bm2, err := ca.GetOrLoadImage(path, 0, 0)
	if err != nil {
		t.Fatalf("second load: %v", err)
	}

	if bm1 != bm2 {
		t.Error("cache should return same pointer")
	}
}

func TestLoadImage_NoDownsampleWhenSmaller(t *testing.T) {
	dir := t.TempDir()
	img := image.NewNRGBA(image.Rect(0, 0, 3, 3))
	path := writePNG(t, dir, "small.png", img)

	bm, err := LoadImage(path, 10, 10)
	if err != nil {
		t.Fatalf("LoadImage: %v", err)
	}

	// Image is already smaller than max — should not be scaled.
	if bm.Width != 3 || bm.Height != 3 {
		t.Errorf("size = %dx%d, want 3x3", bm.Width, bm.Height)
	}
}

func absDiff(a, b uint8) uint8 {
	if a > b {
		return a - b
	}
	return b - a
}
