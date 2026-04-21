// Package matchify matches artists, albums, and tracks across different
// music platforms (Spotify, Apple Music, Tidal, Qobuz, SoundCloud, VK
// Music, ...).
//
// The library operates purely on the data the caller passes in — it
// never fetches anything over the network on its own. Optional metadata
// (duration, ISRC, UPC, explicit flag, per-platform IDs, ...) flows in
// via the Tags container, built with NewTags and tag constructors like
// WithDuration or WithISRC. Absence of a tag is unambiguous — accessors
// return (zero, false) when a field is not set — so "zero seconds" and
// "unknown duration" do not collide.
//
// The scoring model distinguishes relationship (Relation) from
// confidence (Score.Value). Two tracks can be the same product
// (RelationSame), variants of the same work (RelationVariant: different
// edition, remaster, explicit/clean master, stripped re-recording,
// remix, ...), or unrelated. Callers use Score.Same(threshold) for the
// strict "collapse duplicates" question and Score.Related(threshold) to
// find everything associated with a song or album.
//
// # Top-level usage
//
//	m := matchify.NewTrackMatcher()
//	score := m.Match(ctx, trackA, trackB)
//	if score.Same(matchify.DefaultTrackThreshold) {
//	    // merge
//	}
//
// # Functional options
//
// Matchers are configured with type-scoped functional options passed to
// the constructor:
//
//	m := matchify.NewTrackMatcher(
//	    matchify.TrackNameWeight(4),
//	    matchify.TrackDurationMismatch(30, 0.5),
//	)
//
// # Tags
//
// Optional metadata on Artist/Album/Track is carried via Tags:
//
//	t := matchify.Track{
//	    Name:    "Despacito",
//	    Artists: []matchify.Artist{{Name: "Luis Fonsi"}},
//	    Tags: matchify.NewTags(
//	        matchify.WithDuration(228 * time.Second),
//	        matchify.WithISRC("USMV10000001"),
//	        matchify.WithExplicit(matchify.ExplicitnessExplicit),
//	        matchify.WithPlatformID(matchify.PlatformSpotify, "1i1fxkWeaMmKEB4T7zqbzK"),
//	    ),
//	}
//
// # Generic helpers
//
// FindBest and Group are generic helpers on top of any Matcher[T]; they
// take an Accept predicate (IsSame or IsRelated) so callers pick their
// own strictness. See the documentation for matchify.FindBest and
// matchify.Group.
package matchify
