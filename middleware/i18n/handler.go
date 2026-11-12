package i18n

import (
	"github.com/kataras/iris/v14/context"

	"github.com/kataras/iris/v14/middleware/i18n/locale"

	"golang.org/x/text/language"
)

func init() {
	context.SetHandlerName("iris/middleware/i18n.Handler*", "iris.i18n.handler")
}

const (
	localizationHandlerContextKey = "iris.i18n.handler"
	localeContextKey              = "iris.i18n.locale"
	languageContextKey            = "iris.i18n.language"
)

// GetMessage is a shortcut of GetLocale(ctx).GetMessage(key, args...).
func GetMessage(ctx *context.Context, key string, args ...any) string {
	if locale := GetLocale(ctx); locale != nil {
		return locale.GetMessage(key, args...)
	}

	return ""
}

// SetLanguage sets the language code to the context, it overrides any request's language codes.
func SetLanguage(ctx *context.Context, languageCode string) {
	if languageCode == "" {
		ctx.Values().Remove(languageContextKey)
		return
	}

	ctx.Values().Set(languageContextKey, languageCode)
}

// SetLocaleByLanguage sets and returns the locale to the context based on the language code.
func SetLocaleByLanguage(ctx *context.Context, languageCode string) locale.Locale {
	if handler := GetHandler(ctx); handler != nil {
		if locale, ok := handler.GetLocale(languageCode); ok {
			SetLocale(ctx, locale)
			return locale
		}
	}

	return nil
}

// SetLocale sets the locale to the context.
func SetLocale(ctx *context.Context, locale locale.Locale) {
	if locale == nil {
		ctx.Values().Remove(localeContextKey)
		return
	}

	ctx.Values().Set(localeContextKey, locale)
}

// GetLocale returns the locale from the context.
func GetLocale(ctx *context.Context) locale.Locale {
	if v := ctx.Values().Get(localeContextKey); v != nil {
		return v.(locale.Locale)
	}

	return nil
}

// SetHandler sets the i18n handler to the context.
func SetHandler(ctx *context.Context, h *Handler) {
	if h == nil {
		ctx.Values().Remove(localizationHandlerContextKey)
		return
	}

	ctx.Values().Set(localizationHandlerContextKey, h)
}

// GetHandler returns the i18n handler from the context.
func GetHandler(ctx *context.Context) *Handler {
	if v := ctx.Values().Get(localizationHandlerContextKey); v != nil {
		return v.(*Handler)
	}

	return nil
}

type (
	// LocaleMatcher is the interface which contains the supported languages and the logic to match a language based on the request.
	// It's being used by the `Handler` to extract the language from the request.
	LocaleMatcher interface {
		// Tags returns the list of language tags that are supported by the application.
		Tags() []language.Tag

		// TryMatchString should return the language tag and the confidence of the match based on the given string.
		TryMatchString(string) (language.Tag, int, bool)
		// GetLocaleByIndex should return the Locale based on the given index.
		// GetLocaleByIndex(int) (Locale, bool)
		GetDefaultLocale() locale.Locale
		// GetLocale should return the Locale based on the given language tag.
		GetLocale(string) (locale.Locale, bool)
	}
)

// HandlerOptions is the options for the i18n handler.
type HandlerOptions struct {
	// ExtractFunc is the type signature for declaring custom logic
	// to extract the language tag name.
	// By-default the handler will retrieve the language code by the Accept-Language request header.
	//
	// Defaults to nil.
	ExtractFunc func(ctx *context.Context) string `yaml:"-"`
	// If not empty, it is language identifier by url query.
	//
	// Defaults to empty.
	URLParameter string `yaml:"URLParameter"`
	// If not empty, it is language identifier by cookie of this name.
	//
	// Defaults to empty.
	Cookie string `yaml:"Cookie"`
	// If true then a subdomain can be a language identifier.
	//
	// Defaults to false.
	Subdomain bool `yaml:"Subdomain"`
}

// Handler is the main structure of the i18n handler.
// It's responsible to extract the language from the request and set the locale.
//
// Retrieve the locale through the `GetLocale` and translate messages through `GetMessage` package-level functions.
//
// Look the `NewHandler` package-level function for more.
type Handler struct {
	LocaleMatcher `yaml:"-"`

	opts *HandlerOptions
}

// NewHandler returns a new i18n handler.
// It's the main structure of the i18n middleware.
// It's responsible to extract the language from the request and set the matched
// Locale to the Context, so it can be retrieved through the `GetLocale` package-level function.
// Look the `NewRouter` package-level function for path and subdomain based language redirects.
//
// Usage:
//
//	i18nInstance := New()
//	i18nInstance.Load("./locales/*/*", "en-US", "el-GR", "zh-CN")
//	i18nInstance.SetDefault("en-US")
//
//	i18nHandler := NewHandler(i18nInstance, HandlerOptions{URLParameter: "lang", Cookie: "", Subdomain: true})
//
//	app := iris.New()
//	app.Use(i18nHandler)
func NewHandler(i LocaleMatcher, opts HandlerOptions) context.Handler {
	handler := &Handler{
		LocaleMatcher: i,
		opts:          &opts,
	}

	return handler.handleRequest
}

// GetURLParameter returns the URL parameter name which is used to extract the language.
func (h *Handler) GetURLParameter() string {
	return h.opts.URLParameter
}

// GetCookie returns the cookie name which is used to extract the language.
func (h *Handler) GetCookie() string {
	return h.opts.Cookie
}

// CanUseSubdomainAsLanguage returns true if the subdomain can be used as a language identifier.
func (h *Handler) CanUseSubdomainAsLanguage() bool {
	return h.opts.Subdomain
}

// handler is the main function of the i18n Handler.
// It sets the language to the request and response based on the configuration.
// It also sets itself (instance) to the context.
func (h *Handler) handleRequest(ctx *context.Context) {
	SetHandler(ctx, h)
	SetLocale(ctx, h.extractLocale(ctx))

	ctx.Next()
}

const acceptLanguageHeaderKey = "Accept-Language"

// extractLocale extracts the locale of a request.
// It will return the first registered language if nothing else matched.
func (h *Handler) extractLocale(ctx *context.Context) locale.Locale {
	if serverLangCode := ctx.Values().GetString(languageContextKey); serverLangCode != "" {
		// If force-set by server, then this is the language and we don't count any requested one,
		// if not found then we directly return nil.
		if locale, ok := h.LocaleMatcher.GetLocale(serverLangCode); ok {
			return locale
		}

		return nil
	}

	if fn := h.opts.ExtractFunc; fn != nil {
		if v := fn(ctx); v != "" {
			if locale, ok := h.LocaleMatcher.GetLocale(v); ok {
				return locale
			}
		}
	}

	if param := h.opts.URLParameter; param != "" {
		if v := ctx.Query().Get(param); v != "" {
			if locale, ok := h.LocaleMatcher.GetLocale(v); ok {
				return locale
			}
		}
	}

	if cookieName := h.opts.Cookie; cookieName != "" {
		if v := ctx.CookieValue(cookieName); v != "" {
			if locale, ok := h.LocaleMatcher.GetLocale(v); ok { // url.QueryUnescape(cookie.Value)
				return locale
			}
		}
	}

	if h.opts.Subdomain {
		if v := ctx.Subdomain(); v != "" {
			if locale, ok := h.LocaleMatcher.GetLocale(v); ok {
				return locale
			}
		}
	}

	if v := ctx.Header(acceptLanguageHeaderKey); v != "" {
		if locale, ok := h.LocaleMatcher.GetLocale(v); ok {
			return locale
		}
	}

	// Return the default one, which is the first registered language.
	return h.LocaleMatcher.GetDefaultLocale()
	// locale := i.localizer.GetLocale(index)
	// ctx.Values().Set(ctx.Application().ConfigurationReadOnly().GetLocaleContextKey(), locale)
	//
	// Set the user input we wanna use it on DefaultMessageFunc.
	// Even if matched because it may be en-gb or en but if there is a language registered
	// as en-us it will be successfully matched ( see TrymatchString and Low conf).
	// ctx.Values().Set(context.LanguageInputContextKey, extractedLang)
	//
	// if index == 0 then it defaults to the first language.
}
