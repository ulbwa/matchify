package matchify

import (
	"context"
	"time"

	"github.com/ulbwa/matchify/internal/explicitparse"
	"github.com/ulbwa/matchify/internal/featparse"
	"github.com/ulbwa/matchify/internal/recordingparse"
	"github.com/ulbwa/matchify/internal/textnorm"
)

// DefaultTrackThreshold is the score value above which two tracks are
// considered a likely match under the default weights.
const DefaultTrackThreshold = 0.85

// trackConfig holds the tunable parameters of a TrackMatcher. It is
// unexported; construct a TrackMatcher with NewTrackMatcher and zero or
// more TrackOption values.
type trackConfig struct {
	nameWeight              float64
	artistWeight            float64
	durationWeight          float64
	albumWeight             float64
	trackPositionWeight     float64
	isrcMismatchCap         float64
	platformIDMismatchCap   float64
	durationMismatchSeconds float64
	durationMismatchCap     float64
}

// defaultTrackConfig returns the baseline config applied before any
// caller options run.
func defaultTrackConfig() trackConfig {
	return trackConfig{
		nameWeight:              3,
		artistWeight:            3,
		durationWeight:          1,
		albumWeight:             1,
		trackPositionWeight:     0.5,
		isrcMismatchCap:         0.4,
		platformIDMismatchCap:   0.4,
		durationMismatchSeconds: 30,
		durationMismatchCap:     0.55,
	}
}

// TrackOption is a functional option for NewTrackMatcher.
type TrackOption func(*trackConfig)

// TrackNameWeight sets the weight of the normalised title similarity
// (default 3). Larger values make the title more dominant in the score.
func TrackNameWeight(w float64) TrackOption {
	return func(c *trackConfig) { c.nameWeight = w }
}

// TrackArtistWeight sets the weight of the merged artist-list similarity
// (default 3).
func TrackArtistWeight(w float64) TrackOption {
	return func(c *trackConfig) { c.artistWeight = w }
}

// TrackDurationWeight sets the weight of the duration agreement signal
// (default 1). Only takes effect when both sides provide a duration via
// WithDuration.
func TrackDurationWeight(w float64) TrackOption {
	return func(c *trackConfig) { c.durationWeight = w }
}

// TrackAlbumWeight sets the weight of album-title agreement (default 1).
// Only takes effect when both sides set a non-nil Album.
func TrackAlbumWeight(w float64) TrackOption {
	return func(c *trackConfig) { c.albumWeight = w }
}

// TrackPositionWeight sets the weight of matching disc/track numbers
// (default 0.5).
func TrackPositionWeight(w float64) TrackOption {
	return func(c *trackConfig) { c.trackPositionWeight = w }
}

// TrackISRCMismatchCap caps the score when both sides declare an ISRC
// via WithISRC and the codes differ (default 0.4). Set to 1 to disable.
func TrackISRCMismatchCap(cap float64) TrackOption {
	return func(c *trackConfig) { c.isrcMismatchCap = cap }
}

// TrackPlatformIDMismatchCap caps the score when both sides share a
// platform key via WithPlatformID and that platform's IDs disagree
// (default 0.4). Set to 1 to disable.
func TrackPlatformIDMismatchCap(cap float64) TrackOption {
	return func(c *trackConfig) { c.platformIDMismatchCap = cap }
}

// TrackDurationMismatch configures the duration disagreement cap.
// seconds is the tolerance (in seconds, fractional allowed) beyond which
// a score is capped; cap is the maximum score allowed past that
// tolerance. Defaults are 30 s and 0.55.
func TrackDurationMismatch(seconds float64, cap float64) TrackOption {
	return func(c *trackConfig) {
		c.durationMismatchSeconds = seconds
		c.durationMismatchCap = cap
	}
}

// TrackMatcher scores the similarity of two Track values.
type TrackMatcher struct {
	cfg trackConfig
}

// NewTrackMatcher returns a TrackMatcher with the given options applied
// on top of the baseline defaults.
func NewTrackMatcher(opts ...TrackOption) *TrackMatcher {
	cfg := defaultTrackConfig()
	for _, fn := range opts {
		if fn != nil {
			fn(&cfg)
		}
	}
	return &TrackMatcher{cfg: cfg}
}

// Match scores the similarity of a and b.
//
// Authoritative short-circuits (each returns Relation=Same, Value≈1):
//  1. ISRC tags match — ISRCs identify a single recording uniquely.
//  2. MBID tags match — MusicBrainz recording identifiers.
//  3. Platform-ID tags match on at least one shared platform.
//
// Otherwise the score is a weighted combination of normalised title
// similarity, merged artist-list overlap, duration agreement, album-title
// similarity, disc/track positions. If a recording-variant marker or
// explicit flag disagrees between sides, the Score is downgraded to
// RelationVariant and capped accordingly.
func (m *TrackMatcher) Match(ctx context.Context, a, b Track) Score {
	if isrcA, okA := a.Tags.ISRC(); okA {
		if isrcB, okB := b.Tags.ISRC(); okB && isrcA == isrcB {
			return authoritativeScore("isrc", "ISRC match", 1)
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

	cleanedA, featsA := featparse.ExtractFeatures(a.Name)
	cleanedB, featsB := featparse.ExtractFeatures(b.Name)
	artistsA := mergeFeatureArtists(a.Artists, featsA)
	artistsB := mergeFeatureArtists(b.Artists, featsB)

	signals := []Signal{
		{Name: "name", Value: nameSimilarity(cleanedA, cleanedB), Weight: m.cfg.nameWeight},
		artistListSignal(artistsA, artistsB, m.cfg.artistWeight),
	}

	durA, _ := a.Tags.Duration()
	durB, _ := b.Tags.Duration()
	signals = append(signals, durationSignal(durA, durB, m.cfg.durationWeight))

	if a.Album != nil && b.Album != nil && a.Album.Name != "" && b.Album.Name != "" {
		signals = append(signals, Signal{
			Name:   "album",
			Value:  nameSimilarity(a.Album.Name, b.Album.Name),
			Weight: m.cfg.albumWeight,
		})
	}

	tnA, okTnA := a.Tags.TrackNumber()
	tnB, okTnB := b.Tags.TrackNumber()
	if okTnA && okTnB {
		value := 0.0
		if tnA == tnB {
			value = 1
		}
		if dnA, okA := a.Tags.DiscNumber(); okA {
			if dnB, okB := b.Tags.DiscNumber(); okB && dnA != dnB {
				value = 0
			}
		}
		signals = append(signals, Signal{
			Name:   "track_number",
			Value:  value,
			Weight: m.cfg.trackPositionWeight,
		})
	}

	score := scoreOf(signals...)

	// Recording-variant markers (live/acoustic/remix/stripped/extended
	// cut/...) mean the two sides are recording variants of the same
	// work rather than the same product. Only Relation is downgraded
	// — Value remains as a measure of how confident we are in the
	// variant relation.
	if !recordingparse.SameVariant(a.Name, b.Name) {
		score.Relation = RelationVariant
		score.Signals = append(score.Signals, Signal{
			Name: "variant", Value: 0, Weight: 0, Note: variantMismatchNote(a.Name, b.Name),
		})
	}

	// Explicit/clean disagreement: same song, different master — a
	// variant, not the same product.
	if effectiveExplicitnessDiffers(a, b) {
		score.Relation = RelationVariant
		score.Signals = append(score.Signals, Signal{
			Name: "explicit", Value: 0, Weight: 0, Note: "explicit/clean differs",
		})
	}

	// Large duration disagreement: likely different recordings entirely
	// — a 30s clip is not the "same" as a 4min track even if name and
	// artists happen to agree. Mark as unrelated and cap the score.
	if durA > 0 && durB > 0 {
		diff := durA - durB
		if diff < 0 {
			diff = -diff
		}
		limit := time.Duration(m.cfg.durationMismatchSeconds * float64(time.Second))
		if diff > limit {
			score.Relation = RelationUnrelated
			score.Signals = append(score.Signals, Signal{
				Name: "duration_cap", Value: 0, Weight: 0, Note: "large duration mismatch",
			})
			if score.Value > m.cfg.durationMismatchCap {
				score.Value = m.cfg.durationMismatchCap
			}
		}
	}

	// Authoritative-ID disagreements mean definitely-different products.
	// Cap the score aggressively and mark as unrelated.
	if isrcA, okA := a.Tags.ISRC(); okA {
		if isrcB, okB := b.Tags.ISRC(); okB && isrcA != isrcB {
			score.Relation = RelationUnrelated
			score.Signals = append(score.Signals, Signal{
				Name: "isrc", Value: 0, Weight: 0, Note: "ISRC mismatch",
			})
			if score.Value > m.cfg.isrcMismatchCap {
				score.Value = m.cfg.isrcMismatchCap
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

// mergeFeatureArtists returns artists extended with feature names
// extracted from the title that aren't already present.
func mergeFeatureArtists(artists []Artist, features []string) []Artist {
	if len(features) == 0 {
		return artists
	}
	out := append(make([]Artist, 0, len(artists)+len(features)), artists...)
	for _, f := range features {
		if f == "" {
			continue
		}
		if containsArtistByName(out, f) {
			continue
		}
		out = append(out, Artist{Name: f})
	}
	return out
}

func containsArtistByName(list []Artist, candidate string) bool {
	normCand := textnorm.Normalize(candidate)
	if normCand == "" {
		return false
	}
	for _, a := range list {
		if textnorm.Normalize(a.Name) == normCand {
			return true
		}
		for _, alias := range a.Tags.Aliases() {
			if textnorm.Normalize(alias) == normCand {
				return true
			}
		}
	}
	return false
}

// effectiveExplicitnessDiffers reports whether two tracks disagree on
// their explicit/clean status. A side's effective explicitness is its
// WithExplicit tag value when set, otherwise the value parsed from the
// name. Two sides "differ" only when both have a known effective value
// and those values disagree.
func effectiveExplicitnessDiffers(a, b Track) bool {
	ea := effectiveTrackExplicit(a)
	eb := effectiveTrackExplicit(b)
	return ea != ExplicitnessUnknown && eb != ExplicitnessUnknown && ea != eb
}

func effectiveTrackExplicit(t Track) Explicitness {
	if e, ok := t.Tags.Explicit(); ok {
		return e
	}
	switch explicitparse.Detect(t.Name) {
	case explicitparse.Clean:
		return ExplicitnessClean
	case explicitparse.Explicit:
		return ExplicitnessExplicit
	default:
		return ExplicitnessUnknown
	}
}

func variantMismatchNote(a, b string) string {
	ma := recordingparse.Markers(a)
	mb := recordingparse.Markers(b)
	return markersToString(ma) + " vs " + markersToString(mb)
}

func markersToString(m []recordingparse.Marker) string {
	if len(m) == 0 {
		return "studio"
	}
	out := string(m[0])
	for _, x := range m[1:] {
		out += "+" + string(x)
	}
	return out
}
