package errors

import (
	"sync"

	"github.com/kataras/iris/v14/context"
)

// ErrorResolver inspects an error inside HandleError and reports whether it
// produced a client response. Resolvers run after the per-route Options code
// map and before the built-in classification, in registration order.
//
// They are the integration point for datastore- or driver-specific error
// mapping without linking the driver into every application: see the
// pgerrors subpackage for the PostgreSQL (github.com/kataras/pg) mapping.
type ErrorResolver func(ctx *context.Context, err error) bool

var (
	errorResolversMu sync.RWMutex
	errorResolvers   []ErrorResolver
)

// RegisterErrorResolver registers fn to run inside HandleError for errors the
// per-route Options code map did not claim. Register at init time or before
// serving; the first resolver that reports true stops the chain.
func RegisterErrorResolver(fn ErrorResolver) {
	if fn == nil {
		return
	}

	errorResolversMu.Lock()
	errorResolvers = append(errorResolvers, fn)
	errorResolversMu.Unlock()
}

// resolveError runs the registered resolvers in order.
func resolveError(ctx *context.Context, err error) bool {
	errorResolversMu.RLock()
	resolvers := errorResolvers
	errorResolversMu.RUnlock()

	for _, resolve := range resolvers {
		if resolve(ctx, err) {
			return true
		}
	}

	return false
}
