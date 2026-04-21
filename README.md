# matchify

[![Go Reference](https://pkg.go.dev/badge/github.com/ulbwa/matchify.svg)](https://pkg.go.dev/github.com/ulbwa/matchify)

Cross-platform matching for artists, albums, and tracks across music
services (Spotify, Apple Music, Tidal, Qobuz, Deezer, SoundCloud, VK
Music, and anything else you describe as a `Platform`).

`matchify` does **not** fetch anything over the network — it is a pure
library that works on the data you hand it. When you want the artist
matcher to disambiguate ambiguous name matches by looking at
discographies, you supply a `ReleaseProvider` implementation.

## Installation

```sh
go get github.com/ulbwa/matchify
```

Requires Go 1.26 or newer.

## Design

### Core types are minimal

Only the identifying fields are positional. Everything else — duration,
ISRC, UPC, release date, explicit flag, per-platform IDs, aliases — is
carried in a typed `Tags` value built with `NewTags` and tag
constructors:

```go
track := matchify.Track{
    Name:    "Despacito",
    Artists: []matchify.Artist{
        {Name: "Luis Fonsi"},
        {Name: "Daddy Yankee"},
        {Name: "Justin Bieber"},
    },
    Tags: matchify.NewTags(
        matchify.WithDuration(228 * time.Second),
        matchify.WithISRC("USMV10000001"),
        matchify.WithExplicit(matchify.ExplicitnessExplicit),
        matchify.WithPlatformID(matchify.PlatformSpotify, "1i1fxkWe..."),
    ),
}
```

This lets platforms with thin metadata (SoundCloud, VK Music) coexist
with platforms that ship rich data (Qobuz, Apple Music) — an unset tag
is unambiguous:

```go
if d, ok := track.Tags.Duration(); ok {
    // platform supplied a duration
} else {
    // platform didn't
}
```

### Relation vs Value

A `Score` carries both a **Relation** (what kind of relationship was
inferred) and a **Value** (how confident we are in that relation):

```go
type Score struct {
    Value    float64   // confidence in [0, 1]
    Relation Relation  // Unrelated | Variant | Same
    Signals  []Signal  // per-signal contributions, for debugging
}
```

- `RelationSame` — the two sides are the same product (same recording,
  same master, same edition).
- `RelationVariant` — the two sides are variants of the same underlying
  work: plain vs Deluxe edition, studio vs Live, explicit vs clean
  master, original vs Stripped re-recording, remaster vs original.
- `RelationUnrelated` — the two sides are different works, or an
  authoritative identifier (ISRC, UPC, platform ID) disagrees.

Callers express their intent explicitly:

```go
score := matcher.Match(ctx, a, b)

if score.Same(matchify.DefaultTrackThreshold) {
    // collapse duplicates
}
if score.Related(matchify.DefaultTrackThreshold) {
    // show as "other versions of this song"
}
```

### Functional options

Matchers are configured with type-scoped functional options:

```go
m := matchify.NewTrackMatcher(
    matchify.TrackNameWeight(4),
    matchify.TrackDurationMismatch(30, 0.5),
)
```

Each matcher has its own option family (`TrackXxx`, `AlbumXxx`,
`ArtistXxx`) so the compiler catches misuse.

## Features

- **Adaptive scoring.** The score uses whatever signals the caller
  supplied; missing data contributes nothing rather than penalising.
- **Authoritative-ID short-circuits.** Matching ISRC, UPC, MBID, or
  per-platform IDs short-circuit to `RelationSame` with value ≈ 1.
  Mismatching IDs short-circuit to `RelationUnrelated`.
- **Feat-aware.** The matcher handles every convention: Spotify's
  artist-array style, Apple Music's inline "(feat. X)" suffix,
  Tidal/Qobuz mixes, and Deezer's composite strings
  ("A feat. B", "A & B") inside a single artist entry.
- **Edition-aware.** Deluxe / Anniversary / Japanese Edition /
  Remastered are detected. Matching edition markers on both sides
  keep `RelationSame`; one side carrying an edition marker the other
  doesn't downgrades to `RelationVariant`.
- **Variant-aware.** Live / Acoustic / Remix / Demo / Unplugged /
  Stripped / Extended Cut / Taylor's Version and similar markers
  trigger a `RelationVariant` downgrade so different recordings of
  the same song don't silently merge.
- **Explicit-aware.** Clean and explicit masters are never the same
  product, even when names otherwise match. Works from both a
  structured `WithExplicit` tag and from "(Clean)" / "(Explicit)"
  markers in the name.
- **Artist matcher with release probe.** When artist names fall into
  an ambiguous band, the matcher optionally consults a
  `ReleaseProvider` for each artist's discography and looks for a
  single overlapping release. One match lifts the score decisively.
  A built-in `singleflight` cache deduplicates concurrent provider
  fetches.
- **Generic helpers.** `FindBest[T]` and `Group[T]` work with any
  `Matcher[T]`, taking an `Accept` predicate — `IsSame(threshold)`
  for strict duplicate detection, `IsRelated(threshold)` to also
  capture variants.

## Quick tour

### Track matching

```go
m := matchify.NewTrackMatcher()

spotify := matchify.Track{
    Name:    "Despacito",
    Artists: []matchify.Artist{
        {Name: "Luis Fonsi"},
        {Name: "Daddy Yankee"},
        {Name: "Justin Bieber"},
    },
    Tags: matchify.NewTags(matchify.WithDuration(228 * time.Second)),
}
appleMusic := matchify.Track{
    Name:    "Despacito (feat. Justin Bieber)",
    Artists: []matchify.Artist{
        {Name: "Luis Fonsi"},
        {Name: "Daddy Yankee"},
    },
    Tags: matchify.NewTags(matchify.WithDuration(229 * time.Second)),
}

score := m.Match(ctx, spotify, appleMusic)
// score.Same(matchify.DefaultTrackThreshold) == true
```

### Finding the best candidate

```go
idx, score, ok := matchify.FindBest(
    ctx, trackMatcher, target, candidates,
    matchify.IsSame(matchify.DefaultTrackThreshold),
)
```

### Grouping across a playlist

```go
groups := matchify.Group(
    ctx, trackMatcher, tracks,
    matchify.IsSame(matchify.DefaultTrackThreshold),
)
// groups is [][]int — indices per cluster.
```

Use `IsRelated` instead of `IsSame` to also group variants together.

### Artist matching with release probe

```go
provider := matchify.ReleaseProviderFunc(func(ctx context.Context, a matchify.Artist) ([]matchify.Album, error) {
    return fetchDiscography(ctx, a)
})

am := matchify.NewArtistMatcher(
    matchify.ArtistReleaseProvider(provider),
)

score := am.Match(ctx, matchify.Artist{Name: "Maneskin"}, matchify.Artist{Name: "Måneskin"})
```

If the two names fall into the ambiguous zone the matcher fetches each
artist's releases once, runs them through the album matcher pairwise,
and if any pair matches above the album threshold lifts the score
decisively.

## Scoring internals

Every `Match` returns a populated `Score.Signals` so you can inspect
the reasoning:

```go
fmt.Println(score)
// same 0.947 {name=1.00*3.00, artists=1.00*3.00, duration=1.00*1.00}
```

Each matcher exposes its weights via functional options. Zero values
receive sensible defaults; pass a non-zero value to override. The
high-level knobs:

| Option                                  | Purpose                                              |
|-----------------------------------------|------------------------------------------------------|
| `TrackNameWeight` / `AlbumNameWeight`   | Title similarity strength                            |
| `TrackArtistWeight` / `AlbumArtistWeight` | Artist list similarity strength                    |
| `TrackDurationWeight`                   | Duration-agreement contribution (when both set)      |
| `TrackAlbumWeight`                      | Album-title agreement for tracks (when both set)     |
| `AlbumYearWeight`                       | Release-year agreement                               |
| `AlbumTrackCountWeight`                 | Track-count agreement                                |
| `TrackISRCMismatchCap`                  | Cap when both ISRCs disagree (default 0.4)           |
| `AlbumUPCMismatchCap`                   | Cap when both UPCs disagree                          |
| `TrackPlatformIDMismatchCap`            | Cap when platform IDs disagree                       |
| `TrackDurationMismatch(seconds, cap)`   | Duration-disagreement cap (marks RelationUnrelated)  |
| `ArtistReleaseProvider`                 | Hook for release-probe on ambiguous name matches     |
| `ArtistReleaseProbeBand(min, max)`      | Name-similarity band where release probing runs      |

## What the library does *not* do

- **No data loading.** No HTTP, no platform SDKs. Hand over
  `Artist`/`Album`/`Track` values and ask the library to compare them.
- **No transliteration.** If Cyrillic ⇄ Latin ⇄ Japanese matters,
  attach aliases with `WithAlias`.
- **No ML models.** All heuristics are deterministic, explainable, and
  run in microseconds. The string-similarity primitive is Jaro-Winkler
  via [adrg/strutil](https://github.com/adrg/strutil).

## Performance

Approximate numbers on Apple M1 Pro, `go test -bench`:

| Operation                                      | Time    |
|------------------------------------------------|---------|
| Track match, full weighted scoring             | ~60 µs  |
| Track match, authoritative-ID short circuit    | ~25 ns  |
| Album match, full weighted scoring             | ~60 µs  |
| Artist match, name-only                        | ~5 µs   |
| Group 100 tracks (4,950 comparisons, O(n²))    | ~70 ms  |

For large inputs to `Group`, pre-bucket items by a deterministic key
(e.g. normalised artist name) and call `Group` on each bucket.

## License

Apache 2.0 — see [LICENSE](LICENSE).
