package matchify

import (
	"fmt"
	"strings"
)

// Relation classifies how two entities are related — distinct from just
// how confident we are in that relation.
type Relation uint8

// Known relation kinds.
const (
	// RelationUnrelated means the two entities are different works or
	// different artists, not variants of the same thing.
	RelationUnrelated Relation = iota

	// RelationVariant means the entities are variants of the same
	// underlying work: the same song or album but in a different
	// recording, edition, explicit/clean master, or similar. They are
	// NOT the same product, but they are linked. Matchers use this to
	// signal cases like "plain album vs Deluxe Edition" or "studio
	// track vs Stripped re-recording".
	RelationVariant

	// RelationSame means the entities are the same product — a
	// high-confidence Value here indicates that merging the two sides
	// is safe.
	RelationSame
)

// String returns a lower-case label for the Relation.
func (r Relation) String() string {
	switch r {
	case RelationVariant:
		return "variant"
	case RelationSame:
		return "same"
	default:
		return "unrelated"
	}
}

// Score is the result of comparing two entities.
//
// Value and Relation convey two different things. Relation says what kind
// of relationship (if any) the matcher inferred; Value says how confident
// the matcher is in that inference. Callers typically use Same or Related
// to turn a Score into a yes/no decision rather than comparing Value to a
// bare threshold, so "same" and "variant" can be distinguished cleanly.
type Score struct {
	// Value is the match confidence in [0, 1]. A value of 1 means
	// authoritative identifiers agreed; 0 means no evidence at all.
	Value float64

	// Relation classifies the match kind. See Relation.
	Relation Relation

	// Signals records the individual contributions that make up Value.
	// Matchers always populate this so callers can inspect the reasoning
	// behind a score.
	Signals []Signal
}

// Same reports whether the score indicates the two sides are the same
// product at or above threshold confidence. This is the right check when
// the caller wants to collapse duplicates — e.g. grouping tracks in a
// cross-platform playlist.
func (s Score) Same(threshold float64) bool {
	return s.Relation == RelationSame && s.Value >= threshold
}

// Related reports whether the score indicates the two sides are at
// least variants of the same underlying work. Use this to find anything
// associated with a song or album — the original, remasters, live
// renditions, stripped re-recordings, different editions.
func (s Score) Related(threshold float64) bool {
	return s.Relation >= RelationVariant && s.Value >= threshold
}

// Signal records the contribution of a single matching signal to the
// overall score. Signal values are normalised to [0, 1]; Weight values
// are the relative weights used to combine them.
type Signal struct {
	// Name identifies the signal (e.g., "isrc", "name", "duration").
	Name string

	// Value is the per-signal contribution in [0, 1].
	Value float64

	// Weight is the relative weight of this signal among all signals
	// in the score. A weight of 0 means the signal is informational
	// only and does not influence Value.
	Weight float64

	// Note is an optional human-readable note (e.g. "within 2s",
	// "isrc mismatch"). Empty when no extra context is available.
	Note string
}

// String returns a compact representation of the score, suitable for
// logs and debugging.
func (s Score) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s %.3f", s.Relation, s.Value)
	if len(s.Signals) == 0 {
		return b.String()
	}
	b.WriteString(" {")
	for i, sig := range s.Signals {
		if i > 0 {
			b.WriteString(", ")
		}
		fmt.Fprintf(&b, "%s=%.2f*%.2f", sig.Name, sig.Value, sig.Weight)
		if sig.Note != "" {
			fmt.Fprintf(&b, " (%s)", sig.Note)
		}
	}
	b.WriteString("}")
	return b.String()
}

// combineSignals computes the weighted average of the given signals.
// Signals with zero weight are ignored. If no signal has positive weight,
// the result is zero.
func combineSignals(signals []Signal) float64 {
	var totalWeighted, totalWeight float64
	for _, s := range signals {
		if s.Weight <= 0 {
			continue
		}
		totalWeighted += s.Value * s.Weight
		totalWeight += s.Weight
	}
	if totalWeight == 0 {
		return 0
	}
	return totalWeighted / totalWeight
}

// scoreOf constructs a Score with the given signals at RelationSame,
// combining them into a weighted average for Value. The caller is free
// to downgrade Relation or override Value afterwards.
func scoreOf(signals ...Signal) Score {
	return Score{
		Value:    combineSignals(signals),
		Relation: RelationSame,
		Signals:  signals,
	}
}

// authoritativeScore returns a fixed-value Score built from a single
// authoritative signal (ISRC, UPC, MBID, ...). Relation is RelationSame.
func authoritativeScore(name, note string, value float64) Score {
	return Score{
		Value:    value,
		Relation: RelationSame,
		Signals: []Signal{
			{Name: name, Value: value, Weight: 1, Note: note},
		},
	}
}
