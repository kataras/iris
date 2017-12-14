package nethttp

import (
	"net/http"
	"time"

	"github.com/geekypanda/httpcache/cfg"
	"github.com/geekypanda/httpcache/entry"
	"github.com/geekypanda/httpcache/nethttp/rule"
)

// Handler the local cache service handler contains
// the original bodyHandler, the memory cache entry and
// the validator for each of the incoming requests and post responses
type Handler struct {

	// bodyHandler the original route's handler
	bodyHandler http.Handler

	// Rule optional validators for pre cache and post cache actions
	//
	// See more at ruleset.go
	rule rule.Rule

	// entry is the memory cache entry
	entry *entry.Entry
}

// NewHandler returns a new cached handler
func NewHandler(bodyHandler http.Handler,
	expireDuration time.Duration) *Handler {

	e := entry.NewEntry(expireDuration)

	return &Handler{
		bodyHandler: bodyHandler,
		rule:        DefaultRuleSet,
		entry:       e,
	}
}

// Rule sets the ruleset for this handler.
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

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// check for pre-cache validators, if at least one of them return false
	// for this specific request, then skip the whole cache
	if !h.rule.Claim(r) {
		h.bodyHandler.ServeHTTP(w, r)
		return
	}

	// check if we have a stored response( it is not expired)
	res, exists := h.entry.Response()
	if !exists {
		// if it's not exists, then execute the original handler
		// with our custom response recorder response writer
		// because the net/http doesn't give us
		// a built'n way to get the status code & body
		recorder := AcquireResponseRecorder(w)
		defer ReleaseResponseRecorder(recorder)
		h.bodyHandler.ServeHTTP(recorder, r)

		// now that we have recordered the response,
		// we are ready to check if that specific response is valid to be stored.

		// check if it's a valid response, if it's not then just return.
		if !h.rule.Valid(recorder, r) {
			return
		}

		// no need to copy the body, its already done inside
		body := recorder.Body()
		if len(body) == 0 {
			// if no body then just exit
			return
		}

		// check for an expiration time if the
		// given expiration was not valid then check for GetMaxAge &
		// update the response & release the recorder
		h.entry.Reset(recorder.StatusCode(), recorder.ContentType(), body, GetMaxAge(r))
		return
	}

	// if it's valid then just write the cached results
	w.Header().Set(cfg.ContentTypeHeader, res.ContentType())
	w.WriteHeader(res.StatusCode())
	w.Write(res.Body())
}
