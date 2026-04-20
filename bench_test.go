package matchify

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func BenchmarkTrackMatcher_Match_SameTrack(b *testing.B) {
	m := NewTrackMatcher(TrackMatcherOptions{})
	ctx := context.Background()
	ta := Track{
		Name:     "Despacito",
		Artists:  []Artist{{Name: "Luis Fonsi"}, {Name: "Daddy Yankee"}, {Name: "Justin Bieber"}},
		Duration: 228 * time.Second,
	}
	tb := Track{
		Name:     "Despacito (feat. Justin Bieber)",
		Artists:  []Artist{{Name: "Luis Fonsi"}, {Name: "Daddy Yankee"}},
		Duration: 229 * time.Second,
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.Match(ctx, ta, tb)
	}
}

func BenchmarkTrackMatcher_Match_ShortCircuit(b *testing.B) {
	m := NewTrackMatcher(TrackMatcherOptions{})
	ctx := context.Background()
	ta := Track{Name: "x", ISRC: "ABC"}
	tb := Track{Name: "y", ISRC: "ABC"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.Match(ctx, ta, tb)
	}
}

func BenchmarkAlbumMatcher_Match(b *testing.B) {
	m := NewAlbumMatcher(AlbumMatcherOptions{})
	ctx := context.Background()
	aa := Album{
		Name:        "Dark Side of the Moon",
		Artists:     []Artist{{Name: "Pink Floyd"}},
		ReleaseDate: time.Date(1973, 3, 1, 0, 0, 0, 0, time.UTC),
		TrackCount:  10,
	}
	bb := Album{
		Name:        "The Dark Side of the Moon (2011 Remastered Version)",
		Artists:     []Artist{{Name: "Pink Floyd"}},
		ReleaseDate: time.Date(1973, 3, 1, 0, 0, 0, 0, time.UTC),
		TrackCount:  10,
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.Match(ctx, aa, bb)
	}
}

func BenchmarkArtistMatcher_Match(b *testing.B) {
	m := NewArtistMatcher(ArtistMatcherOptions{})
	ctx := context.Background()
	aa := Artist{Name: "The Beatles"}
	bb := Artist{Name: "Beatles"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.Match(ctx, aa, bb)
	}
}

func BenchmarkGroup_100Tracks(b *testing.B) {
	m := NewTrackMatcher(TrackMatcherOptions{})
	ctx := context.Background()
	items := make([]Track, 100)
	for i := range items {
		items[i] = Track{
			Name:     fmt.Sprintf("Song %d", i),
			Artists:  []Artist{{Name: fmt.Sprintf("Artist %d", i)}},
			Duration: time.Duration(180+i) * time.Second,
		}
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Group(ctx, m, items, DefaultTrackThreshold)
	}
}
