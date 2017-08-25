package host

import (
	"crypto/tls"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/kataras/iris/core/netutil"
)

func singleJoiningSlash(a, b string) string {
	aslash := strings.HasSuffix(a, "/")
	bslash := strings.HasPrefix(b, "/")
	switch {
	case aslash && bslash:
		return a + b[1:]
	case !aslash && !bslash:
		return a + "/" + b
	}
	return a + b
}

// ProxyHandler returns a new ReverseProxy that rewrites
// URLs to the scheme, host, and base path provided in target. If the
// target's path is "/base" and the incoming request was for "/dir",
// the target request will be for /base/dir.
//
// Relative to httputil.NewSingleHostReverseProxy with some additions.
func ProxyHandler(target *url.URL) *httputil.ReverseProxy {
	targetQuery := target.RawQuery
	director := func(req *http.Request) {
		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host
		req.Host = target.Host
		req.URL.Path = singleJoiningSlash(target.Path, req.URL.Path)
		if targetQuery == "" || req.URL.RawQuery == "" {
			req.URL.RawQuery = targetQuery + req.URL.RawQuery
		} else {
			req.URL.RawQuery = targetQuery + "&" + req.URL.RawQuery
		}

		if _, ok := req.Header["User-Agent"]; !ok {
			// explicitly disable User-Agent so it's not set to default value
			req.Header.Set("User-Agent", "")
		}
	}
	p := &httputil.ReverseProxy{Director: director}

	if netutil.IsLoopbackHost(target.Host) {
		transport := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		p.Transport = transport
	}

	return p
}

// NewProxy returns a new host (server supervisor) which
// proxies all requests to the target.
// It uses the httputil.NewSingleHostReverseProxy.
//
// Usage:
// target, _ := url.Parse("https://mydomain.com")
// proxy := NewProxy("mydomain.com:80", target)
// proxy.ListenAndServe() // use of `proxy.Shutdown` to close the proxy server.
func NewProxy(hostAddr string, target *url.URL) *Supervisor {
	proxyHandler := ProxyHandler(target)
	proxy := New(&http.Server{
		Addr:    hostAddr,
		Handler: proxyHandler,
	})

	return proxy
}

// NewRedirection returns a new host (server supervisor) which
// redirects all requests to the target.
// Usage:
// target, _ := url.Parse("https://mydomain.com")
// r := NewRedirection(":80", target, 307)
// r.ListenAndServe() // use of `r.Shutdown` to close this server.
func NewRedirection(hostAddr string, target *url.URL, redirectStatus int) *Supervisor {
	targetURI := target.String()
	if redirectStatus <= 300 {
		// here we should use StatusPermanentRedirect but
		// that may result on unexpected behavior
		// for end-developers who might change their minds
		// after a while, so keep status temporary.
		// Note thatwe could also use StatusFound
		// as we do on the `Context#Redirect`.
		// It will also help us to prevent any post data issues.
		redirectStatus = http.StatusTemporaryRedirect
	}

	redirectSrv := &http.Server{
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
		Addr:         hostAddr,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			redirectTo := singleJoiningSlash(targetURI, r.URL.Path)
			if len(r.URL.RawQuery) > 0 {
				redirectTo += "?" + r.URL.RawQuery
			}
			http.Redirect(w, r, redirectTo, redirectStatus)
		}),
	}

	return New(redirectSrv)
}
