package core

import "flicker/fmath"

// Camera holds orthographic camera parameters. The zero value is usable:
// a Zoom of 0 is treated as 1.0 (no magnification).
type Camera struct {
	Zoom float64
}

// ViewMatrix computes the world-to-screen matrix for an orthographic camera.
//
//	Translate(screenW/2, screenH/2) × Scale(zoom, zoom) × Rotate(-rotation) × Translate(-posX, -posY)
//
// This centers the camera's world position on screen, then applies zoom
// (Zoom=2 → things appear 2× bigger).
func ViewMatrix(cam *Camera, tr *Transform, screenW, screenH int) fmath.Mat3 {
	if cam == nil || tr == nil {
		return fmath.Mat3Identity()
	}

	zoom := cam.Zoom
	if zoom <= 0 {
		zoom = 1.0
	}

	halfW := float64(screenW) / 2.0
	halfH := float64(screenH) / 2.0

	return fmath.Mat3Translate(halfW, halfH).
		Multiply(fmath.Mat3Scale(zoom, zoom)).
		Multiply(fmath.Mat3Rotate(-tr.Rotation)).
		Multiply(fmath.Mat3Translate(-tr.Position.X, -tr.Position.Y))
}
