package context

import (
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/kataras/iris/v12/macro"
)

// RouteReadOnly allows decoupled access to the current route
// inside the context.
type RouteReadOnly interface {
	// Name returns the route's name.
	Name() string

	// Method returns the route's method.
	Method() string

	// Subdomains returns the route's subdomain.
	Subdomain() string

	// Path returns the route's original registered path.
	Path() string

	// String returns the form of METHOD, SUBDOMAIN, TMPL PATH.
	String() string

	// IsOnline returns true if the route is marked as "online" (state).
	IsOnline() bool

	// IsStatic reports whether this route is a static route.
	// Does not contain dynamic path parameters,
	// is online and registered on GET HTTP Method.
	IsStatic() bool
	// StaticPath returns the static part of the original, registered route path.
	// if /user/{id} it will return /user
	// if /user/{id}/friend/{friendid:uint64} it will return /user too
	// if /assets/{filepath:path} it will return /assets.
	StaticPath() string

	// ResolvePath returns the formatted path's %v replaced with the args.
	ResolvePath(args ...string) string
	// Trace returns some debug infos as a string sentence.
	// Should be called after Build.
	Trace() string

	// Tmpl returns the path template,
	// it contains the parsed template
	// for the route's path.
	// May contain zero named parameters.
	//
	// Available after the build state, i.e a request handler or Iris Configurator.
	Tmpl() macro.Template

	// MainHandlerName returns the first registered handler for the route.
	MainHandlerName() string

	// StaticSites if not empty, refers to the system (or virtual if embedded) directory
	// and sub directories that this "GET" route was registered to serve files and folders
	// that contain index.html (a site). The index handler may registered by other
	// route, manually or automatic by the framework,
	// get the route by `Application#GetRouteByPath(staticSite.RequestPath)`.
	StaticSites() []StaticSite

	// Sitemap properties: https://www.sitemaps.org/protocol.html

	// GetLastMod returns the date of last modification of the file served by this route.
	GetLastMod() time.Time
	// GetChangeFreq returns the the page frequently is likely to change.
	GetChangeFreq() string
	// GetPriority returns the priority of this route's URL relative to other URLs on your site.
	GetPriority() float32
}

// StaticSite is a structure which is used as field on the `Route`
// and route registration on the `APIBuilder#HandleDir`.
// See `GetStaticSites` and `APIBuilder#HandleDir`.
type StaticSite struct {
	Dir         string `json:"dir"`
	RequestPath string `json:"requestPath"`
}

// GetStaticSites search for a relative filename of "indexName" in "rootDir" and all its subdirectories
// and returns a list of structures which contains the directory found an "indexName" and the request path
// that a route should be registered to handle this "indexName".
// The request path is given by the directory which an index exists on.
func GetStaticSites(rootDir, rootRequestPath, indexName string) (sites []StaticSite) {
	f, err := os.Open(rootDir)
	if err != nil {
		return nil
	}

	list, err := f.Readdir(-1)
	f.Close()
	if err != nil {
		return nil
	}

	if len(list) == 0 {
		return nil
	}

	for _, l := range list {
		dir := filepath.Join(rootDir, l.Name())

		if l.IsDir() {
			sites = append(sites, GetStaticSites(dir, path.Join(rootRequestPath, l.Name()), indexName)...)
			continue
		}

		if l.Name() == strings.TrimPrefix(indexName, "/") {
			sites = append(sites, StaticSite{
				Dir:         filepath.FromSlash(rootDir),
				RequestPath: rootRequestPath,
			})
			continue
		}
	}

	return
}
