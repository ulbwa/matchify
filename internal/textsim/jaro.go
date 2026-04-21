// Package textsim provides thin, library-backed string similarity helpers
// used by matchers.
//
// The underlying metric is Jaro-Winkler from github.com/adrg/strutil,
// wrapped here so the rest of the library depends on a stable internal
// surface even if the upstream package is swapped out later.
package textsim

import (
	"sync"

	"github.com/adrg/strutil/metrics"
)

// jwPool serves cheap, reusable JaroWinkler configurations. metrics.JaroWinkler
// carries no mutable state once constructed, so pooling avoids allocating
// one per call without requiring a package-level singleton.
var jwPool = sync.Pool{
	New: func() any { return metrics.NewJaroWinkler() },
}

// JaroWinkler returns the Jaro-Winkler similarity of two strings in [0, 1].
// The underlying implementation is rune-aware, so multi-byte inputs behave
// correctly.
func JaroWinkler(a, b string) float64 {
	jw := jwPool.Get().(*metrics.JaroWinkler)
	defer jwPool.Put(jw)
	return jw.Compare(a, b)
}
