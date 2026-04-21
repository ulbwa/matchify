package versionparse

import (
	"reflect"
	"testing"
)

func TestMarkers(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		in   string
		want []Marker
	}{
		{"none", "1989", nil},
		{"deluxe paren", "1989 (Deluxe Edition)", []Marker{MarkerDeluxe}},
		{"deluxe word", "1989 Deluxe", []Marker{MarkerDeluxe}},
		{"super deluxe", "Thriller (Super Deluxe Edition)", []Marker{MarkerSuperDeluxe}},
		{"expanded", "Album (Expanded Edition)", []Marker{MarkerExpanded}},
		{"extended", "Album (Extended Edition)", []Marker{MarkerExtended}},
		{"25th anniversary", "Thriller (25th Anniversary Edition)", []Marker{MarkerAnniversary}},
		{"anniversary only", "Album (Anniversary Edition)", []Marker{MarkerAnniversary}},
		{"remastered", "Album (Remastered)", []Marker{MarkerRemaster}},
		{"remastered year", "Album (Remastered 2011)", []Marker{MarkerRemaster}},
		{"year remaster", "Album (2011 Remaster)", []Marker{MarkerRemaster}},
		{"digital remaster", "Album - Digital Remaster", []Marker{MarkerRemaster}},
		{"japan", "Album (Japanese Edition)", []Marker{MarkerJapan}},
		{"japan short", "Album (Japan Edition)", []Marker{MarkerJapan}},
		{"international", "Album (International Edition)", []Marker{MarkerInternational}},
		{"limited", "Album (Limited Edition)", []Marker{MarkerLimited}},
		{"collector", "Album (Collector's Edition)", []Marker{MarkerCollector}},
		{"collector plural", "Album (Collectors Edition)", []Marker{MarkerCollector}},
		{"special", "Album (Special Edition)", []Marker{MarkerSpecial}},
		{"deluxe not super deluxe", "Album (Deluxe)", []Marker{MarkerDeluxe}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := Markers(tc.in)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("Markers(%q) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}

func TestMarkersSuperDeluxeIsNotAlsoDeluxe(t *testing.T) {
	t.Parallel()
	// Regression: "Super Deluxe" should be recognised as super_deluxe only,
	// not also as deluxe (otherwise the set comparison in SameEdition would
	// see "Super Deluxe" and "Deluxe" as overlapping).
	got := Markers("Album (Super Deluxe Edition)")
	for _, m := range got {
		if m == MarkerDeluxe {
			t.Errorf("Super Deluxe should not report MarkerDeluxe; got %v", got)
		}
	}
}

func TestSameEdition(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		a, b string
		want bool
	}{
		{"both plain", "1989", "1989", true},
		{"both deluxe", "1989 (Deluxe Edition)", "1989 - Deluxe", true},
		{"one deluxe one plain", "1989 (Deluxe Edition)", "1989", false},
		{"different markers", "1989 (Deluxe)", "1989 (Remastered)", false},
		{"both remaster different years", "Album (2009 Remaster)", "Album (Remastered 2011)", true},
		{"anniversary same", "Thriller (25th Anniversary)", "Thriller (Anniversary Edition)", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := SameEdition(tc.a, tc.b); got != tc.want {
				t.Errorf("SameEdition(%q, %q) = %v, want %v", tc.a, tc.b, got, tc.want)
			}
		})
	}
}

func TestHasAny(t *testing.T) {
	t.Parallel()
	if !HasAny("Album (Deluxe)") {
		t.Error("expected HasAny to be true for Album (Deluxe)")
	}
	if HasAny("Album") {
		t.Error("expected HasAny to be false for plain Album")
	}
}

func TestMarkerOrderStable(t *testing.T) {
	t.Parallel()
	// Multiple markers must return in the order defined by markerPatterns
	// (more specific before less specific), so that callers can rely on
	// a stable canonical ordering. Do NOT sort before comparing — that
	// would defeat the purpose of the test.
	got := Markers("Album (Deluxe, Remastered)")
	want := []Marker{MarkerDeluxe, MarkerRemaster}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got markers %v, want %v (canonical order is pattern-definition order)", got, want)
	}

	// Reversing the markers in the input must not change the output order.
	got2 := Markers("Album (Remastered, Deluxe)")
	if !reflect.DeepEqual(got2, want) {
		t.Errorf("input order should not affect output: got %v, want %v", got2, want)
	}
}
