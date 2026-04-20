// Package explicitparse detects explicit/clean markers in a release or
// track name.
//
// Clean and explicit masters are distinct products — never the same
// recording — so matchers must be able to tell them apart even when the
// caller didn't populate a structured explicit flag. This package returns
// an Explicitness value parsed from the name itself, which matchers then
// combine with whatever explicit flag the caller provided.
package explicitparse

import "regexp"

// Explicitness is a tri-state: Unknown (no marker detected), Clean, or
// Explicit. It mirrors matchify.Explicitness but lives in an internal
// package to avoid an import cycle.
type Explicitness uint8

// Possible detected values.
const (
	Unknown Explicitness = iota
	Clean
	Explicit
)

var (
	// explicitPattern matches explicit markers inside parentheses,
	// brackets, or after a dash separator. "edited" is treated as a
	// clean indicator (common in hip-hop releases where "edited" means
	// censored).
	explicitPattern = regexp.MustCompile(
		`(?i)(?:\(|\[|\s[-–—]\s+)` +
			`\s*(?:explicit(?:\s+version)?)\s*` +
			`(?:\)|\]|$)`,
	)

	cleanPattern = regexp.MustCompile(
		`(?i)(?:\(|\[|\s[-–—]\s+)` +
			`\s*(?:clean(?:\s+version)?|edited(?:\s+version)?)\s*` +
			`(?:\)|\]|$)`,
	)
)

// Detect parses explicit/clean markers from a release or track name and
// returns the detected Explicitness. If neither marker is present (or
// both are — a pathological case we treat as Unknown), Detect returns
// Unknown.
func Detect(name string) Explicitness {
	hasClean := cleanPattern.MatchString(name)
	hasExplicit := explicitPattern.MatchString(name)
	switch {
	case hasClean && hasExplicit:
		return Unknown
	case hasClean:
		return Clean
	case hasExplicit:
		return Explicit
	default:
		return Unknown
	}
}

// Differ reports whether a and b are both known and disagree. Two tracks
// or releases whose explicit/clean labels differ must not be treated as
// the same product.
func Differ(a, b Explicitness) bool {
	return a != Unknown && b != Unknown && a != b
}
