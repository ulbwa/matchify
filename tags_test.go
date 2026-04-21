package matchify

import (
	"testing"
	"time"
)

func TestTags_EmptyAccessors(t *testing.T) {
	t.Parallel()
	var tags Tags
	if _, ok := tags.Duration(); ok {
		t.Error("empty Tags should report no Duration")
	}
	if _, ok := tags.ISRC(); ok {
		t.Error("empty Tags should report no ISRC")
	}
	if a := tags.Aliases(); len(a) != 0 {
		t.Errorf("empty Tags Aliases should be empty, got %v", a)
	}
	if p := tags.PlatformIDs(); len(p) != 0 {
		t.Errorf("empty Tags PlatformIDs should be empty, got %v", p)
	}
}

func TestTags_SetAndRead(t *testing.T) {
	t.Parallel()
	tags := NewTags(
		WithDuration(120*time.Second),
		WithDiscNumber(2),
		WithTrackNumber(5),
		WithTrackCount(10),
		WithExplicit(ExplicitnessClean),
		WithISRC("USMV10000001"),
		WithUPC("012345678905"),
		WithMBID("abc-123"),
		WithReleaseDate(time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)),
		WithReleaseType(ReleaseTypeAlbum),
		WithAlias("Alt1", "Alt2"),
		WithPlatformID(PlatformSpotify, "sp1"),
		WithPlatformID(PlatformTidal, "td1"),
	)

	if d, ok := tags.Duration(); !ok || d != 120*time.Second {
		t.Errorf("Duration: got %v, %v", d, ok)
	}
	if n, ok := tags.DiscNumber(); !ok || n != 2 {
		t.Errorf("DiscNumber: got %v, %v", n, ok)
	}
	if n, ok := tags.TrackNumber(); !ok || n != 5 {
		t.Errorf("TrackNumber: got %v, %v", n, ok)
	}
	if n, ok := tags.TrackCount(); !ok || n != 10 {
		t.Errorf("TrackCount: got %v, %v", n, ok)
	}
	if e, ok := tags.Explicit(); !ok || e != ExplicitnessClean {
		t.Errorf("Explicit: got %v, %v", e, ok)
	}
	if s, ok := tags.ISRC(); !ok || s != "USMV10000001" {
		t.Errorf("ISRC: got %v, %v", s, ok)
	}
	if s, ok := tags.UPC(); !ok || s != "012345678905" {
		t.Errorf("UPC: got %v, %v", s, ok)
	}
	if s, ok := tags.MBID(); !ok || s != "abc-123" {
		t.Errorf("MBID: got %v, %v", s, ok)
	}
	if y, ok := tags.ReleaseYear(); !ok || y != 2020 {
		t.Errorf("ReleaseYear: got %v, %v", y, ok)
	}
	if r, ok := tags.ReleaseType(); !ok || r != ReleaseTypeAlbum {
		t.Errorf("ReleaseType: got %v, %v", r, ok)
	}

	aliases := tags.Aliases()
	if len(aliases) != 2 || aliases[0] != "Alt1" || aliases[1] != "Alt2" {
		t.Errorf("Aliases: got %v", aliases)
	}
	if id, ok := tags.PlatformID(PlatformSpotify); !ok || id != "sp1" {
		t.Errorf("PlatformID(Spotify): got %v, %v", id, ok)
	}
	if id, ok := tags.PlatformID(PlatformTidal); !ok || id != "td1" {
		t.Errorf("PlatformID(Tidal): got %v, %v", id, ok)
	}
	if _, ok := tags.PlatformID(PlatformQobuz); ok {
		t.Errorf("PlatformID(Qobuz) should be absent")
	}
}

func TestTags_ZeroValueIsValid(t *testing.T) {
	t.Parallel()
	// Zero-value Explicit is distinguishable from unset — a tag of
	// WithExplicit(ExplicitnessUnknown) must still report ok=true.
	tags := NewTags(WithExplicit(ExplicitnessUnknown))
	if _, ok := tags.Explicit(); !ok {
		t.Error("WithExplicit(Unknown) should still set the tag")
	}
}

func TestTags_EmptyStringsIgnored(t *testing.T) {
	t.Parallel()
	tags := NewTags(WithAlias("", "Real"), WithPlatformID(PlatformSpotify, ""))
	aliases := tags.Aliases()
	if len(aliases) != 1 || aliases[0] != "Real" {
		t.Errorf("empty alias should be skipped, got %v", aliases)
	}
	if _, ok := tags.PlatformID(PlatformSpotify); ok {
		t.Error("empty platform ID should not be stored")
	}
}

func TestTags_LaterOverrides(t *testing.T) {
	t.Parallel()
	tags := NewTags(
		WithDuration(60*time.Second),
		WithDuration(120*time.Second),
		WithPlatformID(PlatformSpotify, "v1"),
		WithPlatformID(PlatformSpotify, "v2"),
	)
	if d, _ := tags.Duration(); d != 120*time.Second {
		t.Errorf("later WithDuration should win, got %v", d)
	}
	if id, _ := tags.PlatformID(PlatformSpotify); id != "v2" {
		t.Errorf("later WithPlatformID should win, got %v", id)
	}
}
