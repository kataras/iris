package context

// MinimalRestErrorRenderer exposes the built-in fallback renderer to the
// external tests. The canonical rest/errors renderer replaces it at init
// time whenever that package is linked in, so the fallback's own behavior
// is only reachable by calling it directly.
var MinimalRestErrorRenderer RestErrorRenderer = minimalRestErrorRenderer{}
