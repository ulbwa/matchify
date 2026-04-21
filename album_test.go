package matchify

import (
	"context"
	"testing"
	"time"
)

func releasedIn(year int) Tag {
	return WithReleaseDate(time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC))
}

func TestAlbumMatcher_AuthoritativeIDs(t *testing.T) {
	t.Parallel()
	m := NewAlbumMatcher()
	ctx := context.Background()

	t.Run("upc match short-circuits", func(t *testing.T) {
		a := Album{Name: "x", Tags: NewTags(WithUPC("012345678905"))}
		b := Album{Name: "y", Tags: NewTags(WithUPC("012345678905"))}
		if s := m.Match(ctx, a, b); !s.Same(0.99) {
			t.Errorf("UPC match: got %v", s)
		}
	})

	t.Run("mbid match short-circuits", func(t *testing.T) {
		a := Album{Name: "x", Tags: NewTags(WithMBID("abc"))}
		b := Album{Name: "y", Tags: NewTags(WithMBID("abc"))}
		if s := m.Match(ctx, a, b); !s.Same(0.99) {
			t.Errorf("MBID match: got %v", s)
		}
	})

	t.Run("upc mismatch marks unrelated", func(t *testing.T) {
		a := Album{
			Name:    "1989",
			Artists: []Artist{{Name: "Taylor Swift"}},
			Tags:    NewTags(WithUPC("111"), releasedIn(2014)),
		}
		b := Album{
			Name:    "1989",
			Artists: []Artist{{Name: "Taylor Swift"}},
			Tags:    NewTags(WithUPC("222"), releasedIn(2014)),
		}
		s := m.Match(ctx, a, b)
		if s.Relation != RelationUnrelated {
			t.Errorf("expected Unrelated on UPC mismatch, got %v", s)
		}
	})
}

func TestAlbumMatcher_CrossPlatformSame(t *testing.T) {
	t.Parallel()
	m := NewAlbumMatcher()
	ctx := context.Background()

	cases := []struct {
		name string
		a, b Album
	}{
		{
			name: "accented name",
			a: Album{
				Name:    "Björk - Homogenic",
				Artists: []Artist{{Name: "Björk"}},
			},
			b: Album{
				Name:    "Bjork - Homogenic",
				Artists: []Artist{{Name: "Bjork"}},
			},
		},
		{
			name: "year difference of 1",
			a: Album{
				Name:    "Random Access Memories",
				Artists: []Artist{{Name: "Daft Punk"}},
				Tags:    NewTags(releasedIn(2013)),
			},
			b: Album{
				Name:    "Random Access Memories",
				Artists: []Artist{{Name: "Daft Punk"}},
				Tags:    NewTags(releasedIn(2014)),
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			s := m.Match(ctx, tc.a, tc.b)
			if !s.Same(DefaultAlbumThreshold) {
				t.Errorf("expected Same, got %v", s)
			}
		})
	}
}

// TestAlbumMatcher_PlainVsRemasteredIsVariant documents the deliberate
// choice that a remastered release is a *variant* of the original, not
// the same product. Matching these would accidentally merge distinct
// masters in a cross-platform library.
func TestAlbumMatcher_PlainVsRemasteredIsVariant(t *testing.T) {
	t.Parallel()
	m := NewAlbumMatcher()
	ctx := context.Background()

	a := Album{
		Name:    "Dark Side of the Moon",
		Artists: []Artist{{Name: "Pink Floyd"}},
		Tags:    NewTags(releasedIn(1973), WithTrackCount(10)),
	}
	b := Album{
		Name:    "The Dark Side of the Moon (2011 Remastered Version)",
		Artists: []Artist{{Name: "Pink Floyd"}},
		Tags:    NewTags(releasedIn(1973), WithTrackCount(10)),
	}
	s := m.Match(ctx, a, b)
	if s.Relation != RelationVariant {
		t.Errorf("plain vs remastered should be RelationVariant, got %v", s)
	}
	if s.Same(DefaultAlbumThreshold) {
		t.Errorf("plain vs remastered should NOT be Same, got %v", s)
	}
	// They should still be related with confidence.
	if !s.Related(DefaultAlbumThreshold) {
		t.Errorf("plain vs remastered should be Related, got %v", s)
	}
}

func TestAlbumMatcher_DeluxeEditionRelation(t *testing.T) {
	t.Parallel()
	m := NewAlbumMatcher()
	ctx := context.Background()

	plain := Album{
		Name:    "1989",
		Artists: []Artist{{Name: "Taylor Swift"}},
		Tags:    NewTags(releasedIn(2014)),
	}
	deluxe := Album{
		Name:    "1989 (Deluxe Edition)",
		Artists: []Artist{{Name: "Taylor Swift"}},
		Tags:    NewTags(releasedIn(2014)),
	}
	deluxe2 := Album{
		Name:    "1989 (Deluxe Edition)",
		Artists: []Artist{{Name: "Taylor Swift"}},
		Tags:    NewTags(releasedIn(2014)),
	}

	if s := m.Match(ctx, deluxe, deluxe2); !s.Same(DefaultAlbumThreshold) {
		t.Errorf("deluxe vs deluxe should be Same, got %v", s)
	}
	s := m.Match(ctx, plain, deluxe)
	if s.Relation != RelationVariant {
		t.Errorf("plain vs deluxe should be Variant, got %v", s)
	}
	if s.Same(DefaultAlbumThreshold) {
		t.Errorf("plain vs deluxe should NOT be Same, got %v", s)
	}
	if !s.Related(DefaultAlbumThreshold) {
		t.Errorf("plain vs deluxe should be Related (same family), got %v", s)
	}
}

func TestAlbumMatcher_Unrelated(t *testing.T) {
	t.Parallel()
	m := NewAlbumMatcher()
	ctx := context.Background()

	a := Album{Name: "1989", Artists: []Artist{{Name: "Taylor Swift"}}}
	b := Album{Name: "Red", Artists: []Artist{{Name: "Taylor Swift"}}}
	if s := m.Match(ctx, a, b); s.Same(DefaultAlbumThreshold) {
		t.Errorf("different albums same artist should not match, got %v", s)
	}
}

func TestAlbumMatcher_ExplicitMismatch(t *testing.T) {
	t.Parallel()
	m := NewAlbumMatcher()
	ctx := context.Background()

	a := Album{Name: "Album", Artists: []Artist{{Name: "Artist"}},
		Tags: NewTags(WithExplicit(ExplicitnessExplicit))}
	b := Album{Name: "Album", Artists: []Artist{{Name: "Artist"}},
		Tags: NewTags(WithExplicit(ExplicitnessClean))}
	s := m.Match(ctx, a, b)
	if s.Same(DefaultAlbumThreshold) {
		t.Errorf("explicit vs clean albums should not match as Same, got %v", s)
	}
	if s.Relation != RelationVariant {
		t.Errorf("explicit vs clean should be Variant, got %v", s)
	}
}
