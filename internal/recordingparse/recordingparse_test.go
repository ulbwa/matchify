package recordingparse

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
		{"none", "Blinding Lights", nil},
		{"live paren", "Bohemian Rhapsody (Live at Wembley)", []Marker{MarkerLive}},
		{"live short", "Song (Live)", []Marker{MarkerLive}},
		{"live session", "Song (Live Session)", []Marker{MarkerLive}},
		{"unplugged", "Song (Unplugged)", []Marker{MarkerUnplugged}},
		{"acoustic", "Song (Acoustic)", []Marker{MarkerAcoustic}},
		{"acoustic version", "Song (Acoustic Version)", []Marker{MarkerAcoustic}},
		{"remix", "Song (Remix)", []Marker{MarkerRemix}},
		{"specific remix", "Song (Aoki Remix)", []Marker{MarkerRemix}},
		{"demo", "Song (Demo)", []Marker{MarkerDemo}},
		{"instrumental", "Song (Instrumental)", []Marker{MarkerInstrumental}},
		{"karaoke", "Song (Karaoke)", []Marker{MarkerKaraoke}},
		{"piano version", "Song (Piano Version)", []Marker{MarkerPiano}},
		{"orchestral", "Song (Orchestral Version)", []Marker{MarkerOrchestral}},
		{"extended mix", "Song (Extended Mix)", []Marker{MarkerExtendedMix}},
		{"radio edit", "Song (Radio Edit)", []Marker{MarkerRadioEdit}},
		{"spotify session", "Song - Spotify Session", []Marker{MarkerSession}},
		{"taylor's version", "Love Story (Taylor's Version)", []Marker{MarkerRerecording}},
		{"rerecording", "Song (Re-Recorded)", []Marker{MarkerRerecording}},
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

func TestLiveDoesNotFalsePositive(t *testing.T) {
	t.Parallel()
	// "Live" as a substring of larger words must not trigger.
	names := []string{
		"Alive", "Living", "Liver", "Delivery", "Olive",
	}
	for _, n := range names {
		if len(Markers(n)) != 0 {
			t.Errorf("%q: expected no markers, got %v", n, Markers(n))
		}
	}
}

func TestSameVariant(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		a, b string
		want bool
	}{
		{"both studio", "Song", "Song", true},
		{"both live", "Song (Live)", "Song - Live at X", true},
		{"one live", "Song", "Song (Live)", false},
		{"different", "Song (Acoustic)", "Song (Remix)", false},
		{"both acoustic", "Song (Acoustic)", "Song - Acoustic Version", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := SameVariant(tc.a, tc.b); got != tc.want {
				t.Errorf("SameVariant(%q, %q) = %v, want %v", tc.a, tc.b, got, tc.want)
			}
		})
	}
}
