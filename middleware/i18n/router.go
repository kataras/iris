package i18n

import (
	"net/http"
	"strings"

	"github.com/kataras/iris/v14/context"

	"golang.org/x/text/language"
)

func init() {
	context.SetHandlerName("iris/middleware/i18n.Router*", "iris.i18n.router")
}

const (
	localizationRouterContextKey = "iris.i18n.router"
)

// GetRouter returns the i18n router from the context.
func GetRouter(ctx *context.Context) *Router {
	if v := ctx.Values().Get(localizationRouterContextKey); v != nil {
		return v.(*Router)
	}

	return nil
}

// RouterOptions contains the options for the i18n router.
type RouterOptions struct {
	// InjectFunc is a function that can be used to inject the language tag to the context.
	InjectFunc func(ctx *context.Context, tag language.Tag) `yaml:"-"`

	// If not empty, the Router will set the language identifier to the response cookie of this name.
	//
	// Defaults to empty.
	Cookie string `yaml:"Cookie"`
	// If true then a subdomain can be a language identifier,
	// and the requests are being redirected to the main host.
	//
	// Defaults to false.
	Subdomain bool `yaml:"Subdomain"`
}

func (opts *RouterOptions) injectLanguageTag(ctx *context.Context, tag language.Tag) {
	lang := tag.String()
	if opts.Cookie != "" {
		ctx.SetCookie(&http.Cookie{
			Name:  opts.Cookie,
			Value: lang,
			// allow subdomain sharing.
			Domain:   ctx.Domain(),
			SameSite: http.SameSiteLaxMode,
		})
	}

	ctx.Request().Header.Set(acceptLanguageHeaderKey, lang)

	if fn := opts.InjectFunc; fn != nil {
		fn(ctx, tag)
	}
}

// Router is the main structure of the i18n router.
// It's responsible to redirect the request to the correct path based on the language.
//
// Look the `NewRouter` package-level function for more.
type Router struct {
	LocaleMatcher

	opts *RouterOptions
}

// NewRouter returns a nwe Router Handler.
// This will perform router redirects like:
//   - /$lang_prefix/$path to /$path with the language set to $lang_prefix part and
//   - $lang_subdomain.$domain/$path to $domain/$path with the language set to $lang_subdomain part.
//
// Usage:
//
//	i18nInstance := New()
//	i18nInstance.Load("./locales/*/*", "en-US", "el-GR", "zh-CN")
//
//	i18nRouter := NewRouter(i18nInstance, RouterOptions{InjectFunc: nil, Cookie: "lang", Subdomain: true})
//
//	app := iris.New()
//	app.UseRouter(i18nRouter)
func NewRouter(i LocaleMatcher, opts RouterOptions) context.Handler {
	router := &Router{
		LocaleMatcher: i,
		opts:          &opts,
	}

	return router.handleRequest
}

// CanUseSubdomainAsLanguage returns true if the subdomain redirection is enabled.
func (router *Router) CanUseSubdomainAsLanguage() bool {
	return router.opts.Subdomain
}

func (router *Router) handleRequest(ctx *context.Context) {
	r := ctx.Request()

	found := false
	path := r.URL.Path
	if len(path) > 0 && path[0] == '/' {
		path = path[1:]
	}

	if idx := strings.IndexByte(path, '/'); idx > 0 {
		path = path[:idx]
	}

	if path != "" {
		if tag, _, ok := router.LocaleMatcher.TryMatchString(path); ok {
			path = r.URL.Path[len(path)+1:]
			if path == "" {
				path = "/"
			}

			r.RequestURI = path
			r.URL.Path = path
			router.opts.injectLanguageTag(ctx, tag)
			found = true
		}
	}

	if !found && router.opts.Subdomain {
		host := ctx.Host()
		if dotIdx := strings.IndexByte(host, '.'); dotIdx > 0 {
			if subdomain := host[0:dotIdx]; subdomain != "" {
				if tag, _, ok := router.LocaleMatcher.TryMatchString(subdomain); ok {
					host = host[dotIdx+1:]
					r.URL.Host = host
					r.Host = host
					router.opts.injectLanguageTag(ctx, tag)
				}
			}
		}
	}

	ctx.Values().Set(localizationRouterContextKey, router)
	ctx.Next()
}
