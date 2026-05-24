package domain

import "testing"

func TestForgeActionDeltasAndIsHit(t *testing.T) {
	tests := []struct {
		act  ForgeAction
		want int
		hit  bool
	}{
		{WeakHit, -3, true},
		{MediumHit, -6, true},
		{StrongHit, -9, true},
		{Draw, -15, false},
		{Stamp, 2, false},
		{Bend, 7, false},
		{Upset, 13, false},
		{Shrink, 16, false},
	}

	for _, tc := range tests {
		d, ok := ForgeActionDelta[tc.act]
		if !ok {
			t.Fatalf("delta missing for %s", tc.act)
		}
		if d != tc.want {
			t.Fatalf("delta for %s = %d; want %d", tc.act, d, tc.want)
		}
		if IsHit(tc.act) != tc.hit {
			t.Fatalf("IsHit(%s) = %v; want %v", tc.act, IsHit(tc.act), tc.hit)
		}
	}
}
