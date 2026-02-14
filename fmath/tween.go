package fmath

// Tween is a stateful float64 interpolator that tracks elapsed time,
// applies an optional easing function, and returns interpolated values.
type Tween struct {
	From     float64
	To       float64
	Duration float64
	Easing   func(float64) float64 // nil → linear
	elapsed  float64
}

// Update advances elapsed time by dt and returns the current interpolated value.
func (tw *Tween) Update(dt float64) float64 {
	tw.elapsed += dt
	t := Clamp(tw.elapsed/tw.Duration, 0, 1)
	if tw.Easing != nil {
		t = tw.Easing(t)
	}
	return Lerp(tw.From, tw.To, t)
}

// Done reports whether the tween has completed (elapsed >= Duration).
func (tw *Tween) Done() bool {
	return tw.elapsed >= tw.Duration
}

// Reset sets elapsed time back to zero.
func (tw *Tween) Reset() {
	tw.elapsed = 0
}

// TweenVec3 is a stateful Vec3 interpolator with the same API shape as Tween.
type TweenVec3 struct {
	From     Vec3
	To       Vec3
	Duration float64
	Easing   func(float64) float64 // nil → linear
	elapsed  float64
}

// Update advances elapsed time by dt and returns the current interpolated Vec3.
func (tw *TweenVec3) Update(dt float64) Vec3 {
	tw.elapsed += dt
	t := Clamp(tw.elapsed/tw.Duration, 0, 1)
	if tw.Easing != nil {
		t = tw.Easing(t)
	}
	return tw.From.Lerp(tw.To, t)
}

// Done reports whether the tween has completed (elapsed >= Duration).
func (tw *TweenVec3) Done() bool {
	return tw.elapsed >= tw.Duration
}

// Reset sets elapsed time back to zero.
func (tw *TweenVec3) Reset() {
	tw.elapsed = 0
}
