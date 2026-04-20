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

// AlbumMatcherOptions configures an AlbumMatcher.
type AlbumMatcherOptions struct {
	// NameWeight controls how strongly the normalised album title drives
	// the score. Defaults to 3.
	NameWeight float64

	// ArtistWeight controls how strongly the album-artists list drives
	// the score. Defaults to 2.
	ArtistWeight float64

	// YearWeight controls the contribution of release-year agreement.
	// Defaults to 1.
	YearWeight float64

	// TrackCountWeight controls the contribution of track-count
	// agreement. Defaults to 0.5.
	TrackCountWeight float64

	// TypeWeight controls the contribution of release-type agreement.
	// Defaults to 0.3.
	TypeWeight float64

	// EditionPenalty is subtracted from Value when both sides have
	// version markers and they disagree (e.g. plain vs Deluxe). Defaults
	// to 0.1. Set to 0 to treat all editions as equivalent.
	EditionPenalty float64

	// UPCMismatchCap caps the score when both sides declare a UPC and
	// the codes differ. Defaults to 0.4; set to 1 to disable.
	UPCMismatchCap float64

	// VariantMismatchCap caps the score when the album names contain
	// different sets of recording-variant markers (e.g. plain vs
	// Stripped, Extended Cut vs Stripped). Defaults to 0.5. These are
	// distinct versions of the release and must not collapse into one.
	VariantMismatchCap float64

	// ExplicitMismatchCap caps the score when the two albums disagree
	// on their explicit/clean status (either inferred from the name —
	// "(Clean Version)", "(Explicit)" — or provided elsewhere). Clean
	// and explicit masters are distinct products. Defaults to 0.3.
	ExplicitMismatchCap float64
}

// AlbumMatcher scores the similarity of two Album values.
type AlbumMatcher struct {
	opts AlbumMatcherOptions
}

// NewAlbumMatcher returns an AlbumMatcher with the given options.
func NewAlbumMatcher(opts AlbumMatcherOptions) *AlbumMatcher {
	applyDefault(&opts.NameWeight, 3)
	applyDefault(&opts.ArtistWeight, 2)
	applyDefault(&opts.YearWeight, 1)
	applyDefault(&opts.TrackCountWeight, 0.5)
	applyDefault(&opts.TypeWeight, 0.3)
	applyDefault(&opts.EditionPenalty, 0.1)
	applyDefault(&opts.UPCMismatchCap, 0.4)
	applyDefault(&opts.VariantMismatchCap, 0.5)
	applyDefault(&opts.ExplicitMismatchCap, 0.3)
	return &AlbumMatcher{opts: opts}
}

// Match scores the similarity of a and b.
//
// Authoritative short-circuits (in priority order):
//  1. Matching UPC — the same UPC means the same catalog entry.
//  2. Matching MBID — MusicBrainz release identifiers.
//  3. Matching platform ID — same ID on the same platform.
//
// Otherwise the score is a weighted combination of normalised name
// similarity (with edition markers stripped by normalisation), artist-list
// overlap, release-year agreement, track-count agreement, and release-type
// agreement. If both albums carry edition markers and the sets differ
// (e.g. plain vs Deluxe) a small penalty is subtracted.
func (m *AlbumMatcher) Match(ctx context.Context, a, b Album) Score {
	if a.UPC != "" && b.UPC != "" && a.UPC == b.UPC {
		return authoritativeScore("upc", "UPC match", 1)
	}
	if a.MBID != "" && b.MBID != "" && a.MBID == b.MBID {
		return authoritativeScore("mbid", "MBID match", 1)
	}
	if comparePlatformIDs(a.ExternalIDs, b.ExternalIDs) == externalIDEqual {
		return authoritativeScore("platform_id", "platform ID match", 1)
	}

	signals := []Signal{
		{Name: "name", Value: nameSimilarity(a.Name, b.Name), Weight: m.opts.NameWeight},
		artistListSignal(a.Artists, b.Artists, m.opts.ArtistWeight),
		yearSignal(a.ReleaseYear(), b.ReleaseYear(), m.opts.YearWeight),
		trackCountSignal(a.TrackCount, b.TrackCount, m.opts.TrackCountWeight),
	}

	if a.Type != ReleaseTypeUnknown && b.Type != ReleaseTypeUnknown {
		value := 0.0
		if a.Type == b.Type {
			value = 1
		}
		signals = append(signals, Signal{
			Name:   "type",
			Value:  value,
			Weight: m.opts.TypeWeight,
		})
	}

	score := scoreOf(signals...)

	if !versionparse.SameEdition(a.Name, b.Name) {
		markersA := versionparse.Markers(a.Name)
		markersB := versionparse.Markers(b.Name)
		if len(markersA) > 0 || len(markersB) > 0 {
			score.Value -= m.opts.EditionPenalty
			if score.Value < 0 {
				score.Value = 0
			}
			score.Signals = append(score.Signals, Signal{
				Name:   "edition",
				Value:  0,
				Weight: 0,
				Note:   editionNote(markersA, markersB),
			})
		}
	}

	if !recordingparse.SameVariant(a.Name, b.Name) {
		score.Signals = append(score.Signals, Signal{
			Name: "variant", Value: 0, Weight: 0, Note: "recording variant differs",
		})
		if score.Value > m.opts.VariantMismatchCap {
			score.Value = m.opts.VariantMismatchCap
		}
	}

	if explicitparse.Differ(explicitparse.Detect(a.Name), explicitparse.Detect(b.Name)) {
		score.Signals = append(score.Signals, Signal{
			Name: "explicit", Value: 0, Weight: 0, Note: "explicit/clean differs",
		})
		if score.Value > m.opts.ExplicitMismatchCap {
			score.Value = m.opts.ExplicitMismatchCap
		}
	}

	if a.UPC != "" && b.UPC != "" && a.UPC != b.UPC {
		score.Signals = append(score.Signals, Signal{
			Name: "upc", Value: 0, Weight: 0, Note: "UPC mismatch",
		})
		if score.Value > m.opts.UPCMismatchCap {
			score.Value = m.opts.UPCMismatchCap
		}
	}
	if comparePlatformIDs(a.ExternalIDs, b.ExternalIDs) == externalIDDifferent {
		score.Signals = append(score.Signals, Signal{
			Name: "platform_id", Value: 0, Weight: 0, Note: "platform ID mismatch",
		})
		if score.Value > m.opts.UPCMismatchCap {
			score.Value = m.opts.UPCMismatchCap
		}
	}

	return score
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
