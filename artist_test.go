package matchify

import (
	"context"
	"errors"
	"fmt"
	"testing"
)

func TestArtistMatcher_AuthoritativeIDs(t *testing.T) {
	t.Parallel()
	m := NewArtistMatcher(ArtistMatcherOptions{})
	ctx := context.Background()

	t.Run("mbid match", func(t *testing.T) {
		a := Artist{Name: "Alpha", MBID: "id"}
		b := Artist{Name: "Beta", MBID: "id"}
		if s := m.Match(ctx, a, b); s.Value < 0.99 {
			t.Errorf("MBID match: got %v", s)
		}
	})

	t.Run("platform id match", func(t *testing.T) {
		a := Artist{Name: "x", ExternalIDs: map[Platform]string{PlatformSpotify: "s1"}}
		b := Artist{Name: "y", ExternalIDs: map[Platform]string{PlatformSpotify: "s1"}}
		if s := m.Match(ctx, a, b); s.Value < 0.99 {
			t.Errorf("platform match: got %v", s)
		}
	})

	t.Run("platform id mismatch caps", func(t *testing.T) {
		a := Artist{Name: "Taylor Swift", ExternalIDs: map[Platform]string{PlatformSpotify: "s1"}}
		b := Artist{Name: "Taylor Swift", ExternalIDs: map[Platform]string{PlatformSpotify: "s2"}}
		if s := m.Match(ctx, a, b); s.Value > 0.4 {
			t.Errorf("platform ID mismatch should cap: got %v", s)
		}
	})
}

func TestArtistMatcher_NameSimilarity(t *testing.T) {
	t.Parallel()
	m := NewArtistMatcher(ArtistMatcherOptions{})
	ctx := context.Background()

	cases := []struct {
		name        string
		a, b        Artist
		shouldMatch bool
	}{
		{"identical", Artist{Name: "The Beatles"}, Artist{Name: "The Beatles"}, true},
		{"accents", Artist{Name: "Björk"}, Artist{Name: "Bjork"}, true},
		{"the prefix", Artist{Name: "The Beatles"}, Artist{Name: "Beatles"}, true},
		{"punctuation", Artist{Name: "P!nk"}, Artist{Name: "Pink"}, true},
		{"alias", Artist{Name: "The Weeknd", Aliases: []string{"Weeknd"}}, Artist{Name: "Weeknd"}, true},
		{"completely different", Artist{Name: "Beyoncé"}, Artist{Name: "Taylor Swift"}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			s := m.Match(ctx, tc.a, tc.b)
			if tc.shouldMatch && !s.Above(DefaultArtistThreshold) {
				t.Errorf("expected match, got %v", s)
			}
			if !tc.shouldMatch && s.Above(DefaultArtistThreshold) {
				t.Errorf("expected no match, got %v", s)
			}
		})
	}
}

type fakeProvider struct {
	releases map[string][]Album
	calls    int
	err      error
}

func (p *fakeProvider) Releases(ctx context.Context, artist Artist) ([]Album, error) {
	p.calls++
	if p.err != nil {
		return nil, p.err
	}
	return p.releases[artist.Name], nil
}

func TestArtistMatcher_ReleaseOverlap(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	// Two different name spellings that wouldn't match on name alone but
	// share an album. Provider returns each artist's releases.
	provider := &fakeProvider{
		releases: map[string][]Album{
			"Cardi B": {
				{Name: "Invasion of Privacy", Artists: []Artist{{Name: "Cardi B"}}},
				{Name: "WAP", Artists: []Artist{{Name: "Cardi B"}}},
			},
			"Kardi": {
				{Name: "Different Album", Artists: []Artist{{Name: "Kardi"}}},
				{Name: "Invasion of Privacy", Artists: []Artist{{Name: "Kardi"}}},
			},
		},
	}

	// Similar-ish names (Jaro-Winkler > 0.6, < 0.95) — release probe should kick in.
	a := Artist{Name: "Cardi B"}
	b := Artist{Name: "Kardi"}

	withProvider := NewArtistMatcher(ArtistMatcherOptions{
		ReleaseProvider: provider,
	})
	withoutProvider := NewArtistMatcher(ArtistMatcherOptions{})

	withRes := withProvider.Match(ctx, a, b)
	withoutRes := withoutProvider.Match(ctx, a, b)

	if withRes.Value <= withoutRes.Value {
		t.Errorf("provider should lift ambiguous score: with=%v, without=%v", withRes, withoutRes)
	}
}

func TestArtistMatcher_ReleaseProviderCache(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	provider := &fakeProvider{
		releases: map[string][]Album{
			"A": {{Name: "Album 1"}},
			"B": {{Name: "Album 1"}},
		},
	}
	m := NewArtistMatcher(ArtistMatcherOptions{
		ReleaseProvider: provider,
	})

	a := Artist{Name: "A"}
	b := Artist{Name: "B"}

	// Multiple calls shouldn't re-fetch the same artist.
	m.Match(ctx, a, b)
	firstCount := provider.calls
	m.Match(ctx, a, b)
	if provider.calls != firstCount {
		t.Errorf("cache miss on repeat call: was %d, now %d", firstCount, provider.calls)
	}
}

func TestArtistMatcher_ReleaseProviderError(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	provider := &fakeProvider{err: errors.New("boom")}
	m := NewArtistMatcher(ArtistMatcherOptions{ReleaseProvider: provider})

	// Ambiguous name match — provider will fail, matcher should fall back
	// to name signal and not crash.
	a := Artist{Name: "Cardi B"}
	b := Artist{Name: "Kardi"}
	s := m.Match(ctx, a, b)
	if s.Value > DefaultArtistThreshold {
		t.Errorf("fallback to name failure should not pass threshold: %v", s)
	}
}

func TestArtistCacheKey_Stability(t *testing.T) {
	t.Parallel()
	// MBID key should be stable regardless of other fields.
	a := Artist{Name: "A", MBID: "abc", ExternalIDs: map[Platform]string{PlatformSpotify: "s1"}}
	b := Artist{Name: "B", MBID: "abc"}
	if artistCacheKey(a) != artistCacheKey(b) {
		t.Errorf("MBID should dominate cache key")
	}

	// External IDs key should be order-insensitive.
	c := Artist{ExternalIDs: map[Platform]string{PlatformSpotify: "s", PlatformTidal: "t"}}
	d := Artist{ExternalIDs: map[Platform]string{PlatformTidal: "t", PlatformSpotify: "s"}}
	if artistCacheKey(c) != artistCacheKey(d) {
		t.Errorf("external ID key should be deterministic, got %q vs %q",
			artistCacheKey(c), artistCacheKey(d))
	}

	// Name-only key should normalise.
	e := Artist{Name: "Björk"}
	f := Artist{Name: "Bjork"}
	if artistCacheKey(e) != artistCacheKey(f) {
		t.Errorf("name key should normalise diacritics, got %q vs %q",
			artistCacheKey(e), artistCacheKey(f))
	}
}

// Example: release-overlap kicks in when the names are ambiguous.
func Example_artistMatcher_releaseProbe() {
	provider := ReleaseProviderFunc(func(ctx context.Context, a Artist) ([]Album, error) {
		switch a.Name {
		case "Maneskin":
			return []Album{{Name: "Teatro d'Ira"}}, nil
		case "Måneskin":
			return []Album{{Name: "Teatro d'Ira"}}, nil
		}
		return nil, nil
	})
	m := NewArtistMatcher(ArtistMatcherOptions{ReleaseProvider: provider})
	score := m.Match(context.Background(), Artist{Name: "Maneskin"}, Artist{Name: "Måneskin"})
	if score.Above(DefaultArtistThreshold) {
		fmt.Println("match")
	}
	// Output: match
}
