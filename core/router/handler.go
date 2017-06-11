// Copyright 2017 Gerasimos Maropoulos, ΓΜ. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package router

import (
	"fmt"
	"html"
	"net/http"
	"sort"
	"strings"

	"github.com/kataras/iris/context"
	"github.com/kataras/iris/core/nettools"
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

func (h *routerHandler) addRoute(method, subdomain, path string, handlers context.Handlers) error {
	if len(path) == 0 || path[0] != '/' {
		return fmt.Errorf("router: path %q must begin with %q", path, "/")
	}

	t := h.getTree(method, subdomain)

	if t == nil {
		n := make(node.Nodes, 0)
		// first time we register a route to this method with this subdomain
		t = &tree{Method: method, Subdomain: subdomain, Nodes: &n}
		h.trees = append(h.trees, t)
	}

	return t.Nodes.Add(path, handlers)
}

// NewDefaultHandler returns the handler which is responsible
// to map the request with a route (aka mux implementation).
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

		handlers := t.Nodes.Find(path, ctx.Params())
		if len(handlers) > 0 {
			ctx.Do(handlers)
			// found
			return
		}
		// not found or method not allowed.
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
