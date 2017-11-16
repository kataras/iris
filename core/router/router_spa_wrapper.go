package router

import (
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
// It does what StaticWeb or StaticEmbedded expected to do when serving files and routes at the same time
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
				return true // returns true by-default, if false then it fires 404.
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

// Handler serves the asset handler but in addition, it makes some checks before that,
// based on the `AssetValidators` and `IndexNames`.
func (s *SPABuilder) Handler(ctx context.Context) {
	path := ctx.Path()

	// make a validator call, by-default all paths are valid and this codeblock doesn't mean anything
	// but for cases that users wants to bypass an asset she/he can do that by modifiying the `APIBuilder#AssetValidators` field.
	//
	// It's here for backwards compatibility as well, see #803.
	if !s.isAsset(path) {
		// it's not asset, execute the registered route's handlers
		ctx.NotFound()
		return
	}

	for _, index := range s.IndexNames {
		if strings.HasSuffix(path, index) {
			localRedirect(ctx, "./")
			// "/" should be manually  registered.
			// We don't setup an index handler here,
			// let full control to the user
			// (use middleware, ctx.ServeFile or ctx.View and so on...)
			return
		}
	}

	s.AssetHandler(ctx)
}
