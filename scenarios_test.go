package matchify

import (
	"context"
	"testing"
)

// TestScenario_ToveLoBritney pins the expected behaviour for a user-
// supplied scenario comparing editions and variants of two albums from
// different artists. The rules it asserts:
//
//   - "Deluxe Version" and "Deluxe Edition" are equivalent cosmetic
//     labels for the same edition → RelationSame.
//   - An edition marker on only one side (plain vs Deluxe) →
//     RelationVariant — same album, different packaging.
//   - A recording-variant marker on only one side (plain vs Stripped,
//     plain vs Extended Cut) → RelationVariant — different re-recording.
//   - Different artist + different album → RelationUnrelated (name
//     similarity is too low even before relation classification).
func TestScenario_ToveLoBritney(t *testing.T) {
	t.Parallel()
	m := NewAlbumMatcher()
	ctx := context.Background()

	toveLo := []Artist{{Name: "Tove Lo"}}
	britney := []Artist{{Name: "Britney Spears"}}

	dirtFemme := Album{Name: "Dirt Femme", Artists: toveLo}
	dirtFemmeExtended := Album{Name: "Dirt Femme (Extended Cut)", Artists: toveLo}
	dirtFemmeStripped := Album{Name: "Dirt Femme (Stripped)", Artists: toveLo}
	femmeFataleDVer := Album{Name: "Femme Fatale (Deluxe Version)", Artists: britney}
	femmeFataleDEd := Album{Name: "Femme Fatale (Deluxe Edition)", Artists: britney}

	cases := []struct {
		name     string
		a, b     Album
		wantSame bool
	}{
		{"dirt femme vs extended cut", dirtFemme, dirtFemmeExtended, false},
		{"dirt femme vs stripped", dirtFemme, dirtFemmeStripped, false},
		{"dirt femme vs femme fatale deluxe version", dirtFemme, femmeFataleDVer, false},
		{"dirt femme vs femme fatale deluxe edition", dirtFemme, femmeFataleDEd, false},
		{"extended cut vs stripped", dirtFemmeExtended, dirtFemmeStripped, false},
		{"extended cut vs femme fatale deluxe version", dirtFemmeExtended, femmeFataleDVer, false},
		{"extended cut vs femme fatale deluxe edition", dirtFemmeExtended, femmeFataleDEd, false},
		{"stripped vs femme fatale deluxe version", dirtFemmeStripped, femmeFataleDVer, false},
		{"stripped vs femme fatale deluxe edition", dirtFemmeStripped, femmeFataleDEd, false},
		{"deluxe version vs deluxe edition", femmeFataleDVer, femmeFataleDEd, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			score := m.Match(ctx, tc.a, tc.b)
			if got := score.Same(DefaultAlbumThreshold); got != tc.wantSame {
				t.Errorf("Match(%q, %q): Same=%v, want %v (score=%v)",
					tc.a.Name, tc.b.Name, got, tc.wantSame, score)
			}
			if tc.wantSame && score.Relation != RelationSame {
				t.Errorf("expected RelationSame, got %v (score=%v)", score.Relation, score)
			}
		})
	}
}
