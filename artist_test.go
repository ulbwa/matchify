package matchify

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestArtistMatcher_AuthoritativeIDs(t *testing.T) {
	t.Parallel()
	m := NewArtistMatcher()
	ctx := context.Background()

	t.Run("mbid match", func(t *testing.T) {
		a := Artist{Name: "Alpha", Tags: NewTags(WithMBID("id"))}
		b := Artist{Name: "Beta", Tags: NewTags(WithMBID("id"))}
		if s := m.Match(ctx, a, b); !s.Same(0.99) {
			t.Errorf("MBID match: got %v", s)
		}
	})

	t.Run("platform id match", func(t *testing.T) {
		a := Artist{Name: "x", Tags: NewTags(WithPlatformID(PlatformSpotify, "s1"))}
		b := Artist{Name: "y", Tags: NewTags(WithPlatformID(PlatformSpotify, "s1"))}
		if s := m.Match(ctx, a, b); !s.Same(0.99) {
			t.Errorf("platform match: got %v", s)
		}
	})

	t.Run("platform id mismatch caps", func(t *testing.T) {
		a := Artist{Name: "Taylor Swift", Tags: NewTags(WithPlatformID(PlatformSpotify, "s1"))}
		b := Artist{Name: "Taylor Swift", Tags: NewTags(WithPlatformID(PlatformSpotify, "s2"))}
		s := m.Match(ctx, a, b)
		if s.Relation != RelationUnrelated {
			t.Errorf("expected Unrelated, got %v", s)
		}
		if s.Value > 0.4 {
			t.Errorf("expected capped score, got %v", s.Value)
		}
	})
}

func TestArtistMatcher_NameSimilarity(t *testing.T) {
	t.Parallel()
	m := NewArtistMatcher()
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
		{"alias via tag", Artist{Name: "The Weeknd", Tags: NewTags(WithAlias("Weeknd"))}, Artist{Name: "Weeknd"}, true},
		{"completely different", Artist{Name: "Beyoncé"}, Artist{Name: "Taylor Swift"}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			s := m.Match(ctx, tc.a, tc.b)
			got := s.Same(DefaultArtistThreshold)
			if got != tc.shouldMatch {
				t.Errorf("got Same=%v want %v (%v)", got, tc.shouldMatch, s)
			}
		})
	}
}

type fakeProvider struct {
	mu       sync.Mutex
	releases map[string][]Album
	calls    int
	err      error
	delay    time.Duration
}

func (p *fakeProvider) Releases(ctx context.Context, artist Artist) ([]Album, error) {
	p.mu.Lock()
	p.calls++
	p.mu.Unlock()
	if p.delay > 0 {
		time.Sleep(p.delay)
	}
	if p.err != nil {
		return nil, p.err
	}
	return p.releases[artist.Name], nil
}

func (p *fakeProvider) Calls() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.calls
}

func TestArtistMatcher_ReleaseOverlap(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

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

	a := Artist{Name: "Cardi B"}
	b := Artist{Name: "Kardi"}

	withProvider := NewArtistMatcher(ArtistReleaseProvider(provider))
	withoutProvider := NewArtistMatcher()

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
	m := NewArtistMatcher(ArtistReleaseProvider(provider))

	a := Artist{Name: "A"}
	b := Artist{Name: "B"}

	m.Match(ctx, a, b)
	first := provider.Calls()
	m.Match(ctx, a, b)
	if provider.Calls() != first {
		t.Errorf("cache miss on repeat call: was %d, now %d", first, provider.Calls())
	}
}

func TestArtistMatcher_ProviderErrorDoesNotPenalise(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	nameOnly := NewArtistMatcher()
	withFailing := NewArtistMatcher(ArtistReleaseProvider(&fakeProvider{err: errors.New("boom")}))

	a := Artist{Name: "Cardi B"}
	b := Artist{Name: "Kardi"}
	nameScore := nameOnly.Match(ctx, a, b).Value
	failScore := withFailing.Match(ctx, a, b).Value
	if failScore < nameScore-1e-9 {
		t.Errorf("failing provider lowered score: name=%v fail=%v", nameScore, failScore)
	}
}

func TestArtistMatcher_ConcurrentFetchDedup(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	provider := &fakeProvider{
		releases: map[string][]Album{
			"Cardi B": {{Name: "Shared"}},
			"Kardi":   {{Name: "Shared"}},
		},
		delay: 10 * time.Millisecond,
	}
	m := NewArtistMatcher(ArtistReleaseProvider(provider))

	a := Artist{Name: "Cardi B"}
	b := Artist{Name: "Kardi"}

	var wg sync.WaitGroup
	const workers = 16
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			m.Match(ctx, a, b)
		}()
	}
	wg.Wait()

	if got := provider.Calls(); got > 4 {
		t.Errorf("singleflight dedup broken: %d provider calls for %d workers (2 unique artists)", got, workers)
	}
}

func TestArtistCacheKey_Stability(t *testing.T) {
	t.Parallel()
	// MBID dominates.
	a := Artist{Name: "A", Tags: NewTags(WithMBID("abc"), WithPlatformID(PlatformSpotify, "s1"))}
	b := Artist{Name: "B", Tags: NewTags(WithMBID("abc"))}
	if artistCacheKey(a) != artistCacheKey(b) {
		t.Errorf("MBID should dominate cache key")
	}

	// External IDs order-insensitive.
	c := Artist{Tags: NewTags(WithPlatformID(PlatformSpotify, "s"), WithPlatformID(PlatformTidal, "t"))}
	d := Artist{Tags: NewTags(WithPlatformID(PlatformTidal, "t"), WithPlatformID(PlatformSpotify, "s"))}
	if artistCacheKey(c) != artistCacheKey(d) {
		t.Errorf("external ID key not order-insensitive: %q vs %q", artistCacheKey(c), artistCacheKey(d))
	}

	// Name-only normalises.
	e := Artist{Name: "Björk"}
	f := Artist{Name: "Bjork"}
	if artistCacheKey(e) != artistCacheKey(f) {
		t.Errorf("name key not normalised: %q vs %q", artistCacheKey(e), artistCacheKey(f))
	}
}

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
	m := NewArtistMatcher(ArtistReleaseProvider(provider))
	score := m.Match(context.Background(), Artist{Name: "Maneskin"}, Artist{Name: "Måneskin"})
	if score.Same(DefaultArtistThreshold) {
		fmt.Println("match")
	}
	// Output: match
}
