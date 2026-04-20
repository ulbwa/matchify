package matchify_test

import (
	"context"
	"fmt"
	"time"

	"github.com/ulbwa/matchify"
)

// ExampleTrackMatcher_Match demonstrates matching the same track as
// rendered by two platforms — one listing the featured artist in the title
// (Apple Music style), the other in the artist array (Spotify style).
func ExampleTrackMatcher_Match() {
	m := matchify.NewTrackMatcher(matchify.TrackMatcherOptions{})

	spotify := matchify.Track{
		Name: "Despacito",
		Artists: []matchify.Artist{
			{Name: "Luis Fonsi"},
			{Name: "Daddy Yankee"},
			{Name: "Justin Bieber"},
		},
		Duration: 228 * time.Second,
	}
	appleMusic := matchify.Track{
		Name: "Despacito (feat. Justin Bieber)",
		Artists: []matchify.Artist{
			{Name: "Luis Fonsi"},
			{Name: "Daddy Yankee"},
		},
		Duration: 229 * time.Second,
	}

	score := m.Match(context.Background(), spotify, appleMusic)
	fmt.Printf("match=%v", score.Above(matchify.DefaultTrackThreshold))
	// Output: match=true
}

// ExampleFindBest demonstrates picking the best candidate from a list.
func ExampleFindBest() {
	m := matchify.NewTrackMatcher(matchify.TrackMatcherOptions{})

	target := matchify.Track{
		Name:    "Shape of You",
		Artists: []matchify.Artist{{Name: "Ed Sheeran"}},
	}
	candidates := []matchify.Track{
		{Name: "Thinking Out Loud", Artists: []matchify.Artist{{Name: "Ed Sheeran"}}},
		{Name: "Shape of You (Remastered)", Artists: []matchify.Artist{{Name: "Ed Sheeran"}}},
		{Name: "Photograph", Artists: []matchify.Artist{{Name: "Ed Sheeran"}}},
	}

	idx, _, ok := matchify.FindBest(context.Background(), m, target, candidates, matchify.DefaultTrackThreshold)
	fmt.Printf("ok=%v idx=%d", ok, idx)
	// Output: ok=true idx=1
}

// ExampleGroup demonstrates clustering tracks from multiple platforms into
// groups that represent the same song.
func ExampleGroup() {
	m := matchify.NewTrackMatcher(matchify.TrackMatcherOptions{})

	items := []matchify.Track{
		{Name: "Blinding Lights", Artists: []matchify.Artist{{Name: "The Weeknd"}}, Duration: 200 * time.Second},
		{Name: "Blinding Lights", Artists: []matchify.Artist{{Name: "The Weeknd"}}, Duration: 200 * time.Second},
		{Name: "Save Your Tears", Artists: []matchify.Artist{{Name: "The Weeknd"}}, Duration: 216 * time.Second},
	}

	groups := matchify.Group(context.Background(), m, items, matchify.DefaultTrackThreshold)
	fmt.Println(groups)
	// Output: [[0 1] [2]]
}
