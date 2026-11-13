// Package ipaccess allows or blocks requests by the client's IP address, given
// as literals, CIDR blocks or ranges.
//
// The address is the one Context.RemoteAddr returns: the address of the
// connection or, when Options.Context.RemoteAddrHeaders is set, the one a
// proxy wrote in such a header. See the SECURITY WARNING on New.
//
// Its entry point is New(allowed, blocked []string) (Handler, error), which
// reports a malformed pattern; Must(allowed, blocked) is the panicking
// companion.
package ipaccess

import (
	"fmt"
	"net"
	"net/http"
	"regexp"
	"strings"

	"github.com/kataras/iris/v14/context"
)

// ipPattern represents a pattern for matching IP addresses.
type ipPattern struct {
	exact net.IP
	cidr  *net.IPNet
	regex *regexp.Regexp
}

// newIPPattern creates a new IPPattern from a string pattern.
//
// A pattern is one of:
//
//   - an exact IP address, e.g. "10.0.0.7" or "::1",
//   - a CIDR block, e.g. "10.0.0.0/8",
//   - a regular expression, e.g. `^10\.0\.0\.\d{1,3}$`, optionally wrapped in
//     slashes.
//
// Note(@kataras): the regexp form is anchored AT THE START.
//
// It used to be compiled as-is and matched with MatchString, which succeeds on a
// substring anywhere in the address. The natural-looking allow pattern
// "10.0.0.*" is neither an IP nor a CIDR, so it fell through to the regexp
// branch, where '.' means "any character" and the match may start anywhere - it
// accepted "110.0.0.99", "210.0.0.5" and any public address whose text merely
// contains the pattern. An operator writing the most obvious thing got an
// allowlist that let strangers in, silently.
//
// Only the start is anchored, deliberately. A prefix pattern such as
// `^198\.51\.100\.` - the documented way to cover a /24 in a blocklist - is
// meant to stay open at the end, and anchoring both ends would silently stop it
// matching anything at all. Anchor the end yourself with '$' when you want an
// exact match. The leading anchor is what closes the bypass, because the input
// is always a normalized address.
func newIPPattern(pattern string) (*ipPattern, error) {
	if ip := net.ParseIP(pattern); ip != nil {
		return &ipPattern{exact: ip}, nil
	}

	if _, cidr, err := net.ParseCIDR(pattern); err == nil {
		return &ipPattern{cidr: cidr}, nil
	}

	// Accept an optional /.../ wrapping so a regexp can be written unambiguously.
	expr := pattern
	if inner, ok := strings.CutPrefix(expr, "/"); ok {
		if inner, ok := strings.CutSuffix(inner, "/"); ok {
			expr = inner
		}
	}

	regex, err := regexp.Compile(`\A(?:` + expr + `)`)
	if err != nil {
		return nil, fmt.Errorf("invalid IP pattern: %s: expected an IP address, a CIDR block or a regular expression: %w", pattern, err)
	}

	return &ipPattern{regex: regex}, nil
}

// match checks if the given IP matches the pattern.
func (p *ipPattern) match(ip string) bool {
	if p.exact != nil {
		// Compare parsed addresses, not text: "::1" and "0:0:0:0:0:0:0:1" are the
		// same address written two ways, and an IPv4-mapped IPv6 address is the
		// same as its IPv4 form.
		return p.exact.Equal(net.ParseIP(ip))
	}

	if p.cidr != nil {
		parsed := net.ParseIP(ip)
		return parsed != nil && p.cidr.Contains(parsed)
	}

	if p.regex != nil {
		return p.regex.MatchString(ip)
	}

	return false
}

// New returns a middleware that allows or blocks requests by client IP address.
//
// An empty "allowedPatterns" means "allow everything that is not blocked"; a
// non-empty one means "allow only these".
//
// The client IP comes from Context.RemoteAddr. With no
// Options.Context.RemoteAddrHeaders it is the address of the connection. With
// one, the header is read from the right, where a proxy appends the address it
// got the request from: the first public address is the client IP, and when no
// header gives one it is the address of the connection.
//
// SECURITY WARNING: Iris has no trusted-proxy list yet, so with
// RemoteAddrHeaders set the client IP is only as good as the proxy in front of
// the application:
//
//   - A client that reaches the server directly, with no proxy in front,
//     writes the whole header and with it the client IP. Such a server MUST be
//     reachable through the proxy only.
//   - A header that the proxy passes on as the client sent it is the
//     client's. Name only headers the proxy writes: a header is read when the
//     ones named before it give no public address.
//   - A client inside your network is appended as a private address, which is
//     skipped as one of your own proxies: the entry on its left is read, and
//     the client wrote that one. A proxy that OVERWRITES the header closes
//     this.
//   - A private address in the header is never the client IP, so a pattern
//     such as "10.0.0.0/8" does not match a client that an internal proxy
//     reports as 10.0.0.5: the address of the connection, the proxy, is
//     matched. RemoteAddrHeadersForce takes the first address of a header that
//     has no public one, which the client can write, and an empty
//     RemoteAddrPrivateSubnets takes the right-most one, which the nearest
//     proxy appended. A loopback address is skipped whatever that list holds,
//     so only RemoteAddrHeadersForce takes it.
//   - A proxy with a public address in front of your own one, a CDN for
//     example, is appended as a public address: it is the client IP, and the
//     client it got the request from is not read. Add its range to
//     RemoteAddrPrivateSubnets to have it skipped. That list is checked for
//     IPv4 addresses only.
//
// The client IP used to come from Context.Addr, which returns the FIRST entry
// of the header, so sending the header was enough to pass an allowlist or to
// escape a blocklist.
func New(allowedPatterns []string, blockedPatterns []string) (context.Handler, error) {
	allowedIPPatterns := make([]*ipPattern, len(allowedPatterns))
	for i, pattern := range allowedPatterns {
		ipPattern, err := newIPPattern(pattern)
		if err != nil {
			return nil, err
		}
		allowedIPPatterns[i] = ipPattern
	}

	blockedIPPatterns := make([]*ipPattern, len(blockedPatterns))
	for i, pattern := range blockedPatterns {
		ipPattern, err := newIPPattern(pattern)
		if err != nil {
			return nil, err
		}
		blockedIPPatterns[i] = ipPattern
	}

	handler := func(ctx *context.Context) {
		// See the SECURITY WARNING on New: with RemoteAddrHeaders set this
		// value is only as trustworthy as the proxy in front of the
		// application.
		clientIP := ctx.RemoteAddr()
		// An address taken from a header comes as it is spelled there, such
		// as "::ffff:198.51.100.9". The regular expressions are written for
		// the one spelling net.IP.String gives, see newIPPattern.
		if ip := net.ParseIP(clientIP); ip != nil {
			clientIP = ip.String()
		}

		for _, pattern := range blockedIPPatterns {
			if pattern.match(clientIP) {
				ctx.StopWithStatus(http.StatusForbidden)
				return
			}
		}

		// An empty allowlist means "no allowlist", not "deny everything".
		// New(nil, blocked) is the natural way to write a blocklist-only policy,
		// and it used to fall through and 403 every single request.
		if len(allowedIPPatterns) == 0 {
			ctx.Next()
			return
		}

		for _, pattern := range allowedIPPatterns {
			if pattern.match(clientIP) {
				ctx.Next()
				return
			}
		}

		ctx.StopWithStatus(http.StatusForbidden)
	}
	return handler, nil
}

// Must is New for a caller that has nothing to do with a malformed pattern: it
// panics instead of reporting one, so a registration stays a single expression.
//
//	app.Use(ipaccess.Must([]string{"127.0.0.1", "10.0.0.0/8"}, nil))
//
// The patterns are usually literals or configuration read at startup, and a bad
// one there is a broken deployment rather than a request-time failure. Use New
// when they come from somewhere that can be wrong at run time.
func Must(allowedPatterns []string, blockedPatterns []string) context.Handler {
	handler, err := New(allowedPatterns, blockedPatterns)
	if err != nil {
		panic(err)
	}

	return handler
}
