package asset

import (
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg"
	_ "image/png"
	"os"

	"flicker/core"
	"flicker/core/bitmap"
)

// LoadImage decodes a PNG or JPEG file into a Bitmap.
// If maxWidth or maxHeight > 0, the image is downsampled (nearest-neighbor)
// to fit within those bounds while preserving aspect ratio.
func LoadImage(path string, maxWidth, maxHeight int) (*bitmap.Bitmap, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()

	img, _, err := image.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("decode %s: %w", path, err)
	}

	bounds := img.Bounds()
	srcW := bounds.Dx()
	srcH := bounds.Dy()

	dstW, dstH := srcW, srcH
	if maxWidth > 0 && maxHeight > 0 && (srcW > maxWidth || srcH > maxHeight) {
		scaleW := float64(maxWidth) / float64(srcW)
		scaleH := float64(maxHeight) / float64(srcH)
		scale := scaleW
		if scaleH < scale {
			scale = scaleH
		}
		dstW = int(float64(srcW) * scale)
		dstH = int(float64(srcH) * scale)
		if dstW < 1 {
			dstW = 1
		}
		if dstH < 1 {
			dstH = 1
		}
	}

	bm := bitmap.New(dstW, dstH)
	for y := range dstH {
		for x := range dstW {
			// Nearest-neighbor sampling.
			srcX := x * srcW / dstW
			srcY := y * srcH / dstH
			nrgba := color.NRGBAModel.Convert(img.At(bounds.Min.X+srcX, bounds.Min.Y+srcY)).(color.NRGBA)
			c := core.Color{R: nrgba.R, G: nrgba.G, B: nrgba.B}
			a := float64(nrgba.A) / 255.0
			bm.Set(x, y, c, a)
		}
	}
	return bm, nil
}

func (c *Cache) GetOrLoadImage(path string, maxW, maxH int) (*bitmap.Bitmap, error) {
	key := fmt.Sprintf("%s@%dx%d", path, maxW, maxH)
	if v, ok := c.Get(key); ok {
		return v.(*bitmap.Bitmap), nil
	}
	bm, err := LoadImage(path, maxW, maxH)
	if err != nil {
		return nil, err
	}
	c.Put(key, bm)
	return bm, nil
}
