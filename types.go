package matchify

// Platform identifies a music platform. It is a free-form string so callers
// can register their own platform names; the constants defined here cover
// the platforms the library is tested against.
type Platform string

// Built-in Platform identifiers. Callers may also define their own.
const (
	PlatformSpotify    Platform = "spotify"
	PlatformAppleMusic Platform = "apple_music"
	PlatformTidal      Platform = "tidal"
	PlatformQobuz      Platform = "qobuz"
	PlatformDeezer     Platform = "deezer"
	PlatformYouTube    Platform = "youtube_music"
	PlatformAmazon     Platform = "amazon_music"
	PlatformSoundCloud Platform = "soundcloud"
	PlatformVK         Platform = "vk_music"
)

// ReleaseType categorises a release.
type ReleaseType uint8

// Known release-type categories. ReleaseTypeUnknown is the zero value and
// means the caller did not supply this information.
const (
	ReleaseTypeUnknown ReleaseType = iota
	ReleaseTypeAlbum
	ReleaseTypeSingle
	ReleaseTypeEP
	ReleaseTypeCompilation
)

// String returns the lower-case name of the release type.
func (r ReleaseType) String() string {
	switch r {
	case ReleaseTypeAlbum:
		return "album"
	case ReleaseTypeSingle:
		return "single"
	case ReleaseTypeEP:
		return "ep"
	case ReleaseTypeCompilation:
		return "compilation"
	default:
		return "unknown"
	}
}

// Explicitness is a tri-state indicator of whether a recording or release
// contains explicit content. Clean and explicit masters are distinct
// products — a mismatch between two sides' explicitness is treated as
// evidence that the sides are different products, not the same.
type Explicitness uint8

// Explicitness values. ExplicitnessUnknown is the zero value.
const (
	ExplicitnessUnknown Explicitness = iota
	ExplicitnessClean
	ExplicitnessExplicit
)

// String returns the lower-case name of the explicitness value.
func (e Explicitness) String() string {
	switch e {
	case ExplicitnessClean:
		return "clean"
	case ExplicitnessExplicit:
		return "explicit"
	default:
		return "unknown"
	}
}

// Artist represents a performer. Only Name is required; all further
// attributes (MBID, per-platform IDs, aliases) go through Tags.
//
// Because artist-level identifiers are frequently missing on platforms
// like SoundCloud or VK Music, matching often has to work from the name
// alone. ArtistMatcher can use an optional ReleaseProvider to disambiguate
// in that case — see NewArtistMatcher.
type Artist struct {
	// Name is the primary display name from the source platform.
	Name string

	// Tags carries any optional metadata the platform provided
	// (MBID, per-platform IDs, aliases, ...). Use NewTags to build.
	Tags Tags
}

// Album represents a release (album, single, EP, compilation). Only Name
// and Artists are required; further attributes (release date, type,
// track count, UPC, MBID, per-platform IDs, ...) go through Tags.
type Album struct {
	// Name is the release title.
	Name string

	// Artists lists the primary artists credited on the release. At
	// least one artist is expected — matching an album against another
	// with no artists is allowed but yields a low score.
	Artists []Artist

	// Tags carries any optional metadata the platform provided.
	Tags Tags
}

// Track represents a single recording on a release. Only Name and
// Artists are required; an Album pointer is optional context, and all
// further attributes (duration, ISRC, explicit flag, disc/track number,
// per-platform IDs, ...) go through Tags.
type Track struct {
	// Name is the track title. It may contain embedded edition or
	// feature markers — matchers deal with those internally.
	Name string

	// Artists lists all credited artists (primary and featured). The
	// matcher reconciles feature credits encoded in the title against
	// the artist list and against composite-credit strings ("A feat. B"
	// as a single artist entry, as seen on Deezer).
	Artists []Artist

	// Album is optional context: the release this track belongs to.
	// When both sides provide an album it is a useful match signal;
	// when either side leaves it nil, it contributes nothing.
	Album *Album

	// Tags carries any optional metadata the platform provided.
	Tags Tags
}
