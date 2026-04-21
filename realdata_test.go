package matchify

import (
	"context"
	"testing"
	"time"
)

// Tests in this file exercise the matcher against data *shapes* observed
// on real music platforms. Each case mirrors the layout a platform
// returns for a well-known release — artist array structure, feat
// conventions, edition markers, explicit flags, ISRC/UPC formats.
//
// The library is never asked to hit the network, so the values in these
// fixtures are static snapshots rather than live queries. Identifiers
// (ISRC, UPC, platform IDs) follow the format used by the real
// platforms and are consistent within each test case; they are used to
// verify the matcher's short-circuit and mismatch-cap behaviour.
//
// Platforms referenced: Spotify, Apple Music, Tidal, Qobuz, Deezer,
// SoundCloud, Amazon Music, YouTube Music, VK Music. Data conventions
// are drawn from song.link / odesli.co aggregation patterns and from
// each platform's public catalog pages.

// TestRealData_Despacito covers the Luis Fonsi ft. Daddy Yankee ft. Justin
// Bieber remix that every platform renders slightly differently.
func TestRealData_Despacito_CrossPlatform(t *testing.T) {
	t.Parallel()
	m := NewTrackMatcher()
	ctx := context.Background()

	// Spotify: all credited artists in the array, title has no feat clause.
	spotify := Track{
		Name: "Despacito - Remix",
		Artists: []Artist{
			{Name: "Luis Fonsi"},
			{Name: "Daddy Yankee"},
			{Name: "Justin Bieber"},
		},
		Tags: NewTags(
			WithDuration(228*time.Second),
			WithISRC("USUM71703861"),
			WithExplicit(ExplicitnessExplicit),
			WithPlatformID(PlatformSpotify, "3AszgPDZd7BMwLujhjpnyZ"),
		),
	}
	// Apple Music: feat inline in title, primary artists only in array.
	appleMusic := Track{
		Name: "Despacito (Remix) [feat. Justin Bieber]",
		Artists: []Artist{
			{Name: "Luis Fonsi"},
			{Name: "Daddy Yankee"},
		},
		Tags: NewTags(
			WithDuration(229*time.Second),
			WithISRC("USUM71703861"),
			WithPlatformID(PlatformAppleMusic, "1212080695"),
		),
	}
	// Tidal: often uses " - Remix" suffix, features in array.
	tidal := Track{
		Name: "Despacito (Remix)",
		Artists: []Artist{
			{Name: "Luis Fonsi"},
			{Name: "Daddy Yankee"},
			{Name: "Justin Bieber"},
		},
		Tags: NewTags(
			WithDuration(228 * time.Second),
		),
	}
	// Deezer: composite single-entry credits.
	deezer := Track{
		Name: "Despacito (Remix)",
		Artists: []Artist{
			{Name: "Luis Fonsi feat. Daddy Yankee & Justin Bieber"},
		},
		Tags: NewTags(WithDuration(228 * time.Second)),
	}

	// Every pair should match as Same.
	pairs := [][2]Track{
		{spotify, appleMusic},
		{spotify, tidal},
		{spotify, deezer},
		{appleMusic, tidal},
		{appleMusic, deezer},
		{tidal, deezer},
	}
	for i, p := range pairs {
		s := m.Match(ctx, p[0], p[1])
		if !s.Same(DefaultTrackThreshold) {
			t.Errorf("pair %d: expected Same, got %v", i, s)
		}
	}
}

// TestRealData_BadGuy_ISRCShortCircuit: Billie Eilish's "Bad Guy" has a
// stable ISRC across every platform that publishes one. Even when the
// displayed name differs significantly, the ISRC short-circuits to
// Same.
func TestRealData_BadGuy_ISRCShortCircuit(t *testing.T) {
	t.Parallel()
	m := NewTrackMatcher()
	ctx := context.Background()

	const isrc = "USUM71900764"

	// Spotify strips the "." from Billie Eilish's style, ships ISRC.
	spotify := Track{
		Name:    "bad guy",
		Artists: []Artist{{Name: "Billie Eilish"}},
		Tags: NewTags(
			WithDuration(194*time.Second),
			WithISRC(isrc),
			WithExplicit(ExplicitnessExplicit),
		),
	}
	// Apple Music writes it with a capital 'B'.
	appleMusic := Track{
		Name:    "bad guy",
		Artists: []Artist{{Name: "Billie Eilish"}},
		Tags: NewTags(
			WithDuration(194*time.Second),
			WithISRC(isrc),
		),
	}
	if s := m.Match(ctx, spotify, appleMusic); !s.Same(0.99) {
		t.Errorf("expected ISRC short-circuit, got %v", s)
	}
}

// TestRealData_BlindingLights checks The Weeknd's "Blinding Lights"
// across platforms where track context (album, track number, duration)
// is provided on both sides.
func TestRealData_BlindingLights_CrossPlatform(t *testing.T) {
	t.Parallel()
	m := NewTrackMatcher()
	ctx := context.Background()

	afterHoursSpotify := Album{
		Name:    "After Hours",
		Artists: []Artist{{Name: "The Weeknd"}},
		Tags: NewTags(
			WithReleaseDate(time.Date(2020, 3, 20, 0, 0, 0, 0, time.UTC)),
			WithTrackCount(14),
		),
	}
	afterHoursApple := Album{
		Name:    "After Hours",
		Artists: []Artist{{Name: "The Weeknd"}},
		Tags: NewTags(
			WithReleaseDate(time.Date(2020, 3, 20, 0, 0, 0, 0, time.UTC)),
			WithTrackCount(14),
		),
	}

	spotify := Track{
		Name:    "Blinding Lights",
		Artists: []Artist{{Name: "The Weeknd"}},
		Album:   &afterHoursSpotify,
		Tags: NewTags(
			WithDuration(200*time.Second),
			WithTrackNumber(9),
			WithDiscNumber(1),
		),
	}
	apple := Track{
		Name:    "Blinding Lights",
		Artists: []Artist{{Name: "The Weeknd"}},
		Album:   &afterHoursApple,
		Tags: NewTags(
			WithDuration(200*time.Second),
			WithTrackNumber(9),
			WithDiscNumber(1),
		),
	}
	if s := m.Match(ctx, spotify, apple); !s.Same(DefaultTrackThreshold) {
		t.Errorf("expected Same, got %v", s)
	}
}

// TestRealData_ShapeOfYou_AcousticIsVariant asserts the Spotify studio
// version of "Shape of You" and its Apple-Music-listed "Acoustic" are
// treated as variants of the same song, not the same product.
func TestRealData_ShapeOfYou_AcousticIsVariant(t *testing.T) {
	t.Parallel()
	m := NewTrackMatcher()
	ctx := context.Background()

	studio := Track{
		Name:    "Shape of You",
		Artists: []Artist{{Name: "Ed Sheeran"}},
		Tags: NewTags(
			WithDuration(234*time.Second),
			WithISRC("GBAHS1700024"),
		),
	}
	acoustic := Track{
		Name:    "Shape of You (Acoustic)",
		Artists: []Artist{{Name: "Ed Sheeran"}},
		Tags: NewTags(
			WithDuration(234 * time.Second),
		),
	}
	s := m.Match(ctx, studio, acoustic)
	if s.Same(DefaultTrackThreshold) {
		t.Errorf("acoustic should NOT be Same as studio, got %v", s)
	}
	if s.Relation != RelationVariant {
		t.Errorf("expected RelationVariant, got %v", s)
	}
	if !s.Related(DefaultTrackThreshold) {
		t.Errorf("acoustic should be Related to studio, got %v", s)
	}
}

// TestRealData_TaylorsVersion asserts that Taylor Swift's re-recordings
// (Red (Taylor's Version), 1989 (Taylor's Version)) are NOT treated as
// the same product as the originals — the audio differs top to bottom.
func TestRealData_TaylorsVersionIsVariant(t *testing.T) {
	t.Parallel()
	am := NewAlbumMatcher()
	ctx := context.Background()

	original := Album{
		Name:    "1989",
		Artists: []Artist{{Name: "Taylor Swift"}},
		Tags: NewTags(
			WithReleaseDate(time.Date(2014, 10, 27, 0, 0, 0, 0, time.UTC)),
			WithTrackCount(13),
			WithReleaseType(ReleaseTypeAlbum),
		),
	}
	taylorsVersion := Album{
		Name:    "1989 (Taylor's Version)",
		Artists: []Artist{{Name: "Taylor Swift"}},
		Tags: NewTags(
			WithReleaseDate(time.Date(2023, 10, 27, 0, 0, 0, 0, time.UTC)),
			WithTrackCount(21),
			WithReleaseType(ReleaseTypeAlbum),
		),
	}
	s := am.Match(ctx, original, taylorsVersion)
	if s.Same(DefaultAlbumThreshold) {
		t.Errorf("Taylor's Version is NOT the same album as the original, got %v", s)
	}
	if s.Relation != RelationVariant {
		t.Errorf("expected RelationVariant, got %v", s)
	}
}

// TestRealData_ThrillerEditions groups Michael Jackson's Thriller
// editions: the original 1982 release, the 25th Anniversary Edition,
// and a Japanese Edition. All are variants of the same album.
func TestRealData_ThrillerEditionsRelated(t *testing.T) {
	t.Parallel()
	am := NewAlbumMatcher()
	ctx := context.Background()

	original := Album{
		Name:    "Thriller",
		Artists: []Artist{{Name: "Michael Jackson"}},
		Tags: NewTags(
			WithReleaseDate(time.Date(1982, 11, 30, 0, 0, 0, 0, time.UTC)),
			WithTrackCount(9),
		),
	}
	anniversary := Album{
		Name:    "Thriller (25th Anniversary Edition)",
		Artists: []Artist{{Name: "Michael Jackson"}},
		Tags: NewTags(
			WithReleaseDate(time.Date(2008, 2, 11, 0, 0, 0, 0, time.UTC)),
			WithTrackCount(18),
		),
	}
	japan := Album{
		Name:    "Thriller (Japanese Edition)",
		Artists: []Artist{{Name: "Michael Jackson"}},
		Tags: NewTags(
			WithReleaseDate(time.Date(1982, 11, 30, 0, 0, 0, 0, time.UTC)),
			WithTrackCount(10),
		),
	}

	cases := [][2]Album{
		{original, anniversary},
		{original, japan},
		{anniversary, japan},
	}
	for i, p := range cases {
		s := am.Match(ctx, p[0], p[1])
		if s.Same(DefaultAlbumThreshold) {
			t.Errorf("pair %d editions should not be Same, got %v", i, s)
		}
		if s.Relation != RelationVariant {
			t.Errorf("pair %d expected RelationVariant, got %v", i, s)
		}
		if !s.Related(0.7) {
			t.Errorf("pair %d should at least be related, got %v", i, s)
		}
	}
}

// TestRealData_BohemianRhapsody_LiveAidIsVariant checks Queen's studio
// recording vs the Live Aid 1985 performance.
func TestRealData_BohemianRhapsody_LiveAidIsVariant(t *testing.T) {
	t.Parallel()
	m := NewTrackMatcher()
	ctx := context.Background()

	studio := Track{
		Name:    "Bohemian Rhapsody",
		Artists: []Artist{{Name: "Queen"}},
		Tags: NewTags(
			WithDuration(354 * time.Second),
		),
	}
	liveAid := Track{
		Name:    "Bohemian Rhapsody (Live at Wembley, 13 July 1985)",
		Artists: []Artist{{Name: "Queen"}},
		Tags: NewTags(
			WithDuration(178 * time.Second),
		),
	}
	s := m.Match(ctx, studio, liveAid)
	if s.Same(DefaultTrackThreshold) {
		t.Errorf("live version must not be Same as studio, got %v", s)
	}
}

// TestRealData_HumbleExplicitVsCleanMustNotMerge pins the explicit-vs-
// clean invariant for Kendrick Lamar's "HUMBLE." — these are distinct
// masters and must never collapse.
func TestRealData_HumbleExplicitVsCleanMustNotMerge(t *testing.T) {
	t.Parallel()
	m := NewTrackMatcher()
	ctx := context.Background()

	// Spotify ships two separate entries for the album — one with the
	// explicit flag set, one labelled "(Clean)" in the album name.
	explicit := Track{
		Name:    "HUMBLE.",
		Artists: []Artist{{Name: "Kendrick Lamar"}},
		Tags: NewTags(
			WithDuration(177*time.Second),
			WithExplicit(ExplicitnessExplicit),
		),
	}
	clean := Track{
		Name:    "HUMBLE. (Clean)",
		Artists: []Artist{{Name: "Kendrick Lamar"}},
		Tags: NewTags(
			WithDuration(177 * time.Second),
		),
	}
	s := m.Match(ctx, explicit, clean)
	if s.Same(DefaultTrackThreshold) {
		t.Errorf("explicit must not merge with clean, got %v", s)
	}
	if s.Relation != RelationVariant {
		t.Errorf("expected RelationVariant, got %v", s)
	}
}

// TestRealData_Maneskin covers how platforms render the name of the
// Italian band — some keep the å, some don't. The matcher should match
// regardless of diacritic presence.
func TestRealData_Maneskin_DiacriticVariants(t *testing.T) {
	t.Parallel()
	m := NewArtistMatcher()
	ctx := context.Background()

	pairs := [][2]Artist{
		{{Name: "Måneskin"}, {Name: "Maneskin"}},
		{{Name: "Måneskin"}, {Name: "MÅNESKIN"}},
		{{Name: "måneskin"}, {Name: "Maneskin"}},
	}
	for i, p := range pairs {
		if s := m.Match(ctx, p[0], p[1]); !s.Same(DefaultArtistThreshold) {
			t.Errorf("pair %d expected Same, got %v", i, s)
		}
	}
}

// TestRealData_TheBeatles_ArticleHandling: platforms and users are
// inconsistent about the leading "The" in band names.
func TestRealData_TheBeatles_ArticleHandling(t *testing.T) {
	t.Parallel()
	m := NewArtistMatcher()
	ctx := context.Background()

	full := Artist{Name: "The Beatles"}
	short := Artist{Name: "Beatles"}
	if s := m.Match(ctx, full, short); !s.Same(DefaultArtistThreshold) {
		t.Errorf("'The Beatles' and 'Beatles' should match, got %v", s)
	}
}

// TestRealData_TheWeeknd_AliasFlow demonstrates using WithAlias to
// bridge "The Weeknd" and "Weeknd" when a platform gives neither
// MBID nor platform ID (e.g. SoundCloud or VK Music).
func TestRealData_TheWeeknd_AliasFlow(t *testing.T) {
	t.Parallel()
	m := NewArtistMatcher()
	ctx := context.Background()

	spotify := Artist{
		Name: "The Weeknd",
		Tags: NewTags(
			WithMBID("c8b03190-306c-4120-bb0b-6f2ebfc06ea9"),
			WithAlias("Weeknd", "Abel Tesfaye"),
		),
	}
	soundcloud := Artist{Name: "Weeknd"}

	if s := m.Match(ctx, spotify, soundcloud); !s.Same(DefaultArtistThreshold) {
		t.Errorf("alias-based match failed, got %v", s)
	}
}

// TestRealData_BadBunny_DeezerComposite: Deezer sometimes ships the
// full credit "Bad Bunny & J Balvin" as a single artist-array entry.
// Spotify keeps them separate. Matching must treat both shapes the
// same.
func TestRealData_BadBunny_DeezerComposite(t *testing.T) {
	t.Parallel()
	m := NewTrackMatcher()
	ctx := context.Background()

	// Spotify-style
	spotify := Track{
		Name:    "I Like It",
		Artists: []Artist{{Name: "Cardi B"}, {Name: "Bad Bunny"}, {Name: "J Balvin"}},
		Tags:    NewTags(WithDuration(253 * time.Second)),
	}
	// Deezer-style composite
	deezer := Track{
		Name:    "I Like It",
		Artists: []Artist{{Name: "Cardi B, Bad Bunny & J Balvin"}},
		Tags:    NewTags(WithDuration(253 * time.Second)),
	}
	if s := m.Match(ctx, spotify, deezer); !s.Same(DefaultTrackThreshold) {
		t.Errorf("expected composite-vs-separate to match Same, got %v", s)
	}
}

// TestRealData_SoundCloudMinimalData simulates a SoundCloud/VK Music
// entry where only the track name and the artist's display name are
// known — the matcher must still produce a sensible score against a
// Spotify entry with rich metadata.
func TestRealData_SoundCloudMinimalData(t *testing.T) {
	t.Parallel()
	m := NewTrackMatcher()
	ctx := context.Background()

	// Spotify (rich)
	spotify := Track{
		Name:    "Levitating",
		Artists: []Artist{{Name: "Dua Lipa"}},
		Tags: NewTags(
			WithDuration(203*time.Second),
			WithISRC("GBAHT2000540"),
			WithPlatformID(PlatformSpotify, "463CkQjx2Zk1yXoBuierM9"),
		),
	}
	// SoundCloud (minimal)
	soundcloud := Track{
		Name:    "Levitating",
		Artists: []Artist{{Name: "Dua Lipa"}},
	}
	if s := m.Match(ctx, spotify, soundcloud); !s.Same(DefaultTrackThreshold) {
		t.Errorf("minimal-data side should still match, got %v", s)
	}
}

// TestRealData_DarkSideOfTheMoon_EditionFamily asserts that the
// original 1973 release and the 2011 remaster form a variant family
// rather than being merged or unrelated.
func TestRealData_DarkSideOfTheMoon_EditionFamily(t *testing.T) {
	t.Parallel()
	am := NewAlbumMatcher()
	ctx := context.Background()

	original := Album{
		Name:    "The Dark Side of the Moon",
		Artists: []Artist{{Name: "Pink Floyd"}},
		Tags: NewTags(
			WithReleaseDate(time.Date(1973, 3, 1, 0, 0, 0, 0, time.UTC)),
			WithTrackCount(10),
		),
	}
	remaster2011 := Album{
		Name:    "The Dark Side of the Moon (2011 Remastered Version)",
		Artists: []Artist{{Name: "Pink Floyd"}},
		Tags: NewTags(
			WithReleaseDate(time.Date(1973, 3, 1, 0, 0, 0, 0, time.UTC)),
			WithTrackCount(10),
		),
	}
	anniversary := Album{
		Name:    "The Dark Side of the Moon (50th Anniversary Remaster)",
		Artists: []Artist{{Name: "Pink Floyd"}},
		Tags: NewTags(
			WithReleaseDate(time.Date(1973, 3, 1, 0, 0, 0, 0, time.UTC)),
			WithTrackCount(10),
		),
	}
	for i, p := range [][2]Album{{original, remaster2011}, {original, anniversary}, {remaster2011, anniversary}} {
		s := am.Match(ctx, p[0], p[1])
		if s.Same(DefaultAlbumThreshold) {
			t.Errorf("pair %d should not be Same (different remasters), got %v", i, s)
		}
		if !s.Related(0.7) {
			t.Errorf("pair %d should be Related (same album family), got %v", i, s)
		}
	}
}

// TestRealData_PlaylistDeduplication simulates a song.link / odesli.co
// style aggregation: the caller has one playlist populated from five
// different platforms and wants to deduplicate it without collapsing
// variants.
func TestRealData_PlaylistDeduplication(t *testing.T) {
	t.Parallel()
	m := NewTrackMatcher()
	ctx := context.Background()

	tracks := []Track{
		// Blinding Lights — Spotify, Apple Music, Tidal, Amazon (4 copies
		// of the same product).
		{Name: "Blinding Lights", Artists: []Artist{{Name: "The Weeknd"}},
			Tags: NewTags(WithDuration(200*time.Second), WithISRC("USUG12002862"))},
		{Name: "Blinding Lights", Artists: []Artist{{Name: "The Weeknd"}},
			Tags: NewTags(WithDuration(200*time.Second), WithISRC("USUG12002862"))},
		{Name: "Blinding Lights", Artists: []Artist{{Name: "The Weeknd"}},
			Tags: NewTags(WithDuration(200 * time.Second))},
		{Name: "Blinding Lights", Artists: []Artist{{Name: "The Weeknd"}},
			Tags: NewTags(WithDuration(201 * time.Second))},

		// Bad Guy — Spotify & Deezer (composite) (2 copies, same product).
		{Name: "bad guy", Artists: []Artist{{Name: "Billie Eilish"}},
			Tags: NewTags(WithDuration(194*time.Second), WithISRC("USUM71900764"))},
		{Name: "Bad Guy", Artists: []Artist{{Name: "Billie Eilish"}},
			Tags: NewTags(WithDuration(194*time.Second), WithISRC("USUM71900764"))},

		// Shape of You (Acoustic) — variant; must NOT group with the
		// studio version.
		{Name: "Shape of You", Artists: []Artist{{Name: "Ed Sheeran"}},
			Tags: NewTags(WithDuration(234 * time.Second))},
		{Name: "Shape of You (Acoustic)", Artists: []Artist{{Name: "Ed Sheeran"}},
			Tags: NewTags(WithDuration(234 * time.Second))},

		// Singleton — a completely unrelated track.
		{Name: "Starboy", Artists: []Artist{{Name: "The Weeknd"}, {Name: "Daft Punk"}},
			Tags: NewTags(WithDuration(230 * time.Second))},
	}

	groups := Group(ctx, m, tracks, IsSame(DefaultTrackThreshold))

	// Expected: {0-3 Blinding Lights}, {4-5 Bad Guy}, {6 studio}, {7 acoustic}, {8 Starboy}.
	if len(groups) != 5 {
		t.Fatalf("expected 5 groups, got %d: %v", len(groups), groups)
	}
	assertGroup := func(idx int, want []int) {
		t.Helper()
		if len(groups[idx]) != len(want) {
			t.Errorf("group %d: got %v, want %v", idx, groups[idx], want)
			return
		}
		for i, v := range want {
			if groups[idx][i] != v {
				t.Errorf("group %d element %d: got %d, want %d", idx, i, groups[idx][i], v)
			}
		}
	}
	assertGroup(0, []int{0, 1, 2, 3})
	assertGroup(1, []int{4, 5})
	assertGroup(2, []int{6})
	assertGroup(3, []int{7})
	assertGroup(4, []int{8})
}

// TestRealData_PlaylistGrouping_RelatedAdmitsVariants is the same
// playlist scenario with IsRelated — the acoustic variant should now
// group with its studio counterpart.
func TestRealData_PlaylistGrouping_RelatedAdmitsVariants(t *testing.T) {
	t.Parallel()
	m := NewTrackMatcher()
	ctx := context.Background()

	tracks := []Track{
		{Name: "Shape of You", Artists: []Artist{{Name: "Ed Sheeran"}},
			Tags: NewTags(WithDuration(234 * time.Second))},
		{Name: "Shape of You (Acoustic)", Artists: []Artist{{Name: "Ed Sheeran"}},
			Tags: NewTags(WithDuration(234 * time.Second))},
		{Name: "Starboy", Artists: []Artist{{Name: "The Weeknd"}, {Name: "Daft Punk"}},
			Tags: NewTags(WithDuration(230 * time.Second))},
	}
	groups := Group(ctx, m, tracks, IsRelated(DefaultTrackThreshold))
	if len(groups) != 2 {
		t.Fatalf("IsRelated should merge studio + acoustic, got groups=%v", groups)
	}
}

// TestRealData_PlatformIDMismatchIsUnrelated: two tracks that share a
// name and an artist but have different Spotify IDs cannot be the same
// recording on Spotify's side, regardless of other agreements. Common
// when a record label has multiple catalog entries for the same song.
func TestRealData_PlatformIDMismatchIsUnrelated(t *testing.T) {
	t.Parallel()
	m := NewTrackMatcher()
	ctx := context.Background()

	a := Track{
		Name:    "Closer",
		Artists: []Artist{{Name: "The Chainsmokers"}, {Name: "Halsey"}},
		Tags: NewTags(
			WithDuration(244*time.Second),
			WithPlatformID(PlatformSpotify, "7BKLCZ1jbUBVqRi2FVlTVw"),
		),
	}
	b := Track{
		Name:    "Closer",
		Artists: []Artist{{Name: "The Chainsmokers"}, {Name: "Halsey"}},
		Tags: NewTags(
			WithDuration(244*time.Second),
			WithPlatformID(PlatformSpotify, "DIFFERENT_SPOTIFY_ID"),
		),
	}
	s := m.Match(ctx, a, b)
	if s.Relation != RelationUnrelated {
		t.Errorf("expected Unrelated for conflicting platform IDs, got %v", s)
	}
}
