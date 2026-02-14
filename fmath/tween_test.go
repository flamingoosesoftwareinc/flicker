package fmath

import (
	"math"
	"testing"
)

func TestClamp(t *testing.T) {
	tests := []struct {
		v, lo, hi, want float64
	}{
		{0.5, 0, 1, 0.5},
		{-1, 0, 1, 0},
		{2, 0, 1, 1},
		{0, 0, 1, 0},
		{1, 0, 1, 1},
		{5, 3, 7, 5},
		{1, 3, 7, 3},
		{10, 3, 7, 7},
	}
	for _, tc := range tests {
		got := Clamp(tc.v, tc.lo, tc.hi)
		if got != tc.want {
			t.Errorf("Clamp(%v, %v, %v) = %v, want %v", tc.v, tc.lo, tc.hi, got, tc.want)
		}
	}
}

func TestTweenLinear(t *testing.T) {
	tw := Tween{From: 0, To: 100, Duration: 1.0}

	tests := []struct {
		dt   float64
		want float64
		done bool
	}{
		{0.0, 0, false},
		{0.25, 25, false},
		{0.25, 50, false},
		{0.25, 75, false},
		{0.25, 100, true},
	}
	for i, tc := range tests {
		got := tw.Update(tc.dt)
		if !approxEqual(got, tc.want) {
			t.Errorf("step %d: Update(%v) = %v, want %v", i, tc.dt, got, tc.want)
		}
		if tw.Done() != tc.done {
			t.Errorf("step %d: Done() = %v, want %v", i, tw.Done(), tc.done)
		}
	}
}

func TestTweenWithEasing(t *testing.T) {
	tw := Tween{From: 0, To: 100, Duration: 1.0, Easing: EaseInQuad}

	// At t=0.5 linear → 0.5, EaseInQuad(0.5) = 0.25, so value = 25.
	got := tw.Update(0.5)
	want := 25.0
	if !approxEqual(got, want) {
		t.Errorf("Update(0.5) with EaseInQuad = %v, want %v", got, want)
	}

	// At t=1.0, easing must reach 1.0, so value = 100.
	got = tw.Update(0.5)
	want = 100.0
	if !approxEqual(got, want) {
		t.Errorf("Update(0.5) at end with EaseInQuad = %v, want %v", got, want)
	}
	if !tw.Done() {
		t.Error("expected Done() = true")
	}
}

func TestTweenOvershoot(t *testing.T) {
	tw := Tween{From: 0, To: 10, Duration: 1.0}

	// Overshoot: dt larger than remaining duration should clamp to To.
	got := tw.Update(5.0)
	if !approxEqual(got, 10) {
		t.Errorf("overshoot: Update(5.0) = %v, want 10", got)
	}
	if !tw.Done() {
		t.Error("expected Done() = true after overshoot")
	}
}

func TestTweenReset(t *testing.T) {
	tw := Tween{From: 0, To: 10, Duration: 1.0}

	tw.Update(1.0)
	if !tw.Done() {
		t.Fatal("expected Done() after full duration")
	}

	tw.Reset()
	if tw.Done() {
		t.Error("expected Done() = false after Reset")
	}

	got := tw.Update(0.5)
	if !approxEqual(got, 5) {
		t.Errorf("after Reset, Update(0.5) = %v, want 5", got)
	}
}

func TestTweenReverseRange(t *testing.T) {
	tw := Tween{From: 100, To: 0, Duration: 2.0}

	got := tw.Update(1.0)
	if !approxEqual(got, 50) {
		t.Errorf("reverse tween at midpoint = %v, want 50", got)
	}

	got = tw.Update(1.0)
	if !approxEqual(got, 0) {
		t.Errorf("reverse tween at end = %v, want 0", got)
	}
}

func TestTweenVec3Linear(t *testing.T) {
	tw := TweenVec3{
		From:     Vec3{X: 0, Y: 0, Z: 0},
		To:       Vec3{X: 10, Y: 20, Z: 30},
		Duration: 1.0,
	}

	got := tw.Update(0.5)
	want := Vec3{X: 5, Y: 10, Z: 15}
	if !approxEqualVec3(got, want) {
		t.Errorf("TweenVec3 at 0.5 = %v, want %v", got, want)
	}

	got = tw.Update(0.5)
	want = Vec3{X: 10, Y: 20, Z: 30}
	if !approxEqualVec3(got, want) {
		t.Errorf("TweenVec3 at 1.0 = %v, want %v", got, want)
	}
	if !tw.Done() {
		t.Error("expected Done() = true")
	}
}

func TestTweenVec3WithEasing(t *testing.T) {
	tw := TweenVec3{
		From:     Vec3{X: 0, Y: 0, Z: 0},
		To:       Vec3{X: 100, Y: 100, Z: 100},
		Duration: 1.0,
		Easing:   EaseInQuad,
	}

	got := tw.Update(0.5)
	// EaseInQuad(0.5) = 0.25
	want := Vec3{X: 25, Y: 25, Z: 25}
	if !approxEqualVec3(got, want) {
		t.Errorf("TweenVec3 with EaseInQuad at 0.5 = %v, want %v", got, want)
	}
}

func TestTweenVec3Reset(t *testing.T) {
	tw := TweenVec3{
		From:     Vec3{X: 0, Y: 0, Z: 0},
		To:       Vec3{X: 10, Y: 10, Z: 10},
		Duration: 1.0,
	}

	tw.Update(1.0)
	if !tw.Done() {
		t.Fatal("expected Done() after full duration")
	}

	tw.Reset()
	if tw.Done() {
		t.Error("expected Done() = false after Reset")
	}

	got := tw.Update(0.0)
	want := Vec3{X: 0, Y: 0, Z: 0}
	if !approxEqualVec3(got, want) {
		t.Errorf("after Reset, Update(0) = %v, want %v", got, want)
	}
}

func approxEqualVec3(a, b Vec3) bool {
	return math.Abs(a.X-b.X) < epsilon &&
		math.Abs(a.Y-b.Y) < epsilon &&
		math.Abs(a.Z-b.Z) < epsilon
}
