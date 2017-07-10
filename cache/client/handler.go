package client

import (
	"time"

	"github.com/kataras/iris/cache/cfg"
	"github.com/kataras/iris/cache/client/rule"
	"github.com/kataras/iris/cache/entry"
	"github.com/kataras/iris/context"
)

// Handler the local cache service handler contains
// the original bodyHandler, the memory cache entry and
// the validator for each of the incoming requests and post responses
type Handler struct {

	// bodyHandler the original route's handler.
	// If nil then it tries to take the next handler from the chain.
	bodyHandler context.Handler

	// Rule optional validators for pre cache and post cache actions
	//
	// See more at ruleset.go
	rule rule.Rule

	// entry is the memory cache entry
	entry *entry.Entry
}

// NewHandler returns a new cached handler for the "bodyHandler"
// which expires every "expiration".
func NewHandler(bodyHandler context.Handler,
	expiration time.Duration) *Handler {

	e := entry.NewEntry(expiration)

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
		// if nothing passed then use the allow-everything rule
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

func (h *Handler) ServeHTTP(ctx context.Context) {
	// check for pre-cache validators, if at least one of them return false
	// for this specific request, then skip the whole cache
	bodyHandler := h.bodyHandler

	if bodyHandler == nil {
		if nextHandler := ctx.NextHandler(); nextHandler != nil {
			// skip prepares the context to move to the next handler if the "nextHandler" has a ctx.Next() inside it,
			// even if it's not executed because it's cached.
			ctx.Skip()
			bodyHandler = nextHandler
		} else {
			ctx.StatusCode(500)
			ctx.WriteString("cache: empty body handler")
			ctx.StopExecution()
			return
		}
	}

	if !h.rule.Claim(ctx) {
		bodyHandler(ctx)
		return
	}

	// check if we have a stored response( it is not expired)
	res, exists := h.entry.Response()
	if !exists {

		// if it's not exists, then execute the original handler
		// with our custom response recorder response writer
		// because the net/http doesn't give us
		// a built'n way to get the status code & body
		recorder := ctx.Recorder()
		bodyHandler(ctx)

		// now that we have recordered the response,
		// we are ready to check if that specific response is valid to be stored.

		// check if it's a valid response, if it's not then just return.
		if !h.rule.Valid(ctx) {
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
		h.entry.Reset(recorder.StatusCode(), recorder.Header().Get(cfg.ContentTypeHeader), body, GetMaxAge(ctx.Request()))
		return
	}

	// if it's valid then just write the cached results
	ctx.ContentType(res.ContentType())
	ctx.StatusCode(res.StatusCode())
	ctx.Write(res.Body())
}
