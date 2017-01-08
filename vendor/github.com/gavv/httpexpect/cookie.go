package httpexpect

import (
	"net/http"
	"time"
)

// Cookie provides methods to inspect attached http.Cookie value.
type Cookie struct {
	chain chain
	value *http.Cookie
}

// NewCookie returns a new Cookie object given a reporter used to report
// failures and cookie value to be inspected.
//
// reporter and value should not be nil.
//
// Example:
//   cookie := NewCookie(reporter, &http.Cookie{...})
//   cookie.Domain().Equal("example.com")
//   cookie.Path().Equal("/")
//   cookie.Expires().InRange(time.Now(), time.Now().Add(time.Hour * 24))
func NewCookie(reporter Reporter, value *http.Cookie) *Cookie {
	chain := makeChain(reporter)
	if value == nil {
		chain.fail("expected non-nil cookie")
	}
	return &Cookie{chain, value}
}

// Raw returns underlying http.Cookie value attached to Cookie.
// This is the value originally passed to NewCookie.
//
// Example:
//  cookie := NewCookie(t, c)
//  assert.Equal(t, c, cookie.Raw())
func (c *Cookie) Raw() *http.Cookie {
	return c.value
}

// Name returns a new String object that may be used to inspect
// cookie name.
//
// Example:
//  cookie := NewCookie(t, &http.Cookie{...})
//  cookie.Name().Equal("session")
func (c *Cookie) Name() *String {
	if c.chain.failed() {
		return &String{c.chain, ""}
	}
	return &String{c.chain, c.value.Name}
}

// Value returns a new String object that may be used to inspect
// cookie value.
//
// Example:
//  cookie := NewCookie(t, &http.Cookie{...})
//  cookie.Value().Equal("gH6z7Y")
func (c *Cookie) Value() *String {
	if c.chain.failed() {
		return &String{c.chain, ""}
	}
	return &String{c.chain, c.value.Value}
}

// Domain returns a new String object that may be used to inspect
// cookie domain.
//
// Example:
//  cookie := NewCookie(t, &http.Cookie{...})
//  cookie.Domain().Equal("example.com")
func (c *Cookie) Domain() *String {
	if c.chain.failed() {
		return &String{c.chain, ""}
	}
	return &String{c.chain, c.value.Domain}
}

// Path returns a new String object that may be used to inspect
// cookie path.
//
// Example:
//  cookie := NewCookie(t, &http.Cookie{...})
//  cookie.Path().Equal("/foo")
func (c *Cookie) Path() *String {
	if c.chain.failed() {
		return &String{c.chain, ""}
	}
	return &String{c.chain, c.value.Path}
}

// Expires returns a new DateTime object that may be used to inspect
// cookie expiration date.
//
// Example:
//  cookie := NewCookie(t, &http.Cookie{...})
//  cookie.Expires().InRange(time.Now(), time.Now().Add(time.Hour * 24))
func (c *Cookie) Expires() *DateTime {
	if c.chain.failed() {
		return &DateTime{c.chain, time.Unix(0, 0)}
	}
	return &DateTime{c.chain, c.value.Expires}
}
