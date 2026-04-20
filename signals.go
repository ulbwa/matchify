package matchify

import (
	"time"

	"github.com/ulbwa/matchify/internal/featparse"
	"github.com/ulbwa/matchify/internal/textnorm"
	"github.com/ulbwa/matchify/internal/textsim"
)

// nameSimilarity returns a strict similarity score in [0, 1] between two
// display names, after normalisation. It uses Jaro-Winkler, which preserves
// the significance of word-level differences such as "(Live)" or
// "(Acoustic)" that distinguish recordings. This is the appropriate metric
// for track and album titles.
func nameSimilarity(a, b string) float64 {
	na := textnorm.Normalize(a)
	nb := textnorm.Normalize(b)
	if na == "" && nb == "" {
		return 1
	}
	if na == "" || nb == "" {
		return 0
	}
	if na == nb {
		return 1
	}
	return textsim.JaroWinkler(na, nb)
}

// artistSingleNameSimilarity returns a loose similarity score suitable for
// matching artist names, which benefit from insensitivity to word order and
// missing articles (e.g., "The Beatles" vs "Beatles"). It combines Jaro-
// Winkler with token-set-ratio.
func artistSingleNameSimilarity(a, b string) float64 {
	na := textnorm.Normalize(a)
	nb := textnorm.Normalize(b)
	if na == "" && nb == "" {
		return 1
	}
	if na == "" || nb == "" {
		return 0
	}
	if na == nb {
		return 1
	}
	jw := textsim.JaroWinkler(na, nb)
	ts := textsim.TokenSetRatio(na, nb)
	if jw > ts {
		return jw
	}
	return ts
}

// artistNameSimilarity returns the best similarity between two Artist values,
// considering their names and aliases.
func artistNameSimilarity(a, b Artist) float64 {
	namesA := append([]string{a.Name}, a.Aliases...)
	namesB := append([]string{b.Name}, b.Aliases...)
	best := 0.0
	for _, na := range namesA {
		for _, nb := range namesB {
			if s := artistSingleNameSimilarity(na, nb); s > best {
				best = s
			}
		}
	}
	return best
}

// artistListSignal returns a signal representing the similarity between two
// lists of artists. Both lists are first normalised via expandComposites so
// that a Deezer-style "A feat. B" or "A & B" single entry is split into its
// constituent names; this makes the signal symmetrical against Spotify-
// style lists where each credited artist gets its own entry.
//
// The algorithm is greedy assignment — for each artist in the shorter list,
// pick the best remaining match in the longer list. The final value is the
// sum of best pairwise similarities divided by the length of the longer
// list, which penalises extra artists on either side.
func artistListSignal(a, b []Artist, weight float64) Signal {
	a = expandComposites(a)
	b = expandComposites(b)
	switch {
	case len(a) == 0 && len(b) == 0:
		return Signal{Name: "artists", Weight: 0, Note: "both empty"}
	case len(a) == 0 || len(b) == 0:
		return Signal{Name: "artists", Value: 0, Weight: weight, Note: "one side empty"}
	}

	short, long := a, b
	if len(long) < len(short) {
		short, long = long, short
	}
	used := make([]bool, len(long))
	total := 0.0
	for _, s := range short {
		bestScore := 0.0
		bestIdx := -1
		for i, l := range long {
			if used[i] {
				continue
			}
			sc := artistNameSimilarity(s, l)
			if sc > bestScore {
				bestScore = sc
				bestIdx = i
			}
		}
		if bestIdx >= 0 {
			used[bestIdx] = true
			total += bestScore
		}
	}
	return Signal{
		Name:   "artists",
		Value:  total / float64(len(long)),
		Weight: weight,
	}
}

// expandComposites walks an Artist list and replaces any entry whose name
// contains feature or list markers with multiple name-only Artist entries,
// one per extracted name. Entries whose names are atomic pass through
// unchanged; the original metadata (MBID, ExternalIDs, Aliases) is
// preserved for those. Split entries do not inherit metadata because it
// applied to the composite identity rather than each individual.
func expandComposites(artists []Artist) []Artist {
	var out []Artist
	for _, a := range artists {
		names := featparse.SplitCompositeName(a.Name)
		if len(names) <= 1 {
			out = append(out, a)
			continue
		}
		for _, n := range names {
			out = append(out, Artist{Name: n})
		}
	}
	return out
}

// durationSignal scores the similarity of two track durations. A difference
// of up to 2 seconds scores full; the score degrades to zero past 15 seconds.
// If either side is unknown (zero), the signal is emitted with zero weight
// so it doesn't affect the weighted average.
func durationSignal(a, b time.Duration, weight float64) Signal {
	if a == 0 || b == 0 {
		return Signal{Name: "duration", Weight: 0, Note: "unknown"}
	}
	diff := a - b
	if diff < 0 {
		diff = -diff
	}
	switch {
	case diff <= 2*time.Second:
		return Signal{Name: "duration", Value: 1.0, Weight: weight}
	case diff <= 5*time.Second:
		return Signal{Name: "duration", Value: 0.75, Weight: weight}
	case diff <= 15*time.Second:
		return Signal{Name: "duration", Value: 0.3, Weight: weight}
	default:
		return Signal{Name: "duration", Value: 0, Weight: weight}
	}
}

// yearSignal scores the similarity of two release years. A difference of up
// to 1 year scores highly (label bureaucracy causes this); beyond 3 years
// the signal is zero. Unknown years yield zero weight.
func yearSignal(a, b int, weight float64) Signal {
	if a == 0 || b == 0 {
		return Signal{Name: "year", Weight: 0, Note: "unknown"}
	}
	diff := a - b
	if diff < 0 {
		diff = -diff
	}
	switch {
	case diff == 0:
		return Signal{Name: "year", Value: 1.0, Weight: weight}
	case diff == 1:
		return Signal{Name: "year", Value: 0.75, Weight: weight}
	case diff <= 3:
		return Signal{Name: "year", Value: 0.3, Weight: weight}
	default:
		return Signal{Name: "year", Value: 0, Weight: weight}
	}
}

// trackCountSignal scores the similarity of two track-count values. Even for
// the "same" album across platforms the count can differ by a few when
// bonuses or region-locked tracks are added, so small differences are
// tolerated. Unknown counts yield zero weight.
func trackCountSignal(a, b int, weight float64) Signal {
	if a == 0 || b == 0 {
		return Signal{Name: "track_count", Weight: 0, Note: "unknown"}
	}
	diff := a - b
	if diff < 0 {
		diff = -diff
	}
	switch {
	case diff == 0:
		return Signal{Name: "track_count", Value: 1.0, Weight: weight}
	case diff <= 2:
		return Signal{Name: "track_count", Value: 0.75, Weight: weight}
	case diff <= 5:
		return Signal{Name: "track_count", Value: 0.3, Weight: weight}
	default:
		return Signal{Name: "track_count", Value: 0, Weight: weight}
	}
}

// externalIDVerdict is the outcome of comparing two ExternalIDs maps for the
// same platform.
type externalIDVerdict int

const (
	externalIDNone      externalIDVerdict = iota // no shared platform
	externalIDEqual                              // shared platform, same ID
	externalIDDifferent                          // shared platform, different ID
)

// comparePlatformIDs walks the intersection of a's and b's platform keys and
// reports whether any shared platform had matching IDs or all shared
// platforms disagreed.
func comparePlatformIDs(a, b map[Platform]string) externalIDVerdict {
	if len(a) == 0 || len(b) == 0 {
		return externalIDNone
	}
	var sawDifferent bool
	for platform, idA := range a {
		idB, ok := b[platform]
		if !ok || idA == "" || idB == "" {
			continue
		}
		if idA == idB {
			return externalIDEqual
		}
		sawDifferent = true
	}
	if sawDifferent {
		return externalIDDifferent
	}
	return externalIDNone
}

// explicitnessSignal scores the difference of two explicitness values. A
// difference is *not* strong evidence against — clean and explicit masters
// of the same recording exist. The signal contributes a small amount when
// both sides are known and agree.
func explicitnessSignal(a, b Explicitness, weight float64) Signal {
	if a == ExplicitnessUnknown || b == ExplicitnessUnknown {
		return Signal{Name: "explicit", Weight: 0, Note: "unknown"}
	}
	if a == b {
		return Signal{Name: "explicit", Value: 1.0, Weight: weight}
	}
	// Different: slight negative but don't punish hard.
	return Signal{Name: "explicit", Value: 0.6, Weight: weight, Note: "different masters"}
}
