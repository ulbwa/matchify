package matchify

import (
	"context"
	"time"

	"github.com/ulbwa/matchify/internal/featparse"
	"github.com/ulbwa/matchify/internal/recordingparse"
	"github.com/ulbwa/matchify/internal/textnorm"
)

// DefaultTrackThreshold is the score value above which two tracks are
// considered a likely match under the default weights.
const DefaultTrackThreshold = 0.85

// TrackMatcherOptions configures a TrackMatcher.
type TrackMatcherOptions struct {
	// NameWeight controls how strongly the normalised track title drives
	// the score. Defaults to 3.
	NameWeight float64

	// ArtistWeight controls how strongly the merged artist list drives
	// the score. Defaults to 3.
	ArtistWeight float64

	// DurationWeight controls the contribution of duration agreement when
	// both sides provide a duration. Defaults to 1.
	DurationWeight float64

	// AlbumWeight controls the contribution of album-name agreement when
	// both sides provide an Album. Defaults to 1.
	AlbumWeight float64

	// TrackPositionWeight controls the contribution of matching track/disc
	// numbers when both sides provide them. Defaults to 0.5.
	TrackPositionWeight float64

	// ExplicitWeight controls the contribution of explicitness agreement.
	// Defaults to 0.2 — a small bonus, since clean and explicit masters
	// of the same song should still match.
	ExplicitWeight float64

	// ISRCMismatchCap caps the score when both sides declare an ISRC and
	// the codes differ. Defaults to 0.4; set to 1 to disable the cap.
	ISRCMismatchCap float64

	// VariantMismatchCap caps the score when the two titles contain
	// different sets of recording-variant markers (live, acoustic, remix,
	// demo, ...). Defaults to 0.5; set to 1 to disable the cap.
	VariantMismatchCap float64

	// DurationMismatchSeconds, if non-zero, is the duration difference (in
	// seconds) beyond which the score is capped at DurationMismatchCap.
	// Defaults to 30.
	DurationMismatchSeconds float64

	// DurationMismatchCap caps the score when duration disagreement
	// exceeds DurationMismatchSeconds. Defaults to 0.55.
	DurationMismatchCap float64
}

// TrackMatcher scores the similarity of two Track values.
type TrackMatcher struct {
	opts TrackMatcherOptions
}

// NewTrackMatcher returns a TrackMatcher with the given options. Zero-valued
// options receive sensible defaults.
func NewTrackMatcher(opts TrackMatcherOptions) *TrackMatcher {
	applyDefault(&opts.NameWeight, 3)
	applyDefault(&opts.ArtistWeight, 3)
	applyDefault(&opts.DurationWeight, 1)
	applyDefault(&opts.AlbumWeight, 1)
	applyDefault(&opts.TrackPositionWeight, 0.5)
	applyDefault(&opts.ExplicitWeight, 0.2)
	applyDefault(&opts.ISRCMismatchCap, 0.4)
	applyDefault(&opts.VariantMismatchCap, 0.5)
	applyDefault(&opts.DurationMismatchSeconds, 30)
	applyDefault(&opts.DurationMismatchCap, 0.55)
	return &TrackMatcher{opts: opts}
}

// Match scores the similarity of a and b. The returned Score's Value is in
// [0, 1] where 1 means authoritative identifiers agreed.
//
// Authoritative short-circuits (in priority order):
//  1. Matching ISRC — ISRCs uniquely identify a recording.
//  2. Matching MBID — MusicBrainz recording identifiers.
//  3. Matching platform ID — two tracks with the same Spotify ID, Apple
//     Music ID, etc. must be the same track.
//
// Otherwise the score is a weighted combination of normalised title
// similarity, merged artist-list overlap, duration agreement, album-name
// similarity, track/disc positions, and explicitness agreement.
func (m *TrackMatcher) Match(ctx context.Context, a, b Track) Score {
	if a.ISRC != "" && b.ISRC != "" && a.ISRC == b.ISRC {
		return authoritativeScore("isrc", "ISRC match", 1)
	}
	if a.MBID != "" && b.MBID != "" && a.MBID == b.MBID {
		return authoritativeScore("mbid", "MBID match", 1)
	}
	if comparePlatformIDs(a.ExternalIDs, b.ExternalIDs) == externalIDEqual {
		return authoritativeScore("platform_id", "platform ID match", 1)
	}

	cleanedA, featsA := featparse.ExtractFeatures(a.Name)
	cleanedB, featsB := featparse.ExtractFeatures(b.Name)
	artistsA := mergeFeatureArtists(a.Artists, featsA)
	artistsB := mergeFeatureArtists(b.Artists, featsB)

	signals := []Signal{
		{Name: "name", Value: nameSimilarity(cleanedA, cleanedB), Weight: m.opts.NameWeight},
		artistListSignal(artistsA, artistsB, m.opts.ArtistWeight),
		durationSignal(a.Duration, b.Duration, m.opts.DurationWeight),
	}

	if a.Album != nil && b.Album != nil && a.Album.Name != "" && b.Album.Name != "" {
		signals = append(signals, Signal{
			Name:   "album",
			Value:  nameSimilarity(a.Album.Name, b.Album.Name),
			Weight: m.opts.AlbumWeight,
		})
	}

	if a.TrackNumber > 0 && b.TrackNumber > 0 {
		value := 0.0
		if a.TrackNumber == b.TrackNumber {
			value = 1
		}
		if a.DiscNumber > 0 && b.DiscNumber > 0 && a.DiscNumber != b.DiscNumber {
			value = 0
		}
		signals = append(signals, Signal{
			Name:   "track_number",
			Value:  value,
			Weight: m.opts.TrackPositionWeight,
		})
	}

	signals = append(signals, explicitnessSignal(a.Explicit, b.Explicit, m.opts.ExplicitWeight))

	score := scoreOf(signals...)

	if !recordingparse.SameVariant(a.Name, b.Name) {
		score.Signals = append(score.Signals, Signal{
			Name: "variant", Value: 0, Weight: 0, Note: variantMismatchNote(a.Name, b.Name),
		})
		if score.Value > m.opts.VariantMismatchCap {
			score.Value = m.opts.VariantMismatchCap
		}
	}

	if a.Duration > 0 && b.Duration > 0 {
		diff := a.Duration - b.Duration
		if diff < 0 {
			diff = -diff
		}
		limit := time.Duration(m.opts.DurationMismatchSeconds) * time.Second
		if diff > limit {
			score.Signals = append(score.Signals, Signal{
				Name: "duration", Value: 0, Weight: 0, Note: "large duration mismatch",
			})
			if score.Value > m.opts.DurationMismatchCap {
				score.Value = m.opts.DurationMismatchCap
			}
		}
	}

	if a.ISRC != "" && b.ISRC != "" && a.ISRC != b.ISRC {
		score.Signals = append(score.Signals, Signal{
			Name:   "isrc",
			Value:  0,
			Weight: 0,
			Note:   "ISRC mismatch",
		})
		if score.Value > m.opts.ISRCMismatchCap {
			score.Value = m.opts.ISRCMismatchCap
		}
	}
	if comparePlatformIDs(a.ExternalIDs, b.ExternalIDs) == externalIDDifferent {
		score.Signals = append(score.Signals, Signal{
			Name:   "platform_id",
			Value:  0,
			Weight: 0,
			Note:   "platform ID mismatch",
		})
		if score.Value > m.opts.ISRCMismatchCap {
			score.Value = m.opts.ISRCMismatchCap
		}
	}

	return score
}

// variantMismatchNote describes the variant differences between two titles
// for inclusion in the Score.Signals log.
func variantMismatchNote(a, b string) string {
	ma := recordingparse.Markers(a)
	mb := recordingparse.Markers(b)
	left := markersToString(ma)
	right := markersToString(mb)
	return left + " vs " + right
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

// mergeFeatureArtists returns artists extended with any feature names
// extracted from the track title that aren't already present.
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

// containsArtistByName reports whether any artist in list has a name similar
// to candidate. Used to avoid duplicating feature artists that are already
// credited.
func containsArtistByName(list []Artist, candidate string) bool {
	normCand := textnorm.Normalize(candidate)
	if normCand == "" {
		return false
	}
	for _, a := range list {
		if textnorm.Normalize(a.Name) == normCand {
			return true
		}
		for _, alias := range a.Aliases {
			if textnorm.Normalize(alias) == normCand {
				return true
			}
		}
	}
	return false
}

// applyDefault sets *v to def if *v is zero.
func applyDefault(v *float64, def float64) {
	if *v == 0 {
		*v = def
	}
}
