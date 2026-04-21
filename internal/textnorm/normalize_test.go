package textnorm

import "testing"

func TestStripAccents(t *testing.T) {
	t.Parallel()
	tests := []struct{ in, want string }{
		{"Björk", "Bjork"},
		{"Beyoncé", "Beyonce"},
		{"Sigur Rós", "Sigur Ros"},
		{"Mötley Crüe", "Motley Crue"},
		{"", ""},
		{"ASCII only", "ASCII only"},
		{"Ça va", "Ca va"},
	}
	for _, tc := range tests {
		if got := StripAccents(tc.in); got != tc.want {
			t.Errorf("StripAccents(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestNormalizePunctuation(t *testing.T) {
	t.Parallel()
	tests := []struct{ in, want string }{
		{"don\u2019t", "don't"},
		{"\u201Chello\u201D", `"hello"`},
		{"a \u2013 b", "a - b"},
		{"a \u2014 b", "a - b"},
		{"foo\u2026", "foo..."},
	}
	for _, tc := range tests {
		if got := NormalizePunctuation(tc.in); got != tc.want {
			t.Errorf("NormalizePunctuation(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestCollapseWhitespace(t *testing.T) {
	t.Parallel()
	tests := []struct{ in, want string }{
		{"  hello   world  ", "hello world"},
		{"\thello\nworld", "hello world"},
		{"", ""},
		{"single", "single"},
	}
	for _, tc := range tests {
		if got := CollapseWhitespace(tc.in); got != tc.want {
			t.Errorf("CollapseWhitespace(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestNormalize(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"identical", "Hello World", "hello world"},
		{"accents", "Björk", "bjork"},
		{"remastered paren", "Come Together (Remastered 2009)", "come together"},
		{"remastered dash", "Come Together - Remastered 2009", "come together"},
		{"remaster paren", "Come Together (Remaster)", "come together"},
		{"explicit paren preserved", "HUMBLE. (Explicit)", "humble explicit"},
		{"clean paren preserved", "HUMBLE. (Clean)", "humble clean"},
		{"explicit version preserved", "HUMBLE. (Explicit Version)", "humble explicit version"},
		{"deluxe paren", "1989 (Deluxe Edition)", "1989"},
		{"deluxe dash", "1989 - Deluxe Edition", "1989"},
		{"anniversary", "Thriller (25th Anniversary Edition)", "thriller"},
		{"stacked suffixes", "Song (Remastered) - 2011 Remaster", "song"},
		{"live preserved", "Wonderwall (Live at Wembley)", "wonderwall live at wembley"},
		{"acoustic preserved", "Shape of You (Acoustic)", "shape of you acoustic"},
		{"remix preserved", "Take On Me (Remix)", "take on me remix"},
		{"punctuation", "P!nk", "p nk"},
		{"apostrophe", "Don't Stop Me Now", "don t stop me now"},
		{"ac/dc", "AC/DC", "ac dc"},
		{"fancy quotes", "\u201cHello\u201d", "hello"},
		{"multiple spaces", "hello   world", "hello world"},
		{"digital remaster", "Bohemian Rhapsody - Digital Remaster", "bohemian rhapsody"},
		{"brackets explicit preserved", "Song [Explicit]", "song explicit"},
		{"year remaster", "Song (2011 Remaster)", "song"},
		{"strip leading the", "The Beatles", "beatles"},
		{"keep the in middle", "The Dark Side of the Moon", "dark side of the moon"},
		{"just the alone", "The", "the"},
		{"The The", "The The", "the"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := Normalize(tc.in); got != tc.want {
				t.Errorf("Normalize(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestNormalizeIdempotent(t *testing.T) {
	t.Parallel()
	inputs := []string{
		"Hello World",
		"Björk - Army of Me",
		"Beyoncé (Deluxe)",
		"Song (Remastered 2011)",
		"Don't Stop Me Now",
		"",
		"   ",
		"AC/DC - Back in Black",
	}
	for _, in := range inputs {
		first := Normalize(in)
		second := Normalize(first)
		if first != second {
			t.Errorf("not idempotent: Normalize(%q)=%q, Normalize again=%q", in, first, second)
		}
	}
}

func FuzzNormalizeIdempotent(f *testing.F) {
	f.Add("Hello")
	f.Add("Björk - Army of Me")
	f.Add("Song (Remastered)")
	f.Add("")
	f.Fuzz(func(t *testing.T, s string) {
		first := Normalize(s)
		second := Normalize(first)
		if first != second {
			t.Errorf("not idempotent: Normalize(%q)=%q, Normalize again=%q", s, first, second)
		}
	})
}
