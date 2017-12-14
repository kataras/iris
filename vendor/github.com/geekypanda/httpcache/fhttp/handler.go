package fhttp

import (
	"github.com/geekypanda/httpcache/entry"
	"github.com/geekypanda/httpcache/fhttp/rule"
	"github.com/valyala/fasthttp"
	"time"
)

// Handler the fasthttp cache service handler
type Handler struct {

	// bodyHandler the original route's handler
	bodyHandler fasthttp.RequestHandler

	// Rule optional validators for pre cache and post cache actions
	//
	// See more at rule.go
	rule rule.Rule

	// entry is the memory cache entry
	entry *entry.Entry
}

// NewHandler returns a new cached handler
func NewHandler(bodyHandler fasthttp.RequestHandler,
	expireDuration time.Duration) *Handler {
	e := entry.NewEntry(expireDuration)

	return &Handler{
		bodyHandler: bodyHandler,
		rule:        DefaultRuleSet,
		entry:       e,
	}
}

// Rule sets the rule for this handler,
// see internal/net/http/rule.go for more information.
//
// returns itself.
func (h *Handler) Rule(r rule.Rule) *Handler {
	if r == nil {
		// if nothing passed then use the allow-everyting rule
		r = rule.Satisfied()
	}
	h.rule = r

	return h
}

// AddRule adds a rule in the chain, the default rules are executed first.
//
// returns itself.
func (h *Handler) AddRule(r rule.Rule) *Handler {
	if r == nil {
		return h
	}

	h.rule = rule.Chained(h.rule, r)
	return h
}

func (h *Handler) ServeHTTP(reqCtx *fasthttp.RequestCtx) {

	// check for pre-cache validators, if at least one of them return false
	// for this specific request, then skip the whole cache
	if !h.rule.Claim(reqCtx) {
		h.bodyHandler(reqCtx)
		return
	}

	// check if we have a stored response( it is not expired)
	res, exists := h.entry.Response()
	if !exists {
		// if it's not valid then execute the original handler
		h.bodyHandler(reqCtx)

		// check if it's a valid response, if it's not then just return.
		if !h.rule.Valid(reqCtx) {
			return
		}

		// no need to copy the body, its already done inside
		body := reqCtx.Response.Body()
		if len(body) == 0 {
			// if no body then just exit
			return
		}

		// and re-new the entry's response with the new data
		statusCode := reqCtx.Response.StatusCode()
		contentType := string(reqCtx.Response.Header.ContentType())

		// check for an expiration time if the
		// given expiration was not valid &
		// update the response & release the recorder
		h.entry.Reset(statusCode, contentType, body, GetMaxAge(reqCtx))
		return
	}

	// if it's valid then just write the cached results
	reqCtx.SetStatusCode(res.StatusCode())
	reqCtx.SetContentType(res.ContentType())
	reqCtx.SetBody(res.Body())
}
