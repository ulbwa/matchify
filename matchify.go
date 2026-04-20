// Package matchify matches artists, albums, and tracks across different
// music platforms (Spotify, Apple Music, Tidal, Qobuz, ...).
//
// The library operates purely on the data passed to it. It does not fetch
// anything over the network on its own. If a caller wants artist matching
// to take release overlap into account, it supplies a ReleaseProvider
// implementation; the library will call it as needed.
//
// The scoring scheme is adaptive: it uses whatever signals are present on
// the input entities. When authoritative identifiers such as ISRC (for
// tracks), UPC (for albums), or MusicBrainz IDs are available, they
// short-circuit the comparison. Otherwise the score is a weighted
// combination of normalized-name similarity and contextual signals
// (artist match, duration, release year, track position, ...).
//
// # Top-level usage
//
//	trackMatcher := matchify.NewTrackMatcher(matchify.TrackMatcherOptions{})
//	score := trackMatcher.Match(ctx, trackA, trackB)
//	if score.Above(matchify.DefaultThreshold) {
//	    // ...
//	}
//
// # Generic helpers
//
// FindBest and Group are generic helpers on top of any Matcher[T]. See the
// documentation for matchify.FindBest and matchify.Group.
package matchify

// DefaultThreshold is the score value above which two entities are
// considered a high-confidence match. Individual matchers expose their own
// defaults via the Default*Threshold constants — the value here matches
// the default used by FindBest and Group when no explicit threshold is
// given.
const DefaultThreshold = 0.85
