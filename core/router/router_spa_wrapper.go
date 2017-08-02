package router

import (
	"net/http"
	"strings"

	"github.com/kataras/iris/context"
)

// AssetValidator returns true if "filename"
// is asset, i.e: strings.Contains(filename, ".").
type AssetValidator func(filename string) bool

// SPABuilder helps building a single page application server
// which serves both routes and files from the root path.
type SPABuilder struct {
	IndexNames      []string
	AssetHandler    context.Handler
	AssetValidators []AssetValidator
}

// NewSPABuilder returns a new Single Page Application builder
// It does what StaticWeb expected to do when serving files and routes at the same time
// from the root "/" path.
//
// Accepts a static asset handler, which can be an app.StaticHandler, app.StaticEmbeddedHandler...
func NewSPABuilder(assetHandler context.Handler) *SPABuilder {
	if assetHandler == nil {
		assetHandler = func(ctx context.Context) {
			ctx.Writef("empty asset handler")
		}
	}

	return &SPABuilder{
		IndexNames:   []string{"index.html"},
		AssetHandler: assetHandler,
		AssetValidators: []AssetValidator{
			func(path string) bool {
				return strings.Contains(path, ".")
			},
		},
	}
}

func (s *SPABuilder) isAsset(reqPath string) bool {
	for _, v := range s.AssetValidators {
		if !v(reqPath) {
			return false
		}
	}
	return true
}

// BuildWrapper returns a wrapper which serves the single page application
// with the declared configuration.
//
// It should be passed to the router's `WrapRouter`:
// https://godoc.org/github.com/kataras/iris/core/router#Router.WrapRouter
//
// Example: https://github.com/kataras/iris/tree/master/_examples/file-server/single-page-application-builder
func (s *SPABuilder) BuildWrapper(cPool *context.Pool) WrapperFunc {

	fileServer := s.AssetHandler
	indexNames := s.IndexNames

	wrapper := func(w http.ResponseWriter, r *http.Request, router http.HandlerFunc) {
		path := r.URL.Path

		if !s.isAsset(path) {
			// it's not asset, execute the registered route's handlers
			router(w, r)
			return
		}

		ctx := cPool.Acquire(w, r)

		for _, index := range indexNames {
			if strings.HasSuffix(path, index) {
				localRedirect(ctx, "./")
				cPool.Release(ctx)
				// "/" should be manually  registered.
				// We don't setup an index handler here,
				// let full control to the user
				// (use middleware, ctx.ServeFile or ctx.View and so on...)
				return
			}
		}

		// execute file server for path
		fileServer(ctx)

		cPool.Release(ctx)
	}

	return wrapper
}
