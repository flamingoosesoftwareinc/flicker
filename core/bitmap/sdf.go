package bitmap

import (
	"math"

	"flicker/core"
	"flicker/fmath"
)

// SDF holds a signed distance field computed from a bitmap.
// Convention: positive = outside the shape, negative = inside.
type SDF struct {
	Width, Height int
	Dist          []float64 // row-major; positive = outside, negative = inside
	MaxDist       float64
}

// At returns the signed distance at (x, y). Out-of-bounds returns MaxDist.
func (s *SDF) At(x, y int) float64 {
	if x < 0 || x >= s.Width || y < 0 || y >= s.Height {
		return s.MaxDist
	}
	return s.Dist[y*s.Width+x]
}

// Gradient returns the central-difference gradient of the SDF at (x, y).
func (s *SDF) Gradient(x, y int) fmath.Vec2 {
	dx := (s.At(x+1, y) - s.At(x-1, y)) / 2.0
	dy := (s.At(x, y+1) - s.At(x, y-1)) / 2.0
	return fmath.Vec2{X: dx, Y: dy}
}

// Bounds represents the axis-aligned bounding box of a shape.
type Bounds struct {
	MinX, MinY int
	MaxX, MaxY int
	Empty      bool // true if no content was found
}

// Bounds returns the bounding box of the shape in the SDF.
// It finds the min/max X and Y coordinates where the SDF distance is <= 0
// (inside or on the boundary of the shape).
func (s *SDF) Bounds() Bounds {
	minX, minY := s.Width, s.Height
	maxX, maxY := -1, -1

	for y := 0; y < s.Height; y++ {
		for x := 0; x < s.Width; x++ {
			if s.At(x, y) <= 0 {
				if x < minX {
					minX = x
				}
				if x > maxX {
					maxX = x
				}
				if y < minY {
					minY = y
				}
				if y > maxY {
					maxY = y
				}
			}
		}
	}

	if maxX < 0 {
		return Bounds{Empty: true}
	}

	return Bounds{
		MinX:  minX,
		MinY:  minY,
		MaxX:  maxX,
		MaxY:  maxY,
		Empty: false,
	}
}

// HalfBlockThreshold returns a material that reveals half-block encoded content
// using SDF threshold animation. Cells where the SDF distance exceeds
// *threshold are hidden. Each half (top/bottom) is masked independently.
func HalfBlockThreshold(s *SDF, threshold *float64) core.Material {
	return func(f core.Fragment) core.Cell {
		topD := s.At(f.X, f.Y*2)
		botD := s.At(f.X, f.Y*2+1)
		t := *threshold
		topVis := topD <= t
		botVis := botD <= t
		if !topVis && !botVis {
			return core.Cell{}
		}
		cell := f.Cell
		if !topVis {
			cell.FGAlpha = 0
		}
		if !botVis {
			cell.BGAlpha = 0
		}
		return cell
	}
}

// BrailleThreshold returns a material that reveals braille-encoded content
// using SDF threshold animation. Samples the SDF at the cell center
// (2x4 dots per cell) and hides the entire cell if above *threshold.
func BrailleThreshold(s *SDF, threshold *float64) core.Material {
	return func(f core.Fragment) core.Cell {
		px := f.X*2 + 1
		py := f.Y*4 + 2
		if s.At(px, py) > *threshold {
			return core.Cell{}
		}
		return f.Cell
	}
}

// ComputeSDF computes a signed distance field from a bitmap using the
// 8SSEDT (8-point Sequential Signed Euclidean Distance Transform) algorithm.
// maxDist clamps the result range to [-maxDist, +maxDist].
// A pixel is considered "inside" if its alpha > 0.
func ComputeSDF(bm *Bitmap, maxDist float64) *SDF {
	w, h := bm.Width, bm.Height
	n := w * h

	// Classify pixels: inside (alpha > 0) vs outside.
	inside := make([]bool, n)
	for i := range n {
		inside[i] = bm.Alpha[i] > 0
	}

	// Compute two unsigned distance fields:
	// dOut[p] = distance from outside pixel p to nearest inside pixel
	// dIn[p]  = distance from inside pixel p to nearest outside pixel
	dOut := edt(inside, w, h, maxDist)         // seeds are inside pixels
	dIn := edt(notBool(inside), w, h, maxDist) // seeds are outside pixels

	// Combine: SDF = dOut - dIn (positive outside, negative inside)
	dist := make([]float64, n)
	for i := range n {
		d := dOut[i] - dIn[i]
		if d > maxDist {
			d = maxDist
		}
		if d < -maxDist {
			d = -maxDist
		}
		dist[i] = d
	}

	return &SDF{
		Width:   w,
		Height:  h,
		Dist:    dist,
		MaxDist: maxDist,
	}
}

// notBool returns a negated copy of a boolean slice.
func notBool(a []bool) []bool {
	b := make([]bool, len(a))
	for i, v := range a {
		b[i] = !v
	}
	return b
}

// edt computes unsigned Euclidean distance to the nearest seed pixel using
// 8SSEDT. seed[i]=true marks a seed pixel (distance=0). Non-seed pixels
// get their distance to the nearest seed.
func edt(seed []bool, w, h int, maxDist float64) []float64 {
	n := w * h
	inf := maxDist + 1

	// dx, dy store the vector to the nearest seed pixel.
	dx := make([]float64, n)
	dy := make([]float64, n)

	// Initialize: seeds get (0,0), non-seeds get (inf,inf).
	for i := range n {
		if seed[i] {
			dx[i] = 0
			dy[i] = 0
		} else {
			dx[i] = inf
			dy[i] = inf
		}
	}

	// Forward pass: top-left to bottom-right.
	for y := range h {
		for x := range w {
			i := y*w + x
			propagate(dx, dy, w, h, x, y, i, -1, 0, inf)  // left
			propagate(dx, dy, w, h, x, y, i, -1, -1, inf) // top-left
			propagate(dx, dy, w, h, x, y, i, 0, -1, inf)  // top
			propagate(dx, dy, w, h, x, y, i, 1, -1, inf)  // top-right
		}
	}

	// Backward pass: bottom-right to top-left.
	for y := h - 1; y >= 0; y-- {
		for x := w - 1; x >= 0; x-- {
			i := y*w + x
			propagate(dx, dy, w, h, x, y, i, 1, 0, inf)  // right
			propagate(dx, dy, w, h, x, y, i, 1, 1, inf)  // bottom-right
			propagate(dx, dy, w, h, x, y, i, 0, 1, inf)  // bottom
			propagate(dx, dy, w, h, x, y, i, -1, 1, inf) // bottom-left
		}
	}

	// Convert (dx, dy) vectors to distances.
	dist := make([]float64, n)
	for i := range n {
		d := math.Sqrt(dx[i]*dx[i] + dy[i]*dy[i])
		if d > maxDist {
			d = maxDist
		}
		dist[i] = d
	}
	return dist
}

// propagate checks if going through neighbor (x+ox, y+oy) yields a shorter
// path to the nearest seed for pixel at index i.
func propagate(dx, dy []float64, w, h, x, y, i, ox, oy int, inf float64) {
	nx, ny := x+ox, y+oy
	if nx < 0 || nx >= w || ny < 0 || ny >= h {
		return
	}
	ni := ny*w + nx
	// Candidate vector: neighbor's vector + offset back to current pixel.
	cdx := dx[ni] - float64(ox)
	cdy := dy[ni] - float64(oy)
	cDist := cdx*cdx + cdy*cdy
	myDist := dx[i]*dx[i] + dy[i]*dy[i]
	if cDist < myDist {
		dx[i] = cdx
		dy[i] = cdy
	}
}
