package router

import (
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
}

type tree struct {
	Method string
	// subdomain is empty for default-hostname routes,
	// ex: mysubdomain.
	Subdomain string
	Nodes     *node.Nodes
}

type routerHandler struct {
	trees []*tree
	hosts bool // true if at least one route contains a Subdomain.
}

var _ RequestHandler = &routerHandler{}

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

	t := h.getTree(method, subdomain)

	if t == nil {
		n := node.Nodes{}
		// first time we register a route to this method with this subdomain
		t = &tree{Method: method, Subdomain: subdomain, Nodes: &n}
		h.trees = append(h.trees, t)
	}
	return t.Nodes.Add(routeName, path, handlers)
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
}

func (h *routerHandler) Build(provider RoutesProvider) error {
	registeredRoutes := provider.GetRoutes()
	h.trees = h.trees[0:0] // reset, inneed when rebuilding.

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

	for i := range h.trees {
		t := h.trees[i]
		if method != t.Method {
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
		routeName, handlers := t.Nodes.Find(path, ctx.Params())
		if len(handlers) > 0 {
			ctx.SetCurrentRouteName(routeName)
			ctx.Do(handlers)
			// found
			return
		}
		// not found or method not allowed.
		break
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

	ctx.StatusCode(http.StatusNotFound)
}
