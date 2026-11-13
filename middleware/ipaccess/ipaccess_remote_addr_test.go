package ipaccess_test

import (
	"net/http"
	nethttptest "net/http/httptest"
	"slices"
	"testing"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/core/pkg/netutil"
	"github.com/kataras/iris/v14/middleware/ipaccess"
)

// The ways the tests of this file configure where the client's address comes
// from.
var (
	// no header is read.
	connectionOnly = func(*iris.Options) {}
	// X-Forwarded-For is read.
	forwardedFor = func(opts *iris.Options) {
		opts.Context.RemoteAddrHeaders = []string{"X-Forwarded-For"}
	}
	// and its first address is taken when it has no public one.
	forwardedForForce = func(opts *iris.Options) {
		forwardedFor(opts)
		opts.Context.RemoteAddrHeadersForce = true
	}
	// and no range is a private one.
	forwardedForNoPrivateSubnets = func(opts *iris.Options) {
		forwardedFor(opts)
		opts.Context.RemoteAddrPrivateSubnets = []netutil.IPRange{}
	}
)

// status answers one request through a new application that runs the
// middleware with the given patterns, and returns the status code. The
// connection address of a net/http/httptest request is 192.0.2.1:1234, which
// stands for the proxy when a header is read.
func status(t *testing.T, options func(*iris.Options), allowed, blocked []string, forwardedFor string) int {
	t.Helper()

	app := iris.New(iris.Defaults(options))
	app.Use(ipaccess.Must(allowed, blocked))
	app.Get("/", func(ctx iris.Context) { ctx.WriteString("ok") })
	if err := app.Build(); err != nil {
		t.Fatal(err)
	}

	req := nethttptest.NewRequest(http.MethodGet, "/", nil)
	if forwardedFor != "" {
		req.Header.Set("X-Forwarded-For", forwardedFor)
	}
	rec := nethttptest.NewRecorder()
	app.ServeHTTP(rec, req)
	return rec.Code
}

// With RemoteAddrHeaders set the middleware decides on Context.RemoteAddr,
// which reads the header from the right, where the proxy appends the address
// it got the request from. It used to decide on Context.Addr, which reads it
// from the left, where the client writes.
func TestDecidesOnTheEntryTheProxyAppended(t *testing.T) {
	for _, tt := range []struct {
		name         string
		allowed      []string
		blocked      []string
		forwardedFor string
		want         int
	}{
		// the client wrote an allowed address and the proxy appended the
		// real one. It was allowed.
		{"allowlist, written address", []string{"203.0.113.0/24"}, nil, "203.0.113.7, 198.51.100.9", http.StatusForbidden},
		// the client is blocked and wrote another address in front of the
		// one the proxy appended. It passed.
		{"blocklist, written address", nil, []string{"198.51.100.9"}, "1.1.1.1, 198.51.100.9", http.StatusForbidden},
		// an entry that is not an address ends the reading, so the entry on
		// its left is never read: the connection decides. It was allowed.
		{"allowlist, not an address on the right", []string{"203.0.113.0/24"}, nil, "203.0.113.7, unknown", http.StatusForbidden},
		// the client wrote nothing, the proxy sent its address. Same as before.
		{"allowlist, one entry", []string{"198.51.100.0/24"}, nil, "198.51.100.9", http.StatusOK},
		{"allowlist, one entry outside of it", []string{"203.0.113.0/24"}, nil, "198.51.100.9", http.StatusForbidden},
		{"allowlist, one entry with a port", []string{"198.51.100.0/24"}, nil, "198.51.100.9:4321", http.StatusOK},
		{"blocklist, one entry", nil, []string{"198.51.100.9"}, "198.51.100.9", http.StatusForbidden},
		{"blocklist, another entry", nil, []string{"198.51.100.9"}, "198.51.100.10", http.StatusOK},
		// no header: the connection. Same as before.
		{"allowlist, no header", []string{"192.0.2.0/24"}, nil, "", http.StatusOK},
		{"blocklist, no header", nil, []string{"192.0.2.1"}, "", http.StatusForbidden},
	} {
		if got := status(t, forwardedFor, tt.allowed, tt.blocked, tt.forwardedFor); got != tt.want {
			t.Errorf("%s: X-Forwarded-For %q, allowed %v, blocked %v: status %d, want %d",
				tt.name, tt.forwardedFor, tt.allowed, tt.blocked, got, tt.want)
		}
	}
}

// A private address in the header is one of our own proxies for
// Context.RemoteAddr, which skips it. A proxy that sends the private address
// of its client is therefore not enough to match a pattern for that address:
// the connection, which is the proxy, decides, unless RemoteAddrHeadersForce
// is set or the address is taken out of RemoteAddrPrivateSubnets.
func TestPrivateAddressInTheHeader(t *testing.T) {
	for _, tt := range []struct {
		name         string
		options      func(*iris.Options)
		allowed      []string
		blocked      []string
		forwardedFor string
		want         int
	}{
		// it was allowed, through Context.Addr.
		{"allowlist", forwardedFor, []string{"10.0.0.0/8"}, nil, "10.0.0.5", http.StatusForbidden},
		{"allowlist, force", forwardedForForce, []string{"10.0.0.0/8"}, nil, "10.0.0.5", http.StatusOK},
		{"allowlist, no private subnets", forwardedForNoPrivateSubnets, []string{"10.0.0.0/8"}, nil, "10.0.0.5", http.StatusOK},
		// it was blocked, through Context.Addr.
		{"blocklist", forwardedFor, nil, []string{"10.0.0.5"}, "10.0.0.5", http.StatusOK},
		{"blocklist, force", forwardedForForce, nil, []string{"10.0.0.5"}, "10.0.0.5", http.StatusForbidden},
		{"blocklist, no private subnets", forwardedForNoPrivateSubnets, nil, []string{"10.0.0.5"}, "10.0.0.5", http.StatusForbidden},
		// the connection decides, so a pattern for the proxy matches.
		{"allowlist with the proxy", forwardedFor, []string{"10.0.0.0/8", "192.0.2.1"}, nil, "10.0.0.5", http.StatusOK},
		// neither option brings back the entry the client wrote when the
		// proxy appended a public address. It was allowed.
		{"allowlist, force, written address", forwardedForForce, []string{"10.0.0.0/8"}, nil, "10.0.0.5, 198.51.100.9", http.StatusForbidden},
		{"allowlist, no private subnets, written address", forwardedForNoPrivateSubnets, []string{"10.0.0.0/8"}, nil, "10.0.0.5, 198.51.100.9", http.StatusForbidden},
		// the proxy appended a private address, 10.0.0.66, to the one the
		// client wrote, 10.0.0.1: the option takes the first and an empty
		// RemoteAddrPrivateSubnets the last.
		{"blocklist, force, written private address", forwardedForForce, nil, []string{"10.0.0.66"}, "10.0.0.1, 10.0.0.66", http.StatusOK},
		{"blocklist, no private subnets, written private address", forwardedForNoPrivateSubnets, nil, []string{"10.0.0.66"}, "10.0.0.1, 10.0.0.66", http.StatusForbidden},
	} {
		if got := status(t, tt.options, tt.allowed, tt.blocked, tt.forwardedFor); got != tt.want {
			t.Errorf("%s: X-Forwarded-For %q, allowed %v, blocked %v: status %d, want %d",
				tt.name, tt.forwardedFor, tt.allowed, tt.blocked, got, tt.want)
		}
	}
}

// A proxy with a public address in front of our own one, a CDN for example,
// is the first public address from the right. Context.RemoteAddr gives that
// one, not the client the CDN got the request from, until the range of the
// CDN is one of RemoteAddrPrivateSubnets. Context.Addr gave the client.
func TestPublicProxyInFront(t *testing.T) {
	// 198.51.100.9 is the client and 203.0.113.50 the CDN, which our proxy
	// appended.
	const chain = "198.51.100.9, 203.0.113.50"

	forwardedForSkipCDN := func(opts *iris.Options) {
		forwardedFor(opts)
		opts.Context.RemoteAddrPrivateSubnets = append(slices.Clone(opts.Context.RemoteAddrPrivateSubnets),
			netutil.IPRange{Start: "203.0.113.0", End: "203.0.113.255"})
	}

	for _, tt := range []struct {
		name         string
		options      func(*iris.Options)
		allowed      []string
		blocked      []string
		forwardedFor string
		want         int
	}{
		// it was allowed, through Context.Addr.
		{"allowlist", forwardedFor, []string{"198.51.100.0/24"}, nil, chain, http.StatusForbidden},
		// it was blocked, through Context.Addr.
		{"blocklist", forwardedFor, nil, []string{"198.51.100.9"}, chain, http.StatusOK},
		// the CDN decides, so a pattern for it matches.
		{"allowlist with the CDN", forwardedFor, []string{"203.0.113.50"}, nil, chain, http.StatusOK},
		// the CDN is skipped as one of our own proxies. Same as before.
		{"allowlist, CDN skipped", forwardedForSkipCDN, []string{"198.51.100.0/24"}, nil, chain, http.StatusOK},
		{"blocklist, CDN skipped", forwardedForSkipCDN, nil, []string{"198.51.100.9"}, chain, http.StatusForbidden},
		// and the entry the client wrote in front is not read. It passed.
		{"blocklist, CDN skipped, written address", forwardedForSkipCDN, nil, []string{"198.51.100.9"}, "198.51.100.77, " + chain, http.StatusForbidden},
	} {
		if got := status(t, tt.options, tt.allowed, tt.blocked, tt.forwardedFor); got != tt.want {
			t.Errorf("%s: X-Forwarded-For %q, allowed %v, blocked %v: status %d, want %d",
				tt.name, tt.forwardedFor, tt.allowed, tt.blocked, got, tt.want)
		}
	}
}

// Every header named in RemoteAddrHeaders is read as one a proxy wrote. One
// that the proxy passes on as the client sent it, X-Real-Ip here, is the
// client's. This pins what Context.RemoteAddr does with it, as the SECURITY
// WARNING on New says: such a header decides when it is named first and, named
// second, when X-Forwarded-For gives no public address.
func TestHeaderTheProxyDoesNotWrite(t *testing.T) {
	for _, tt := range []struct {
		name         string
		headers      []string
		forwardedFor string // what the proxy sent.
		want         int
	}{
		// named first, it is read first. Same as before.
		{"first", []string{"X-Real-Ip", "X-Forwarded-For"}, "198.51.100.9", http.StatusOK},
		// named second, the public address the proxy sent decides. Same as before.
		{"second, public address", []string{"X-Forwarded-For", "X-Real-Ip"}, "198.51.100.9", http.StatusForbidden},
		// named second, the private address the proxy sent is skipped and
		// the next header is read. Context.Addr stopped at the private
		// address: it was 403.
		{"second, private address", []string{"X-Forwarded-For", "X-Real-Ip"}, "10.0.0.66", http.StatusOK},
	} {
		app := iris.New(iris.Defaults(func(opts *iris.Options) {
			opts.Context.RemoteAddrHeaders = tt.headers
		}))
		app.Use(ipaccess.Must([]string{"203.0.113.0/24"}, nil))
		app.Get("/", func(ctx iris.Context) { ctx.WriteString("ok") })
		if err := app.Build(); err != nil {
			t.Fatal(err)
		}

		req := nethttptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("X-Forwarded-For", tt.forwardedFor)
		req.Header.Set("X-Real-Ip", "203.0.113.7") // the client wrote it.
		rec := nethttptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if got := rec.Code; got != tt.want {
			t.Errorf("%s: RemoteAddrHeaders %v, X-Forwarded-For %q, X-Real-Ip \"203.0.113.7\", allowed [203.0.113.0/24]: status %d, want %d",
				tt.name, tt.headers, tt.forwardedFor, got, tt.want)
		}
	}
}

// Context.RemoteAddr gives an address taken from a header as it is spelled
// there. A regular expression is matched against text, so the middleware
// brings the address to its one canonical spelling first, the one
// Context.Addr gave.
func TestSpellingOfAnAddressInTheHeader(t *testing.T) {
	for _, tt := range []struct {
		name         string
		allowed      []string
		blocked      []string
		forwardedFor string
		want         int
	}{
		{"blocklist regexp, IPv4-mapped IPv6", nil, []string{`^198\.51\.100\.`}, "::ffff:198.51.100.9", http.StatusForbidden},
		{"blocklist regexp, IPv4-mapped IPv6 with a port", nil, []string{`^198\.51\.100\.`}, "[::ffff:198.51.100.9]:4321", http.StatusForbidden},
		{"blocklist regexp, IPv6 in full", nil, []string{`^2001:db8::`}, "2001:0DB8:0000:0000:0000:0000:0000:0001", http.StatusForbidden},
		{"allowlist regexp, IPv6 in full", []string{`^2001:db8::\d$`}, nil, "2001:0DB8:0000:0000:0000:0000:0000:0001", http.StatusOK},
		// an exact address and a CIDR block compare parsed addresses.
		{"blocklist exact, IPv4-mapped IPv6", nil, []string{"198.51.100.9"}, "::ffff:198.51.100.9", http.StatusForbidden},
		{"blocklist CIDR, IPv4-mapped IPv6", nil, []string{"198.51.100.0/24"}, "::ffff:198.51.100.9", http.StatusForbidden},
		{"allowlist exact, IPv6 in full", []string{"2001:db8::1"}, nil, "2001:0DB8:0000:0000:0000:0000:0000:0001", http.StatusOK},
	} {
		if got := status(t, forwardedFor, tt.allowed, tt.blocked, tt.forwardedFor); got != tt.want {
			t.Errorf("%s: X-Forwarded-For %q, allowed %v, blocked %v: status %d, want %d",
				tt.name, tt.forwardedFor, tt.allowed, tt.blocked, got, tt.want)
		}
	}
}

// Without RemoteAddrHeaders no header is read: the connection decides,
// whatever the client sends.
func TestConnectionAddressWithoutRemoteAddrHeaders(t *testing.T) {
	for _, tt := range []struct {
		name         string
		allowed      []string
		blocked      []string
		forwardedFor string
		want         int
	}{
		{"allowlist, connection in it", []string{"192.0.2.0/24"}, nil, "", http.StatusOK},
		{"allowlist, connection in it, header outside of it", []string{"192.0.2.0/24"}, nil, "203.0.113.7", http.StatusOK},
		{"allowlist, header in it", []string{"203.0.113.0/24"}, nil, "203.0.113.7", http.StatusForbidden},
		{"blocklist, connection in it", nil, []string{"192.0.2.1"}, "", http.StatusForbidden},
		{"blocklist, connection in it, another header", nil, []string{"192.0.2.1"}, "1.1.1.1", http.StatusForbidden},
		{"blocklist, header in it", nil, []string{"1.1.1.1"}, "1.1.1.1", http.StatusOK},
	} {
		if got := status(t, connectionOnly, tt.allowed, tt.blocked, tt.forwardedFor); got != tt.want {
			t.Errorf("%s: X-Forwarded-For %q, allowed %v, blocked %v: status %d, want %d",
				tt.name, tt.forwardedFor, tt.allowed, tt.blocked, got, tt.want)
		}
	}
}
