package matchify

import (
	"context"
	"testing"
	"time"
)

// Track test cases are structured as cross-platform pairs of the same or
// different tracks. Names use the real conventions each platform ships
// (Spotify keeps features in the artist array, Apple Music tends to inline
// them, Tidal/Qobuz vary). The matcher should treat these as the same
// track without the caller doing per-platform preprocessing.

func TestTrackMatcher_AuthoritativeIDs(t *testing.T) {
	t.Parallel()
	m := NewTrackMatcher(TrackMatcherOptions{})

	t.Run("isrc match short-circuits", func(t *testing.T) {
		a := Track{Name: "Entirely Different", ISRC: "USUM71703861"}
		b := Track{Name: "Also Different", ISRC: "USUM71703861"}
		s := m.Match(context.Background(), a, b)
		if s.Value < 0.99 {
			t.Errorf("expected ISRC match to return ~1, got %v", s)
		}
	})

	t.Run("mbid match short-circuits", func(t *testing.T) {
		a := Track{Name: "x", MBID: "abc-123"}
		b := Track{Name: "y", MBID: "abc-123"}
		s := m.Match(context.Background(), a, b)
		if s.Value < 0.99 {
			t.Errorf("expected MBID match to return ~1, got %v", s)
		}
	})

	t.Run("platform id match short-circuits", func(t *testing.T) {
		a := Track{Name: "x", ExternalIDs: map[Platform]string{PlatformSpotify: "abc"}}
		b := Track{Name: "y", ExternalIDs: map[Platform]string{PlatformSpotify: "abc"}}
		s := m.Match(context.Background(), a, b)
		if s.Value < 0.99 {
			t.Errorf("expected platform ID match to return ~1, got %v", s)
		}
	})

	t.Run("isrc mismatch caps the score", func(t *testing.T) {
		// Strong text agreement but ISRCs disagree — should be capped below
		// the ISRCMismatchCap default (0.4).
		a := Track{
			Name:    "Shape of You",
			Artists: []Artist{{Name: "Ed Sheeran"}},
			ISRC:    "GBAHS1700024",
		}
		b := Track{
			Name:    "Shape of You",
			Artists: []Artist{{Name: "Ed Sheeran"}},
			ISRC:    "GBAHS9999999",
		}
		s := m.Match(context.Background(), a, b)
		if s.Value > 0.4 {
			t.Errorf("expected ISRC mismatch to cap score <= 0.4, got %v", s)
		}
	})
}

func TestTrackMatcher_CrossPlatformSameTrack(t *testing.T) {
	t.Parallel()
	m := NewTrackMatcher(TrackMatcherOptions{})
	ctx := context.Background()

	cases := []struct {
		name string
		a, b Track
	}{
		{
			name: "spotify vs apple music — feat in title vs feat in artists",
			a: Track{
				Name:     "Despacito",
				Artists:  []Artist{{Name: "Luis Fonsi"}, {Name: "Daddy Yankee"}, {Name: "Justin Bieber"}},
				Duration: 228 * time.Second,
			},
			b: Track{
				Name:     "Despacito (feat. Justin Bieber)",
				Artists:  []Artist{{Name: "Luis Fonsi"}, {Name: "Daddy Yankee"}},
				Duration: 229 * time.Second,
			},
		},
		{
			name: "ft. vs feat.",
			a: Track{
				Name:     "Mi Gente (ft. Beyoncé)",
				Artists:  []Artist{{Name: "J Balvin"}, {Name: "Willy William"}},
				Duration: 204 * time.Second,
			},
			b: Track{
				Name:     "Mi Gente (feat. Beyonce)",
				Artists:  []Artist{{Name: "J Balvin"}, {Name: "Willy William"}},
				Duration: 204 * time.Second,
			},
		},
		{
			name: "remastered suffix",
			a: Track{
				Name:     "Come Together",
				Artists:  []Artist{{Name: "The Beatles"}},
				Duration: 259 * time.Second,
			},
			b: Track{
				Name:     "Come Together - Remastered 2009",
				Artists:  []Artist{{Name: "The Beatles"}},
				Duration: 259 * time.Second,
			},
		},
		{
			name: "accented artist",
			a: Track{
				Name:     "Partition",
				Artists:  []Artist{{Name: "Beyoncé"}},
				Duration: 317 * time.Second,
			},
			b: Track{
				Name:     "Partition",
				Artists:  []Artist{{Name: "Beyonce"}},
				Duration: 317 * time.Second,
			},
		},
		{
			name: "both explicit agree",
			a: Track{
				Name:     "HUMBLE.",
				Artists:  []Artist{{Name: "Kendrick Lamar"}},
				Explicit: ExplicitnessExplicit,
				Duration: 177 * time.Second,
			},
			b: Track{
				Name:     "HUMBLE. (Explicit)",
				Artists:  []Artist{{Name: "Kendrick Lamar"}},
				Duration: 177 * time.Second,
			},
		},
		{
			name: "both clean agree",
			a: Track{
				Name:     "HUMBLE. (Clean)",
				Artists:  []Artist{{Name: "Kendrick Lamar"}},
				Duration: 177 * time.Second,
			},
			b: Track{
				Name:     "HUMBLE. (Clean Version)",
				Artists:  []Artist{{Name: "Kendrick Lamar"}},
				Explicit: ExplicitnessClean,
				Duration: 177 * time.Second,
			},
		},
		{
			name: "typographic apostrophe",
			a: Track{
				Name:     "Don't Stop Me Now",
				Artists:  []Artist{{Name: "Queen"}},
				Duration: 209 * time.Second,
			},
			b: Track{
				Name:     "Don\u2019t Stop Me Now",
				Artists:  []Artist{{Name: "Queen"}},
				Duration: 209 * time.Second,
			},
		},
		{
			name: "multiple features reordered",
			a: Track{
				Name:    "Lean On",
				Artists: []Artist{{Name: "Major Lazer"}, {Name: "DJ Snake"}, {Name: "MØ"}},
			},
			b: Track{
				Name:    "Lean On (feat. MØ & DJ Snake)",
				Artists: []Artist{{Name: "Major Lazer"}},
			},
		},
		{
			name: "duration tolerance 1 second",
			a: Track{
				Name:     "Song",
				Artists:  []Artist{{Name: "Artist"}},
				Duration: 180 * time.Second,
			},
			b: Track{
				Name:     "Song",
				Artists:  []Artist{{Name: "Artist"}},
				Duration: 181 * time.Second,
			},
		},
		{
			// Deezer-style composite: feat is inside the single artist
			// entry, not in the title or the artist list.
			name: "deezer feat in artist name vs spotify separate entries",
			a: Track{
				Name:     "Despacito",
				Artists:  []Artist{{Name: "Luis Fonsi feat. Daddy Yankee"}},
				Duration: 228 * time.Second,
			},
			b: Track{
				Name:     "Despacito",
				Artists:  []Artist{{Name: "Luis Fonsi"}, {Name: "Daddy Yankee"}},
				Duration: 228 * time.Second,
			},
		},
		{
			name: "ampersand composite vs separate entries",
			a: Track{
				Name:     "Tokyo Drifting",
				Artists:  []Artist{{Name: "Glass Animals & Denzel Curry"}},
				Duration: 210 * time.Second,
			},
			b: Track{
				Name:     "Tokyo Drifting",
				Artists:  []Artist{{Name: "Glass Animals"}, {Name: "Denzel Curry"}},
				Duration: 210 * time.Second,
			},
		},
		{
			name: "comma composite vs separate entries",
			a: Track{
				Name:     "No Role Modelz",
				Artists:  []Artist{{Name: "J. Cole, Young Thug"}},
				Duration: 287 * time.Second,
			},
			b: Track{
				Name:     "No Role Modelz",
				Artists:  []Artist{{Name: "J. Cole"}, {Name: "Young Thug"}},
				Duration: 287 * time.Second,
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			s := m.Match(ctx, tc.a, tc.b)
			if !s.Above(DefaultTrackThreshold) {
				t.Errorf("expected match for %s, got %v", tc.name, s)
			}
		})
	}
}

func TestTrackMatcher_ShouldNotMatch(t *testing.T) {
	t.Parallel()
	m := NewTrackMatcher(TrackMatcherOptions{})
	ctx := context.Background()

	cases := []struct {
		name string
		a, b Track
	}{
		{
			name: "live version is different recording",
			a: Track{
				Name:     "Bohemian Rhapsody",
				Artists:  []Artist{{Name: "Queen"}},
				Duration: 355 * time.Second,
			},
			b: Track{
				Name:     "Bohemian Rhapsody (Live at Wembley '86)",
				Artists:  []Artist{{Name: "Queen"}},
				Duration: 348 * time.Second,
			},
		},
		{
			name: "acoustic version is different recording",
			a: Track{
				Name:     "Shape of You",
				Artists:  []Artist{{Name: "Ed Sheeran"}},
				Duration: 233 * time.Second,
			},
			b: Track{
				Name:     "Shape of You (Acoustic)",
				Artists:  []Artist{{Name: "Ed Sheeran"}},
				Duration: 233 * time.Second,
			},
		},
		{
			name: "remix is different recording",
			a: Track{
				Name:    "Take On Me",
				Artists: []Artist{{Name: "a-ha"}},
			},
			b: Track{
				Name:    "Take On Me (Remix)",
				Artists: []Artist{{Name: "a-ha"}},
			},
		},
		{
			name: "different artists same title",
			a: Track{
				Name:    "Hallelujah",
				Artists: []Artist{{Name: "Leonard Cohen"}},
			},
			b: Track{
				Name:    "Hallelujah",
				Artists: []Artist{{Name: "Jeff Buckley"}},
			},
		},
		{
			name: "wildly different duration",
			a: Track{
				Name:     "Intro",
				Artists:  []Artist{{Name: "Artist"}},
				Duration: 30 * time.Second,
			},
			b: Track{
				Name:     "Intro",
				Artists:  []Artist{{Name: "Artist"}},
				Duration: 360 * time.Second,
			},
		},
		{
			name: "explicit flag mismatch",
			a: Track{
				Name:     "HUMBLE.",
				Artists:  []Artist{{Name: "Kendrick Lamar"}},
				Explicit: ExplicitnessExplicit,
				Duration: 177 * time.Second,
			},
			b: Track{
				Name:     "HUMBLE.",
				Artists:  []Artist{{Name: "Kendrick Lamar"}},
				Explicit: ExplicitnessClean,
				Duration: 177 * time.Second,
			},
		},
		{
			name: "explicit field vs clean label",
			a: Track{
				Name:     "HUMBLE.",
				Artists:  []Artist{{Name: "Kendrick Lamar"}},
				Explicit: ExplicitnessExplicit,
				Duration: 177 * time.Second,
			},
			b: Track{
				Name:     "HUMBLE. (Clean)",
				Artists:  []Artist{{Name: "Kendrick Lamar"}},
				Duration: 177 * time.Second,
			},
		},
		{
			name: "explicit paren vs clean paren",
			a: Track{
				Name:     "HUMBLE. (Explicit)",
				Artists:  []Artist{{Name: "Kendrick Lamar"}},
				Duration: 177 * time.Second,
			},
			b: Track{
				Name:     "HUMBLE. (Clean)",
				Artists:  []Artist{{Name: "Kendrick Lamar"}},
				Duration: 177 * time.Second,
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			s := m.Match(ctx, tc.a, tc.b)
			if s.Above(DefaultTrackThreshold) {
				t.Errorf("expected non-match for %s, got %v", tc.name, s)
			}
		})
	}
}

// TestTrackMatcher_FractionalDurationMismatch asserts that fractional
// DurationMismatchSeconds values aren't truncated by the Duration
// conversion. Regression test for the review comment on track.go:168.
func TestTrackMatcher_FractionalDurationMismatch(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	// Two tracks 3s apart with a fractional 2.5s cap should trip the
	// cap (3 > 2.5); if the float were truncated to 2s, the check would
	// trip too, so invert: two tracks 2s apart should NOT trip a 2.5s
	// cap. If truncated to 2s, the 2s diff would trip.
	m := NewTrackMatcher(TrackMatcherOptions{
		DurationMismatchSeconds: 2.5,
		DurationMismatchCap:     0.1,
	})
	a := Track{
		Name:     "Song",
		Artists:  []Artist{{Name: "Artist"}},
		Duration: 180 * time.Second,
	}
	b := Track{
		Name:     "Song",
		Artists:  []Artist{{Name: "Artist"}},
		Duration: 182 * time.Second,
	}
	s := m.Match(ctx, a, b)
	if s.Value <= 0.5 {
		t.Errorf("fractional cap was likely truncated; expected no cap for 2s diff with 2.5s limit, got %v", s)
	}
}

func TestTrackMatcher_MissingFields(t *testing.T) {
	t.Parallel()
	m := NewTrackMatcher(TrackMatcherOptions{})
	ctx := context.Background()

	t.Run("only names", func(t *testing.T) {
		a := Track{Name: "Thriller", Artists: []Artist{{Name: "Michael Jackson"}}}
		b := Track{Name: "Thriller", Artists: []Artist{{Name: "Michael Jackson"}}}
		s := m.Match(ctx, a, b)
		if !s.Above(DefaultTrackThreshold) {
			t.Errorf("expected match on names only, got %v", s)
		}
	})

	t.Run("missing duration", func(t *testing.T) {
		a := Track{Name: "Thriller", Artists: []Artist{{Name: "Michael Jackson"}}}
		b := Track{Name: "Thriller", Artists: []Artist{{Name: "Michael Jackson"}}, Duration: 357 * time.Second}
		s := m.Match(ctx, a, b)
		if !s.Above(DefaultTrackThreshold) {
			t.Errorf("expected match with partial duration, got %v", s)
		}
	})
}
