package sessions

import (
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/kataras/iris/context"
)

var (
	// CookieExpireDelete may be set on Cookie.Expire for expiring the given cookie.
	CookieExpireDelete = time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)

	// CookieExpireUnlimited indicates that the cookie doesn't expire.
	CookieExpireUnlimited = time.Now().AddDate(24, 10, 10)
)

// GetCookie returns cookie's value by it's name
// returns empty string if nothing was found
func GetCookie(ctx context.Context, name string) string {
	c, err := ctx.Request().Cookie(name)
	if err != nil {
		return ""
	}

	return c.Value
}

// AddCookie adds a cookie
func AddCookie(ctx context.Context, cookie *http.Cookie, reclaim bool) {
	if reclaim {
		ctx.Request().AddCookie(cookie)
	}
	ctx.SetCookie(cookie)
}

// RemoveCookie deletes a cookie by it's name/key
// If "purge" is true then it removes the, temp, cookie from the request as well.
func RemoveCookie(ctx context.Context, config Config) {
	cookie, err := ctx.Request().Cookie(config.Cookie)
	if err != nil {
		return
	}

	cookie.Expires = CookieExpireDelete
	// MaxAge<0 means delete cookie now, equivalently 'Max-Age: 0'
	cookie.MaxAge = -1
	cookie.Value = ""
	cookie.Path = "/"
	cookie.Domain = formatCookieDomain(ctx, config.DisableSubdomainPersistence)

	AddCookie(ctx, cookie, config.AllowReclaim)

	if config.AllowReclaim {
		// delete request's cookie also, which is temporary available.
		ctx.Request().Header.Set("Cookie", "")
	}
}

// IsValidCookieDomain returns true if the receiver is a valid domain to set
// valid means that is recognised as 'domain' by the browser, so it(the cookie) can be shared with subdomains also
func IsValidCookieDomain(domain string) bool {
	if net.IP([]byte(domain)).IsLoopback() {
		// for these type of hosts, we can't allow subdomains persistence,
		// the web browser doesn't understand the mysubdomain.0.0.0.0 and mysubdomain.127.0.0.1 mysubdomain.32.196.56.181. as scorrectly ubdomains because of the many dots
		// so don't set a cookie domain here, let browser handle this
		return false
	}

	dotLen := strings.Count(domain, ".")
	if dotLen == 0 {
		// we don't have a domain, maybe something like 'localhost', browser doesn't see the .localhost as wildcard subdomain+domain
		return false
	}
	if dotLen >= 3 {
		if lastDotIdx := strings.LastIndexByte(domain, '.'); lastDotIdx != -1 {
			// chekc the last part, if it's number then propably it's ip
			if len(domain) > lastDotIdx+1 {
				_, err := strconv.Atoi(domain[lastDotIdx+1:])
				if err == nil {
					return false
				}
			}
		}
	}

	return true
}

func formatCookieDomain(ctx context.Context, disableSubdomainPersistence bool) string {
	if disableSubdomainPersistence {
		return ""
	}

	requestDomain := ctx.Host()
	if portIdx := strings.IndexByte(requestDomain, ':'); portIdx > 0 {
		requestDomain = requestDomain[0:portIdx]
	}

	if !IsValidCookieDomain(requestDomain) {
		return ""
	}

	// RFC2109, we allow level 1 subdomains, but no further
	// if we have localhost.com , we want the localhost.com.
	// so if we have something like: mysubdomain.localhost.com we want the localhost here
	// if we have mysubsubdomain.mysubdomain.localhost.com we want the .mysubdomain.localhost.com here
	// slow things here, especially the 'replace' but this is a good and understable( I hope) way to get the be able to set cookies from subdomains & domain with 1-level limit
	if dotIdx := strings.IndexByte(requestDomain, '.'); dotIdx > 0 {
		// is mysubdomain.localhost.com || mysubsubdomain.mysubdomain.localhost.com
		if strings.IndexByte(requestDomain[dotIdx+1:], '.') > 0 {
			requestDomain = requestDomain[dotIdx+1:]
		}
	}
	// finally set the .localhost.com (for(1-level) || .mysubdomain.localhost.com (for 2-level subdomain allow)
	return "." + requestDomain // . to allow persistence
}
