package client

import (
	"net/http"
	"strings"
	"time"

	"github.com/kataras/iris/v12/cache/client/rule"
	"github.com/kataras/iris/v12/cache/entry"
	"github.com/kataras/iris/v12/context"
)

func init() {
	context.SetHandlerName("iris/cache/client.(*Handler).ServeHTTP-fm", "iris.cache")
}

// Handler the local cache service handler contains
// the original response, the memory cache entry and
// the validator for each of the incoming requests and post responses
type Handler struct {
	// Rule optional validators for pre cache and post cache actions
	//
	// See more at ruleset.go
	rule rule.Rule
	// when expires.
	maxAgeFunc MaxAgeFunc
	// entries the memory cache stored responses.
	entryPool  *entry.Pool
	entryStore entry.Store
}

type MaxAgeFunc func(*context.Context) time.Duration

// NewHandler returns a new Server-side cached handler for the "bodyHandler"
// which expires every "expiration".
func NewHandler(maxAgeFunc MaxAgeFunc) *Handler {
	return &Handler{
		rule:       DefaultRuleSet,
		maxAgeFunc: maxAgeFunc,

		entryPool:  entry.NewPool(),
		entryStore: entry.NewMemStore(),
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

// Store sets a custom store for this handler.
func (h *Handler) Store(store entry.Store) *Handler {
	h.entryStore = store
	return h
}

// MaxAge customizes the expiration duration for this handler.
func (h *Handler) MaxAge(fn MaxAgeFunc) *Handler {
	h.maxAgeFunc = fn
	return h
}

var emptyHandler = func(ctx *context.Context) {
	ctx.StopWithText(500, "cache: empty body handler")
}

const entryKeyContextKey = "iris.cache.server.entry.key"

// SetKey sets a custom entry key for cached pages.
// See root package-level `WithKey` instead.
func SetKey(ctx *context.Context, key string) {
	ctx.Values().Set(entryKeyContextKey, key)
}

// GetKey returns the entry key for the current page.
func GetKey(ctx *context.Context) string {
	return ctx.Values().GetString(entryKeyContextKey)
}

func getOrSetKey(ctx *context.Context) string {
	if key := GetKey(ctx); key != "" {
		return key
	}

	// Note: by-default the rules(ruleset pkg)
	// explicitly ignores the cache handler
	// execution on authenticated requests
	// and immediately runs the next handler:
	// if !h.rule.Claim(ctx) ...see `Handler` method.
	// So the below two lines are useless,
	// however we add it for cases
	// that the end-developer messedup with the rules
	// and by accident allow authenticated cached results.
	username, password, _ := ctx.Request().BasicAuth()
	authPart := username + strings.Repeat("*", len(password))

	key := ctx.Method() + authPart

	u := ctx.Request().URL
	if !u.IsAbs() {
		key += ctx.Scheme() + ctx.Host()
	}
	key += u.String()

	SetKey(ctx, key)
	return key
}

func (h *Handler) ServeHTTP(ctx *context.Context) {
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

	key := getOrSetKey(ctx) // unique per subdomains and paths with different url query.

	e := h.entryStore.Get(key)
	if e == nil {
		// if it's expired, then execute the original handler
		// with our custom response recorder response writer
		// because the net/http doesn't give us
		// a builtin way to get the status code & body
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

		// fmt.Printf("reset cache entry\n")
		// fmt.Printf("key: %s\n", key)
		// fmt.Printf("content type: %s\n", recorder.Header().Get(cfg.ContentTypeHeader))
		// fmt.Printf("body len: %d\n", len(body))

		r := entry.NewResponse(recorder.StatusCode(), recorder.Header(), body)
		e = h.entryPool.Acquire(h.maxAgeFunc(ctx), r, func() {
			h.entryStore.Delete(key)
		})

		h.entryStore.Set(key, e)
		return
	}

	// if it's valid then just write the cached results
	r := e.Response()
	// if !ok {
	// 	// it shouldn't be happen because if it's not valid (= expired)
	// 	// then it shouldn't be found on the store, we return as it is, the body was written.
	// 	return
	// }

	copyHeaders(ctx.ResponseWriter().Header(), r.Headers())
	ctx.SetLastModified(e.LastModified)
	ctx.StatusCode(r.StatusCode())
	ctx.Write(r.Body())

	// fmt.Printf("key: %s\n", key)
	// fmt.Printf("write content type: %s\n", response.Headers()["ContentType"])
	// fmt.Printf("write body len: %d\n", len(response.Body()))
}

func copyHeaders(dst, src http.Header) {
	// Clone returns a copy of h or nil if h is nil.
	if src == nil {
		return
	}

	// Find total number of values.
	nv := 0
	for _, vv := range src {
		nv += len(vv)
	}

	sv := make([]string, nv) // shared backing array for headers' values
	for k, vv := range src {
		if vv == nil {
			// Preserve nil values. ReverseProxy distinguishes
			// between nil and zero-length header values.
			dst[k] = nil
			continue
		}

		n := copy(sv, vv)
		dst[k] = sv[:n:n]
		sv = sv[n:]
	}
}
