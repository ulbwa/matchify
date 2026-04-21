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

// ExampleFindBest demonstrates picking the best candidate from a list.
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

// ExampleGroup demonstrates clustering tracks into groups that represent
// the same song across platforms.
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
