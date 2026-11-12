// Package apikey guards routes with a shared secret that a caller presents
// on every request, which is how services, cron jobs and webhooks
// authenticate to each other when there is no user to sign in.
//
// A key is read from a header (Authorization: Bearer <key> by default) or,
// when configured, from a query parameter. It is compared against the
// configured keys in constant time, or resolved through a Lookup function
// when the keys live in a database. The principal that matched is stored on
// the Context, so a handler and an access log can name the caller.
//
// The default is to deny: a middleware with no keys and no Lookup rejects
// every request instead of quietly letting everyone through.
package apikey

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	stdErrors "errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/kataras/iris/v14/context"
	"github.com/kataras/iris/v14/rest/errors"
)

func init() {
	context.SetHandlerName("iris/middleware/apikey.*", "iris.apikey")
}

// NoScheme disables the auth-scheme prefix, for a header such as X-API-Key
// which carries a bare key. It is a sentinel rather than the empty string so
// that an omitted Scheme still gets the Bearer default.
const NoScheme = "-"

// DefaultHeader and DefaultScheme are the defaults of Options.Header and
// Options.Scheme. Infrastructure (proxies, CDNs, log scrubbers) redacts the
// Authorization header by default, which a custom header does not get.
const (
	DefaultHeader = "Authorization"
	DefaultScheme = "Bearer"
)

// The reasons a request is rejected. A custom ErrorHandler can tell them
// apart with errors.Is, e.g. to log a missing key at a lower level than an
// invalid one.
var (
	// ErrMissing is the reason when the request carries no key at all, or
	// one the configured scheme does not fit.
	ErrMissing = stdErrors.New("apikey: credentials are missing")
	// ErrInvalid is the reason when the request carries a key that matched
	// nothing.
	ErrInvalid = stdErrors.New("apikey: credentials are invalid")
)

// Options is the configuration of the middleware. The zero value denies
// every request; see AllowAnonymous.
type Options struct {
	// Header is the request header that carries the key.
	// Defaults to "Authorization".
	Header string `yaml:"Header" json:"header"`
	// Scheme is the auth-scheme prefix expected on the header value, per
	// RFC 7235. Defaults to "Bearer" and is matched case-insensitively.
	// Set it to NoScheme to accept a bare value, which is what a custom
	// header such as X-API-Key carries.
	Scheme string `yaml:"Scheme" json:"scheme"`
	// Query is an optional query parameter name to read the key from when
	// the header is absent. It is off when empty.
	//
	// A key in a URL ends up in browser history, proxy logs, referrers and,
	// when middleware/accesslog is registered, in this application's own
	// access log, which records every query argument. Its Redact hook covers
	// bodies only, so drop or mask the parameter in a custom accesslog
	// formatter before you enable this.
	Query string `yaml:"Query" json:"query"`
	// Keys maps a principal name to the secret it presents, or to the
	// secret's digest when Hash is set. One entry per consumer of the API,
	// so that a leak can be traced and one key revoked without breaking
	// the others. An entry with an empty secret is dropped.
	//
	// Give every consumer its own secret. Two principals sharing one secret
	// authenticate each other's requests, and which of the two is reported
	// then depends on the map's iteration order at construction.
	Keys map[string]string `yaml:"Keys" json:"keys"`
	// Hash, when set, is applied to the presented key before it is
	// compared, so Keys hold Hash(secret) instead of the secret itself and
	// the configuration file never holds a usable credential.
	// Use apikey.SHA256 unless you have a reason not to.
	Hash func(string) string `yaml:"-" json:"-"`
	// Lookup resolves a key that none of the Keys matched, for keys that
	// live in a database. It reports the principal the key belongs to. An
	// error is a server error (500), not a rejection: a store that is down
	// must not read as an invalid key. When Hash is set, Lookup receives
	// Hash(key), never the raw key, so a store of hashed keys compares
	// directly and must not hash again.
	Lookup func(ctx *context.Context, key string) (principal string, ok bool, err error) `yaml:"-" json:"-"`
	// AllowAnonymous lets every request through when there is neither a Key
	// nor a Lookup, for local development. It is ignored as soon as one of
	// them exists.
	//
	// It must never be true in production, and the middleware logs a
	// warning on the first request when it is.
	AllowAnonymous bool `yaml:"AllowAnonymous" json:"allow_anonymous"`
	// ErrorHandler renders a rejection. The default renders the
	// rest/errors Unauthenticated envelope with the reason
	// (ErrMissing or ErrInvalid) as its message.
	ErrorHandler func(ctx *context.Context, err error) `yaml:"-" json:"-"`
}

// principalContextKey is unexported so that only Get can read the principal
// and no handler can forge one by setting a context value.
const principalContextKey = "iris.apikey.principal"

// Get returns the principal name the request authenticated as, and false
// when the request did not go through this middleware or went through it
// anonymously.
func Get(ctx *context.Context) (string, bool) {
	return ctx.ValueAs[string](principalContextKey)
}

// SHA256 returns the hex-encoded SHA-256 digest of the secret, ready for
// Options.Hash and for generating the values of Options.Keys.
//
// A shared secret is a high-entropy random string, not a password, so a
// plain digest is the right hash here: there is nothing to brute-force.
func SHA256(secret string) string {
	sum := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(sum[:])
}

// digest reduces a value to a fixed 32 bytes, so that every comparison the
// middleware makes is over the same number of bytes no matter how long the
// presented key or the configured secret is.
//
// subtle.ConstantTimeCompare returns 0 immediately when the two lengths
// differ, so comparing the raw values would leak the length of the
// configured keys: a presented key whose length matches one of them costs
// measurably more than one that matches none.
func digest(s string) []byte {
	sum := sha256.Sum256([]byte(s))
	return sum[:]
}

// compare is subtle.ConstantTimeCompare behind a variable so that the test
// can observe that every comparison receives two equal-length inputs. It is
// never reassigned outside the test.
var compare = subtle.ConstantTimeCompare

// keyEntry is one configured credential, with the principal it belongs to.
// The secret is stored as its digest; see digest.
type keyEntry struct {
	principal string
	secret    []byte
}

// New returns the middleware that enforces opts. Register it on a router or
// on a route:
//
//	app.Router("/api", apikey.New(apikey.Options{
//		Keys: map[string]string{"reporting": os.Getenv("REPORTING_KEY")},
//	}))
//
// Behaviour:
//
//   - Keys or Lookup configured: the presented key is compared against every
//     configured key in constant time and then, if nothing matched, handed
//     to Lookup. A match calls the next handler, anything else is a 401.
//   - Neither configured, AllowAnonymous false: every request is a 401
//     (default-deny), with a warning logged on the first one so that a
//     missing configuration is visible.
//   - Neither configured, AllowAnonymous true: every request passes, with a
//     loud warning. Development only.
func New(opts Options) context.Handler {
	header := opts.Header
	if header == "" {
		header = DefaultHeader
	}

	scheme := opts.Scheme
	if scheme == "" {
		scheme = DefaultScheme
	}
	if scheme == NoScheme {
		scheme = ""
	}

	errorHandler := opts.ErrorHandler
	if errorHandler == nil {
		errorHandler = defaultErrorHandler
	}

	entries := make([]keyEntry, 0, len(opts.Keys))
	for principal, secret := range opts.Keys {
		if secret == "" { // a blank secret is not a credential, it is a mistake.
			continue
		}

		entries = append(entries, keyEntry{principal: principal, secret: digest(secret)})
	}

	if len(entries) == 0 && opts.Lookup == nil {
		return newUnconfigured(opts.AllowAnonymous, errorHandler, opts.ErrorHandler == nil)
	}

	hash := opts.Hash
	query := opts.Query

	return func(ctx *context.Context) {
		key, ok := readKey(ctx, header, scheme, query)
		if !ok {
			reject(ctx, errorHandler, fmt.Errorf("apikey: no credentials presented: %w", ErrMissing))
			return
		}

		presented := key
		if hash != nil {
			presented = hash(key)
		}

		if principal, ok := match(entries, presented); ok {
			authenticate(ctx, principal, scheme)
			return
		}

		if opts.Lookup != nil {
			principal, ok, err := opts.Lookup(ctx, presented)
			if err != nil {
				// The store failed; that is our fault, not the caller's, and
				// a 401 here would send them off to rotate a valid key. The
				// render comes before StopExecution because the error
				// pipeline declines to write on a stopped Context.
				errors.HandleError(ctx, err)
				ctx.StopExecution()
				return
			}

			if ok {
				authenticate(ctx, principal, scheme)
				return
			}
		}

		reject(ctx, errorHandler, fmt.Errorf("apikey: the presented key matched nothing: %w", ErrInvalid))
	}
}

// newUnconfigured returns the handler for a middleware with no keys and no
// Lookup: either default-deny or, when the operator asked for it, wide open.
// The warning is logged on the first request because New has no application
// to log through.
func newUnconfigured(allowAnonymous bool, errorHandler func(*context.Context, error), isDefaultHandler bool) context.Handler {
	var once sync.Once

	if allowAnonymous {
		return func(ctx *context.Context) {
			once.Do(func() {
				ctx.Application().Logger().Warnf("apikey: no keys and no Lookup configured and AllowAnonymous is true: every request to the guarded routes is accepted (do not use in production)")
			})

			ctx.Next()
		}
	}

	return func(ctx *context.Context) {
		once.Do(func() {
			ctx.Application().Logger().Warnf("apikey: no keys and no Lookup configured: every request to the guarded routes is rejected with 401")
		})

		err := fmt.Errorf("apikey: authentication is not configured: %w", ErrMissing)
		if isDefaultHandler {
			// The client is not told that the deployment is misconfigured;
			// ErrPrivate keeps the reason in the log and renders the generic
			// Unauthenticated message.
			errors.Unauthenticated.ErrPrivate(ctx, err)
			ctx.StopExecution()
			return
		}

		reject(ctx, errorHandler, err)
	}
}

// readKey returns the key the request presents, from the header or, when the
// header is absent and a query parameter is configured, from the query.
//
// Repeated values are joined by ctx.Header and by the query entries, so a
// request presenting two credentials fails the comparison: two keys on one
// request is not a valid presentation of either.
func readKey(ctx *context.Context, header, scheme, query string) (string, bool) {
	if raw := ctx.Header(header); raw != "" {
		return stripScheme(raw, scheme)
	}

	if query != "" {
		// The query parameter carries a bare key: a scheme prefix belongs to
		// a header, and accepting one here would let "Bearer x" and "x" be
		// two spellings of the same credential.
		if raw := ctx.Query().GetString(query); raw != "" {
			return raw, true
		}
	}

	return "", false
}

// match compares the presented key against every configured key with no
// early exit and with no branch on the result, so that the time taken does
// not reveal which key, or which key length, the request came closest to.
// Both sides are 32-byte digests, so the length of the real secret never
// enters the comparison.
func match(entries []keyEntry, presented string) (string, bool) {
	value := digest(presented)

	matches := 0
	index := -1

	for i, entry := range entries {
		equal := compare(value, entry.secret)
		matches += equal
		// ConstantTimeSelect rather than an if: the winning index is picked
		// without a branch that depends on the secret.
		index = subtle.ConstantTimeSelect(equal, i, index)
	}

	if matches == 0 {
		return "", false
	}

	return entries[index].principal, true
}

// authenticate stores the principal and hands the request to the next
// handler. It also fills a SimpleUser when nothing else did, so that code
// downstream which reads ctx.User() sees the caller.
func authenticate(ctx *context.Context, principal, scheme string) {
	ctx.Values().Set(principalContextKey, principal)

	if ctx.User() == nil {
		authorization := scheme
		if authorization == "" {
			authorization = "API Key"
		}

		ctx.SetUser(&context.SimpleUser{
			Authorization: authorization,
			AuthorizedAt:  time.Now(),
			ID:            principal,
			Username:      principal,
		})
	}

	ctx.Next()
}

// reject renders the reason and stops the request. The order matters: the
// rest/errors renderers decline to write on a Context that is already
// stopped.
func reject(ctx *context.Context, errorHandler func(*context.Context, error), err error) {
	errorHandler(ctx, err)
	ctx.StopExecution()
}

// defaultErrorHandler renders the rest/errors Unauthenticated envelope, the
// same 401 shape the rest of the framework answers with.
func defaultErrorHandler(ctx *context.Context, err error) {
	errors.Unauthenticated.Err(ctx, err)
}

// stripScheme removes the auth-scheme prefix from a header value. The scheme
// is public, so matching it in variable time leaks nothing. An empty scheme
// accepts the raw value.
func stripScheme(raw, scheme string) (string, bool) {
	if scheme == "" {
		return raw, raw != ""
	}

	rest, ok := cutPrefixFold(raw, scheme+" ")
	if !ok {
		return "", false
	}

	// Tolerate padding between the scheme and the key, but nothing else: a
	// key never legitimately starts or ends with a space.
	rest = strings.TrimLeft(rest, " ")
	return rest, rest != ""
}

// cutPrefixFold is strings.CutPrefix with a case-insensitive comparison, as
// RFC 7235 makes the auth-scheme case-insensitive.
func cutPrefixFold(s, prefix string) (string, bool) {
	if len(s) < len(prefix) || !strings.EqualFold(s[:len(prefix)], prefix) {
		return "", false
	}

	return s[len(prefix):], true
}
