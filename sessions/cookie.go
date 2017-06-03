// Copyright 2017 Gerasimos Maropoulos, ΓΜ. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sessions

import (
	"net/http"
	"strconv"
	"strings"
	"time"
)

var (
	// CookieExpireDelete may be set on Cookie.Expire for expiring the given cookie.
	CookieExpireDelete = time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)

	// CookieExpireUnlimited indicates that the cookie doesn't expire.
	CookieExpireUnlimited = time.Now().AddDate(24, 10, 10)
)

// GetCookie returns cookie's value by it's name
// returns empty string if nothing was found
func GetCookie(name string, req *http.Request) string {
	c, err := req.Cookie(name)
	if err != nil {
		return ""
	}
	return c.Value
}

// AddCookie adds a cookie
func AddCookie(cookie *http.Cookie, res http.ResponseWriter) {
	if v := cookie.String(); v != "" {
		http.SetCookie(res, cookie)
	}
}

// RemoveCookie deletes a cookie by it's name/key
func RemoveCookie(name string, res http.ResponseWriter, req *http.Request) {
	c, err := req.Cookie(name)
	if err != nil {
		return
	}

	c.Expires = CookieExpireDelete
	// MaxAge<0 means delete cookie now, equivalently 'Max-Age: 0'
	c.MaxAge = -1
	c.Value = ""
	c.Path = "/"
	AddCookie(c, res)
}

// IsValidCookieDomain returns true if the receiver is a valid domain to set
// valid means that is recognised as 'domain' by the browser, so it(the cookie) can be shared with subdomains also
func IsValidCookieDomain(domain string) bool {
	if domain == "0.0.0.0" || domain == "127.0.0.1" {
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
