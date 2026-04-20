// Package textsim provides string similarity primitives used by matchers.
package textsim

// Jaro returns the Jaro similarity of two strings in [0, 1]. The inputs are
// compared as sequences of runes, so multi-byte characters are handled
// correctly.
func Jaro(a, b string) float64 {
	ra := []rune(a)
	rb := []rune(b)
	return jaroRunes(ra, rb)
}

// JaroWinkler returns the Jaro-Winkler similarity of two strings in [0, 1]. It
// applies the standard 0.1 prefix scaling factor over a maximum prefix of 4
// runes. A Jaro similarity below 0.7 is not boosted.
func JaroWinkler(a, b string) float64 {
	ra := []rune(a)
	rb := []rune(b)
	j := jaroRunes(ra, rb)
	if j < 0.7 {
		return j
	}
	p := 0
	for p < 4 && p < len(ra) && p < len(rb) && ra[p] == rb[p] {
		p++
	}
	return j + float64(p)*0.1*(1-j)
}

func jaroRunes(a, b []rune) float64 {
	la, lb := len(a), len(b)
	if la == 0 && lb == 0 {
		return 1
	}
	if la == 0 || lb == 0 {
		return 0
	}
	matchDist := max(la, lb)/2 - 1
	if matchDist < 0 {
		matchDist = 0
	}

	ma := make([]bool, la)
	mb := make([]bool, lb)
	matches := 0
	for i, r := range a {
		lo := i - matchDist
		if lo < 0 {
			lo = 0
		}
		hi := i + matchDist + 1
		if hi > lb {
			hi = lb
		}
		for j := lo; j < hi; j++ {
			if mb[j] || b[j] != r {
				continue
			}
			ma[i] = true
			mb[j] = true
			matches++
			break
		}
	}
	if matches == 0 {
		return 0
	}

	k := 0
	transpositions := 0
	for i := 0; i < la; i++ {
		if !ma[i] {
			continue
		}
		for !mb[k] {
			k++
		}
		if a[i] != b[k] {
			transpositions++
		}
		k++
	}

	m := float64(matches)
	return ((m / float64(la)) + (m / float64(lb)) + ((m - float64(transpositions)/2) / m)) / 3
}
