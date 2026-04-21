package matchify

import (
	"context"
	"testing"
	"time"
)

func mustDate(year int) time.Time {
	return time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
}

func TestAlbumMatcher_AuthoritativeIDs(t *testing.T) {
	t.Parallel()
	m := NewAlbumMatcher(AlbumMatcherOptions{})
	ctx := context.Background()

	t.Run("upc match short-circuits", func(t *testing.T) {
		a := Album{Name: "x", UPC: "012345678905"}
		b := Album{Name: "y", UPC: "012345678905"}
		if s := m.Match(ctx, a, b); s.Value < 0.99 {
			t.Errorf("UPC match: got %v", s)
		}
	})

	t.Run("mbid match short-circuits", func(t *testing.T) {
		a := Album{Name: "x", MBID: "abc"}
		b := Album{Name: "y", MBID: "abc"}
		if s := m.Match(ctx, a, b); s.Value < 0.99 {
			t.Errorf("MBID match: got %v", s)
		}
	})

	t.Run("upc mismatch caps the score", func(t *testing.T) {
		a := Album{
			Name:        "1989",
			Artists:     []Artist{{Name: "Taylor Swift"}},
			UPC:         "111",
			ReleaseDate: mustDate(2014),
		}
		b := Album{
			Name:        "1989",
			Artists:     []Artist{{Name: "Taylor Swift"}},
			UPC:         "222",
			ReleaseDate: mustDate(2014),
		}
		if s := m.Match(ctx, a, b); s.Value > 0.4 {
			t.Errorf("expected UPC mismatch to cap, got %v", s)
		}
	})
}

func TestAlbumMatcher_CrossPlatformSame(t *testing.T) {
	t.Parallel()
	m := NewAlbumMatcher(AlbumMatcherOptions{})
	ctx := context.Background()

	cases := []struct {
		name string
		a, b Album
	}{
		{
			name: "plain vs remastered",
			a: Album{
				Name:        "Dark Side of the Moon",
				Artists:     []Artist{{Name: "Pink Floyd"}},
				ReleaseDate: mustDate(1973),
				TrackCount:  10,
			},
			b: Album{
				Name:        "The Dark Side of the Moon (2011 Remastered Version)",
				Artists:     []Artist{{Name: "Pink Floyd"}},
				ReleaseDate: mustDate(1973),
				TrackCount:  10,
			},
		},
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
				Name:        "Random Access Memories",
				Artists:     []Artist{{Name: "Daft Punk"}},
				ReleaseDate: mustDate(2013),
			},
			b: Album{
				Name:        "Random Access Memories",
				Artists:     []Artist{{Name: "Daft Punk"}},
				ReleaseDate: mustDate(2014),
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if s := m.Match(ctx, tc.a, tc.b); !s.Above(DefaultAlbumThreshold) {
				t.Errorf("expected match, got %v", s)
			}
		})
	}
}

func TestAlbumMatcher_DeluxeEditionPenalty(t *testing.T) {
	t.Parallel()
	m := NewAlbumMatcher(AlbumMatcherOptions{})
	ctx := context.Background()

	a := Album{
		Name:        "1989",
		Artists:     []Artist{{Name: "Taylor Swift"}},
		ReleaseDate: mustDate(2014),
	}
	b := Album{
		Name:        "1989 (Deluxe Edition)",
		Artists:     []Artist{{Name: "Taylor Swift"}},
		ReleaseDate: mustDate(2014),
	}
	aa := Album{
		Name:        "1989 (Deluxe Edition)",
		Artists:     []Artist{{Name: "Taylor Swift"}},
		ReleaseDate: mustDate(2014),
	}

	plainVsDeluxe := m.Match(ctx, a, b)
	deluxeVsDeluxe := m.Match(ctx, b, aa)

	if !(deluxeVsDeluxe.Value > plainVsDeluxe.Value) {
		t.Errorf("deluxe vs deluxe should score higher than plain vs deluxe, got %v vs %v",
			deluxeVsDeluxe, plainVsDeluxe)
	}
	// Plain vs deluxe should still be above threshold — they're the same album family.
	if !plainVsDeluxe.Above(0.7) {
		t.Errorf("plain vs deluxe should be reasonable match, got %v", plainVsDeluxe)
	}
}

func TestAlbumMatcher_DisableEditionPenalty(t *testing.T) {
	t.Parallel()
	plain := Album{Name: "1989", Artists: []Artist{{Name: "Taylor Swift"}}}
	deluxe := Album{Name: "1989 (Deluxe Edition)", Artists: []Artist{{Name: "Taylor Swift"}}}
	ctx := context.Background()

	with := NewAlbumMatcher(AlbumMatcherOptions{})
	without := NewAlbumMatcher(AlbumMatcherOptions{DisableEditionPenalty: true})

	swith := with.Match(ctx, plain, deluxe).Value
	swithout := without.Match(ctx, plain, deluxe).Value

	if swithout <= swith {
		t.Errorf("DisableEditionPenalty should eliminate the penalty: with=%v, without=%v", swith, swithout)
	}
	if delta := swithout - swith; delta < 0.09 || delta > 0.11 {
		t.Errorf("expected penalty ≈ 0.1, got delta=%v (with=%v, without=%v)", delta, swith, swithout)
	}
}

func TestAlbumMatcher_DifferentAlbumsDoNotMatch(t *testing.T) {
	t.Parallel()
	m := NewAlbumMatcher(AlbumMatcherOptions{})
	ctx := context.Background()

	a := Album{Name: "1989", Artists: []Artist{{Name: "Taylor Swift"}}}
	b := Album{Name: "Red", Artists: []Artist{{Name: "Taylor Swift"}}}
	if s := m.Match(ctx, a, b); s.Above(DefaultAlbumThreshold) {
		t.Errorf("different albums same artist should not match, got %v", s)
	}
}
