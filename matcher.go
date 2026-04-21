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

// Accept is a predicate on a Score. FindBest and Group take one of these
// to decide which scores count as "a match". Common choices are
// IsSame(threshold) (strict: only RelationSame) and IsRelated(threshold)
// (lenient: RelationSame or RelationVariant). Callers are free to define
// their own.
type Accept func(Score) bool

// IsSame returns an Accept predicate that admits only same-product
// matches at or above threshold.
func IsSame(threshold float64) Accept {
	return func(s Score) bool { return s.Same(threshold) }
}

// IsRelated returns an Accept predicate that admits both same-product
// and variant matches at or above threshold.
func IsRelated(threshold float64) Accept {
	return func(s Score) bool { return s.Related(threshold) }
}

// FindBest iterates over candidates and returns the index of the
// candidate with the highest Score.Value, along with the score itself.
// The result is reported only if the score satisfies accept; otherwise
// the returned ok is false.
//
// To find only same-product matches use IsSame(threshold); to find
// variants too use IsRelated(threshold).
func FindBest[T any](ctx context.Context, m Matcher[T], target T, candidates []T, accept Accept) (int, Score, bool) {
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
	if bestIdx == -1 || !accept(bestScore) {
		return -1, Score{}, false
	}
	return bestIdx, bestScore, true
}

// Group clusters items into groups of indices such that every adjacent
// pair (in the transitive sense) satisfies accept. The returned groups
// partition [0, len(items)) and are ordered by the smallest index they
// contain; singletons are included.
//
// Group runs in O(n²) comparisons. For large inputs, pre-bucket items by
// a deterministic key (e.g. normalised artist name) before calling
// Group on each bucket.
func Group[T any](ctx context.Context, m Matcher[T], items []T, accept Accept) [][]int {
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
			if accept(m.Match(ctx, items[i], items[j])) {
				union(i, j)
			}
		}
	}

	groups := make(map[int][]int)
	for i := 0; i < n; i++ {
		root := find(i)
		groups[root] = append(groups[root], i)
	}

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
