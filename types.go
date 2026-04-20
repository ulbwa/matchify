package matchify

import "time"

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

// Explicitness is a tri-state indicator of whether a track contains
// explicit content.
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

// Artist represents a performer. The Name field is required; every other
// field is optional. A zero value on an optional field is interpreted as
// "unknown".
type Artist struct {
	// Name is the primary display name used by the source platform.
	Name string

	// Aliases are alternative spellings or transliterations the caller
	// wants considered equivalent (e.g. "P!nk" alongside "Pink"). They are
	// compared together with Name during matching.
	Aliases []string

	// MBID is the MusicBrainz artist identifier, if known. Treated as
	// authoritative when present on both sides.
	MBID string

	// ExternalIDs maps a Platform to that platform's artist identifier.
	// Two artists with the same ID on the same platform are considered a
	// definite match.
	ExternalIDs map[Platform]string
}

// Album represents a release (album, single, EP, compilation).
type Album struct {
	// Name is the release title.
	Name string

	// Artists lists the primary artists credited on the release.
	Artists []Artist

	// Type categorises the release. ReleaseTypeUnknown means the caller did
	// not supply this information.
	Type ReleaseType

	// ReleaseDate is the original release date. A zero time means unknown.
	// Callers who only know the year can pass time.Date(year, 1, 1, ...).
	ReleaseDate time.Time

	// TrackCount is the number of tracks on the release. Zero means
	// unknown.
	TrackCount int

	// UPC is the Universal Product Code / EAN for the release. Treated as
	// authoritative when present on both sides.
	UPC string

	// MBID is the MusicBrainz release (or release-group) identifier, if
	// known.
	MBID string

	// ExternalIDs maps a Platform to that platform's album identifier.
	ExternalIDs map[Platform]string
}

// ReleaseYear returns the year of ReleaseDate, or 0 if unknown.
func (a Album) ReleaseYear() int {
	if a.ReleaseDate.IsZero() {
		return 0
	}
	return a.ReleaseDate.Year()
}

// Track represents a single recording on a release.
type Track struct {
	// Name is the track title.
	Name string

	// Artists lists all credited artists (primary and featured). The
	// matcher will attempt to reconcile feature credits encoded in the
	// track title with those listed here.
	Artists []Artist

	// Album is the release the track belongs to, if known. Providing it
	// strengthens the match when track-level information is ambiguous.
	Album *Album

	// Duration is the track duration. Zero means unknown.
	Duration time.Duration

	// DiscNumber is the 1-based disc number. Zero means unknown.
	DiscNumber int

	// TrackNumber is the 1-based position of the track on its disc. Zero
	// means unknown.
	TrackNumber int

	// Explicit indicates whether the recording contains explicit content.
	// Matchers treat different-but-known values as the same track on the
	// assumption that clean and explicit masters of the same recording
	// exist; the Explicit signal contributes to disambiguation, not
	// rejection.
	Explicit Explicitness

	// ISRC is the International Standard Recording Code for the
	// recording. Treated as authoritative when present on both sides.
	ISRC string

	// MBID is the MusicBrainz recording identifier, if known.
	MBID string

	// ExternalIDs maps a Platform to that platform's track identifier.
	ExternalIDs map[Platform]string
}
