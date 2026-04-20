package textsim

import (
	"math"
	"testing"
)

func TestTokenSetRatio(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		a, b   string
		minVal float64
	}{
		{"identical", "the beatles", "the beatles", 0.99},
		{"reordered", "lennon john", "john lennon", 0.99},
		{"extra word", "the beatles", "beatles", 0.99},
		{"completely different", "hello world", "goodbye planet", 0},
		{"both empty", "", "", 0.99},
		{"one empty", "abc", "", 0},
		{"duplicates", "hello hello world", "world hello", 0.99},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := TokenSetRatio(tc.a, tc.b)
			if got+math.SmallestNonzeroFloat64 < tc.minVal {
				t.Errorf("TokenSetRatio(%q, %q) = %v, want >= %v", tc.a, tc.b, got, tc.minVal)
			}
		})
	}
}

func TestTokenSetRatioRange(t *testing.T) {
	t.Parallel()
	inputs := []struct{ a, b string }{
		{"", ""},
		{"a", ""},
		{"hello world", "hello"},
		{"the quick brown fox", "the fox jumps"},
	}
	for _, in := range inputs {
		v := TokenSetRatio(in.a, in.b)
		if v < 0 || v > 1 {
			t.Errorf("TokenSetRatio(%q, %q) = %v, out of [0,1]", in.a, in.b, v)
		}
	}
}
