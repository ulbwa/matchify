package featparse

import (
	"reflect"
	"testing"
)

func TestExtractFeatures(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		in        string
		wantTitle string
		wantFeats []string
	}{
		{"no feature", "Blinding Lights", "Blinding Lights", nil},
		{"paren feat.", "Despacito (feat. Justin Bieber)", "Despacito", []string{"Justin Bieber"}},
		{"paren ft.", "Despacito (ft. Justin Bieber)", "Despacito", []string{"Justin Bieber"}},
		{"paren featuring", "Despacito (featuring Justin Bieber)", "Despacito", []string{"Justin Bieber"}},
		{"paren feat no dot", "Despacito (feat Justin Bieber)", "Despacito", []string{"Justin Bieber"}},
		{"paren with", "Track (with Justin Bieber)", "Track", []string{"Justin Bieber"}},
		{"bracket feat.", "Despacito [feat. Justin Bieber]", "Despacito", []string{"Justin Bieber"}},
		{"dash feat.", "Despacito - feat. Justin Bieber", "Despacito", []string{"Justin Bieber"}},
		{"inline feat.", "Despacito feat. Justin Bieber", "Despacito", []string{"Justin Bieber"}},
		{"inline ft.", "Despacito ft. Justin Bieber", "Despacito", []string{"Justin Bieber"}},
		{"multiple ampersand", "Lean On (feat. MØ & DJ Snake)", "Lean On", []string{"MØ", "DJ Snake"}},
		{"multiple comma", "Track (feat. A, B, C)", "Track", []string{"A", "B", "C"}},
		{"multiple comma and", "Track (feat. A, B and C)", "Track", []string{"A", "B", "C"}},
		{"multiple comma ampersand", "Track (feat. A, B & C)", "Track", []string{"A", "B", "C"}},
		{"preserves other paren", "Despacito (Remix) [feat. Justin Bieber]", "Despacito (Remix)", []string{"Justin Bieber"}},
		{"case insensitive", "Track (FEAT. Artist)", "Track", []string{"Artist"}},
		{"inline with not matched", "Walking with Friends", "Walking with Friends", nil},
		{"dash with not matched", "Song - with Artist", "Song - with Artist", nil},
		{"multiple feat clauses", "Track (feat. A) (feat. B)", "Track", []string{"A", "B"}},
		{"trims result", "Track (feat. X)   ", "Track", []string{"X"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			title, feats := ExtractFeatures(tc.in)
			if title != tc.wantTitle {
				t.Errorf("title: got %q, want %q", title, tc.wantTitle)
			}
			if !reflect.DeepEqual(feats, tc.wantFeats) {
				t.Errorf("features: got %v, want %v", feats, tc.wantFeats)
			}
		})
	}
}

func TestSplitCompositeName(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		in   string
		want []string
	}{
		{"atomic", "Luis Fonsi", []string{"Luis Fonsi"}},
		{"empty", "", nil},
		{"whitespace only", "   ", nil},
		{"feat.", "Luis Fonsi feat. Daddy Yankee", []string{"Luis Fonsi", "Daddy Yankee"}},
		{"ft.", "Drake ft. Future", []string{"Drake", "Future"}},
		{"featuring", "Eminem featuring Rihanna", []string{"Eminem", "Rihanna"}},
		{"with", "Drake with DJ Khaled", []string{"Drake", "DJ Khaled"}},
		{"ampersand", "Glass Animals & Denzel Curry", []string{"Glass Animals", "Denzel Curry"}},
		{"comma", "A, B, C", []string{"A", "B", "C"}},
		{"and", "A and B", []string{"A", "B"}},
		{"complex", "A feat. B, C & D", []string{"A", "B", "C", "D"}},
		{"dedup", "A & A", []string{"A"}},
		{"case sensitive dedup", "A & a", []string{"A"}},
		{"trim", "  A   &   B  ", []string{"A", "B"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := SplitCompositeName(tc.in)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("SplitCompositeName(%q) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}

func FuzzExtractFeaturesStability(f *testing.F) {
	f.Add("Despacito (feat. Justin Bieber)")
	f.Add("Blinding Lights")
	f.Add("Track")
	f.Add("")
	f.Fuzz(func(t *testing.T, s string) {
		title, feats := ExtractFeatures(s)
		// Second pass should yield no new features; the cleaned title
		// should be stable under re-extraction.
		title2, feats2 := ExtractFeatures(title)
		if title2 != title {
			t.Errorf("not stable: %q -> %q -> %q", s, title, title2)
		}
		if len(feats2) != 0 {
			t.Errorf("second pass extracted features from cleaned title: %v (orig feats %v)", feats2, feats)
		}
	})
}
