package matchify

import (
	"context"
	"sort"
)

// Matcher compares two entities of type T and returns a Score. All matchers
// in this package satisfy Matcher[T] for their respective entity type, so
// the generic helpers FindBest and Group work uniformly across tracks,
// albums, and artists.
type Matcher[T any] interface {
	Match(ctx context.Context, a, b T) Score
}

// FindBest iterates over candidates and returns the index of the candidate
// with the highest score against target, along with the score itself. The
// result is reported only if the score meets threshold; otherwise the
// returned ok is false.
//
// Pass a threshold of 0 to always receive the best-scoring candidate.
func FindBest[T any](ctx context.Context, m Matcher[T], target T, candidates []T, threshold float64) (int, Score, bool) {
	bestIdx := -1
	var bestScore Score
	for i, c := range candidates {
		if err := ctx.Err(); err != nil {
			break
		}
		s := m.Match(ctx, target, c)
		if s.Value > bestScore.Value || bestIdx == -1 {
			bestIdx = i
			bestScore = s
		}
	}
	if bestIdx == -1 || !bestScore.Above(threshold) {
		return -1, Score{}, false
	}
	return bestIdx, bestScore, true
}

// Group clusters items into groups of indices such that every adjacent pair
// (in the transitive sense) scores at or above threshold. The returned
// groups partition the indices [0, len(items)) and are ordered by the first
// index they contain; singletons are included.
//
// Group runs in O(n²) comparisons — each pair is compared once via m.Match.
// For large inputs, callers may want to pre-bucket items by some
// deterministic key (e.g. normalized artist name) before calling Group on
// each bucket.
func Group[T any](ctx context.Context, m Matcher[T], items []T, threshold float64) [][]int {
	n := len(items)
	if n == 0 {
		return nil
	}

	parent := make([]int, n)
	for i := range parent {
		parent[i] = i
	}
	var find func(int) int
	find = func(x int) int {
		if parent[x] != x {
			parent[x] = find(parent[x])
		}
		return parent[x]
	}
	union := func(x, y int) {
		px, py := find(x), find(y)
		if px != py {
			parent[px] = py
		}
	}

	for i := 0; i < n; i++ {
		if err := ctx.Err(); err != nil {
			break
		}
		for j := i + 1; j < n; j++ {
			if m.Match(ctx, items[i], items[j]).Above(threshold) {
				union(i, j)
			}
		}
	}

	groups := make(map[int][]int)
	for i := 0; i < n; i++ {
		root := find(i)
		groups[root] = append(groups[root], i)
	}

	// Order groups deterministically by their smallest index.
	heads := make([]int, 0, len(groups))
	for root := range groups {
		heads = append(heads, root)
	}
	sort.Slice(heads, func(i, j int) bool {
		return groups[heads[i]][0] < groups[heads[j]][0]
	})

	out := make([][]int, 0, len(heads))
	for _, h := range heads {
		out = append(out, groups[h])
	}
	return out
}
