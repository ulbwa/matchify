package matchify

import "context"

// ReleaseProvider is an optional hook that lets ArtistMatcher fetch an
// artist's discography when name-based comparison is ambiguous. The library
// never calls this automatically outside of the artist matcher; passing nil
// (or leaving the field unset) disables release-based cross-checking.
//
// Implementations should honour ctx cancellation. Returning an error from a
// provider causes the matcher to fall back to the name-based score.
type ReleaseProvider interface {
	Releases(ctx context.Context, artist Artist) ([]Album, error)
}

// ReleaseProviderFunc adapts an ordinary function to the ReleaseProvider
// interface.
type ReleaseProviderFunc func(ctx context.Context, artist Artist) ([]Album, error)

// Releases calls f.
func (f ReleaseProviderFunc) Releases(ctx context.Context, artist Artist) ([]Album, error) {
	return f(ctx, artist)
}
