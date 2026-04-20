# matchify

[![Go Reference](https://pkg.go.dev/badge/github.com/ulbwa/matchify.svg)](https://pkg.go.dev/github.com/ulbwa/matchify)

Cross-platform matching for artists, albums, and tracks across music
services (Spotify, Apple Music, Tidal, Qobuz, and anything else you care
to describe as a `Platform`).

`matchify` does **not** fetch anything over the network — it is a pure
library that works on the data you hand it. When you do have something
that can load discographies on demand (for the artist matcher), you
supply it via the `ReleaseProvider` interface.

## Installation

```sh
go get github.com/ulbwa/matchify
```

Requires Go 1.26 or newer.

## Highlights

- **Adaptive scoring.** The matchers use whatever signals are present on
  the input and ignore the rest. An `ISRC` on both sides short-circuits
  comparison; otherwise names, artists, duration, release year, track
  position, etc. are combined with tunable weights.
- **Authoritative-ID short-circuits.** `ISRC` (tracks), `UPC` (albums),
  `MBID`, and per-platform external IDs are treated as ground truth when
  both sides provide them.
- **Feat-aware.** Spotify keeps featured artists in the artist array,
  Apple Music inlines them in the title, Tidal/Qobuz vary, Deezer often
  puts the whole credit ("Artist1 feat. Artist2" or "Artist1 & Artist2")
  into a single artist entry. `matchify` extracts feature clauses from
  titles (`feat.`, `ft.`, `featuring`, `with`, paren/bracket/dash/inline
  forms), splits composite artist entries on feature markers and on
  `&` / `,` / `and`, and merges everything into a canonical list before
  comparing.
- **Edition / variant aware.**
  - Cosmetic markers like `(Remastered)`, `(Explicit)`, `- Digital
    Remaster` normalise out so they don't affect the score.
  - Edition markers (`Deluxe`, `25th Anniversary`, `Japanese Edition`,
    ...) are detected — matching editions score higher than
    plain-vs-deluxe.
  - Recording variants (`(Live)`, `(Acoustic)`, `(Remix)`, `(Demo)`,
    `(Unplugged)`, `(Taylor's Version)`, ...) are detected and cap the
    score so that two recordings of the same song don't silently merge.
- **Artist matcher with release probe.** When artist names are
  ambiguous, the matcher optionally calls a `ReleaseProvider` for each
  artist's discography and looks for a single overlapping release — one
  is enough to lift the score above threshold. Results are cached per
  matcher instance.
- **Generic helpers.** `FindBest[T]` and `Group[T]` work with any
  `Matcher[T]`, so the same primitives do per-track, per-album, and
  per-artist clustering.

## Quick tour

### Track matching

```go
import (
    "context"
    "time"

    "github.com/ulbwa/matchify"
)

m := matchify.NewTrackMatcher(matchify.TrackMatcherOptions{})

spotify := matchify.Track{
    Name: "Despacito",
    Artists: []matchify.Artist{
        {Name: "Luis Fonsi"},
        {Name: "Daddy Yankee"},
        {Name: "Justin Bieber"},
    },
    Duration: 228 * time.Second,
}

appleMusic := matchify.Track{
    Name: "Despacito (feat. Justin Bieber)",
    Artists: []matchify.Artist{
        {Name: "Luis Fonsi"},
        {Name: "Daddy Yankee"},
    },
    Duration: 229 * time.Second,
}

score := m.Match(context.Background(), spotify, appleMusic)
// score.Above(matchify.DefaultTrackThreshold) == true
```

### Finding the best candidate

```go
idx, score, ok := matchify.FindBest(
    ctx, trackMatcher, target, candidates, matchify.DefaultTrackThreshold,
)
if ok {
    fmt.Println("match:", candidates[idx], "score:", score)
}
```

### Grouping duplicates across a playlist

```go
groups := matchify.Group(ctx, trackMatcher, tracks, matchify.DefaultTrackThreshold)
// groups is [][]int — indices per cluster.
```

### Artist matching with release probe

```go
provider := matchify.ReleaseProviderFunc(func(ctx context.Context, a matchify.Artist) ([]matchify.Album, error) {
    // fetch the artist's discography from whatever source you have
    return fetchDiscography(ctx, a), nil
})

am := matchify.NewArtistMatcher(matchify.ArtistMatcherOptions{
    ReleaseProvider: provider,
})

score := am.Match(ctx, matchify.Artist{Name: "Maneskin"}, matchify.Artist{Name: "Måneskin"})
```

If the two names fall into the ambiguous zone (Jaro-Winkler 0.6–0.95),
the matcher fetches each artist's releases once, runs them through the
album matcher pairwise, and if any pair matches above the album
threshold, the score is lifted decisively above threshold.

## Scoring model

Every `Match` returns a `Score`:

```go
type Score struct {
    Value   float64   // 0..1 confidence
    Signals []Signal  // per-signal contributions
}
```

`Signals` is always populated so you can inspect the reasoning:

```go
fmt.Println(score)
// 0.947 {name=1.00*3.00, artists=1.00*3.00, duration=1.00*1.00, explicit=1.00*0.20}
```

Each matcher exposes its weights via its `*MatcherOptions` type. Zero
values receive sensible defaults; a non-zero value overrides. The
high-level knobs are:

| Option                    | Purpose                                                            |
|---------------------------|--------------------------------------------------------------------|
| `NameWeight`              | How strongly the title drives the score                            |
| `ArtistWeight`            | How strongly the artist list drives the score                      |
| `DurationWeight`          | Track duration agreement (track matcher)                           |
| `AlbumWeight`             | Album-title agreement (track matcher)                              |
| `YearWeight`              | Release-year agreement (album matcher)                             |
| `TrackCountWeight`        | Track-count agreement (album matcher)                              |
| `EditionPenalty`          | Penalty subtracted when album editions differ (plain vs Deluxe)    |
| `ReleaseOverlapWeight`    | Strength of release-probe result (artist matcher)                  |
| `VariantMismatchCap`      | Score cap when track variants differ (e.g., studio vs live)        |
| `DurationMismatchCap`     | Score cap when durations disagree beyond `DurationMismatchSeconds` |
| `ISRCMismatchCap`         | Score cap when both tracks' ISRCs disagree                         |
| `UPCMismatchCap`          | Same, for album UPCs                                               |

## What the library does *not* do

- **No data loading.** No HTTP, no platform SDKs. You hand over `Track`,
  `Album`, and `Artist` values and ask the library to compare them.
- **No transliteration.** If Cyrillic/Latin/Japanese transliteration
  matters for your data, attach `Aliases` to the `Artist` values.
- **No ML models.** All heuristics are deterministic, explainable, and
  run in microseconds.

## Performance

Approximate numbers on Apple M1 Pro, `go test -bench`:

| Operation                                         | Time       |
|---------------------------------------------------|------------|
| Track match, full weighted scoring                | ~60 µs     |
| Track match, authoritative-ID short circuit       | ~25 ns     |
| Album match, full weighted scoring                | ~60 µs     |
| Artist match, name-only                           | ~5 µs      |
| Group 100 tracks (4,950 comparisons, O(n²))       | ~70 ms     |

For large inputs to `Group`, pre-bucket your items by a deterministic
key (e.g., normalised artist name) and call `Group` on each bucket.

## License

Apache 2.0 — see [LICENSE](LICENSE).
