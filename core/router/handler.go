// Copyright 2017 Gerasimos Maropoulos, ΓΜ. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package router

import (
	"html"
	"net/http"
	"sort"
	"strings"
	"sync/atomic"

	"github.com/kataras/iris/context"
	"github.com/kataras/iris/core/nettools"
	"github.com/kataras/iris/core/router/httprouter"
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
	Entry     *httprouter.Node
}

type routerHandler struct {
	trees []*tree
	vhost atomic.Value // is a string setted at the first it founds a subdomain, we need that here in order to reduce the resolveVHost calls
	hosts bool         // true if at least one route contains a Subdomain.
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

func (h *routerHandler) addRoute(method, subdomain, path string, handlers context.Handlers) error {
	// get or create a tree and add the route
	t := h.getTree(method, subdomain)

	if t == nil {
		//first time we register a route to this method with this domain
		t = &tree{Method: method, Subdomain: subdomain, Entry: new(httprouter.Node)}
		h.trees = append(h.trees, t)
	}

	if err := t.Entry.AddRoute(path, handlers); err != nil {
		return err
	}

	return nil
}

func NewDefaultHandler() RequestHandler {
	h := &routerHandler{}
	return h
}

func (h *routerHandler) Build(provider RoutesProvider) error {
	registeredRoutes := provider.GetRoutes()
	h.trees = h.trees[0:0] // reset, inneed when rebuilding.

	// sort, subdomains goes first.
	sort.Slice(registeredRoutes, func(i, j int) bool {
		return len(registeredRoutes[i].Subdomain) >= len(registeredRoutes[j].Subdomain)
	})

	for _, r := range registeredRoutes {
		if r.Subdomain != "" {
			h.hosts = true
		}
		// the only "bad" with this is if the user made an error
		// on route, it will be stacked shown in this build state
		// and no in the lines of the user's action, they should read
		// the docs better. Or TODO: add a link here in order to help new users.
		if err := h.addRoute(r.Method, r.Subdomain, r.Path, r.Handlers); err != nil {
			return err
		}
	}
	return nil
}

func (h *routerHandler) HandleRequest(ctx context.Context) {
	method := ctx.Method()

	for i := range h.trees {
		t := h.trees[i]
		if method != t.Method {
			continue
		}

		// Changed my mind for subdomains, there are unnecessary steps here
		// most servers don't need these and on other servers may force the server to send a 404 not found
		// on a valid subdomain, by commenting my previous implementation we allow any request host to be discovarable for subdomains.
		// if h.hosts && t.Subdomain != "" {

		// 	if h.vhost.Load() == nil {
		// 		h.vhost.Store(nettools.ResolveVHost(ctx.Application().ConfigurationReadOnly().GetAddr()))
		// 	}

		// 	host := h.vhost.Load().(string)
		// 	requestHost := ctx.Host()

		// 	if requestHost != host {
		// 		// we have a subdomain
		// 		if strings.Contains(t.Subdomain, DynamicSubdomainIndicator) {
		// 		} else {
		// 			// if subdomain+host is not the request host
		// 			// and
		// 			// if request host didn't matched the server's host
		// 			// check if reached the server
		// 			// with a local address, this case is the developer him/herself,
		// 			// if both of them failed then continue and ignore this tree.
		// 			if t.Subdomain+host != requestHost && !nettools.IsLoopbackHost(requestHost) {
		// 				// go to the next tree, we have a subdomain but it is not the correct
		// 				continue
		// 			}
		// 		}
		// 	} else {
		// 		//("it's subdomain but the request is not the same as the vhost)
		// 		continue
		// 	}
		// }
		// new, simpler and without the need of known the real host:
		if h.hosts && t.Subdomain != "" {
			requestHost := ctx.Host()
			if nettools.IsLoopbackSubdomain(requestHost) {
				// this fixes a bug when listening on
				// 127.0.0.1:8080 for example
				// and have a wildcard subdomain and a route registered to root domain.
				continue // it's not a subdomain, it's something like 127.0.0.1 probably
			}
			// it's a dynamic wildcard subdomain, we have just to check if ctx.subdomain is not empty
			if t.Subdomain == DynamicSubdomainIndicator {
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

		handlers, mustRedirect := t.Entry.ResolveRoute(ctx)
		if len(handlers) > 0 {
			ctx.SetHandlers(handlers)
			ctx.Handlers()[0](ctx)
			// to remove the .Next(maybe not a good idea), reduces the performance a bit:
			// ctx.Handlers()[0](ctx) // execute the first, as soon as possible
			// // execute the chain of handlers, carefully
			// current := ctx.HandlerIndex(-1)
			// for {
			// 	if ctx.IsStopped() || current >= n {
			// 		break
			// 	}

			// 	ctx.HandlerIndex(current)
			// 	ctx.Handlers()[current](ctx)
			// 	current++
			// 	if i := ctx.HandlerIndex(-1); i > current { // navigate to previous handler is not allowed
			// 		current = i
			// 	}
			// }

			return
		} else if mustRedirect && !ctx.Application().ConfigurationReadOnly().GetDisablePathCorrection() { // && ctx.Method() == MethodConnect {
			urlToRedirect := ctx.Path()
			pathLen := len(urlToRedirect)

			if pathLen > 1 {
				if urlToRedirect[pathLen-1] == '/' {
					urlToRedirect = urlToRedirect[:pathLen-1] // remove the last /
				} else {
					// it has path prefix, it doesn't ends with / and it hasn't be found, then just append the slash
					urlToRedirect = urlToRedirect + "/"
				}

				statusForRedirect := http.StatusMovedPermanently //	StatusMovedPermanently, this document is obselte, clients caches this.
				if t.Method == http.MethodPost ||
					t.Method == http.MethodPut ||
					t.Method == http.MethodDelete {
					statusForRedirect = http.StatusTemporaryRedirect //	To maintain POST data
				}

				ctx.Redirect(urlToRedirect, statusForRedirect)
				// RFC2616 recommends that a short note "SHOULD" be included in the
				// response because older user agents may not understand 301/307.
				// Shouldn't send the response for POST or HEAD; that leaves GET.
				if t.Method == http.MethodGet {
					note := "<a href=\"" +
						html.EscapeString(urlToRedirect) +
						"\">Moved Permanently</a>.\n"

					ctx.ResponseWriter().WriteString(note)
				}
				return
			}
		}
		// not found
		break
	}

	if ctx.Application().ConfigurationReadOnly().GetFireMethodNotAllowed() {
		var methodAllowed string
		for i := range h.trees {
			t := h.trees[i]
			methodAllowed = t.Method // keep track of the allowed method of the last checked tree
			if ctx.Method() != methodAllowed {
				continue
			}
		}
		// RCF rfc2616 https://www.w3.org/Protocols/rfc2616/rfc2616-sec10.html
		// The response MUST include an Allow header containing a list of valid methods for the requested resource.
		ctx.Header("Allow", methodAllowed)
		ctx.StatusCode(http.StatusMethodNotAllowed)
		return
	}
	ctx.StatusCode(http.StatusNotFound)
}
