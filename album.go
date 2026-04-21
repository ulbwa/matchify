package matchify

import (
	"context"

	"github.com/ulbwa/matchify/internal/explicitparse"
	"github.com/ulbwa/matchify/internal/recordingparse"
	"github.com/ulbwa/matchify/internal/versionparse"
)

// DefaultAlbumThreshold is the score value above which two albums are
// considered a likely match under the default weights.
const DefaultAlbumThreshold = 0.85

// albumConfig holds the tunable parameters of an AlbumMatcher.
type albumConfig struct {
	nameWeight            float64
	artistWeight          float64
	yearWeight            float64
	trackCountWeight      float64
	typeWeight            float64
	upcMismatchCap        float64
	platformIDMismatchCap float64
}

func defaultAlbumConfig() albumConfig {
	return albumConfig{
		nameWeight:            3,
		artistWeight:          2,
		yearWeight:            1,
		trackCountWeight:      0.5,
		typeWeight:            0.3,
		upcMismatchCap:        0.4,
		platformIDMismatchCap: 0.4,
	}
}

// AlbumOption is a functional option for NewAlbumMatcher.
type AlbumOption func(*albumConfig)

// AlbumNameWeight sets the weight of the normalised name similarity
// (default 3).
func AlbumNameWeight(w float64) AlbumOption {
	return func(c *albumConfig) { c.nameWeight = w }
}

// AlbumArtistWeight sets the weight of the artist-list similarity
// (default 2).
func AlbumArtistWeight(w float64) AlbumOption {
	return func(c *albumConfig) { c.artistWeight = w }
}

// AlbumYearWeight sets the weight of release-year agreement (default 1).
// Only takes effect when both sides set WithReleaseDate.
func AlbumYearWeight(w float64) AlbumOption {
	return func(c *albumConfig) { c.yearWeight = w }
}

// AlbumTrackCountWeight sets the weight of track-count agreement
// (default 0.5).
func AlbumTrackCountWeight(w float64) AlbumOption {
	return func(c *albumConfig) { c.trackCountWeight = w }
}

// AlbumTypeWeight sets the weight of release-type agreement (album /
// single / EP / compilation) when both sides set WithReleaseType
// (default 0.3).
func AlbumTypeWeight(w float64) AlbumOption {
	return func(c *albumConfig) { c.typeWeight = w }
}

// AlbumUPCMismatchCap caps the score when both sides provide a UPC via
// WithUPC and the codes differ (default 0.4). Set to 1 to disable.
func AlbumUPCMismatchCap(cap float64) AlbumOption {
	return func(c *albumConfig) { c.upcMismatchCap = cap }
}

// AlbumPlatformIDMismatchCap caps the score when both sides share a
// platform key via WithPlatformID and that platform's IDs disagree
// (default 0.4). Set to 1 to disable.
func AlbumPlatformIDMismatchCap(cap float64) AlbumOption {
	return func(c *albumConfig) { c.platformIDMismatchCap = cap }
}

// AlbumMatcher scores the similarity of two Album values.
type AlbumMatcher struct {
	cfg albumConfig
}

// NewAlbumMatcher returns an AlbumMatcher with the given options applied
// on top of the baseline defaults.
func NewAlbumMatcher(opts ...AlbumOption) *AlbumMatcher {
	cfg := defaultAlbumConfig()
	for _, fn := range opts {
		if fn != nil {
			fn(&cfg)
		}
	}
	return &AlbumMatcher{cfg: cfg}
}

// Match scores the similarity of a and b.
//
// Authoritative short-circuits (each returns Relation=Same, Value≈1):
//  1. UPC tags match — same catalog entry.
//  2. MBID tags match — MusicBrainz release identifiers.
//  3. Platform-ID tags match on at least one shared platform.
//
// Otherwise the score is a weighted combination of normalised name
// similarity, artist-list overlap, release-year agreement, track-count
// agreement, and release-type agreement. Recording-variant and explicit
// disagreements downgrade Relation to Variant; authoritative-ID
// disagreements downgrade Relation to Unrelated.
func (m *AlbumMatcher) Match(ctx context.Context, a, b Album) Score {
	if upcA, okA := a.Tags.UPC(); okA {
		if upcB, okB := b.Tags.UPC(); okB && upcA == upcB {
			return authoritativeScore("upc", "UPC match", 1)
		}
	}
	if mbidA, okA := a.Tags.MBID(); okA {
		if mbidB, okB := b.Tags.MBID(); okB && mbidA == mbidB {
			return authoritativeScore("mbid", "MBID match", 1)
		}
	}
	if comparePlatformIDs(a.Tags.PlatformIDs(), b.Tags.PlatformIDs()) == externalIDEqual {
		return authoritativeScore("platform_id", "platform ID match", 1)
	}

	signals := []Signal{
		{Name: "name", Value: nameSimilarity(a.Name, b.Name), Weight: m.cfg.nameWeight},
		artistListSignal(a.Artists, b.Artists, m.cfg.artistWeight),
	}

	yA, _ := a.Tags.ReleaseYear()
	yB, _ := b.Tags.ReleaseYear()
	signals = append(signals, yearSignal(yA, yB, m.cfg.yearWeight))

	cA, _ := a.Tags.TrackCount()
	cB, _ := b.Tags.TrackCount()
	signals = append(signals, trackCountSignal(cA, cB, m.cfg.trackCountWeight))

	tA, okTA := a.Tags.ReleaseType()
	tB, okTB := b.Tags.ReleaseType()
	if okTA && okTB {
		value := 0.0
		if tA == tB {
			value = 1
		}
		signals = append(signals, Signal{
			Name:   "type",
			Value:  value,
			Weight: m.cfg.typeWeight,
		})
	}

	score := scoreOf(signals...)

	// Edition disagreement (plain vs Deluxe, plain vs 2011 Remaster)
	// downgrades to Variant: same underlying album, different packaging.
	// Only Relation changes; Value remains a measure of variant
	// confidence.
	if !versionparse.SameEdition(a.Name, b.Name) {
		markersA := versionparse.Markers(a.Name)
		markersB := versionparse.Markers(b.Name)
		if len(markersA) > 0 || len(markersB) > 0 {
			score.Relation = RelationVariant
			score.Signals = append(score.Signals, Signal{
				Name:   "edition",
				Value:  0,
				Weight: 0,
				Note:   editionNote(markersA, markersB),
			})
		}
	}

	// Recording-variant disagreement: different re-recordings of the
	// same work — stripped, extended cut, live, acoustic.
	if !recordingparse.SameVariant(a.Name, b.Name) {
		score.Relation = RelationVariant
		score.Signals = append(score.Signals, Signal{
			Name: "variant", Value: 0, Weight: 0, Note: "recording variant differs",
		})
	}

	// Explicit/clean disagreement.
	if effectiveAlbumExplicitDiffers(a, b) {
		score.Relation = RelationVariant
		score.Signals = append(score.Signals, Signal{
			Name: "explicit", Value: 0, Weight: 0, Note: "explicit/clean differs",
		})
	}

	// Authoritative-ID disagreements mean definitely-different products.
	if upcA, okA := a.Tags.UPC(); okA {
		if upcB, okB := b.Tags.UPC(); okB && upcA != upcB {
			score.Relation = RelationUnrelated
			score.Signals = append(score.Signals, Signal{
				Name: "upc", Value: 0, Weight: 0, Note: "UPC mismatch",
			})
			if score.Value > m.cfg.upcMismatchCap {
				score.Value = m.cfg.upcMismatchCap
			}
		}
	}
	if comparePlatformIDs(a.Tags.PlatformIDs(), b.Tags.PlatformIDs()) == externalIDDifferent {
		score.Relation = RelationUnrelated
		score.Signals = append(score.Signals, Signal{
			Name: "platform_id", Value: 0, Weight: 0, Note: "platform ID mismatch",
		})
		if score.Value > m.cfg.platformIDMismatchCap {
			score.Value = m.cfg.platformIDMismatchCap
		}
	}

	return score
}

// effectiveAlbumExplicitDiffers applies the same fallback logic as
// tracks: a WithExplicit tag takes priority, otherwise parse the name.
func effectiveAlbumExplicitDiffers(a, b Album) bool {
	ea := effectiveAlbumExplicit(a)
	eb := effectiveAlbumExplicit(b)
	return ea != ExplicitnessUnknown && eb != ExplicitnessUnknown && ea != eb
}

func effectiveAlbumExplicit(a Album) Explicitness {
	if e, ok := a.Tags.Explicit(); ok {
		return e
	}
	switch explicitparse.Detect(a.Name) {
	case explicitparse.Clean:
		return ExplicitnessClean
	case explicitparse.Explicit:
		return ExplicitnessExplicit
	default:
		return ExplicitnessUnknown
	}
}

func editionNote(a, b []versionparse.Marker) string {
	if len(a) == 0 {
		return "base vs " + joinMarkers(b)
	}
	if len(b) == 0 {
		return joinMarkers(a) + " vs base"
	}
	return joinMarkers(a) + " vs " + joinMarkers(b)
}

func joinMarkers(ms []versionparse.Marker) string {
	if len(ms) == 0 {
		return "base"
	}
	out := string(ms[0])
	for _, m := range ms[1:] {
		out += "+" + string(m)
	}
	return out
}
