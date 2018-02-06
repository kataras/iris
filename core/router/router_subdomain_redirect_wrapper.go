package router

import (
	"net/http"
	"strings"

	"github.com/kataras/iris/context"
	"github.com/kataras/iris/core/netutil"
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
// router.WrapRouter(sd)
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
// but in that case, the `WrapRouter` will, simply, ignore that wrapper.
//
// Example: https://github.com/kataras/iris/tree/master/_examples/subdomains/redirect
func NewSubdomainRedirectWrapper(rootDomainGetter func() string, from, to string) WrapperFunc {
	// we can return nil,
	// because if wrapper is nil then it's not be used on the `router#WrapRouter`.
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

const sufscheme = "://"

func getFullScheme(r *http.Request) string {
	if !r.URL.IsAbs() {
		// url scheme is empty.
		return netutil.SchemeHTTP + sufscheme
	}
	return r.URL.Scheme + sufscheme
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
				http.Redirect(w, r, getFullScheme(r)+root+resturi, http.StatusMovedPermanently)
				return
			}
			// from a specific subdomain or any subdomain to a specific subdomain.
			http.Redirect(w, r, getFullScheme(r)+s.to+root+resturi, http.StatusMovedPermanently)
			return
		}

		// the from subdomain is not matched and it's not from root.
		router(w, r)
		return
	}

	if s.isFromRoot {
		resturi := r.URL.RequestURI()
		// we are not inside a subdomain, so we are in the root domain
		// and the redirect is configured to be used from root domain to a subdomain.
		http.Redirect(w, r, getFullScheme(r)+s.to+root+resturi, http.StatusMovedPermanently)
		return
	}

	router(w, r)
}
