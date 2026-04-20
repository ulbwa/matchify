package textsim

import (
	"slices"
	"strings"
)

// TokenSetRatio computes a similarity between two strings based on the sets of
// whitespace-delimited tokens they contain. The tokens are sorted and deduplicated
// before comparison, which makes the metric insensitive to word order and
// duplicated words. Jaro-Winkler is then applied to the canonical joined forms.
//
// This is useful when two names share the same words in a different order
// (e.g. "Lennon John" vs "John Lennon") or where one string contains additional
// connective words.
func TokenSetRatio(a, b string) float64 {
	ta := uniqueSortedTokens(a)
	tb := uniqueSortedTokens(b)
	if len(ta) == 0 && len(tb) == 0 {
		return 1
	}
	if len(ta) == 0 || len(tb) == 0 {
		return 0
	}

	intersection := intersectSorted(ta, tb)
	diffA := subtractSorted(ta, intersection)
	diffB := subtractSorted(tb, intersection)

	joinedIntersect := strings.Join(intersection, " ")
	joinedA := strings.TrimSpace(joinedIntersect + " " + strings.Join(diffA, " "))
	joinedB := strings.TrimSpace(joinedIntersect + " " + strings.Join(diffB, " "))

	s1 := JaroWinkler(joinedIntersect, joinedA)
	s2 := JaroWinkler(joinedIntersect, joinedB)
	s3 := JaroWinkler(joinedA, joinedB)
	return max(s1, max(s2, s3))
}

func uniqueSortedTokens(s string) []string {
	if s == "" {
		return nil
	}
	toks := strings.Fields(s)
	slices.Sort(toks)
	return slices.Compact(toks)
}

// intersectSorted returns tokens present in both sorted slices.
func intersectSorted(a, b []string) []string {
	var out []string
	i, j := 0, 0
	for i < len(a) && j < len(b) {
		switch {
		case a[i] == b[j]:
			out = append(out, a[i])
			i++
			j++
		case a[i] < b[j]:
			i++
		default:
			j++
		}
	}
	return out
}

// subtractSorted returns tokens in a that are not in b.
func subtractSorted(a, b []string) []string {
	var out []string
	i, j := 0, 0
	for i < len(a) {
		switch {
		case j >= len(b) || a[i] < b[j]:
			out = append(out, a[i])
			i++
		case a[i] == b[j]:
			i++
			j++
		default:
			j++
		}
	}
	return out
}
