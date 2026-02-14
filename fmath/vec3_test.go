package fmath

import "testing"

func TestDot(t *testing.T) {
	tests := []struct {
		name string
		a, b Vec3
		want float64
	}{
		{"orthogonal", Vec3{1, 0, 0}, Vec3{0, 1, 0}, 0},
		{"parallel", Vec3{2, 0, 0}, Vec3{3, 0, 0}, 6},
		{"arbitrary", Vec3{1, 2, 3}, Vec3{4, 5, 6}, 32},
		{"anti-parallel", Vec3{1, 0, 0}, Vec3{-1, 0, 0}, -1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.a.Dot(tc.b)
			if !approxEqual(got, tc.want) {
				t.Errorf("Dot = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestCross(t *testing.T) {
	tests := []struct {
		name string
		a, b Vec3
		want Vec3
	}{
		{"i x j = k", Vec3{1, 0, 0}, Vec3{0, 1, 0}, Vec3{0, 0, 1}},
		{"j x i = -k", Vec3{0, 1, 0}, Vec3{1, 0, 0}, Vec3{0, 0, -1}},
		{"j x k = i", Vec3{0, 1, 0}, Vec3{0, 0, 1}, Vec3{1, 0, 0}},
		{"k x i = j", Vec3{0, 0, 1}, Vec3{1, 0, 0}, Vec3{0, 1, 0}},
		{"parallel = zero", Vec3{2, 0, 0}, Vec3{3, 0, 0}, Vec3{0, 0, 0}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.a.Cross(tc.b)
			if !approxEqual(got.X, tc.want.X) || !approxEqual(got.Y, tc.want.Y) ||
				!approxEqual(got.Z, tc.want.Z) {
				t.Errorf("Cross = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestCrossAnticommutativity(t *testing.T) {
	a := Vec3{1, 2, 3}
	b := Vec3{4, 5, 6}
	ab := a.Cross(b)
	ba := b.Cross(a)
	if !approxEqual(ab.X, -ba.X) || !approxEqual(ab.Y, -ba.Y) || !approxEqual(ab.Z, -ba.Z) {
		t.Errorf("a×b = %v, b×a = %v; expected a×b = -(b×a)", ab, ba)
	}
}
