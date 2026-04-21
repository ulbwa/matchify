package matchify

import "time"

// Tag is a functional option applied to a Tags value by NewTags.
//
// A Tag represents one piece of optional metadata that a platform may or
// may not ship alongside the core identifying fields of an Artist, Album,
// or Track. Users construct a Tags value by passing whichever tag
// constructors are applicable — absent tags stay absent, and the matcher
// scores from the signals that are present rather than penalising missing
// ones.
type Tag func(*tagsData)

// Tags is an opaque, immutable container of optional entity metadata.
//
// Use NewTags to build a Tags value; use the typed accessors
// (Duration, ISRC, MBID, PlatformID, ...) to read individual tags. Each
// accessor returns (value, true) if the tag is set and (zero, false) if
// it is not — this makes presence unambiguous and avoids the zero-value
// trap where 0 could mean either "zero seconds" or "unknown".
//
// The zero Tags is valid and represents "nothing beyond the core fields
// is known".
type Tags struct {
	data *tagsData
}

// tagsData holds the actual values for every known optional field. A nil
// pointer for a singular field means "not set"; an empty slice/map means
// "not set" for their respective collection tags.
type tagsData struct {
	duration    *time.Duration
	discNumber  *int
	trackNumber *int
	trackCount  *int
	explicit    *Explicitness
	isrc        *string
	upc         *string
	mbid        *string
	releaseDate *time.Time
	releaseType *ReleaseType
	aliases     []string
	platformIDs map[Platform]string
}

// NewTags builds a Tags value by applying the given tag constructors in
// order. Calling NewTags with no arguments yields a zero Tags.
func NewTags(tags ...Tag) Tags {
	if len(tags) == 0 {
		return Tags{}
	}
	d := &tagsData{}
	for _, fn := range tags {
		if fn != nil {
			fn(d)
		}
	}
	return Tags{data: d}
}

// ---- Tag constructors ----

// WithDuration records the playback duration of a track.
func WithDuration(d time.Duration) Tag {
	return func(t *tagsData) { t.duration = &d }
}

// WithDiscNumber records the 1-based disc number of a track.
func WithDiscNumber(n int) Tag {
	return func(t *tagsData) { t.discNumber = &n }
}

// WithTrackNumber records the 1-based track position within its disc.
func WithTrackNumber(n int) Tag {
	return func(t *tagsData) { t.trackNumber = &n }
}

// WithTrackCount records the number of tracks on an album.
func WithTrackCount(n int) Tag {
	return func(t *tagsData) { t.trackCount = &n }
}

// WithExplicit records the explicit/clean status of a recording or
// release. Two entities whose explicit tags disagree are treated as
// distinct products — they must not be merged.
func WithExplicit(e Explicitness) Tag {
	return func(t *tagsData) { t.explicit = &e }
}

// WithISRC records the International Standard Recording Code of a track.
func WithISRC(code string) Tag {
	return func(t *tagsData) { t.isrc = &code }
}

// WithUPC records the Universal Product Code / EAN of an album.
func WithUPC(code string) Tag {
	return func(t *tagsData) { t.upc = &code }
}

// WithMBID records the MusicBrainz identifier of an artist, album, or
// track recording.
func WithMBID(id string) Tag {
	return func(t *tagsData) { t.mbid = &id }
}

// WithReleaseDate records the first-publication date of an album.
func WithReleaseDate(date time.Time) Tag {
	return func(t *tagsData) { t.releaseDate = &date }
}

// WithReleaseType records the release-type categorisation of an album
// (album, single, EP, compilation).
func WithReleaseType(r ReleaseType) Tag {
	return func(t *tagsData) { t.releaseType = &r }
}

// WithAlias appends alternative names that matchers should consider
// equivalent to the primary name. Suitable for known transliterations
// ("P!nk" / "Pink"), former band names, or region-specific renderings.
// Multiple WithAlias calls accumulate; an empty name is ignored.
func WithAlias(names ...string) Tag {
	return func(t *tagsData) {
		for _, n := range names {
			if n != "" {
				t.aliases = append(t.aliases, n)
			}
		}
	}
}

// WithPlatformID records a per-platform identifier for this entity. Two
// entities that share the same (platform, id) pair are treated as a
// definite match; two that share a platform but differ on the id are
// treated as definitely-different on that platform. Multiple
// WithPlatformID calls set one id per platform; a later call for the
// same platform overrides the earlier value.
func WithPlatformID(p Platform, id string) Tag {
	return func(t *tagsData) {
		if id == "" {
			return
		}
		if t.platformIDs == nil {
			t.platformIDs = map[Platform]string{}
		}
		t.platformIDs[p] = id
	}
}

// ---- Accessors ----

// Duration returns the duration tag if set.
func (t Tags) Duration() (time.Duration, bool) {
	if t.data == nil || t.data.duration == nil {
		return 0, false
	}
	return *t.data.duration, true
}

// DiscNumber returns the disc-number tag if set.
func (t Tags) DiscNumber() (int, bool) {
	if t.data == nil || t.data.discNumber == nil {
		return 0, false
	}
	return *t.data.discNumber, true
}

// TrackNumber returns the track-number tag if set.
func (t Tags) TrackNumber() (int, bool) {
	if t.data == nil || t.data.trackNumber == nil {
		return 0, false
	}
	return *t.data.trackNumber, true
}

// TrackCount returns the track-count tag if set.
func (t Tags) TrackCount() (int, bool) {
	if t.data == nil || t.data.trackCount == nil {
		return 0, false
	}
	return *t.data.trackCount, true
}

// Explicit returns the explicit tag if set.
func (t Tags) Explicit() (Explicitness, bool) {
	if t.data == nil || t.data.explicit == nil {
		return ExplicitnessUnknown, false
	}
	return *t.data.explicit, true
}

// ISRC returns the ISRC tag if set.
func (t Tags) ISRC() (string, bool) {
	if t.data == nil || t.data.isrc == nil {
		return "", false
	}
	return *t.data.isrc, true
}

// UPC returns the UPC tag if set.
func (t Tags) UPC() (string, bool) {
	if t.data == nil || t.data.upc == nil {
		return "", false
	}
	return *t.data.upc, true
}

// MBID returns the MusicBrainz identifier tag if set.
func (t Tags) MBID() (string, bool) {
	if t.data == nil || t.data.mbid == nil {
		return "", false
	}
	return *t.data.mbid, true
}

// ReleaseDate returns the release-date tag if set.
func (t Tags) ReleaseDate() (time.Time, bool) {
	if t.data == nil || t.data.releaseDate == nil {
		return time.Time{}, false
	}
	return *t.data.releaseDate, true
}

// ReleaseYear is a convenience that extracts the year component of the
// release-date tag if set.
func (t Tags) ReleaseYear() (int, bool) {
	d, ok := t.ReleaseDate()
	if !ok {
		return 0, false
	}
	return d.Year(), true
}

// ReleaseType returns the release-type tag if set.
func (t Tags) ReleaseType() (ReleaseType, bool) {
	if t.data == nil || t.data.releaseType == nil {
		return ReleaseTypeUnknown, false
	}
	return *t.data.releaseType, true
}

// Aliases returns the alias tags. The slice is always non-nil but may be
// empty.
func (t Tags) Aliases() []string {
	if t.data == nil {
		return nil
	}
	return t.data.aliases
}

// PlatformIDs returns all platform-id tags as a map. The result is always
// non-nil but may be empty; callers must not mutate it.
func (t Tags) PlatformIDs() map[Platform]string {
	if t.data == nil || t.data.platformIDs == nil {
		return map[Platform]string{}
	}
	return t.data.platformIDs
}

// PlatformID returns the id recorded for the given platform if set.
func (t Tags) PlatformID(p Platform) (string, bool) {
	if t.data == nil || t.data.platformIDs == nil {
		return "", false
	}
	id, ok := t.data.platformIDs[p]
	return id, ok
}
