package client

import (
	"sync"
	"time"

	"github.com/kataras/iris/cache/client/rule"
	"github.com/kataras/iris/cache/entry"
	"github.com/kataras/iris/context"
)

// Handler the local cache service handler contains
// the original response, the memory cache entry and
// the validator for each of the incoming requests and post responses
type Handler struct {
	// Rule optional validators for pre cache and post cache actions
	//
	// See more at ruleset.go
	rule rule.Rule
	// when expires.
	expiration time.Duration
	// entries the memory cache stored responses.
	entries map[string]*entry.Entry
	mu      sync.RWMutex
}

// NewHandler returns a new cached handler for the "bodyHandler"
// which expires every "expiration".
func NewHandler(expiration time.Duration) *Handler {
	return &Handler{
		rule:       DefaultRuleSet,
		expiration: expiration,
		entries:    make(map[string]*entry.Entry, 0),
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

var emptyHandler = func(ctx context.Context) {
	ctx.StatusCode(500)
	ctx.WriteString("cache: empty body handler")
	ctx.StopExecution()
}

func parseLifeChanger(ctx context.Context) entry.LifeChanger {
	return func() time.Duration {
		return time.Duration(ctx.MaxAge()) * time.Second
	}
}

///TODO: debug this and re-run the parallel tests on larger scale,
// because I think we have a bug here when `core/router#StaticWeb` is used after this middleware.
func (h *Handler) ServeHTTP(ctx context.Context) {
	// check for pre-cache validators, if at least one of them return false
	// for this specific request, then skip the whole cache
	bodyHandler := ctx.NextHandler()
	if bodyHandler == nil {
		emptyHandler(ctx)
		return
	}
	// skip prepares the context to move to the next handler if the "nextHandler" has a ctx.Next() inside it,
	// even if it's not executed because it's cached.
	ctx.Skip()

	if !h.rule.Claim(ctx) {
		bodyHandler(ctx)
		return
	}

	scheme := "http"
	if ctx.Request().TLS != nil {
		scheme = "https"
	}

	var (
		response *entry.Response
		valid    = false
		// unique per subdomains and paths with different url query.
		key = scheme + ctx.Host() + ctx.Request().URL.RequestURI()
	)

	h.mu.RLock()
	e, found := h.entries[key]
	h.mu.RUnlock()

	if found {
		// the entry is here, .Response will give us
		// if it's expired or no
		response, valid = e.Response()
	} else {
		// create the entry now.
		// fmt.Printf("create new cache entry\n")
		// fmt.Printf("key: %s\n", key)

		e = entry.NewEntry(h.expiration)
		h.mu.Lock()
		h.entries[key] = e
		h.mu.Unlock()
	}

	if !valid {
		// if it's expired, then execute the original handler
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
			// if no body then just exit.
			return
		}

		// check for an expiration time if the
		// given expiration was not valid then check for GetMaxAge &
		// update the response & release the recorder
		e.Reset(
			recorder.StatusCode(),
			recorder.Header(),
			body,
			parseLifeChanger(ctx),
		)

		// fmt.Printf("reset cache entry\n")
		// fmt.Printf("key: %s\n", key)
		// fmt.Printf("content type: %s\n", recorder.Header().Get(cfg.ContentTypeHeader))
		// fmt.Printf("body len: %d\n", len(body))
		return
	}

	// if it's valid then just write the cached results
	entry.CopyHeaders(ctx.ResponseWriter().Header(), response.Headers())
	ctx.SetLastModified(e.LastModified)
	ctx.StatusCode(response.StatusCode())
	ctx.Write(response.Body())

	// fmt.Printf("key: %s\n", key)
	// fmt.Printf("write content type: %s\n", response.Headers()["ContentType"])
	// fmt.Printf("write body len: %d\n", len(response.Body()))

}
