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
	// Root  defaults to "/", it's the root path that explicitly set-ed,
	// this can be changed if more than SPAs are used on the same
	// iris router instance.
	Root string
	// emptyRoot can be changed with `ChangeRoot` only,
	// is, statically, true if root is empty
	// and if root is empty then let 404 fire from server-side anyways if
	// the passed `AssetHandler` returns 404 for a specific request path.
	// Defaults to false.
	emptyRoot bool

	IndexNames      []string
	AssetHandler    context.Handler
	AssetValidators []AssetValidator
}

// AddIndexName will add an index name.
// If path == $filename then it redirects to Root, which defaults to "/".
//
// It can be called BEFORE the server start.
func (s *SPABuilder) AddIndexName(filename string) *SPABuilder {
	s.IndexNames = append(s.IndexNames, filename)
	return s
}

// ChangeRoot modifies the `Root` request path that is
// explicitly set-ed if the `AssetHandler` gave a Not Found (404)
// previously, if request's path is the passed "path"
// then it explicitly sets that and it retries executing the `AssetHandler`.
//
// Empty Root means that let 404 fire from server-side anyways.
//
// Change it ONLY if you use more than one typical SPAs on the same Iris Application instance.
func (s *SPABuilder) ChangeRoot(path string) *SPABuilder {
	s.Root = path
	s.emptyRoot = path == ""
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
		Root:       "/",
		IndexNames: nil,
		// "IndexNames" are empty by-default,
		// if the user wants to redirect to "/" from "/index.html" she/he can chage that to  []string{"index.html"} manually
		// or use the `StaticHandler` as "AssetHandler" which does that already.
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
			if s.emptyRoot {
				ctx.NotFound()
				return
			}
			localRedirect(ctx, "."+s.Root)
			// s.Root should be manually registered to a route
			// (not always, only if custom handler used).
			// We don't setup an index handler here,
			// let full control to the developer via "AssetHandler"
			// (use of middleware, manually call of the ctx.ServeFile or ctx.View etc.)
			return
		}
	}

	s.AssetHandler(ctx)

	if context.StatusCodeNotSuccessful(ctx.GetStatusCode()) && !s.emptyRoot && path != s.Root {
		// If file was not something like a javascript file, or a css or anything that
		// the passed `AssetHandler` scan-ed then re-execute the `AssetHandler`
		// using the `Root` as the request path (virtually).
		//
		// If emptyRoot is true then
		// fire the response as it's, "AssetHandler" is fully responsible for it,
		// client-side's router for invalid paths will not work here else read below.
		//
		// Author's notes:
		// the server doesn't need to know all client routes,
		// client-side router is responsible for any kind of invalid paths,
		// so explicit set to root path.
		//
		// The most simple solution was to use a
		// func(ctx iris.Context) { ctx.ServeFile("$PATH/index.html") } as the "AssetHandler"
		// but many developers use the `StaticHandler` (as shown in the examples)
		// but it was not working as expected because it (correctly) fires
		// a 404 not found if a file based on the request path didn't found.
		//
		// We can't just do it before the "AssetHandler"'s execution
		// for two main reasons:
		// 1. if it's a file serve handler, like `StaticHandler` then it will never serve
		// the corresponding files!
		// 2. it may manually handle those things,
		// don't forget that "AssetHandler" can be
		// ANY iris handler, so we can't be sure what the developer may want to do there.
		//
		// "AssetHandler" as the "StaticHandler" a retry doesn't hurt,
		// it will give us a 404 if the file didn't found very fast WITHOUT moving to the
		// rest of its validation and serving implementation.
		//
		// Another idea would be to modify the "AssetHandler" on every `ChangeRoot`
		// call, which may give us some performance (ns) benefits
		// but this could be bad if root is set-ed before the "AssetHandler",
		// so keep it as it's.
		rootURL, err := ctx.Request().URL.Parse(s.Root)
		if err == nil {
			ctx.Request().URL = rootURL
			s.AssetHandler(ctx)
		}

	}
}
