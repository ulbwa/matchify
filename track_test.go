package matchify

import (
	"context"
	"testing"
	"time"
)

func TestTrackMatcher_AuthoritativeIDs(t *testing.T) {
	t.Parallel()
	m := NewTrackMatcher()
	ctx := context.Background()

	t.Run("isrc match short-circuits", func(t *testing.T) {
		a := Track{Name: "Entirely Different", Tags: NewTags(WithISRC("USUM71703861"))}
		b := Track{Name: "Also Different", Tags: NewTags(WithISRC("USUM71703861"))}
		if s := m.Match(ctx, a, b); !s.Same(0.99) {
			t.Errorf("expected ISRC match to return Same~1, got %v", s)
		}
	})

	t.Run("mbid match short-circuits", func(t *testing.T) {
		a := Track{Name: "x", Tags: NewTags(WithMBID("abc-123"))}
		b := Track{Name: "y", Tags: NewTags(WithMBID("abc-123"))}
		if s := m.Match(ctx, a, b); !s.Same(0.99) {
			t.Errorf("expected MBID match: got %v", s)
		}
	})

	t.Run("platform id match short-circuits", func(t *testing.T) {
		a := Track{Name: "x", Tags: NewTags(WithPlatformID(PlatformSpotify, "abc"))}
		b := Track{Name: "y", Tags: NewTags(WithPlatformID(PlatformSpotify, "abc"))}
		if s := m.Match(ctx, a, b); !s.Same(0.99) {
			t.Errorf("expected platform ID match: got %v", s)
		}
	})

	t.Run("isrc mismatch marks unrelated", func(t *testing.T) {
		a := Track{
			Name:    "Shape of You",
			Artists: []Artist{{Name: "Ed Sheeran"}},
			Tags:    NewTags(WithISRC("GBAHS1700024")),
		}
		b := Track{
			Name:    "Shape of You",
			Artists: []Artist{{Name: "Ed Sheeran"}},
			Tags:    NewTags(WithISRC("GBAHS9999999")),
		}
		s := m.Match(ctx, a, b)
		if s.Relation != RelationUnrelated {
			t.Errorf("expected Unrelated on ISRC mismatch, got %v", s)
		}
		if s.Value > 0.4 {
			t.Errorf("expected score capped at 0.4, got %v", s.Value)
		}
	})
}

func TestTrackMatcher_CrossPlatformSameTrack(t *testing.T) {
	t.Parallel()
	m := NewTrackMatcher()
	ctx := context.Background()

	cases := []struct {
		name string
		a, b Track
	}{
		{
			name: "spotify feat in artists vs apple music feat in title",
			a: Track{
				Name:    "Despacito",
				Artists: []Artist{{Name: "Luis Fonsi"}, {Name: "Daddy Yankee"}, {Name: "Justin Bieber"}},
				Tags:    NewTags(WithDuration(228 * time.Second)),
			},
			b: Track{
				Name:    "Despacito (feat. Justin Bieber)",
				Artists: []Artist{{Name: "Luis Fonsi"}, {Name: "Daddy Yankee"}},
				Tags:    NewTags(WithDuration(229 * time.Second)),
			},
		},
		{
			name: "ft. vs feat.",
			a: Track{
				Name:    "Mi Gente (ft. Beyoncé)",
				Artists: []Artist{{Name: "J Balvin"}, {Name: "Willy William"}},
				Tags:    NewTags(WithDuration(204 * time.Second)),
			},
			b: Track{
				Name:    "Mi Gente (feat. Beyonce)",
				Artists: []Artist{{Name: "J Balvin"}, {Name: "Willy William"}},
				Tags:    NewTags(WithDuration(204 * time.Second)),
			},
		},
		{
			name: "remastered suffix",
			a: Track{
				Name:    "Come Together",
				Artists: []Artist{{Name: "The Beatles"}},
				Tags:    NewTags(WithDuration(259 * time.Second)),
			},
			b: Track{
				Name:    "Come Together - Remastered 2009",
				Artists: []Artist{{Name: "The Beatles"}},
				Tags:    NewTags(WithDuration(259 * time.Second)),
			},
		},
		{
			name: "accented artist",
			a: Track{
				Name:    "Partition",
				Artists: []Artist{{Name: "Beyoncé"}},
				Tags:    NewTags(WithDuration(317 * time.Second)),
			},
			b: Track{
				Name:    "Partition",
				Artists: []Artist{{Name: "Beyonce"}},
				Tags:    NewTags(WithDuration(317 * time.Second)),
			},
		},
		{
			name: "typographic apostrophe",
			a: Track{
				Name:    "Don't Stop Me Now",
				Artists: []Artist{{Name: "Queen"}},
				Tags:    NewTags(WithDuration(209 * time.Second)),
			},
			b: Track{
				Name:    "Don\u2019t Stop Me Now",
				Artists: []Artist{{Name: "Queen"}},
				Tags:    NewTags(WithDuration(209 * time.Second)),
			},
		},
		{
			name: "deezer composite feat in single artist entry",
			a: Track{
				Name:    "Despacito",
				Artists: []Artist{{Name: "Luis Fonsi feat. Daddy Yankee"}},
				Tags:    NewTags(WithDuration(228 * time.Second)),
			},
			b: Track{
				Name:    "Despacito",
				Artists: []Artist{{Name: "Luis Fonsi"}, {Name: "Daddy Yankee"}},
				Tags:    NewTags(WithDuration(228 * time.Second)),
			},
		},
		{
			name: "ampersand composite",
			a: Track{
				Name:    "Tokyo Drifting",
				Artists: []Artist{{Name: "Glass Animals & Denzel Curry"}},
				Tags:    NewTags(WithDuration(210 * time.Second)),
			},
			b: Track{
				Name:    "Tokyo Drifting",
				Artists: []Artist{{Name: "Glass Animals"}, {Name: "Denzel Curry"}},
				Tags:    NewTags(WithDuration(210 * time.Second)),
			},
		},
		{
			name: "both explicit via tag and via title",
			a: Track{
				Name:    "HUMBLE.",
				Artists: []Artist{{Name: "Kendrick Lamar"}},
				Tags:    NewTags(WithExplicit(ExplicitnessExplicit), WithDuration(177*time.Second)),
			},
			b: Track{
				Name:    "HUMBLE. (Explicit)",
				Artists: []Artist{{Name: "Kendrick Lamar"}},
				Tags:    NewTags(WithDuration(177 * time.Second)),
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			s := m.Match(ctx, tc.a, tc.b)
			if !s.Same(DefaultTrackThreshold) {
				t.Errorf("expected Same match, got %v", s)
			}
		})
	}
}

func TestTrackMatcher_DifferentRecordingVariants(t *testing.T) {
	t.Parallel()
	m := NewTrackMatcher()
	ctx := context.Background()

	cases := []struct {
		name string
		a, b Track
		// Expected: variant (same song, different recording) — not Same,
		// but optionally related.
	}{
		{
			name: "studio vs live",
			a: Track{
				Name:    "Bohemian Rhapsody",
				Artists: []Artist{{Name: "Queen"}},
				Tags:    NewTags(WithDuration(355 * time.Second)),
			},
			b: Track{
				Name:    "Bohemian Rhapsody (Live at Wembley '86)",
				Artists: []Artist{{Name: "Queen"}},
				Tags:    NewTags(WithDuration(348 * time.Second)),
			},
		},
		{
			name: "studio vs acoustic",
			a: Track{
				Name:    "Shape of You",
				Artists: []Artist{{Name: "Ed Sheeran"}},
				Tags:    NewTags(WithDuration(233 * time.Second)),
			},
			b: Track{
				Name:    "Shape of You (Acoustic)",
				Artists: []Artist{{Name: "Ed Sheeran"}},
				Tags:    NewTags(WithDuration(233 * time.Second)),
			},
		},
		{
			name: "studio vs remix",
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
			name: "explicit vs clean",
			a: Track{
				Name:    "HUMBLE.",
				Artists: []Artist{{Name: "Kendrick Lamar"}},
				Tags:    NewTags(WithExplicit(ExplicitnessExplicit), WithDuration(177*time.Second)),
			},
			b: Track{
				Name:    "HUMBLE. (Clean)",
				Artists: []Artist{{Name: "Kendrick Lamar"}},
				Tags:    NewTags(WithDuration(177 * time.Second)),
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			s := m.Match(ctx, tc.a, tc.b)
			if s.Same(DefaultTrackThreshold) {
				t.Errorf("expected not Same, got %v", s)
			}
			if s.Relation != RelationVariant {
				t.Errorf("expected RelationVariant, got %v", s)
			}
		})
	}
}

func TestTrackMatcher_Unrelated(t *testing.T) {
	t.Parallel()
	m := NewTrackMatcher()
	ctx := context.Background()

	cases := []struct {
		name string
		a, b Track
	}{
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
				Name:    "Intro",
				Artists: []Artist{{Name: "Artist"}},
				Tags:    NewTags(WithDuration(30 * time.Second)),
			},
			b: Track{
				Name:    "Intro",
				Artists: []Artist{{Name: "Artist"}},
				Tags:    NewTags(WithDuration(360 * time.Second)),
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			s := m.Match(ctx, tc.a, tc.b)
			if s.Same(DefaultTrackThreshold) {
				t.Errorf("expected not Same, got %v", s)
			}
		})
	}
}

// TestTrackMatcher_FractionalDurationMismatch asserts that fractional
// DurationMismatchSeconds values aren't truncated by the Duration
// conversion. Regression test for earlier review feedback.
func TestTrackMatcher_FractionalDurationMismatch(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	// With a 2.5s cap, two tracks 2s apart should NOT trip the cap. If
	// the float were truncated to 2s, they would.
	m := NewTrackMatcher(TrackDurationMismatch(2.5, 0.1))
	a := Track{
		Name:    "Song",
		Artists: []Artist{{Name: "Artist"}},
		Tags:    NewTags(WithDuration(180 * time.Second)),
	}
	b := Track{
		Name:    "Song",
		Artists: []Artist{{Name: "Artist"}},
		Tags:    NewTags(WithDuration(182 * time.Second)),
	}
	s := m.Match(ctx, a, b)
	if s.Value <= 0.5 {
		t.Errorf("fractional cap truncated; expected no cap for 2s diff with 2.5s limit, got %v", s)
	}
}

func TestTrackMatcher_MinimalData(t *testing.T) {
	t.Parallel()
	m := NewTrackMatcher()
	ctx := context.Background()

	// SoundCloud-like case: just name + artist name.
	a := Track{Name: "Thriller", Artists: []Artist{{Name: "Michael Jackson"}}}
	b := Track{Name: "Thriller", Artists: []Artist{{Name: "Michael Jackson"}}}
	s := m.Match(ctx, a, b)
	if !s.Same(DefaultTrackThreshold) {
		t.Errorf("expected match on bare name+artist, got %v", s)
	}
}

func TestTrackMatcher_OptionsApplied(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	a := Track{Name: "X", Artists: []Artist{{Name: "A"}}}
	b := Track{Name: "Y", Artists: []Artist{{Name: "B"}}}

	m := NewTrackMatcher(
		TrackNameWeight(10),
		TrackArtistWeight(0),
	)
	s := m.Match(ctx, a, b)
	// With artist weight zero, only the name dominates; different names
	// yield a low score. Ensure options actually reach the config.
	if s.Value > 0.8 {
		t.Errorf("options not applied: got score %v", s)
	}
}
