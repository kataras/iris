package router

import (
	"bytes"
	"fmt"

	"html"
	"net/http"
	"sort"
	"strings"

	"github.com/kataras/golog"

	"github.com/kataras/iris/context"
	"github.com/kataras/iris/core/errors"
	"github.com/kataras/iris/core/netutil"
	"github.com/kataras/iris/core/router/node"
)

// RequestHandler the middle man between acquiring a context and releasing it.
// By-default is the router algorithm.
type RequestHandler interface {
	// HandleRequest is same as context.Handler but its usage is only about routing,
	// separate the concept here.
	HandleRequest(context.Context)
	// Build  should builds the handler, it's being called on router's BuildRouter.
	Build(provider RoutesProvider) error
	// RouteExists checks if a route exists
	RouteExists(method, path string, ctx context.Context) bool
}

type tree struct {
	Method string
	// subdomain is empty for default-hostname routes,
	// ex: mysubdomain.
	Subdomain string
	Nodes     *node.Nodes
}

func (t *tree) String() string {
	var out bytes.Buffer

	fmt.Fprintf(&out, "[%s] %s", t.Method, t.Subdomain)
	fmt.Fprintln(&out)
	fmt.Fprintln(&out, t.Nodes)

	return out.String()
}

type routerHandler struct {
	trees            []*tree
	hosts            bool // true if at least one route contains a Subdomain.
	fallbackHandlers context.Handlers
}

var _ RequestHandler = &routerHandler{}

// String shows router representation
func (h *routerHandler) String() string {
	res := ""

	for _, t := range h.trees {
		res += t.String()
	}

	return res
}

func (h *routerHandler) getTree(method, subdomain string) *tree {
	for i := range h.trees {
		t := h.trees[i]
		if t.Method == method && t.Subdomain == subdomain {
			return t
		}
	}

	return nil
}

func (h *routerHandler) addRoute(r *Route) error {
	var (
		routeName = r.Name
		method    = r.Method
		subdomain = r.Subdomain
		path      = r.Path
		handlers  = r.Handlers
	)

	if r.isSpecial && (handlers != nil) {
		handlers = append(handlers, func(ctx context.Context) {
			ctx.NotFound()
		})
	}

	t := h.getTree(method, subdomain)

	if t == nil {
		n := node.Nodes{}
		// first time we register a route to this method with this subdomain
		t = &tree{Method: method, Subdomain: subdomain, Nodes: &n}
		h.trees = append(h.trees, t)
	}
	return t.Nodes.Add(routeName, path, handlers, r.isSpecial)
}

// NewDefaultHandler returns the handler which is responsible
// to map the request with a route (aka mux implementation).
func NewDefaultHandler() RequestHandler {
	h := &routerHandler{}
	return h
}

// RoutesProvider should be implemented by
// iteral which contains the registered routes.
type RoutesProvider interface { // api builder
	GetRoutes() []*Route
	GetRoute(routeName string) *Route
	GetGlobalFallbackHandlers() context.Handlers
}

func (h *routerHandler) Build(provider RoutesProvider) error {
	registeredRoutes := provider.GetRoutes()
	h.trees = h.trees[0:0] // reset, inneed when rebuilding.

	fallbackHandlers := provider.GetGlobalFallbackHandlers()
	if fallbackHandlers != nil {
		h.fallbackHandlers = append(fallbackHandlers, func(ctx context.Context) {
			ctx.NotFound()
		})
	}

	// sort, subdomains goes first.
	sort.Slice(registeredRoutes, func(i, j int) bool {
		first, second := registeredRoutes[i], registeredRoutes[j]
		lsub1 := len(first.Subdomain)
		lsub2 := len(second.Subdomain)

		firstSlashLen := strings.Count(first.Path, "/")
		secondSlashLen := strings.Count(second.Path, "/")

		if lsub1 == lsub2 && first.Method == second.Method {
			if secondSlashLen < firstSlashLen {
				// fixes order when wildcard root is registered before other wildcard paths
				return true
			}
			if secondSlashLen == firstSlashLen {
				// fixes order when static path with the same prefix with a wildcard path
				// is registered after the wildcard path, although this is managed
				// by the low-level node but it couldn't work if we registered a root level wildcard, this fixes it.
				if len(first.Tmpl().Params) == 0 {
					return false
				}
				if len(second.Tmpl().Params) == 0 {
					return true
				}
			}
		}

		// the rest are handled inside the node
		return lsub1 > lsub2

	})

	rp := errors.NewReporter()

	for _, r := range registeredRoutes {
		// build the r.Handlers based on begin and done handlers, if any.
		r.BuildHandlers()

		if r.Subdomain != "" {
			h.hosts = true
		}

		// the only "bad" with this is if the user made an error
		// on route, it will be stacked shown in this build state
		// and no in the lines of the user's action, they should read
		// the docs better. Or TODO: add a link here in order to help new users.
		if err := h.addRoute(r); err != nil {
			// node errors:
			rp.Add("%v -> %s", err, r.String())
			continue
		}
		golog.Debugf(r.Trace())
	}

	return rp.Return()
}

func (h *routerHandler) HandleRequest(ctx context.Context) {
	method := ctx.Method()
	path := ctx.Path()
	if !ctx.Application().ConfigurationReadOnly().GetDisablePathCorrection() {

		if len(path) > 1 && path[len(path)-1] == '/' {
			// Remove trailing slash and client-permant rule for redirection,
			// if confgiuration allows that and path has an extra slash.

			// update the new path and redirect.
			r := ctx.Request()
			path = path[:len(path)-1]
			r.URL.Path = path
			url := r.URL.String()

			ctx.Redirect(url, http.StatusMovedPermanently)

			// RFC2616 recommends that a short note "SHOULD" be included in the
			// response because older user agents may not understand 301/307.
			// Shouldn't send the response for POST or HEAD; that leaves GET.
			if method == http.MethodGet {
				note := "<a href=\"" +
					html.EscapeString(url) +
					"\">Moved Permanently</a>.\n"

				ctx.ResponseWriter().WriteString(note)
			}
			return
		}
	}

	var fallbackHandlers context.Handlers

	for i := range h.trees {
		t := h.trees[i]

		switch t.Method {
		case method, "ANY": // Party Routes use ANY
		default:
			continue
		}

		if h.hosts && t.Subdomain != "" {
			requestHost := ctx.Host()
			if netutil.IsLoopbackSubdomain(requestHost) {
				// this fixes a bug when listening on
				// 127.0.0.1:8080 for example
				// and have a wildcard subdomain and a route registered to root domain.
				continue // it's not a subdomain, it's something like 127.0.0.1 probably
			}
			// it's a dynamic wildcard subdomain, we have just to check if ctx.subdomain is not empty
			if t.Subdomain == SubdomainWildcardIndicator {
				// mydomain.com -> invalid
				// localhost -> invalid
				// sub.mydomain.com -> valid
				// sub.localhost -> valid
				serverHost := ctx.Application().ConfigurationReadOnly().GetVHost()
				if serverHost == requestHost {
					continue // it's not a subdomain, it's a full domain (with .com...)
				}

				dotIdx := strings.IndexByte(requestHost, '.')
				slashIdx := strings.IndexByte(requestHost, '/')
				if dotIdx > 0 && (slashIdx == -1 || slashIdx > dotIdx) {
					// if "." was found anywhere but not at the first path segment (host).
				} else {
					continue
				}
				// continue to that, any subdomain is valid.
			} else if !strings.HasPrefix(requestHost, t.Subdomain) { // t.Subdomain contains the dot.
				continue
			}
		}

		routeName, handlers, special := t.Nodes.Find(path, ctx.Params())
		if special {
			if (fallbackHandlers == nil) || (t.Method != "ANY") {
				fallbackHandlers = handlers
			}

			continue
		}

		if len(handlers) > 0 {
			ctx.SetCurrentRouteName(routeName)
			ctx.Do(handlers)
			// found
			return
		}
	}

	if ctx.Application().ConfigurationReadOnly().GetFireMethodNotAllowed() {
		for i := range h.trees {
			t := h.trees[i]
			// a bit slower than previous implementation but @kataras let me to apply this change
			// because it's more reliable.
			//
			// if `Configuration#FireMethodNotAllowed` is kept as defaulted(false) then this function will not
			// run, therefore performance kept as before.
			if t.Nodes.Exists(path) {
				// RCF rfc2616 https://www.w3.org/Protocols/rfc2616/rfc2616-sec10.html
				// The response MUST include an Allow header containing a list of valid methods for the requested resource.
				ctx.Header("Allow", t.Method)
				ctx.StatusCode(http.StatusMethodNotAllowed)
				return
			}
		}
	}

	if fallbackHandlers != nil {
		ctx.Do(fallbackHandlers)
		return
	}

	if h.fallbackHandlers != nil {
		ctx.Do(h.fallbackHandlers)
		return
	}

	ctx.NotFound()
}

// RouteExists checks if a route exists
func (h *routerHandler) RouteExists(method, path string, ctx context.Context) bool {
	for i := range h.trees {
		t := h.trees[i]

		switch t.Method {
		case method, "ANY": // Party Routes use ANY
		default:
			continue
		}

		if h.hosts && t.Subdomain != "" {
			requestHost := ctx.Host()
			if netutil.IsLoopbackSubdomain(requestHost) {
				// this fixes a bug when listening on
				// 127.0.0.1:8080 for example
				// and have a wildcard subdomain and a route registered to root domain.
				continue // it's not a subdomain, it's something like 127.0.0.1 probably
			}
			// it's a dynamic wildcard subdomain, we have just to check if ctx.subdomain is not empty
			if t.Subdomain == SubdomainWildcardIndicator {
				// mydomain.com -> invalid
				// localhost -> invalid
				// sub.mydomain.com -> valid
				// sub.localhost -> valid
				serverHost := ctx.Application().ConfigurationReadOnly().GetVHost()
				if serverHost == requestHost {
					continue // it's not a subdomain, it's a full domain (with .com...)
				}

				dotIdx := strings.IndexByte(requestHost, '.')
				slashIdx := strings.IndexByte(requestHost, '/')
				if dotIdx > 0 && (slashIdx == -1 || slashIdx > dotIdx) {
					// if "." was found anywhere but not at the first path segment (host).
				} else {
					continue
				}
				// continue to that, any subdomain is valid.
			} else if !strings.HasPrefix(requestHost, t.Subdomain) { // t.Subdomain contains the dot.
				continue
			}
		}

		_, handlers, special := t.Nodes.Find(path, ctx.Params())
		if (!special) && (len(handlers) > 0) {
			// found
			return true
		}
	}

	return false
}
