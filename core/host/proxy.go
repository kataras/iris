package host

import (
	"crypto/tls"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

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
// redirects all requests to the target.
// It uses the httputil.NewSingleHostReverseProxy.
//
// Usage:
// target, _ := url.Parse("https://mydomain.com")
// proxy := NewProxy("mydomain.com:80", target)
// proxy.ListenAndServe() // use of proxy.Shutdown to close the proxy server.
func NewProxy(hostAddr string, target *url.URL) *Supervisor {
	proxyHandler := ProxyHandler(target)
	proxy := New(&http.Server{
		Addr:    hostAddr,
		Handler: proxyHandler,
	})

	return proxy
}
