package textsim

import (
	"math"
	"testing"
)

// TestJaroWinkler_SmokeTest verifies the wrapper around the underlying
// library behaves as expected on classic cases. It is intentionally
// minimal — exhaustive testing belongs to the upstream project.
func TestJaroWinkler_SmokeTest(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		a, b string
		want float64
	}{
		{"identical", "hello", "hello", 1.0},
		{"both empty", "", "", 1.0},
		{"one empty", "abc", "", 0.0},
		{"prefix match boosts", "prefix", "prefixes", 0.95},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := JaroWinkler(tc.a, tc.b)
			if math.Abs(got-tc.want) > 0.01 {
				t.Errorf("JaroWinkler(%q, %q) = %v, want ≈ %v", tc.a, tc.b, got, tc.want)
			}
		})
	}
}

func TestJaroWinkler_InRange(t *testing.T) {
	t.Parallel()
	pairs := [][2]string{
		{"the beatles", "beatles"},
		{"bjork", "bjoerk"},
		{"", "x"},
		{"ab", "cd"},
	}
	for _, p := range pairs {
		got := JaroWinkler(p[0], p[1])
		if got < 0 || got > 1 {
			t.Errorf("out of range: JaroWinkler(%q,%q) = %v", p[0], p[1], got)
		}
	}
}

func TestJaroWinkler_Symmetric(t *testing.T) {
	t.Parallel()
	pairs := [][2]string{
		{"hello", "world"},
		{"prefix", "prefixes"},
		{"", ""},
	}
	for _, p := range pairs {
		forward := JaroWinkler(p[0], p[1])
		backward := JaroWinkler(p[1], p[0])
		if math.Abs(forward-backward) > 1e-9 {
			t.Errorf("JaroWinkler asymmetric for %q/%q: %v vs %v", p[0], p[1], forward, backward)
		}
	}
}
