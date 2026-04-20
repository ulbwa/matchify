// Package versionparse identifies release-version markers in album names.
//
// Album names frequently carry edition markers ("Deluxe Edition", "Remastered
// 2011", "25th Anniversary Edition", "Japan Edition"). These markers indicate
// different packagings of what is conceptually the same album. The matcher
// uses them to decide whether two names refer to the same edition or to
// different editions of the same underlying release.
package versionparse

import "regexp"

// Marker is a canonical identifier for a family of edition labels.
type Marker string

const (
	MarkerDeluxe        Marker = "deluxe"
	MarkerSuperDeluxe   Marker = "super_deluxe"
	MarkerExpanded      Marker = "expanded"
	MarkerExtended      Marker = "extended"
	MarkerAnniversary   Marker = "anniversary"
	MarkerRemaster      Marker = "remaster"
	MarkerJapan         Marker = "japan"
	MarkerInternational Marker = "international"
	MarkerLimited       Marker = "limited"
	MarkerCollector     Marker = "collector"
	MarkerSpecial       Marker = "special"
)

// markerPatterns maps each canonical marker to the regex that matches its
// various surface forms. Patterns are case-insensitive. Order matters: the
// more specific patterns (e.g. "super deluxe") must come before the less
// specific ones (e.g. "deluxe") so that they don't shadow each other.
var markerPatterns = []struct {
	marker Marker
	re     *regexp.Regexp
}{
	{MarkerSuperDeluxe, regexp.MustCompile(`(?i)\bsuper\s+deluxe(?:\s+(?:edition|version))?\b`)},
	{MarkerDeluxe, regexp.MustCompile(`(?i)\bdeluxe(?:\s+(?:edition|version))?\b`)},
	{MarkerExpanded, regexp.MustCompile(`(?i)\bexpanded(?:\s+edition)?\b`)},
	{MarkerExtended, regexp.MustCompile(`(?i)\bextended(?:\s+edition)?\b`)},
	{MarkerAnniversary, regexp.MustCompile(`(?i)\b\d+(?:st|nd|rd|th)?\s*anniversary(?:\s+edition)?\b`)},
	{MarkerAnniversary, regexp.MustCompile(`(?i)\banniversary\s+edition\b`)},
	{MarkerRemaster, regexp.MustCompile(`(?i)\b(?:digitally?\s+)?remaster(?:ed)?(?:\s+\d{4})?\b`)},
	{MarkerRemaster, regexp.MustCompile(`(?i)\b\d{4}\s+remaster(?:ed)?\b`)},
	{MarkerJapan, regexp.MustCompile(`(?i)\b(?:japan(?:ese)?)\s+edition\b`)},
	{MarkerInternational, regexp.MustCompile(`(?i)\binternational\s+edition\b`)},
	{MarkerLimited, regexp.MustCompile(`(?i)\blimited\s+edition\b`)},
	{MarkerCollector, regexp.MustCompile(`(?i)\bcollector(?:'?s)?\s+edition\b`)},
	{MarkerSpecial, regexp.MustCompile(`(?i)\bspecial\s+edition\b`)},
}

// Markers returns the distinct edition markers detected in name, in the order
// they are defined by markerPatterns. If no markers are found, the slice is
// empty.
//
// Patterns are applied in order; each match is elided from the working string
// so later patterns cannot overlap with earlier ones. This is what prevents
// "Super Deluxe Edition" from also reporting a MarkerDeluxe.
func Markers(name string) []Marker {
	working := name
	seen := make(map[Marker]struct{})
	var out []Marker
	for _, mp := range markerPatterns {
		if !mp.re.MatchString(working) {
			continue
		}
		if _, ok := seen[mp.marker]; !ok {
			seen[mp.marker] = struct{}{}
			out = append(out, mp.marker)
		}
		working = mp.re.ReplaceAllString(working, " ")
	}
	return out
}

// SameEdition reports whether two names appear to describe the same album
// edition: the same set of markers or both empty. It is insensitive to marker
// ordering.
func SameEdition(a, b string) bool {
	ma := Markers(a)
	mb := Markers(b)
	if len(ma) != len(mb) {
		return false
	}
	set := make(map[Marker]struct{}, len(ma))
	for _, m := range ma {
		set[m] = struct{}{}
	}
	for _, m := range mb {
		if _, ok := set[m]; !ok {
			return false
		}
	}
	return true
}

// HasAny reports whether name contains any known edition marker.
func HasAny(name string) bool {
	return len(Markers(name)) > 0
}

// String returns the canonical string form of the marker.
func (m Marker) String() string {
	return string(m)
}
