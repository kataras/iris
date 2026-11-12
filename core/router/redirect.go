package router

import (
	"html/template"
	"strconv"
	"unicode/utf8"

	"github.com/kataras/iris/v14/context"
)

// RedirectAbsolute replies to the request with a redirect to an absolute URL.
//
// The provided code should be in the 3xx range and is usually
// StatusMovedPermanently, StatusFound or StatusSeeOther.
//
// If the Content-Type header has not been set, Redirect sets it
// to "text/html; charset=utf-8" and writes a small HTML body.
// Setting the Content-Type header to any value, including nil,
// disables that behavior.
func RedirectAbsolute(ctx *context.Context, url string, code int) {
	h := ctx.ResponseWriter().Header()
	method := ctx.Method()
	// RFC 7231 notes that a short HTML body is usually included in
	// the response because older user agents may not understand 301/307.
	// Do it only if the request didn't already have a Content-Type header.
	_, hadCT := h[context.ContentTypeHeaderKey]

	h.Set("Location", hexEscapeNonASCII(url))
	if !hadCT && (method == MethodGet || method == MethodHead) {
		h.Set(context.ContentTypeHeaderKey, "text/html; charset=utf-8")
	}
	ctx.StatusCode(code)

	// Shouldn't send the body for POST or HEAD; that leaves GET.
	if !hadCT && method == MethodGet {
		body := "<a href=\"" + template.HTMLEscapeString(url) + "\">" + context.StatusText(code) + "</a>.\n"
		ctx.WriteString(body)
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
