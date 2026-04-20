// Package featparse extracts featured-artist clauses from track titles.
//
// Music platforms render feature credits inconsistently: Spotify tends to
// keep them in the artist array, Apple Music and Tidal often prepend or
// append them to the track title, sometimes in parentheses, sometimes after
// a dash, sometimes with "ft." vs "feat." vs "featuring". This package
// canonicalises these forms so downstream matchers can compare track titles
// without the feature clause getting in the way and still use the featured
// artists as a matching signal.
package featparse

import (
	"regexp"
	"strings"
)

// explicitMarker matches markers that always indicate a feature credit when
// followed by an artist name: "feat", "feat.", "ft", "ft.", "featuring".
const explicitMarker = `(?:feat\.?|featuring|ft\.?)`

// parenMarker matches the same explicit markers plus "with", which is only
// safe to treat as a feature marker when it appears inside parentheses or
// brackets (outside that context it has too many false positives — e.g.
// song titles containing the word "with").
const parenMarker = `(?:feat\.?|featuring|ft\.?|with)`

var (
	// parenFeatPattern matches parenthesised/bracketed feature clauses
	// anywhere in the title. Non-greedy body so we don't swallow past a
	// closing bracket.
	parenFeatPattern = regexp.MustCompile(
		`(?i)\s*[\(\[]\s*` + parenMarker + `\.?\s+([^)\]]+?)\s*[\)\]]`,
	)

	// dashFeatPattern matches a trailing dash-separated feature clause.
	dashFeatPattern = regexp.MustCompile(
		`(?i)\s+-\s+` + explicitMarker + `\s+(.+)$`,
	)

	// inlineFeatPattern matches a trailing inline feature clause without
	// punctuation. Only explicit markers — "with" is excluded because
	// "Walking with Friends" is a title, not a feature credit.
	inlineFeatPattern = regexp.MustCompile(
		`(?i)\s+` + explicitMarker + `\s+(.+)$`,
	)

	// splitPattern splits a feature body into individual artists on ",",
	// "&", or the word "and".
	splitPattern = regexp.MustCompile(`(?i)\s*(?:,|&|\band\b)\s*`)
)

// ExtractFeatures returns the title with any recognised feature clause
// removed and the list of extracted artist names. If no feature clause is
// present, the cleaned title equals the input (trimmed and whitespace-
// collapsed) and the returned slice is nil.
//
// Recognised forms:
//   - "Song (feat. A)", "Song [ft. A]", "Song (with A)"  (anywhere in title)
//   - "Song - feat. A"                                    (trailing only)
//   - "Song ft. A", "Song featuring A"                    (trailing only)
//
// Multiple artists separated by ",", "&", or "and" are split apart.
func ExtractFeatures(title string) (string, []string) {
	cleaned := title
	var features []string

	for {
		loc := parenFeatPattern.FindStringSubmatchIndex(cleaned)
		if loc == nil {
			break
		}
		features = append(features, splitArtists(cleaned[loc[2]:loc[3]])...)
		cleaned = cleaned[:loc[0]] + cleaned[loc[1]:]
	}

	if loc := dashFeatPattern.FindStringSubmatchIndex(cleaned); loc != nil {
		features = append(features, splitArtists(cleaned[loc[2]:loc[3]])...)
		cleaned = cleaned[:loc[0]]
	} else if loc := inlineFeatPattern.FindStringSubmatchIndex(cleaned); loc != nil {
		features = append(features, splitArtists(cleaned[loc[2]:loc[3]])...)
		cleaned = cleaned[:loc[0]]
	}

	cleaned = strings.Join(strings.Fields(cleaned), " ")
	return cleaned, features
}

func splitArtists(s string) []string {
	parts := splitPattern.Split(s, -1)
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// nameFeatPattern matches a feature marker used inside an artist-name field
// (not a title). In the artist-field context, "with" is a common feature
// marker ("Drake with DJ Khaled") so it is included here — distinguishing
// this from title context where "with" has many false positives.
var nameFeatPattern = regexp.MustCompile(`(?i)\s+(?:feat\.?|ft\.?|featuring|with)\s+`)

// SplitCompositeName splits a single artist-credit string into the
// individual artist names it contains.
//
// Some platforms (notably Deezer) ship a track's artists as a single
// composite string such as "Luis Fonsi feat. Daddy Yankee" or
// "Glass Animals & Denzel Curry" instead of a list of distinct names.
// SplitCompositeName canonicalises such inputs so that downstream matchers
// can compare them against platforms that provide separate entries.
//
// Recognised markers:
//   - "feat.", "ft.", "featuring", "with" clauses anywhere in the name
//   - "&", "," and " and " separators at top level
//
// If the input contains no markers, SplitCompositeName returns a single-
// element slice with the trimmed input.
//
// Caveat: a legitimate band name with an ampersand ("Simon & Garfunkel") is
// split into its parts. In cross-platform matching this is symmetrical and
// rarely produces a false negative — both platforms are likely to represent
// the band the same way.
func SplitCompositeName(name string) []string {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil
	}

	body := name
	var trailing string
	if loc := nameFeatPattern.FindStringIndex(name); loc != nil {
		body = strings.TrimSpace(name[:loc[0]])
		trailing = strings.TrimSpace(name[loc[1]:])
	}

	parts := splitPattern.Split(body, -1)
	if trailing != "" {
		parts = append(parts, splitPattern.Split(trailing, -1)...)
	}

	seen := make(map[string]struct{}, len(parts))
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		key := strings.ToLower(p)
		if _, dup := seen[key]; dup {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, p)
	}
	if len(out) == 0 {
		return []string{name}
	}
	return out
}
