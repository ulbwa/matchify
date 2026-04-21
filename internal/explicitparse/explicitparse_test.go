package explicitparse

import "testing"

func TestDetect(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		in   string
		want Explicitness
	}{
		{"no marker", "HUMBLE.", Unknown},
		{"paren explicit", "HUMBLE. (Explicit)", Explicit},
		{"paren explicit version", "HUMBLE. (Explicit Version)", Explicit},
		{"bracket explicit", "Song [Explicit]", Explicit},
		{"dash explicit", "Song - Explicit", Explicit},
		{"paren clean", "HUMBLE. (Clean)", Clean},
		{"paren clean version", "HUMBLE. (Clean Version)", Clean},
		{"bracket clean", "Song [Clean]", Clean},
		{"paren edited", "Song (Edited)", Clean},
		{"edited version", "Song (Edited Version)", Clean},
		{"case insensitive", "song (EXPLICIT)", Explicit},
		{"both present", "Weird [Clean] (Explicit)", Unknown},
		{"explicit as word without marker", "Explicit Bad Reasons", Unknown},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := Detect(tc.in); got != tc.want {
				t.Errorf("Detect(%q) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}

func TestDiffer(t *testing.T) {
	t.Parallel()
	tests := []struct {
		a, b Explicitness
		want bool
	}{
		{Unknown, Unknown, false},
		{Unknown, Clean, false},
		{Clean, Unknown, false},
		{Clean, Clean, false},
		{Explicit, Explicit, false},
		{Clean, Explicit, true},
		{Explicit, Clean, true},
	}
	for _, tc := range tests {
		if got := Differ(tc.a, tc.b); got != tc.want {
			t.Errorf("Differ(%v, %v) = %v, want %v", tc.a, tc.b, got, tc.want)
		}
	}
}
