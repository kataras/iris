// Package ipaccess allows or blocks requests by the client's IP address, given
// as literals, CIDR blocks or ranges.
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
// SECURITY WARNING: the client IP comes from Context.Addr, which returns the
// FIRST entry of the first configured Options.RemoteAddrHeaders header - a value
// the client writes.
//
// If your application sets RemoteAddrHeaders (the usual "behind a proxy" setup),
// you MUST terminate every request at a proxy that OVERWRITES that header.
// Otherwise any caller can send
//
//	X-Forwarded-For: 10.0.0.1
//
// and walk straight through an admin allowlist. Iris has no trusted-proxy list
// yet, so this middleware cannot verify it for you.
//
// Addr is used rather than RemoteAddr on purpose: RemoteAddr skips private
// subnets while walking the header chain, which would make it impossible to
// allowlist internal addresses - the main thing this middleware is for.
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
		// See the SECURITY WARNING on New: this value is only as trustworthy as
		// the proxy in front of the application.
		clientIP := ctx.Addr()

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
