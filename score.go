package matchify

import (
	"fmt"
	"strings"
)

// Score is the result of comparing two entities.
type Score struct {
	// Value is the match confidence in [0, 1]. A value of 1 means
	// authoritative identifiers agreed; 0 means no evidence of a match.
	Value float64

	// Signals records the individual contributions that made up Value. It
	// is always populated by the built-in matchers so callers can inspect
	// the reasoning without having to re-run scoring.
	Signals []Signal
}

// Above reports whether the score meets the given threshold.
func (s Score) Above(threshold float64) bool { return s.Value >= threshold }

// Signal records the contribution of a single matching signal to the overall
// score. Signal values are normalised to [0, 1]; Weight values are the
// relative weights used to combine them.
type Signal struct {
	// Name identifies the signal (e.g., "isrc", "name", "duration").
	Name string

	// Value is the per-signal contribution in [0, 1].
	Value float64

	// Weight is the relative weight of this signal among all signals in
	// the score.
	Weight float64

	// Note is an optional human-readable note (e.g. "within 2s", "isrc
	// mismatch"). Empty when no extra context is available.
	Note string
}

// String returns a compact representation of the score, suitable for logs
// and debugging.
func (s Score) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%.3f", s.Value)
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

// combineSignals computes the weighted average of the given signals. Signals
// with zero weight are ignored. If no signal has a positive weight, the
// result is zero.
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

// scoreOf constructs a Score with the given signals, combining them into a
// weighted average.
func scoreOf(signals ...Signal) Score {
	return Score{
		Value:   combineSignals(signals),
		Signals: signals,
	}
}

// authoritativeScore returns a fixed-value Score built from a single
// authoritative signal. Used when identifiers like ISRC or UPC agree.
func authoritativeScore(name, note string, value float64) Score {
	return Score{
		Value: value,
		Signals: []Signal{
			{Name: name, Value: value, Weight: 1, Note: note},
		},
	}
}
