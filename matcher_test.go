package matchify

import (
	"context"
	"reflect"
	"testing"
	"time"
)

func TestFindBest_Track(t *testing.T) {
	t.Parallel()
	m := NewTrackMatcher(TrackMatcherOptions{})
	ctx := context.Background()

	target := Track{
		Name:     "Despacito",
		Artists:  []Artist{{Name: "Luis Fonsi"}, {Name: "Daddy Yankee"}},
		Duration: 228 * time.Second,
	}
	candidates := []Track{
		{Name: "Shape of You", Artists: []Artist{{Name: "Ed Sheeran"}}, Duration: 233 * time.Second},
		{Name: "Despacito (feat. Justin Bieber)", Artists: []Artist{{Name: "Luis Fonsi"}, {Name: "Daddy Yankee"}}, Duration: 229 * time.Second},
		{Name: "Havana", Artists: []Artist{{Name: "Camila Cabello"}}, Duration: 217 * time.Second},
	}

	idx, score, ok := FindBest(ctx, m, target, candidates, DefaultTrackThreshold)
	if !ok {
		t.Fatalf("expected to find a match, got none (score %v)", score)
	}
	if idx != 1 {
		t.Errorf("expected index 1, got %d", idx)
	}
}

func TestFindBest_NoMatch(t *testing.T) {
	t.Parallel()
	m := NewTrackMatcher(TrackMatcherOptions{})
	ctx := context.Background()

	target := Track{Name: "Target Song", Artists: []Artist{{Name: "Target Artist"}}}
	candidates := []Track{
		{Name: "Random A", Artists: []Artist{{Name: "Someone"}}},
		{Name: "Random B", Artists: []Artist{{Name: "Someone Else"}}},
	}

	_, _, ok := FindBest(ctx, m, target, candidates, DefaultTrackThreshold)
	if ok {
		t.Error("expected no match above threshold")
	}
}

func TestFindBest_EmptyCandidates(t *testing.T) {
	t.Parallel()
	m := NewTrackMatcher(TrackMatcherOptions{})
	ctx := context.Background()

	_, _, ok := FindBest(ctx, m, Track{Name: "x"}, nil, 0.5)
	if ok {
		t.Error("expected no match on empty candidates")
	}
}

func TestFindBest_ZeroThreshold(t *testing.T) {
	t.Parallel()
	m := NewTrackMatcher(TrackMatcherOptions{})
	ctx := context.Background()

	target := Track{Name: "Target", Artists: []Artist{{Name: "Unique"}}}
	candidates := []Track{
		{Name: "Something Totally Different", Artists: []Artist{{Name: "Other"}}},
	}
	idx, _, ok := FindBest(ctx, m, target, candidates, 0)
	if !ok {
		t.Error("zero threshold should always return best candidate when any exist")
	}
	if idx != 0 {
		t.Errorf("expected index 0 (only candidate), got %d", idx)
	}
}

func TestGroup_Tracks(t *testing.T) {
	t.Parallel()
	m := NewTrackMatcher(TrackMatcherOptions{})
	ctx := context.Background()

	items := []Track{
		// Group A: Despacito
		{Name: "Despacito", Artists: []Artist{{Name: "Luis Fonsi"}, {Name: "Daddy Yankee"}, {Name: "Justin Bieber"}}, Duration: 228 * time.Second},
		{Name: "Despacito (feat. Justin Bieber)", Artists: []Artist{{Name: "Luis Fonsi"}, {Name: "Daddy Yankee"}}, Duration: 229 * time.Second},

		// Group B: Shape of You
		{Name: "Shape of You", Artists: []Artist{{Name: "Ed Sheeran"}}, Duration: 233 * time.Second},
		{Name: "Shape of You (Acoustic)", Artists: []Artist{{Name: "Ed Sheeran"}}, Duration: 233 * time.Second},

		// Singleton: different song
		{Name: "Havana", Artists: []Artist{{Name: "Camila Cabello"}}, Duration: 217 * time.Second},
	}

	groups := Group(ctx, m, items, DefaultTrackThreshold)

	// Expect: {0, 1}, {2}, {3}, {4} — Despacito pair groups, acoustic doesn't group with non-acoustic.
	want := [][]int{{0, 1}, {2}, {3}, {4}}
	if !reflect.DeepEqual(groups, want) {
		t.Errorf("Group: got %v, want %v", groups, want)
	}
}

func TestGroup_Empty(t *testing.T) {
	t.Parallel()
	m := NewTrackMatcher(TrackMatcherOptions{})
	ctx := context.Background()
	if got := Group(ctx, m, nil, DefaultTrackThreshold); got != nil {
		t.Errorf("expected nil, got %v", got)
	}
}

func TestGroup_Singleton(t *testing.T) {
	t.Parallel()
	m := NewTrackMatcher(TrackMatcherOptions{})
	ctx := context.Background()
	got := Group(ctx, m, []Track{{Name: "solo"}}, DefaultTrackThreshold)
	want := [][]int{{0}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("singleton: got %v, want %v", got, want)
	}
}

func TestFindBest_ContextCancelled(t *testing.T) {
	t.Parallel()
	m := NewTrackMatcher(TrackMatcherOptions{})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, _, ok := FindBest(ctx, m, Track{Name: "x"}, []Track{{Name: "y"}}, 0)
	if ok {
		t.Error("cancelled context should break out without success")
	}
}
