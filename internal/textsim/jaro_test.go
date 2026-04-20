package textsim

import (
	"math"
	"testing"
)

func TestJaro(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		a, b    string
		want    float64
		epsilon float64
	}{
		{"both empty", "", "", 1.0, 0.0001},
		{"one empty", "abc", "", 0.0, 0.0001},
		{"identical", "hello", "hello", 1.0, 0.0001},
		{"martha/marhta", "MARTHA", "MARHTA", 0.9444, 0.001},
		{"dixon/dicksonx", "DIXON", "DICKSONX", 0.7667, 0.001},
		{"dwayne/duane", "DWAYNE", "DUANE", 0.8222, 0.001},
		{"different", "abc", "xyz", 0.0, 0.0001},
		{"cyrillic identical", "Пинк", "Пинк", 1.0, 0.0001},
		{"cyrillic transposition", "абвг", "абгв", 0.9167, 0.001},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := Jaro(tc.a, tc.b)
			if math.Abs(got-tc.want) > tc.epsilon {
				t.Errorf("Jaro(%q, %q) = %v, want %v (±%v)", tc.a, tc.b, got, tc.want, tc.epsilon)
			}
		})
	}
}

func TestJaroSymmetric(t *testing.T) {
	t.Parallel()
	pairs := [][2]string{
		{"hello", "world"},
		{"Pink", "P!nk"},
		{"The Weeknd", "Weeknd"},
		{"Björk", "Bjork"},
	}
	for _, p := range pairs {
		forward := Jaro(p[0], p[1])
		backward := Jaro(p[1], p[0])
		if math.Abs(forward-backward) > 0.0001 {
			t.Errorf("Jaro not symmetric for %q/%q: %v vs %v", p[0], p[1], forward, backward)
		}
	}
}

func TestJaroWinkler(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		a, b    string
		want    float64
		epsilon float64
	}{
		{"both empty", "", "", 1.0, 0.0001},
		{"one empty", "abc", "", 0.0, 0.0001},
		{"identical", "hello", "hello", 1.0, 0.0001},
		{"martha/marhta", "MARTHA", "MARHTA", 0.9611, 0.001},
		{"dixon/dicksonx", "DIXON", "DICKSONX", 0.8133, 0.001},
		{"dwayne/duane", "DWAYNE", "DUANE", 0.8400, 0.001},
		{"no prefix match", "abc", "xyz", 0.0, 0.0001},
		{"prefix boost", "prefix", "prefixes", 0.95, 0.002},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := JaroWinkler(tc.a, tc.b)
			if math.Abs(got-tc.want) > tc.epsilon {
				t.Errorf("JaroWinkler(%q, %q) = %v, want %v (±%v)", tc.a, tc.b, got, tc.want, tc.epsilon)
			}
		})
	}
}

func TestJaroWinklerLowSimilarityNoBoost(t *testing.T) {
	t.Parallel()
	// Strings with Jaro < 0.7 should not be boosted (even if they share a prefix).
	a := "abcdef"
	b := "abcxyzqrs"
	jw := JaroWinkler(a, b)
	j := Jaro(a, b)
	if j >= 0.7 {
		t.Skipf("test assumes Jaro < 0.7, got %v", j)
	}
	if jw != j {
		t.Errorf("JaroWinkler should equal Jaro when Jaro < 0.7: got %v vs %v", jw, j)
	}
}

func FuzzJaroSymmetric(f *testing.F) {
	seeds := [][2]string{
		{"hello", "world"},
		{"", ""},
		{"a", "b"},
		{"abc", "abc"},
	}
	for _, s := range seeds {
		f.Add(s[0], s[1])
	}
	f.Fuzz(func(t *testing.T, a, b string) {
		x := Jaro(a, b)
		y := Jaro(b, a)
		if math.Abs(x-y) > 0.0001 {
			t.Errorf("not symmetric: Jaro(%q,%q)=%v, Jaro(%q,%q)=%v", a, b, x, b, a, y)
		}
		if x < 0 || x > 1 {
			t.Errorf("out of range: Jaro(%q,%q)=%v", a, b, x)
		}
	})
}

func FuzzJaroWinklerRange(f *testing.F) {
	f.Add("hello", "world")
	f.Add("", "")
	f.Add("abc", "")
	f.Fuzz(func(t *testing.T, a, b string) {
		got := JaroWinkler(a, b)
		if got < 0 || got > 1 {
			t.Errorf("out of range: JaroWinkler(%q,%q)=%v", a, b, got)
		}
	})
}
