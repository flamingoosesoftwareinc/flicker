package asset

import (
	"flicker/core"
	"flicker/fmath"
)

// RasterizeWireframe projects a mesh through mvp and draws wireframe edges
// into the bitmap using the given color. The bitmap is not cleared first —
// call bm.Clear() beforehand if needed.
//
// NDC coordinates [-1,1] are mapped to [0, Width-1] and [0, Height-1].
// Edges with any vertex behind the camera (w <= 0) are clipped.
func RasterizeWireframe(mesh *Mesh, mvp fmath.Mat4, bm *core.Bitmap, c core.Color) {
	w := float64(bm.Width)
	h := float64(bm.Height)

	// Project all vertices to screen space. Track which are valid (w > 0).
	type projected struct {
		x, y  int
		valid bool
	}
	pts := make([]projected, len(mesh.Vertices))
	for i, v := range mesh.Vertices {
		// Manual transform to get w before divide.
		cx := mvp[0]*v.X + mvp[1]*v.Y + mvp[2]*v.Z + mvp[3]
		cy := mvp[4]*v.X + mvp[5]*v.Y + mvp[6]*v.Z + mvp[7]
		cw := mvp[12]*v.X + mvp[13]*v.Y + mvp[14]*v.Z + mvp[15]

		if cw <= 0 {
			pts[i] = projected{valid: false}
			continue
		}

		ndcX := cx / cw
		ndcY := cy / cw

		// Map NDC [-1,1] to bitmap [0, size-1].
		sx := int((ndcX + 1) * 0.5 * (w - 1))
		sy := int((1 - ndcY) * 0.5 * (h - 1)) // flip Y: NDC up → bitmap down

		pts[i] = projected{x: sx, y: sy, valid: true}
	}

	// Draw edges for each face.
	for _, f := range mesh.Faces {
		a, b, cc := pts[f.V[0]], pts[f.V[1]], pts[f.V[2]]
		if a.valid && b.valid {
			bm.Line(a.x, a.y, b.x, b.y, c)
		}
		if b.valid && cc.valid {
			bm.Line(b.x, b.y, cc.x, cc.y, c)
		}
		if cc.valid && a.valid {
			bm.Line(cc.x, cc.y, a.x, a.y, c)
		}
	}
}
