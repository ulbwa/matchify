package matchify

import (
	"context"
	"sort"
	"strings"
	"sync"

	"github.com/ulbwa/matchify/internal/textnorm"
)

// DefaultArtistThreshold is the score value above which two artists are
// considered a likely match under the default weights.
const DefaultArtistThreshold = 0.85

// ArtistMatcherOptions configures an ArtistMatcher.
type ArtistMatcherOptions struct {
	// ReleaseProvider, if non-nil, is consulted when name-based scoring
	// is ambiguous. The matcher fetches each artist's discography once
	// per unique identity (per matcher instance) and looks for a single
	// release that matches under AlbumMatcher; a match there lifts the
	// artist score above the threshold.
	ReleaseProvider ReleaseProvider

	// AlbumMatcher compares individual releases. If nil and
	// ReleaseProvider is set, a default AlbumMatcher is used.
	AlbumMatcher *AlbumMatcher

	// NameWeight controls how strongly the artist name (and aliases)
	// drives the score. Defaults to 1.
	NameWeight float64

	// ReleaseOverlapWeight controls the contribution when at least one
	// release matches. Defaults to 1 — equal to name weight, so finding
	// an overlapping release brings an ambiguous score decisively above
	// threshold.
	ReleaseOverlapWeight float64

	// ReleaseOverlapThreshold is the album-score threshold used when
	// searching for an overlapping release. Defaults to
	// DefaultAlbumThreshold.
	ReleaseOverlapThreshold float64

	// ReleaseProbeMin is the minimum name similarity at which release
	// probing becomes worthwhile — below this the names are too
	// different to bother. Defaults to 0.6.
	ReleaseProbeMin float64

	// ReleaseProbeMax is the name similarity above which release probing
	// is unnecessary — the names already agree. Defaults to 0.95.
	ReleaseProbeMax float64
}

// ArtistMatcher scores the similarity of two Artist values. If a
// ReleaseProvider is configured, it can resolve ambiguous name matches by
// looking for at least one overlapping release.
type ArtistMatcher struct {
	opts ArtistMatcherOptions

	releaseCacheMu sync.Mutex
	releaseCache   map[string][]Album
	releaseErrors  map[string]error
}

// NewArtistMatcher returns an ArtistMatcher with the given options.
func NewArtistMatcher(opts ArtistMatcherOptions) *ArtistMatcher {
	applyDefault(&opts.NameWeight, 1)
	applyDefault(&opts.ReleaseOverlapWeight, 1)
	applyDefault(&opts.ReleaseOverlapThreshold, DefaultAlbumThreshold)
	applyDefault(&opts.ReleaseProbeMin, 0.6)
	applyDefault(&opts.ReleaseProbeMax, 0.95)
	if opts.ReleaseProvider != nil && opts.AlbumMatcher == nil {
		opts.AlbumMatcher = NewAlbumMatcher(AlbumMatcherOptions{})
	}
	return &ArtistMatcher{
		opts:          opts,
		releaseCache:  map[string][]Album{},
		releaseErrors: map[string]error{},
	}
}

// Match scores the similarity of a and b.
//
// Authoritative short-circuits (in priority order):
//  1. Matching MBID — MusicBrainz artist identifiers.
//  2. Matching platform ID — two artists with the same Spotify ID etc.
//
// Otherwise the score is based on artist-name similarity (including
// aliases). When the name similarity falls into the ambiguous range
// (ReleaseProbeMin..ReleaseProbeMax) and a ReleaseProvider is configured,
// the matcher fetches both discographies and looks for a single
// overlapping release — one is enough to lift the score above threshold.
func (m *ArtistMatcher) Match(ctx context.Context, a, b Artist) Score {
	if a.MBID != "" && b.MBID != "" && a.MBID == b.MBID {
		return authoritativeScore("mbid", "MBID match", 1)
	}
	if comparePlatformIDs(a.ExternalIDs, b.ExternalIDs) == externalIDEqual {
		return authoritativeScore("platform_id", "platform ID match", 1)
	}

	nameSim := artistNameSimilarity(a, b)
	signals := []Signal{
		{Name: "name", Value: nameSim, Weight: m.opts.NameWeight},
	}

	if m.opts.ReleaseProvider != nil &&
		nameSim >= m.opts.ReleaseProbeMin &&
		nameSim < m.opts.ReleaseProbeMax {
		if m.releasesOverlap(ctx, a, b) {
			signals = append(signals, Signal{
				Name:   "release_overlap",
				Value:  1,
				Weight: m.opts.ReleaseOverlapWeight,
				Note:   "shared release found",
			})
		} else {
			signals = append(signals, Signal{
				Name:   "release_overlap",
				Value:  0,
				Weight: m.opts.ReleaseOverlapWeight,
				Note:   "no shared release",
			})
		}
	}

	if comparePlatformIDs(a.ExternalIDs, b.ExternalIDs) == externalIDDifferent {
		// Same platform, different ID — these are certainly different artists
		// on that platform. Cap the score aggressively.
		score := scoreOf(signals...)
		score.Signals = append(score.Signals, Signal{
			Name: "platform_id", Value: 0, Weight: 0, Note: "platform ID mismatch",
		})
		if score.Value > 0.4 {
			score.Value = 0.4
		}
		return score
	}

	return scoreOf(signals...)
}

// releasesOverlap fetches both artists' discographies and returns true as
// soon as any pair of releases matches under the configured
// AlbumMatcher/threshold. Releases are cached per matcher instance.
func (m *ArtistMatcher) releasesOverlap(ctx context.Context, a, b Artist) bool {
	relA, okA := m.releasesOf(ctx, a)
	if !okA {
		return false
	}
	relB, okB := m.releasesOf(ctx, b)
	if !okB {
		return false
	}
	for _, ra := range relA {
		if err := ctx.Err(); err != nil {
			return false
		}
		for _, rb := range relB {
			if m.opts.AlbumMatcher.Match(ctx, ra, rb).Above(m.opts.ReleaseOverlapThreshold) {
				return true
			}
		}
	}
	return false
}

func (m *ArtistMatcher) releasesOf(ctx context.Context, a Artist) ([]Album, bool) {
	key := artistCacheKey(a)
	m.releaseCacheMu.Lock()
	if rel, ok := m.releaseCache[key]; ok {
		m.releaseCacheMu.Unlock()
		return rel, true
	}
	if _, ok := m.releaseErrors[key]; ok {
		m.releaseCacheMu.Unlock()
		return nil, false
	}
	m.releaseCacheMu.Unlock()

	rel, err := m.opts.ReleaseProvider.Releases(ctx, a)
	m.releaseCacheMu.Lock()
	defer m.releaseCacheMu.Unlock()
	if err != nil {
		m.releaseErrors[key] = err
		return nil, false
	}
	m.releaseCache[key] = rel
	return rel, true
}

// artistCacheKey derives a deterministic string key for a given Artist so
// that releasesOf can deduplicate calls to the provider. Priority: MBID >
// external IDs > normalised name.
func artistCacheKey(a Artist) string {
	if a.MBID != "" {
		return "mbid:" + a.MBID
	}
	if len(a.ExternalIDs) > 0 {
		keys := make([]string, 0, len(a.ExternalIDs))
		for p := range a.ExternalIDs {
			keys = append(keys, string(p))
		}
		sort.Strings(keys)
		var sb strings.Builder
		sb.WriteString("ext:")
		for _, p := range keys {
			sb.WriteString(p)
			sb.WriteByte('=')
			sb.WriteString(a.ExternalIDs[Platform(p)])
			sb.WriteByte('|')
		}
		return sb.String()
	}
	return "name:" + textnorm.Normalize(a.Name)
}
