package matchify

import (
	"context"
	"sort"
	"strings"
	"sync"

	"golang.org/x/sync/singleflight"

	"github.com/ulbwa/matchify/internal/textnorm"
)

// DefaultArtistThreshold is the score value above which two artists are
// considered a likely match under the default weights.
const DefaultArtistThreshold = 0.85

// artistConfig holds the tunable parameters of an ArtistMatcher.
type artistConfig struct {
	nameWeight              float64
	releaseOverlapWeight    float64
	releaseOverlapThreshold float64
	releaseProbeMin         float64
	releaseProbeMax         float64
	platformIDMismatchCap   float64
	provider                ReleaseProvider
	albumMatcher            *AlbumMatcher
}

func defaultArtistConfig() artistConfig {
	return artistConfig{
		nameWeight:              1,
		releaseOverlapWeight:    1,
		releaseOverlapThreshold: DefaultAlbumThreshold,
		releaseProbeMin:         0.6,
		releaseProbeMax:         0.95,
		platformIDMismatchCap:   0.4,
	}
}

// ArtistOption is a functional option for NewArtistMatcher.
type ArtistOption func(*artistConfig)

// ArtistNameWeight sets the weight of the artist-name similarity
// (default 1).
func ArtistNameWeight(w float64) ArtistOption {
	return func(c *artistConfig) { c.nameWeight = w }
}

// ArtistReleaseProvider installs a ReleaseProvider that the matcher will
// consult when name-based scoring is ambiguous. Without a provider the
// matcher relies on names and aliases alone.
func ArtistReleaseProvider(p ReleaseProvider) ArtistOption {
	return func(c *artistConfig) { c.provider = p }
}

// ArtistAlbumMatcher installs the AlbumMatcher used when the release
// provider finds candidate overlapping releases. Defaults to a fresh
// NewAlbumMatcher() if unset when a ReleaseProvider is configured.
func ArtistAlbumMatcher(m *AlbumMatcher) ArtistOption {
	return func(c *artistConfig) { c.albumMatcher = m }
}

// ArtistReleaseOverlapWeight sets the weight of the release-probe
// contribution when at least one release overlaps between the two
// artists (default 1).
func ArtistReleaseOverlapWeight(w float64) ArtistOption {
	return func(c *artistConfig) { c.releaseOverlapWeight = w }
}

// ArtistReleaseOverlapThreshold sets the album-score threshold used to
// decide whether two releases are considered "the same" during release
// probing (default DefaultAlbumThreshold).
func ArtistReleaseOverlapThreshold(t float64) ArtistOption {
	return func(c *artistConfig) { c.releaseOverlapThreshold = t }
}

// ArtistReleaseProbeBand sets the range of name-similarity values inside
// which release probing is worthwhile. Below min the names are too
// different to bother; above max they already agree, so probing is
// unnecessary. Defaults to 0.6 and 0.95.
func ArtistReleaseProbeBand(min, max float64) ArtistOption {
	return func(c *artistConfig) {
		c.releaseProbeMin = min
		c.releaseProbeMax = max
	}
}

// ArtistPlatformIDMismatchCap caps the score when both sides share a
// platform key and that platform's IDs disagree (default 0.4).
func ArtistPlatformIDMismatchCap(cap float64) ArtistOption {
	return func(c *artistConfig) { c.platformIDMismatchCap = cap }
}

// ArtistMatcher scores the similarity of two Artist values.
type ArtistMatcher struct {
	cfg artistConfig

	releaseCacheMu sync.Mutex
	releaseCache   map[string][]Album
	releaseErrors  map[string]error
	releaseFetch   singleflight.Group
}

// NewArtistMatcher returns an ArtistMatcher with the given options
// applied on top of the baseline defaults.
func NewArtistMatcher(opts ...ArtistOption) *ArtistMatcher {
	cfg := defaultArtistConfig()
	for _, fn := range opts {
		if fn != nil {
			fn(&cfg)
		}
	}
	if cfg.provider != nil && cfg.albumMatcher == nil {
		cfg.albumMatcher = NewAlbumMatcher()
	}
	return &ArtistMatcher{
		cfg:           cfg,
		releaseCache:  map[string][]Album{},
		releaseErrors: map[string]error{},
	}
}

// Match scores the similarity of a and b.
//
// Authoritative short-circuits (each returns Relation=Same, Value≈1):
//  1. MBID tags match — MusicBrainz artist identifiers.
//  2. Platform-ID tags match on at least one shared platform.
//
// Otherwise the score is based on artist-name similarity (including
// aliases). When the name similarity falls into the ambiguous range
// (ReleaseProbeBand) and a ReleaseProvider is configured, the matcher
// fetches both discographies and looks for a single overlapping release
// — one is enough to lift the score decisively.
//
// Artists do not have recording variants, so this matcher never returns
// RelationVariant — the result is either Same or Unrelated.
func (m *ArtistMatcher) Match(ctx context.Context, a, b Artist) Score {
	if mbidA, okA := a.Tags.MBID(); okA {
		if mbidB, okB := b.Tags.MBID(); okB && mbidA == mbidB {
			return authoritativeScore("mbid", "MBID match", 1)
		}
	}
	if comparePlatformIDs(a.Tags.PlatformIDs(), b.Tags.PlatformIDs()) == externalIDEqual {
		return authoritativeScore("platform_id", "platform ID match", 1)
	}

	nameSim := artistNameSimilarity(a, b)
	signals := []Signal{
		{Name: "name", Value: nameSim, Weight: m.cfg.nameWeight},
	}

	if m.cfg.provider != nil &&
		nameSim >= m.cfg.releaseProbeMin &&
		nameSim < m.cfg.releaseProbeMax {
		overlap, probed := m.releasesOverlap(ctx, a, b)
		switch {
		case !probed:
			// Provider failed — fall back to name-only as documented.
		case overlap:
			signals = append(signals, Signal{
				Name:   "release_overlap",
				Value:  1,
				Weight: m.cfg.releaseOverlapWeight,
				Note:   "shared release found",
			})
		default:
			signals = append(signals, Signal{
				Name:   "release_overlap",
				Value:  0,
				Weight: m.cfg.releaseOverlapWeight,
				Note:   "no shared release",
			})
		}
	}

	score := scoreOf(signals...)

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

// releasesOverlap returns (overlap, probed). probed=false means the
// provider errored and the caller should fall back to name-only.
func (m *ArtistMatcher) releasesOverlap(ctx context.Context, a, b Artist) (overlap, probed bool) {
	relA, okA := m.releasesOf(ctx, a)
	if !okA {
		return false, false
	}
	relB, okB := m.releasesOf(ctx, b)
	if !okB {
		return false, false
	}
	for _, ra := range relA {
		if err := ctx.Err(); err != nil {
			return false, false
		}
		for _, rb := range relB {
			if m.cfg.albumMatcher.Match(ctx, ra, rb).Same(m.cfg.releaseOverlapThreshold) {
				return true, true
			}
		}
	}
	return false, true
}

func (m *ArtistMatcher) releasesOf(ctx context.Context, a Artist) ([]Album, bool) {
	key := artistCacheKey(a)

	// Fast path: already cached.
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

	// Slow path via singleflight so concurrent callers for the same key
	// share a single provider call.
	v, err, _ := m.releaseFetch.Do(key, func() (any, error) {
		m.releaseCacheMu.Lock()
		if rel, ok := m.releaseCache[key]; ok {
			m.releaseCacheMu.Unlock()
			return rel, nil
		}
		if cachedErr, ok := m.releaseErrors[key]; ok {
			m.releaseCacheMu.Unlock()
			return nil, cachedErr
		}
		m.releaseCacheMu.Unlock()

		rel, err := m.cfg.provider.Releases(ctx, a)
		m.releaseCacheMu.Lock()
		defer m.releaseCacheMu.Unlock()
		if err != nil {
			m.releaseErrors[key] = err
			return nil, err
		}
		m.releaseCache[key] = rel
		return rel, nil
	})
	if err != nil {
		return nil, false
	}
	rel, _ := v.([]Album)
	return rel, true
}

// artistCacheKey derives a deterministic key for an artist so
// releasesOf can deduplicate provider calls. Priority: MBID > external
// IDs > normalised name.
func artistCacheKey(a Artist) string {
	if mbid, ok := a.Tags.MBID(); ok && mbid != "" {
		return "mbid:" + mbid
	}
	ids := a.Tags.PlatformIDs()
	if len(ids) > 0 {
		keys := make([]string, 0, len(ids))
		for p := range ids {
			keys = append(keys, string(p))
		}
		sort.Strings(keys)
		var sb strings.Builder
		sb.WriteString("ext:")
		for _, p := range keys {
			sb.WriteString(p)
			sb.WriteByte('=')
			sb.WriteString(ids[Platform(p)])
			sb.WriteByte('|')
		}
		return sb.String()
	}
	return "name:" + textnorm.Normalize(a.Name)
}
