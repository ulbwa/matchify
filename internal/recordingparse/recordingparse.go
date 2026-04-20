// Package recordingparse identifies recording-variant markers in track
// titles — live, acoustic, remix, demo, instrumental, and similar labels
// that denote a distinct recording rather than a cosmetic repackaging.
//
// Two tracks with different sets of recording markers are different
// performances even if the rest of their metadata agrees. Matchers use this
// to penalise such pairs so that, for example, "Bohemian Rhapsody" (the
// studio recording) does not match "Bohemian Rhapsody (Live at Wembley)".
package recordingparse

import "regexp"

// Marker is the canonical identifier of a recording variant.
type Marker string

const (
	MarkerLive         Marker = "live"
	MarkerAcoustic     Marker = "acoustic"
	MarkerUnplugged    Marker = "unplugged"
	MarkerRemix        Marker = "remix"
	MarkerDemo         Marker = "demo"
	MarkerInstrumental Marker = "instrumental"
	MarkerKaraoke      Marker = "karaoke"
	MarkerCover        Marker = "cover"
	MarkerPiano        Marker = "piano"
	MarkerOrchestral   Marker = "orchestral"
	MarkerExtendedMix  Marker = "extended_mix"
	MarkerRadioEdit    Marker = "radio_edit"
	MarkerSession      Marker = "session"
	MarkerRerecording  Marker = "rerecording"
)

// markerPatterns is an ordered list of (marker, regex) pairs. Each pattern
// is applied; when a pattern matches, its region is removed from the
// working string before subsequent patterns are checked, so specific
// markers (e.g. "Extended Mix") aren't also reported as a less specific
// marker (e.g. "remix"? no — they wouldn't overlap, but similar idea).
var markerPatterns = []struct {
	marker Marker
	re     *regexp.Regexp
}{
	{MarkerExtendedMix, regexp.MustCompile(`(?i)\bextended\s+(?:mix|version|edit)\b`)},
	{MarkerRadioEdit, regexp.MustCompile(`(?i)\bradio\s+edit\b`)},
	{MarkerLive, regexp.MustCompile(`(?i)\blive(?:\s+(?:at|in|from)\b|\s*(?:session|version|recording|performance)\b|\s*$|\s*[\)\]])`)},
	{MarkerUnplugged, regexp.MustCompile(`(?i)\bunplugged\b`)},
	{MarkerAcoustic, regexp.MustCompile(`(?i)\bacoustic(?:\s+version)?\b`)},
	{MarkerRerecording, regexp.MustCompile(`(?i)\bre-?(?:recorded|recording)\b|\btaylor'?s\s+version\b`)},
	{MarkerRemix, regexp.MustCompile(`(?i)\b(?:[-\w]+\s+)?remix\b`)},
	{MarkerDemo, regexp.MustCompile(`(?i)\bdemo(?:\s+version)?\b`)},
	{MarkerInstrumental, regexp.MustCompile(`(?i)\binstrumental(?:\s+version|\s+mix)?\b`)},
	{MarkerKaraoke, regexp.MustCompile(`(?i)\bkaraoke\b`)},
	{MarkerPiano, regexp.MustCompile(`(?i)\bpiano\s+(?:version|mix)\b`)},
	{MarkerOrchestral, regexp.MustCompile(`(?i)\borchestral(?:\s+version)?\b`)},
	{MarkerCover, regexp.MustCompile(`(?i)\bcover(?:\s+version)?\b`)},
	{MarkerSession, regexp.MustCompile(`(?i)\b(?:spotify|apple|bbc|abbey\s+road|tiny\s+desk)\s+session\b`)},
}

// Markers returns the distinct recording markers detected in name. Order
// matches the order defined by markerPatterns (specific before general).
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

// SameVariant reports whether two names share the same set of recording
// markers. Two tracks with different sets are different performances.
func SameVariant(a, b string) bool {
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

// String returns the string form of the marker.
func (m Marker) String() string { return string(m) }
