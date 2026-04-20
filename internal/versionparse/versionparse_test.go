package versionparse

import (
	"reflect"
	"sort"
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
	// Multiple markers should return in stable canonical order (order of definition).
	got := Markers("Album (Deluxe, Remastered)")
	if len(got) != 2 {
		t.Fatalf("expected 2 markers, got %v", got)
	}
	// Build sorted canonical for comparison.
	want := []Marker{MarkerDeluxe, MarkerRemaster}
	sort.Slice(got, func(i, j int) bool { return string(got[i]) < string(got[j]) })
	sort.Slice(want, func(i, j int) bool { return string(want[i]) < string(want[j]) })
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got markers %v, want %v", got, want)
	}
}
