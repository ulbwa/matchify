package matchify_test

import (
	"context"
	"fmt"
	"time"

	"github.com/ulbwa/matchify"
)

// ExampleTrackMatcher_Match demonstrates matching the same track as
// rendered by two platforms — Spotify keeps featured artists in the
// artist array, Apple Music tends to append "(feat. X)" to the title.
func ExampleTrackMatcher_Match() {
	m := matchify.NewTrackMatcher()

	spotify := matchify.Track{
		Name: "Despacito",
		Artists: []matchify.Artist{
			{Name: "Luis Fonsi"},
			{Name: "Daddy Yankee"},
			{Name: "Justin Bieber"},
		},
		Tags: matchify.NewTags(matchify.WithDuration(228 * time.Second)),
	}
	appleMusic := matchify.Track{
		Name: "Despacito (feat. Justin Bieber)",
		Artists: []matchify.Artist{
			{Name: "Luis Fonsi"},
			{Name: "Daddy Yankee"},
		},
		Tags: matchify.NewTags(matchify.WithDuration(229 * time.Second)),
	}

	score := m.Match(context.Background(), spotify, appleMusic)
	fmt.Println("same:", score.Same(matchify.DefaultTrackThreshold))
	// Output: same: true
}

// ExampleFindBest demonstrates picking the best candidate from a list
// using IsSame — strict duplicate detection.
func ExampleFindBest() {
	m := matchify.NewTrackMatcher()

	target := matchify.Track{
		Name:    "Shape of You",
		Artists: []matchify.Artist{{Name: "Ed Sheeran"}},
	}
	candidates := []matchify.Track{
		{Name: "Thinking Out Loud", Artists: []matchify.Artist{{Name: "Ed Sheeran"}}},
		{Name: "Shape of You - Remastered", Artists: []matchify.Artist{{Name: "Ed Sheeran"}}},
		{Name: "Photograph", Artists: []matchify.Artist{{Name: "Ed Sheeran"}}},
	}

	idx, _, ok := matchify.FindBest(context.Background(), m, target, candidates,
		matchify.IsSame(matchify.DefaultTrackThreshold))
	fmt.Printf("ok=%v idx=%d", ok, idx)
	// Output: ok=true idx=1
}

// ExampleFindBest_related demonstrates admitting variants (acoustic,
// live, remix, remaster) as matches using IsRelated.
func ExampleFindBest_related() {
	m := matchify.NewTrackMatcher()

	target := matchify.Track{
		Name:    "Shape of You",
		Artists: []matchify.Artist{{Name: "Ed Sheeran"}},
	}
	candidates := []matchify.Track{
		{Name: "Bad Habits", Artists: []matchify.Artist{{Name: "Ed Sheeran"}}},
		{Name: "Shape of You (Acoustic)", Artists: []matchify.Artist{{Name: "Ed Sheeran"}}},
	}

	idx, score, ok := matchify.FindBest(context.Background(), m, target, candidates,
		matchify.IsRelated(matchify.DefaultTrackThreshold))
	fmt.Printf("ok=%v idx=%d relation=%s", ok, idx, score.Relation)
	// Output: ok=true idx=1 relation=variant
}

// ExampleGroup demonstrates clustering tracks into groups that
// represent the same song across platforms.
func ExampleGroup() {
	m := matchify.NewTrackMatcher()

	items := []matchify.Track{
		{Name: "Blinding Lights", Artists: []matchify.Artist{{Name: "The Weeknd"}},
			Tags: matchify.NewTags(matchify.WithDuration(200 * time.Second))},
		{Name: "Blinding Lights", Artists: []matchify.Artist{{Name: "The Weeknd"}},
			Tags: matchify.NewTags(matchify.WithDuration(200 * time.Second))},
		{Name: "Save Your Tears", Artists: []matchify.Artist{{Name: "The Weeknd"}},
			Tags: matchify.NewTags(matchify.WithDuration(216 * time.Second))},
	}

	groups := matchify.Group(context.Background(), m, items,
		matchify.IsSame(matchify.DefaultTrackThreshold))
	fmt.Println(groups)
	// Output: [[0 1] [2]]
}

// ExampleScore_Relation shows how Relation distinguishes Same,
// Variant, and Unrelated — two tracks that share a name and artist
// but differ on explicit/clean master are variants, not the same.
func ExampleScore_Relation() {
	m := matchify.NewTrackMatcher()

	explicit := matchify.Track{
		Name:    "HUMBLE.",
		Artists: []matchify.Artist{{Name: "Kendrick Lamar"}},
		Tags:    matchify.NewTags(matchify.WithExplicit(matchify.ExplicitnessExplicit)),
	}
	clean := matchify.Track{
		Name:    "HUMBLE. (Clean)",
		Artists: []matchify.Artist{{Name: "Kendrick Lamar"}},
	}

	score := m.Match(context.Background(), explicit, clean)
	fmt.Printf("relation=%s same=%v related=%v",
		score.Relation,
		score.Same(matchify.DefaultTrackThreshold),
		score.Related(matchify.DefaultTrackThreshold),
	)
	// Output: relation=variant same=false related=true
}

// ExampleNewTags shows the typed tag constructor pattern for a track
// that carries rich metadata — a case common on Spotify and Qobuz.
func ExampleNewTags() {
	t := matchify.Track{
		Name:    "Bad Guy",
		Artists: []matchify.Artist{{Name: "Billie Eilish"}},
		Tags: matchify.NewTags(
			matchify.WithDuration(194*time.Second),
			matchify.WithTrackNumber(2),
			matchify.WithDiscNumber(1),
			matchify.WithExplicit(matchify.ExplicitnessExplicit),
			matchify.WithISRC("USUM71900764"),
			matchify.WithPlatformID(matchify.PlatformSpotify, "2Fxmhks0bxGSBdJ92vM42m"),
			matchify.WithPlatformID(matchify.PlatformAppleMusic, "1450695739"),
		),
	}

	d, _ := t.Tags.Duration()
	isrc, _ := t.Tags.ISRC()
	spotID, _ := t.Tags.PlatformID(matchify.PlatformSpotify)
	fmt.Printf("duration=%v isrc=%s spotify=%s", d, isrc, spotID)
	// Output: duration=3m14s isrc=USUM71900764 spotify=2Fxmhks0bxGSBdJ92vM42m
}

// ExampleArtistMatcher_Match_withAlias shows how aliases help name-only
// artist matching bridge known transliterations.
func ExampleArtistMatcher_Match_withAlias() {
	m := matchify.NewArtistMatcher()

	a := matchify.Artist{
		Name: "Måneskin",
		Tags: matchify.NewTags(matchify.WithAlias("Maneskin")),
	}
	b := matchify.Artist{Name: "Maneskin"}
	fmt.Println(m.Match(context.Background(), a, b).Same(matchify.DefaultArtistThreshold))
	// Output: true
}
