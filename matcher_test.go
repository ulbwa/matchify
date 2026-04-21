package matchify

import (
	"context"
	"reflect"
	"testing"
	"time"
)

func TestFindBest_Track(t *testing.T) {
	t.Parallel()
	m := NewTrackMatcher()
	ctx := context.Background()

	target := Track{
		Name:    "Despacito",
		Artists: []Artist{{Name: "Luis Fonsi"}, {Name: "Daddy Yankee"}},
		Tags:    NewTags(WithDuration(228 * time.Second)),
	}
	candidates := []Track{
		{Name: "Shape of You", Artists: []Artist{{Name: "Ed Sheeran"}},
			Tags: NewTags(WithDuration(233 * time.Second))},
		{Name: "Despacito (feat. Justin Bieber)",
			Artists: []Artist{{Name: "Luis Fonsi"}, {Name: "Daddy Yankee"}},
			Tags:    NewTags(WithDuration(229 * time.Second))},
		{Name: "Havana", Artists: []Artist{{Name: "Camila Cabello"}},
			Tags: NewTags(WithDuration(217 * time.Second))},
	}

	idx, score, ok := FindBest(ctx, m, target, candidates, IsSame(DefaultTrackThreshold))
	if !ok {
		t.Fatalf("expected match, got none (score %v)", score)
	}
	if idx != 1 {
		t.Errorf("expected index 1, got %d", idx)
	}
}

func TestFindBest_NoMatch(t *testing.T) {
	t.Parallel()
	m := NewTrackMatcher()
	ctx := context.Background()

	target := Track{Name: "Target Song", Artists: []Artist{{Name: "Target Artist"}}}
	candidates := []Track{
		{Name: "Random A", Artists: []Artist{{Name: "Someone"}}},
		{Name: "Random B", Artists: []Artist{{Name: "Someone Else"}}},
	}

	_, _, ok := FindBest(ctx, m, target, candidates, IsSame(DefaultTrackThreshold))
	if ok {
		t.Error("expected no match above threshold")
	}
}

func TestFindBest_IsRelatedFindsVariants(t *testing.T) {
	t.Parallel()
	m := NewTrackMatcher()
	ctx := context.Background()

	target := Track{Name: "Shape of You", Artists: []Artist{{Name: "Ed Sheeran"}}}
	candidates := []Track{
		{Name: "Bad Habits", Artists: []Artist{{Name: "Ed Sheeran"}}},
		{Name: "Shape of You (Acoustic)", Artists: []Artist{{Name: "Ed Sheeran"}}},
	}

	// IsSame rejects the acoustic variant.
	if _, _, ok := FindBest(ctx, m, target, candidates, IsSame(DefaultTrackThreshold)); ok {
		t.Error("IsSame should not admit an acoustic variant of target")
	}
	// IsRelated admits it.
	idx, _, ok := FindBest(ctx, m, target, candidates, IsRelated(DefaultTrackThreshold))
	if !ok || idx != 1 {
		t.Errorf("IsRelated should pick acoustic variant at index 1, got idx=%d ok=%v", idx, ok)
	}
}

func TestGroup_Tracks(t *testing.T) {
	t.Parallel()
	m := NewTrackMatcher()
	ctx := context.Background()

	items := []Track{
		{Name: "Despacito",
			Artists: []Artist{{Name: "Luis Fonsi"}, {Name: "Daddy Yankee"}, {Name: "Justin Bieber"}},
			Tags:    NewTags(WithDuration(228 * time.Second))},
		{Name: "Despacito (feat. Justin Bieber)",
			Artists: []Artist{{Name: "Luis Fonsi"}, {Name: "Daddy Yankee"}},
			Tags:    NewTags(WithDuration(229 * time.Second))},

		{Name: "Shape of You", Artists: []Artist{{Name: "Ed Sheeran"}},
			Tags: NewTags(WithDuration(233 * time.Second))},
		{Name: "Shape of You (Acoustic)", Artists: []Artist{{Name: "Ed Sheeran"}},
			Tags: NewTags(WithDuration(233 * time.Second))},

		{Name: "Havana", Artists: []Artist{{Name: "Camila Cabello"}},
			Tags: NewTags(WithDuration(217 * time.Second))},
	}

	groups := Group(ctx, m, items, IsSame(DefaultTrackThreshold))
	want := [][]int{{0, 1}, {2}, {3}, {4}}
	if !reflect.DeepEqual(groups, want) {
		t.Errorf("Group(IsSame): got %v, want %v", groups, want)
	}

	// IsRelated should group the acoustic variant with the studio.
	groupsR := Group(ctx, m, items, IsRelated(DefaultTrackThreshold))
	wantR := [][]int{{0, 1}, {2, 3}, {4}}
	if !reflect.DeepEqual(groupsR, wantR) {
		t.Errorf("Group(IsRelated): got %v, want %v", groupsR, wantR)
	}
}

func TestGroup_Empty(t *testing.T) {
	t.Parallel()
	m := NewTrackMatcher()
	if got := Group(context.Background(), m, nil, IsSame(DefaultTrackThreshold)); got != nil {
		t.Errorf("expected nil, got %v", got)
	}
}

func TestFindBest_ContextCancelled(t *testing.T) {
	t.Parallel()
	m := NewTrackMatcher()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, _, ok := FindBest(ctx, m, Track{Name: "x"}, []Track{{Name: "y"}}, IsSame(0))
	if ok {
		t.Error("cancelled context should break out without success")
	}
}
