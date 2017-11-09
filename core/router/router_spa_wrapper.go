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

// AddIndexName will add an index name.
// If path == $filename then it redirects to "/".
//
// It can be called after the `BuildWrapper ` as well but BEFORE the server start.
func (s *SPABuilder) AddIndexName(filename string) *SPABuilder {
	s.IndexNames = append(s.IndexNames, filename)
	return s
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
		IndexNames: nil,
		// IndexNames is empty by-default,
		// if the user wants to redirect to "/" from "/index.html" she/he can chage that to  []string{"index.html"} manually.
		AssetHandler: assetHandler,
		AssetValidators: []AssetValidator{
			func(path string) bool {
				return true // returns true by-default
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

	wrapper := func(w http.ResponseWriter, r *http.Request, router http.HandlerFunc) {
		path := r.URL.Path

		// make a validator call, by-default all paths are valid and this codeblock doesn't mean anything
		// but for cases that users wants to bypass an asset she/he can do that by modifiying the `APIBuilder#AssetValidators` field.
		//
		// It's here for backwards compatibility as well, see #803.
		if !s.isAsset(path) {
			// it's not asset, execute the registered route's handlers
			router(w, r)
			return
		}

		for _, index := range s.IndexNames {
			if strings.HasSuffix(path, index) {
				localRedirect(w, r, "./")
				// "/" should be manually  registered.
				// We don't setup an index handler here,
				// let full control to the user
				// (use middleware, ctx.ServeFile or ctx.View and so on...)
				return
			}
		}

		ctx := cPool.Acquire(w, r)
		// convert to a recorder in order to not write the status and body directly but wait for a flush (EndRequest).
		rec := ctx.Recorder() // rec and context.ResponseWriter() is the same thing now.
		// execute the asset handler.
		fileServer(ctx)
		// check if body was written, if not then;
		// 1. reset the whole response writer, its status code, headers and body
		// 2. release only the object,
		//                           so it doesn't fires the status code's handler to the client
		//                          (we are eliminating the multiple response header calls this way)
		// 3. execute the router itself, if route found then it will serve that, otherwise 404 or 405.
		//
		// we could also use the ctx.ResponseWriter().Written() > 0.
		empty := len(rec.Body()) == 0
		if empty {
			rec.Reset()
			cPool.ReleaseLight(ctx)
			router(w, r)
			return
		}
		// if body was written from the file server then release the context as usual,
		// it will send everything to the client and reset the context.
		cPool.Release(ctx)
	}

	return wrapper
}
