package sitemap

import (
	"strings"

	"github.com/kataras/iris/v14/context"
	"github.com/kataras/iris/v14/core/pkg/netutil"
	"github.com/kataras/iris/v14/core/router"

	"github.com/kataras/iris/v14/middleware/i18n"

	"github.com/kataras/sitemap"
	"golang.org/x/text/language"
)

func init() {
	context.SetHandlerName("iris/controller/sitemap.*", "iris.sitemap")
}

// Controller is a router.Controller which can be used to generate and register sitemap(%d).xml endpoint(s)
// based on the registered routes.
type Controller struct {
	domain string

	// localization options.
	defaultLang  string
	languageTags []language.Tag

	isPathRedirectEnabled     bool
	canUseSubdomainAsLanguage bool
	urlParameter              string
	//
}

// New returns a new sitemap router. It accepts the domain name as its only parameter.
// Use the Route's `SetLastMod`, `SetChangeFreq` and `SetPriority` to modify
// the sitemap's URL child element properties .
// Excluded routes:
// - dynamic
// - subdomain
// - offline
// - ExcludeSitemap method called
//
// The "domain" argument is the prefix for the registered routes
// that will be included in the sitemap.
//
// If more than 50,000 static routes are registered then sitemaps will be splitted and a sitemap index will be served in
// /sitemap.xml.
//
// If `Application.I18n.Load/LoadAssets` is called then the sitemap will contain translated links for each static route.
//
// If the result does not complete your needs you can take control
// and use the github.com/kataras/sitemap package to generate a customized one instead.
// Usage:
//
//	app := iris.New()
//	[Register other routes first...]
//	app.Router("/", sitemap.New("https://www.example.com"))
func New(domain string) *Controller {
	return &Controller{
		domain: domain,
	}
}

// WithLocalization adds localization support for the generated links in the sitemap,
// specify the i18n handler or router or both.
func (s *Controller) WithLocalization(handler *i18n.Handler, router *i18n.Router) *Controller {
	var (
		defaultLang  string
		languageTags []language.Tag

		isPathRedirectEnabled     bool
		canUseSubdomainAsLanguage bool
		urlParameter              string
	)

	// Handler or Router can be nil, so we must extract the basic information from both of them.
	var localeMatcher i18n.LocaleMatcher
	if h := handler; h != nil {
		urlParameter = h.GetURLParameter() // only-handler option.
		canUseSubdomainAsLanguage = h.CanUseSubdomainAsLanguage()

		localeMatcher = h.LocaleMatcher
	}

	if r := router; r != nil {
		isPathRedirectEnabled = true
		if !canUseSubdomainAsLanguage {
			canUseSubdomainAsLanguage = r.CanUseSubdomainAsLanguage()
		}

		localeMatcher = r.LocaleMatcher
	}

	if localeMatcher == nil {
		return nil
	}

	defaultLang = localeMatcher.GetDefaultLocale().Language()
	languageTags = localeMatcher.Tags()
	//

	s.defaultLang = defaultLang
	s.languageTags = languageTags
	s.isPathRedirectEnabled = isPathRedirectEnabled
	s.canUseSubdomainAsLanguage = canUseSubdomainAsLanguage
	s.urlParameter = urlParameter
	return s
}

// Routing implements the router.Controller interface.
func (s *Controller) Routing(r router.Router) {
	app := r.Application()

	beforeBuild := func() error {
		handlers := s.makeSitemapHandler(app)
		for path, handler := range handlers {
			r.Handle(router.JoinMethods(router.MethodGet, router.MethodHead, router.MethodOptions), path, handler)
		}

		return nil
	}
	app.RegisterOnBuild(beforeBuild, nil)
}

func (s *Controller) makeSitemapHandler(app context.Application) map[string]context.Handler {
	startURL := s.domain

	sitemaps := sitemap.New(startURL)
	sitemaps.DefaultLang(s.defaultLang)

	for _, r := range app.ListRoutesReadOnly() {
		if !r.IsStatic() || r.GetSubdomain() != "" || !r.IsOnline() || r.IsSitemapDisabled() {
			continue
		}

		loc := r.StaticPath()
		var translatedLinks []sitemap.Link

		for _, tag := range s.languageTags {
			lang := tag.String()
			langPath := lang
			href := ""
			if lang == s.defaultLang {
				// http://domain.com/en-US/path to just http://domain.com/path if en-US is the default language.
				langPath = ""
			}

			if s.isPathRedirectEnabled {
				// then use the path prefix.
				// e.g. http://domain.com/el-GR/path
				if langPath == "" { // fix double slashes http://domain.com// when self-included default language.
					href = loc
				} else {
					href = "/" + langPath + loc
				}
			} else if s.canUseSubdomainAsLanguage {
				// then use the subdomain.
				// e.g. http://el.domain.com/path
				scheme := netutil.ResolveSchemeFromVHost(startURL)
				host := strings.TrimLeft(startURL, scheme)
				if langPath != "" {
					langSubdomain, _, _ := strings.Cut(langPath, "-")
					href = scheme + langSubdomain + "." + host + loc
				} else {
					href = loc
				}

			} else if p := s.urlParameter; p != "" {
				// then use the URL parameter.
				// e.g. http://domain.com/path?lang=el-GR
				href = loc + "?" + p + "=" + lang
			} else {
				// then skip it, we can't generate the link at this state.
				continue
			}

			translatedLinks = append(translatedLinks, sitemap.Link{
				Rel:      "alternate",
				Hreflang: lang,
				Href:     href,
			})
		}

		sitemaps.URL(sitemap.URL{
			Loc:        loc,
			LastMod:    r.GetLastMod(),
			ChangeFreq: r.GetChangeFreq(),
			Priority:   r.GetPriority(),
			Links:      translatedLinks,
		})
	}

	sitemapHandlers := sitemaps.Build()
	handlers := make(map[string]context.Handler, len(sitemapHandlers))
	for _, s := range sitemapHandlers {
		contentCopy := make([]byte, len(s.Content))
		copy(contentCopy, s.Content)

		handler := func(ctx *context.Context) {
			ctx.SetContentType(context.ContentXMLHeaderValue)
			ctx.Write(contentCopy) // nolint:errcheck
		}

		handlers[s.Path] = handler
	}

	return handlers
}
