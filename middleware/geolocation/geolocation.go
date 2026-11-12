// Package geolocation resolves the client's IP address to a country, region and
// city (and, optionally, the country's main language).
//
// Its entry point is New(Options), which returns a Handler that stores an *Info
// on the Context; read it back with Get, or look one up on demand with Fetch.
// Free for non-commercial use, no API key required.
package geolocation

import (
	"errors"
	"fmt"
	"sync"

	"github.com/kataras/iris/v14/core/pkg/cache"

	"github.com/kataras/iris/v14/client"
	"github.com/kataras/iris/v14/context"
)

const middlewareName = "iris.geolocation"

func init() {
	context.SetHandlerName("iris/middleware/geolocation.*", middlewareName)
}

// Err is the error returned when the geolocation information is not available.
// Usage:
//
//	if errors.Is(err, geolocation.Err) {
//	 // handle the error err.Error().
//	}
var Err = errors.New("geolocation error")

// Options is the configuration for the geolocation middleware.
// The zero value is valid and uses the free ip-api.com endpoint, restcountries.com
// for the language and an in-memory cache with no expiration.
//
// It uses two different clients for fetching the geolocation information and the
// country's main language:
//   - ip-api.com API is used to fetch the geolocation information for the IP address,
//   - restcountries.com API is used to fetch the language for the country code.
type Options struct {
	// APIKey, when not empty and LocationClient is nil, points the location
	// lookup at the paid HTTPS endpoint (https://pro.ip-api.com/json) with this
	// key. It is ignored when LocationClient is set.
	//
	// Defaults to empty: the free, plaintext-HTTP endpoint is used.
	APIKey string `yaml:"APIKey"`

	// LocationClient is the HTTP client used to fetch geolocation information from ip-api.com.
	//
	// Absolutely free (with rate limiting) for non-commercial use, no API key required.
	//
	// Defaults to a client for http://ip-api.com/json rate-limited to 45 requests
	// per minute (or the pro endpoint when APIKey is set).
	LocationClient *client.Client `yaml:"-"`

	// LanguageClient is the HTTP client used to fetch the language information from restcountries.com.
	// Please consider donating if you are using it a lot.
	//
	// Defaults to a client for https://restcountries.com/v3.1/alpha.
	// Set DisableLanguage to skip the language lookup entirely.
	LanguageClient *client.Client `yaml:"-"`

	// DisableLanguage skips the country-language lookup, so Info.Language is
	// left empty and no request is sent to restcountries.com.
	//
	// Defaults to false.
	DisableLanguage bool `yaml:"DisableLanguage"`

	// Cache is the cache used to store the geolocation information.
	//
	// Defaults to cache.NewMemoryCache[Info](0).
	// Set DisableCache to look up every request.
	Cache cache.Cache[Info] `yaml:"-"`

	// DisableCache turns the lookup cache off, so every request performs a live
	// lookup. Mind the provider's rate limit.
	//
	// Defaults to false.
	DisableCache bool `yaml:"DisableCache"`
}

// New returns a new geolocation middleware (custom implementation) based on the information retrieved by ip-api.com.
// It fetches the geolocation information for the client's IP address and stores it to the Iris Context.
// Retrieve the geolocation information using the `Get` function.
//
// The default rate limit is 45 requests per minute (1,440 daily) for free version.
// Navigate through https://members.ip-api.com/#pricing to see the prices and the rate limits.
// Its paid plan with unlimited requests is very cheap compared to the rest (e.g. ipgeolocation.io).
func New(opts Options) context.Handler {
	o := makeOptions(opts)
	return o.handleRequest
}

// defaultOptions builds the shared default Options exactly once.
//
// Note(@kataras): Fetch used to call makeOptions() on EVERY call, which built a
// fresh rate limiter and a fresh cache each time - so the provider's 45/min free
// tier limit was never actually enforced and nothing was ever cached across
// requests. The free tier bans on overage.
var defaultOptions = sync.OnceValue(func() *Options {
	return makeOptions(Options{})
})

// makeOptions fills the zero-value fields of "opts" with their documented defaults.
func makeOptions(opts Options) *Options {
	o := opts

	if o.LocationClient == nil {
		if o.APIKey != "" {
			o.LocationClient = client.New(
				client.BaseURL("https://pro.ip-api.com/json"),
				client.PersistentRequestOptions(client.RequestParam("key", o.APIKey)),
			)
		} else {
			// SECURITY: ip-api.com's free tier is plaintext HTTP only.
			//
			// Every visitor's IP address is therefore disclosed to any on-path
			// observer, and the answer - which the application may use for geo-gating,
			// pricing or redirects - is unauthenticated and can be forged in transit.
			// Set APIKey (the HTTPS pro endpoint), or point LocationClient at your own
			// provider, before relying on this for anything that matters.
			o.LocationClient = client.New(client.BaseURL("http://ip-api.com/json"), client.RateLimitPerMinute(45))
		}
	}

	if o.DisableLanguage {
		o.LanguageClient = nil
	} else if o.LanguageClient == nil {
		o.LanguageClient = client.New(client.BaseURL("https://restcountries.com/v3.1/alpha"))
	}

	if o.DisableCache {
		o.Cache = nil
	} else if o.Cache == nil {
		o.Cache = cache.NewMemoryCache[Info](0)
	}

	return &o
}

// Fetch returns the geolocation information for the client's IP address.
// It returns the value already stored in the Iris Context, otherwise it performs
// a live lookup with the default options and stores the result in the Context.
// It returns an error if the information is not available.
// Use the `New` function to register the geolocation middleware
// or the `Get` function to read the stored value without performing a lookup.
func Fetch(ctx *context.Context) (*Info, error) {
	if info := Get(ctx); info != nil {
		return info, nil
	}

	info, err := defaultOptions().fetchInfo(ctx, ctx.RemoteAddr())
	if err != nil {
		return nil, fmt.Errorf("%w: %w", Err, err)
	}

	ctx.Values().Set(infoContextKey, info)
	return info, nil
}

// Get returns the geolocation information from the Iris Context.
func Get(ctx *context.Context) *Info {
	if info, ok := ctx.ValueAs[*Info](infoContextKey); ok {
		return info
	}

	return nil
}

const infoContextKey = "iris.geolocation.info"

func (opts *Options) handleRequest(ctx *context.Context) {
	ip := ctx.RemoteAddr()

	if opts.Cache != nil {
		// check if the geolocation information is already stored in the cache.
		if info, ok, err := opts.Cache.Get(ctx, ip); ok {
			ctx.Values().Set(infoContextKey, info)
			ctx.Next()
			return
		} else if err != nil {
			// A cache read failure is not a reason to reject the request; fall
			// through to a live lookup.
			ctx.OptionsReadOnly().HandleError(ctx, fmt.Errorf("%w: %w", Err, err))
		}
	}

	// get the basic, main geolocation information.
	//
	// Note(@kataras): fail OPEN from here on.
	//
	// Geolocation is enrichment: a provider outage or a rate-limit rejection must
	// not take the whole site down. These paths used to HandleError and return
	// without calling Next, so an ip-api.com blip stopped every request.
	info, err := opts.fetchInfo(ctx, ip)
	if err != nil {
		ctx.OptionsReadOnly().HandleError(ctx, fmt.Errorf("%w: %w", Err, err))
		ctx.Next()
		return
	}

	if opts.Cache != nil {
		// store the geolocation information in the cache.
		if err := opts.Cache.Set(ctx, ip, info); err != nil {
			ctx.OptionsReadOnly().HandleError(ctx, fmt.Errorf("%w: %w", Err, err))
			// The lookup succeeded; only the cache write failed.
		}
	}

	// add geolocation information to the Iris Context.
	ctx.Values().Set(infoContextKey, info)
	ctx.Next()
}

func (opts *Options) fetchInfo(ctx *context.Context, ip string) (*Info, error) {
	if ip == "" {
		return nil, fmt.Errorf("no IP address found")
	}

	// The outbound requests take the request's standard context, never the Iris
	// Context itself: the Iris Context is pooled, and the HTTP transport keeps a
	// reference to the request context for cancel propagation that it may walk
	// AFTER the handler returned and the Context was released. See Context.StdContext.
	stdCtx := ctx.StdContext()

	// get the basic, main geolocation information.
	info, err := fetchGeoLocationInfo(stdCtx, opts.LocationClient, ip)
	if err != nil {
		return nil, fmt.Errorf("location: %s: %w", ip, err)
	}

	if opts.LanguageClient != nil {
		// get the language for the country code.
		language, err := fetchLanguage(stdCtx, opts.LanguageClient, info.CountryCode)
		if err != nil {
			return nil, fmt.Errorf("language: %s: %w", info.CountryCode, err)
		}
		// bind the language to the geolocation information.
		info.Language = language
	}

	return info, nil
}
