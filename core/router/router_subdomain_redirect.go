package router

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"text/template"
	"unicode/utf8"

	"github.com/kataras/iris/v12/context"
	"github.com/kataras/iris/v12/core/netutil"
)

type subdomainRedirectWrapper struct {
	// the func which will give us the root domain,
	// it's declared as a func because in that state the application is not configurated neither ran yet.
	root func() string
	// the from and to locations, if subdomains must end with dot('.').
	from, to string
	// true if from wildcard subdomain is given by 'from' ("*." or '*').
	isFromAny bool
	// true for the location that is the root domain ('/', '.' or "").
	isFromRoot, isToRoot bool
}

func pathIsRootDomain(partyRelPath string) bool {
	return partyRelPath == "/" || partyRelPath == "" || partyRelPath == "."
}

func pathIsWildcard(partyRelPath string) bool {
	return partyRelPath == SubdomainWildcardIndicator || partyRelPath == "*"
}

// NewSubdomainRedirectWrapper returns a router wrapper which
// if it's registered to the router via `router#WrapRouter` it
// redirects(StatusMovedPermanently) a subdomain or the root domain to another subdomain or to the root domain.
//
// It receives three arguments,
// the first one is a function which returns the root domain, (in the application it's the app.ConfigurationReadOnly().GetVHost()).
// The second and third are the from and to locations, 'from' can be a wildcard subdomain as well (*. or *)
// 'to' is not allowed to be a wildcard for obvious reasons,
// 'from' can be the root domain when the 'to' is not the root domain and visa-versa.
// To declare a root domain as 'from' or 'to' you MUST pass an empty string or a slash('/') or a dot('.').
// Important note: the 'from' and 'to' should end with "." like we use the `APIBuilder#Party`, if they are subdomains.
//
// Usage(package-level):
// sd := NewSubdomainRedirectWrapper(func() string { return "mydomain.com" }, ".", "www.")
// router.AddRouterWrapper(sd)
//
// Usage(high-level using `iris#Application.SubdomainRedirect`)
// www := app.Subdomain("www")
// app.SubdomainRedirect(app, www)
// Because app's rel path is "/" it translates it to the root domain
// and www's party's rel path is the "www.", so it's the target subdomain.
//
// All the above code snippets will register a router wrapper which will
// redirect all http(s)://mydomain.com/%anypath% to http(s)://www.mydomain.com/%anypath%.
//
// One or more subdomain redirect wrappers can be used to the same router instance.
//
// NewSubdomainRedirectWrapper may return nil if not allowed input arguments values were received
// but in that case, the `AddRouterWrapper` will, simply, ignore that wrapper.
//
// Example: https://github.com/kataras/iris/tree/main/_examples/routing/subdomains/redirect
func NewSubdomainRedirectWrapper(rootDomainGetter func() string, from, to string) WrapperFunc {
	// we can return nil,
	// because if wrapper is nil then it's not be used on the `router#AddRouterWrapper`.
	if from == to {
		// cannot redirect to the same location, cycle.
		return nil
	}

	if pathIsWildcard(to) {
		// cannot redirect to "any location".
		return nil
	}

	isFromRoot, isToRoot := pathIsRootDomain(from), pathIsRootDomain(to)
	if isFromRoot && isToRoot {
		// cannot redirect to the root domain from the root domain.
		return nil
	}

	sd := &subdomainRedirectWrapper{
		root:       rootDomainGetter,
		from:       from,
		to:         to,
		isFromAny:  pathIsWildcard(from),
		isFromRoot: isFromRoot,
		isToRoot:   isToRoot,
	}

	return sd.Wrapper
}

// Wrapper is the function that is being used to wrap the router with a redirect
// service that is able to redirect between (sub)domains as fast as possible.
// Please take a look at the `NewSubdomainRedirectWrapper` function for more.
func (s *subdomainRedirectWrapper) Wrapper(w http.ResponseWriter, r *http.Request, router http.HandlerFunc) {
	// Author's note:
	// I use the StatusMovedPermanently(301) instead of the the StatusPermanentRedirect(308)
	// because older browsers may not be able to recognise that status code (the RFC 7538, is not so old)
	// although note that move is not the same thing as redirect: move reminds a specific address or location moved while
	// redirect is a new location.
	host := context.GetHost(r)
	root := s.root()
	if loopback := netutil.GetLoopbackSubdomain(root); loopback != "" {
		root = strings.Replace(root, loopback, context.GetDomain(host), 1)
	}

	hasSubdomain := host != root
	if !hasSubdomain && !s.isFromRoot {
		// if the current endpoint is not a subdomain
		// and the redirect is not configured to be used from root domain to a subdomain.
		// This check comes first because it's the most common scenario.
		router(w, r)
		return
	}

	if hasSubdomain {
		// the current endpoint is a subdomain and
		// redirect is used for a subdomain to another subdomain or to its root domain.
		subdomain := strings.TrimSuffix(host, root) // with dot '.'.
		if s.to == subdomain {
			// we are in the subdomain we wanted to be redirected,
			// remember: a redirect response will fire a new request.
			// This check is needed to not allow cycles (too many redirects).
			router(w, r)
			return
		}

		if subdomain == s.from || s.isFromAny {
			resturi := r.URL.RequestURI()
			if s.isToRoot {
				// from a specific subdomain or any subdomain to the root domain.
				RedirectAbsolute(w, r, context.GetScheme(r)+root+resturi, http.StatusMovedPermanently)
				return
			}
			// from a specific subdomain or any subdomain to a specific subdomain.
			RedirectAbsolute(w, r, context.GetScheme(r)+s.to+root+resturi, http.StatusMovedPermanently)
			return
		}

		/* Think of another way. As it's a breaking change.
		if s.isFromRoot && !s.isFromAny {
			// Then we must not continue,
			// the subdomain didn't match the "to" but the from
			// was the application root itself, which is not a wildcard
			// so it shouldn't accept any subdomain, we must fire 404 here.
			// Something like:
			// http://registered_host_but_not_in_app.your.mydomain.com
			http.NotFound(w, r)
			return
		}
		*/

		// the from subdomain is not matched and it's not from root.
		router(w, r)
		return
	}

	if s.isFromRoot {
		resturi := r.URL.RequestURI()
		// we are not inside a subdomain, so we are in the root domain
		// and the redirect is configured to be used from root domain to a subdomain.
		RedirectAbsolute(w, r, context.GetScheme(r)+s.to+root+resturi, http.StatusMovedPermanently)
		return
	}

	router(w, r)
}

// NewSubdomainPartyRedirectHandler returns a handler which can be registered
// through `UseRouter` or `Use` to redirect from the current request's
// subdomain to the one which the given `to` Party can handle.
func NewSubdomainPartyRedirectHandler(to Party) context.Handler {
	return NewSubdomainRedirectHandler(to.GetRelPath())
}

// NewSubdomainRedirectHandler returns a handler which can be registered
// through `UseRouter` or `Use` to redirect from the current request's
// subdomain to the given "toSubdomain".
func NewSubdomainRedirectHandler(toSubdomain string) context.Handler {
	toSubdomain, _ = splitSubdomainAndPath(toSubdomain) // let it here so users can just pass the GetRelPath of a Party.
	if pathIsWildcard(toSubdomain) {
		return nil
	}

	return func(ctx *context.Context) {
		// en-us.test.mydomain.com
		host := ctx.Host()
		fullSubdomain := ctx.SubdomainFull()
		targetHost := strings.Replace(host, fullSubdomain, toSubdomain, 1)
		// resturi := ctx.Request().URL.RequestURI()
		// urlToRedirect := ctx.Scheme() + newHost + resturi
		r := ctx.Request()
		r.Host = targetHost
		r.URL.Host = targetHost
		urlToRedirect := r.URL.String()
		RedirectAbsolute(ctx.ResponseWriter(), r, urlToRedirect, http.StatusMovedPermanently)
	}
}

// RedirectAbsolute replies to the request with a redirect to an absolute URL.
//
// The provided code should be in the 3xx range and is usually
// StatusMovedPermanently, StatusFound or StatusSeeOther.
//
// If the Content-Type header has not been set, Redirect sets it
// to "text/html; charset=utf-8" and writes a small HTML body.
// Setting the Content-Type header to any value, including nil,
// disables that behavior.
func RedirectAbsolute(w http.ResponseWriter, r *http.Request, url string, code int) {
	h := w.Header()

	// RFC 7231 notes that a short HTML body is usually included in
	// the response because older user agents may not understand 301/307.
	// Do it only if the request didn't already have a Content-Type header.
	_, hadCT := h[context.ContentTypeHeaderKey]

	h.Set("Location", hexEscapeNonASCII(url))
	if !hadCT && (r.Method == http.MethodGet || r.Method == http.MethodHead) {
		h.Set(context.ContentTypeHeaderKey, "text/html; charset=utf-8")
	}
	w.WriteHeader(code)

	// Shouldn't send the body for POST or HEAD; that leaves GET.
	if !hadCT && r.Method == "GET" {
		body := "<a href=\"" + template.HTMLEscapeString(url) + "\">" + http.StatusText(code) + "</a>.\n"
		fmt.Fprintln(w, body)
	}
}

func hexEscapeNonASCII(s string) string { // part of the standard library.
	newLen := 0
	for i := 0; i < len(s); i++ {
		if s[i] >= utf8.RuneSelf {
			newLen += 3
		} else {
			newLen++
		}
	}
	if newLen == len(s) {
		return s
	}
	b := make([]byte, 0, newLen)
	for i := 0; i < len(s); i++ {
		if s[i] >= utf8.RuneSelf {
			b = append(b, '%')
			b = strconv.AppendInt(b, int64(s[i]), 16)
		} else {
			b = append(b, s[i])
		}
	}
	return string(b)
}
