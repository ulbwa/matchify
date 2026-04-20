// Package textnorm provides string normalization primitives used by matchers.
//
// Normalization makes two superficially different names (different casing,
// diacritics, quotation styles, cosmetic suffixes) compare as equal while
// preserving meaningful distinctions such as "(Live)" or "(Acoustic)".
package textnorm

import (
	"regexp"
	"strings"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// StripAccents removes combining diacritical marks from s. It applies NFKD
// decomposition first, so "Björk" becomes "Bjork" and "Beyoncé" becomes
// "Beyonce".
func StripAccents(s string) string {
	t := transform.Chain(
		norm.NFKD,
		runes.Remove(runes.In(unicode.Mn)),
		norm.NFC,
	)
	out, _, err := transform.String(t, s)
	if err != nil {
		return s
	}
	return out
}

// CollapseWhitespace replaces runs of whitespace with a single space and trims
// leading/trailing whitespace.
func CollapseWhitespace(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

// Lowercase returns s with all letters mapped to their lower-case equivalents
// using Unicode-aware case folding.
func Lowercase(s string) string {
	return strings.ToLower(s)
}

// quoteReplacer normalises fancy punctuation to plain ASCII equivalents.
var quoteReplacer = strings.NewReplacer(
	"\u2018", "'", // ‘
	"\u2019", "'", // ’
	"\u201A", "'", // ‚
	"\u201B", "'", // ‛
	"\u201C", `"`, // “
	"\u201D", `"`, // ”
	"\u201E", `"`, // „
	"\u201F", `"`, // ‟
	"\u2010", "-", // ‐
	"\u2011", "-", // ‑
	"\u2012", "-", // ‒
	"\u2013", "-", // –
	"\u2014", "-", // —
	"\u2015", "-", // ―
	"\u2026", "...", // …
	"\u00B7", " ", // ·
)

// NormalizePunctuation replaces typographic punctuation with plain ASCII.
func NormalizePunctuation(s string) string {
	return quoteReplacer.Replace(s)
}

// Normalize returns a canonical form of s suitable for similarity comparison.
//
// It performs the following steps:
//  1. Normalises typographic punctuation (fancy quotes, em/en dashes, ellipsis).
//  2. Applies NFKD decomposition and strips combining diacritics.
//  3. Folds to lower case.
//  4. Strips cosmetic suffixes that do not distinguish performances
//     (e.g. "Remastered 2011", "(Explicit)", "- Digital Remaster").
//  5. Replaces remaining punctuation with spaces and collapses whitespace.
//  6. Strips a leading English article ("the ") so that "The Beatles" and
//     "Beatles" collide.
//
// Normalize is conservative: it leaves suffixes that identify different
// performances, such as "(Live)", "(Acoustic)", or "(Remix)", untouched so
// that they continue to distinguish tracks.
func Normalize(s string) string {
	s = NormalizePunctuation(s)
	s = StripAccents(s)
	s = Lowercase(s)
	s = stripCosmeticSuffixes(s)
	s = stripPunctuation(s)
	s = CollapseWhitespace(s)
	s = stripLeadingArticle(s)
	return s
}

// stripLeadingArticle removes a leading English definite article from s.
// The trailing space is required so that "thermodynamics" doesn't lose its
// prefix; only "the " at the start is stripped.
func stripLeadingArticle(s string) string {
	const prefix = "the "
	if strings.HasPrefix(s, prefix) {
		rest := s[len(prefix):]
		if rest != "" {
			return rest
		}
	}
	return s
}

// NormalizeKeepPunctuation is like Normalize but preserves apostrophes. Useful
// when a name contains contractions whose removal loses information
// (e.g. "don't" vs "dont" should not collide with "don").
func NormalizeKeepPunctuation(s string) string {
	s = NormalizePunctuation(s)
	s = StripAccents(s)
	s = Lowercase(s)
	s = stripCosmeticSuffixes(s)
	return CollapseWhitespace(s)
}

// cosmeticPattern matches parenthetical or dash-prefixed suffixes whose
// contents consist solely of cosmetic labels. The labels identify the same
// performance under a different presentation (remaster, explicit flag, etc.)
// rather than a distinct recording.
//
// The pattern is anchored to end-of-string so that we only strip trailing
// noise; mid-string parentheticals are preserved (they may carry meaningful
// information such as album disambiguation).
var cosmeticPattern = regexp.MustCompile(
	`(?i)\s*(?:[-–—]\s*|\(|\[)` +
		`(?:` + cosmeticAlternation + `)` +
		`(?:\s*[-,/]\s*(?:` + cosmeticAlternation + `))*` +
		`(?:\)|\])?\s*$`,
)

// cosmeticAlternation is the set of tokens considered purely cosmetic.
const cosmeticAlternation = `` +
	`\d{4}\s+remaster(?:ed)?|` +
	`remaster(?:ed)?(?:\s+\d{4})?|` +
	`digital(?:ly)?\s+remaster(?:ed)?|` +
	`digital\s+version|` +
	`explicit(?:\s+version)?|` +
	`clean(?:\s+version)?|` +
	`edited(?:\s+version)?|` +
	`album\s+version|` +
	`original\s+version|` +
	`single\s+version|` +
	`bonus\s+track|` +
	`hidden\s+track|` +
	`mono(?:\s+version)?|` +
	`stereo(?:\s+version)?|` +
	`deluxe(?:\s+(?:edition|version))?|` +
	`super\s+deluxe(?:\s+(?:edition|version))?|` +
	`expanded(?:\s+edition)?|` +
	`extended(?:\s+edition)?|` +
	`special\s+edition|` +
	`collectors?\s+edition|` +
	`limited\s+edition|` +
	`anniversary\s+edition|` +
	`\d+(?:st|nd|rd|th)\s+anniversary(?:\s+edition)?|` +
	`japanese\s+edition|` +
	`japan\s+edition|` +
	`international\s+edition`

// stripCosmeticSuffixes repeatedly removes cosmetic tails until no more match.
// Some names have multiple stacked tails ("Song (Remastered) - 2011 Remaster").
func stripCosmeticSuffixes(s string) string {
	for {
		after := cosmeticPattern.ReplaceAllString(s, "")
		after = strings.TrimRight(after, " \t-–—")
		if after == s {
			return s
		}
		s = after
	}
}

// punctRunPattern replaces any run of non-letter, non-digit, non-space runes
// with a single space. This keeps words separated but removes ornamental
// punctuation such as apostrophes, dots, and exclamation marks.
var punctRunPattern = regexp.MustCompile(`[^\p{L}\p{N}\s]+`)

func stripPunctuation(s string) string {
	return punctRunPattern.ReplaceAllString(s, " ")
}
