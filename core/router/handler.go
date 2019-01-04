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
)

// RequestHandler the middle man between acquiring a context and releasing it.
// By-default is the router algorithm.
type RequestHandler interface {
	// HandleRequest should handle the request based on the Context.
	HandleRequest(context.Context)
	// Build should builds the handler, it's being called on router's BuildRouter.
	Build(provider RoutesProvider) error
	// RouteExists reports whether a particular route exists.
	RouteExists(ctx context.Context, method, path string) bool
}

type routerHandler struct {
	trees []*trie
	hosts bool // true if at least one route contains a Subdomain.
}

var _ RequestHandler = &routerHandler{}

func (h *routerHandler) getTree(method, subdomain string) *trie {
	for i := range h.trees {
		t := h.trees[i]
		if t.method == method && t.subdomain == subdomain {
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
		n := newTrieNode()
		// first time we register a route to this method with this subdomain
		t = &trie{method: method, subdomain: subdomain, root: n}
		h.trees = append(h.trees, t)
	}

	t.insert(path, routeName, handlers)
	return nil
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

		if len(path) > 1 && strings.HasSuffix(path, "/") {
			// Remove trailing slash and client-permanent rule for redirection,
			// if confgiuration allows that and path has an extra slash.

			// update the new path and redirect.
			r := ctx.Request()
			// use Trim to ensure there is no open redirect due to two leading slashes
			path = "/" + strings.Trim(path, "/")

			r.URL.Path = path
			if !ctx.Application().ConfigurationReadOnly().GetDisablePathCorrectionRedirection() {
				// do redirect, else continue with the modified path without the last "/".
				url := r.URL.String()

				// Fixes https://github.com/kataras/iris/issues/921
				// This is caused for security reasons, imagine a payment shop,
				// you can't just permantly redirect a POST request, so just 307 (RFC 7231, 6.4.7).
				if method == http.MethodPost || method == http.MethodPut {
					ctx.Redirect(url, http.StatusTemporaryRedirect)
					return
				}

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
	}

	for i := range h.trees {
		t := h.trees[i]
		if method != t.method {
			continue
		}

		if h.hosts && t.subdomain != "" {
			requestHost := ctx.Host()
			if netutil.IsLoopbackSubdomain(requestHost) {
				// this fixes a bug when listening on
				// 127.0.0.1:8080 for example
				// and have a wildcard subdomain and a route registered to root domain.
				continue // it's not a subdomain, it's something like 127.0.0.1 probably
			}
			// it's a dynamic wildcard subdomain, we have just to check if ctx.subdomain is not empty
			if t.subdomain == SubdomainWildcardIndicator {
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
			} else if !strings.HasPrefix(requestHost, t.subdomain) { // t.subdomain contains the dot.
				continue
			}
		}
		n := t.search(path, ctx.Params())
		if n != nil {
			ctx.SetCurrentRouteName(n.RouteName)
			ctx.Do(n.Handlers)
			// found
			return
		}
		// not found or method not allowed.
		break
	}

	if ctx.Application().ConfigurationReadOnly().GetFireMethodNotAllowed() {
		for i := range h.trees {
			t := h.trees[i]
			// if `Configuration#FireMethodNotAllowed` is kept as defaulted(false) then this function will not
			// run, therefore performance kept as before.
			if h.subdomainAndPathAndMethodExists(ctx, t, "", path) {
				// RCF rfc2616 https://www.w3.org/Protocols/rfc2616/rfc2616-sec10.html
				// The response MUST include an Allow header containing a list of valid methods for the requested resource.
				ctx.Header("Allow", t.method)
				ctx.StatusCode(http.StatusMethodNotAllowed)
				return
			}
		}
	}

	ctx.StatusCode(http.StatusNotFound)
}

func (h *routerHandler) subdomainAndPathAndMethodExists(ctx context.Context, t *trie, method, path string) bool {
	if method != "" && method != t.method {
		return false
	}

	if h.hosts && t.subdomain != "" {
		requestHost := ctx.Host()
		if netutil.IsLoopbackSubdomain(requestHost) {
			// this fixes a bug when listening on
			// 127.0.0.1:8080 for example
			// and have a wildcard subdomain and a route registered to root domain.
			return false // it's not a subdomain, it's something like 127.0.0.1 probably
		}
		// it's a dynamic wildcard subdomain, we have just to check if ctx.subdomain is not empty
		if t.subdomain == SubdomainWildcardIndicator {
			// mydomain.com -> invalid
			// localhost -> invalid
			// sub.mydomain.com -> valid
			// sub.localhost -> valid
			serverHost := ctx.Application().ConfigurationReadOnly().GetVHost()
			if serverHost == requestHost {
				return false // it's not a subdomain, it's a full domain (with .com...)
			}

			dotIdx := strings.IndexByte(requestHost, '.')
			slashIdx := strings.IndexByte(requestHost, '/')
			if dotIdx > 0 && (slashIdx == -1 || slashIdx > dotIdx) {
				// if "." was found anywhere but not at the first path segment (host).
			} else {
				return false
			}
			// continue to that, any subdomain is valid.
		} else if !strings.HasPrefix(requestHost, t.subdomain) { // t.subdomain contains the dot.
			return false
		}
	}

	n := t.search(path, ctx.Params())
	return n != nil
}

// RouteExists reports whether a particular route exists
// It will search from the current subdomain of context's host, if not inside the root domain.
func (h *routerHandler) RouteExists(ctx context.Context, method, path string) bool {
	for i := range h.trees {
		t := h.trees[i]
		if h.subdomainAndPathAndMethodExists(ctx, t, method, path) {
			return true
		}
	}

	return false
}
