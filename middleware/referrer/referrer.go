// Package referrer parses the request's Referer header into its source type
// (direct, search, social, email or indirect) and stores it on the Context.
//
// It has no settings, so following the plain-handler rule its entry point is the
// `Handler` function itself; read the result with `Get`.
package referrer

import (
	"github.com/Shopify/goreferrer"

	"github.com/kataras/iris/v14/context"
)

func init() {
	// Note the package path spells "referrer" (double r); the old
	// "iris/middleware/referer.*" pattern never matched the real symbol.
	context.SetHandlerName("iris/middleware/referrer.*", "iris.referer")
}

type (
	// Referrer contains the extracted information from the `Get` function.
	//
	// The structure contains struct tags for JSON, form, XML, YAML and TOML.
	// Look the `Get(ctx *context.Context) Referrer` and `goreferrer` external package.
	Referrer struct {
		// The raw refer(r)er URL.
		Raw        string                   `json:"raw" form:"raw" xml:"Raw" yaml:"Raw" toml:"Raw"`
		Type       ReferrerType             `json:"type" form:"referrer_type" xml:"Type" yaml:"Type" toml:"Type"`
		Label      string                   `json:"label" form:"referrer_form" xml:"Label" yaml:"Label" toml:"Label"`
		URL        string                   `json:"url" form:"referrer_url" xml:"URL" yaml:"URL" toml:"URL"`
		Subdomain  string                   `json:"subdomain" form:"referrer_subdomain" xml:"Subdomain" yaml:"Subdomain" toml:"Subdomain"`
		Domain     string                   `json:"domain" form:"referrer_domain" xml:"Domain" yaml:"Domain" toml:"Domain"`
		Tld        string                   `json:"tld" form:"referrer_tld" xml:"Tld" yaml:"Tld" toml:"Tld"`
		Path       string                   `json:"path" form:"referrer_path" xml:"Path" yaml:"Path" toml:"Path"`
		Query      string                   `json:"query" form:"referrer_query" xml:"Query" yaml:"Query" toml:"GoogleType"`
		GoogleType ReferrerGoogleSearchType `json:"googleType" form:"referrer_google_type" xml:"GoogleType" yaml:"GoogleType" toml:"GoogleType"`
	}

	// ReferrerType is the goreferrer enum for a referrer type (indirect, direct, email, search, social).
	ReferrerType = goreferrer.ReferrerType

	// ReferrerGoogleSearchType is the goreferrer enum for a google search type (organic, adwords).
	ReferrerGoogleSearchType = goreferrer.GoogleSearchType
)

// String returns the raw ref url.
func (ref Referrer) String() string {
	return ref.Raw
}

// Contains the available values of the goreferrer enums.
const (
	ReferrerInvalid ReferrerType = iota
	ReferrerIndirect
	ReferrerDirect
	ReferrerEmail
	ReferrerSearch
	ReferrerSocial

	ReferrerNotGoogleSearch ReferrerGoogleSearchType = iota
	ReferrerGoogleOrganicSearch
	ReferrerGoogleAdwords
)

// unnecessary but good to know the default values upfront.
var emptyReferrer = Referrer{Type: ReferrerInvalid, GoogleType: ReferrerNotGoogleSearch}

const referrerContextKey = "iris.request.referrer"

// Handler is a middleware that extracts the referrer information from the request's headers and query parameters.
// It stores the information to the context values in order to be used by the `Get` function.
func Handler(ctx *context.Context) {
	ctx.Values().Set(referrerContextKey, Get(ctx))
	ctx.Next()
}

// Get extracts and returns the information from the "Referer" (or "Referrer") header
// and url query parameter as specified in https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Referrer-Policy.
func Get(ctx *context.Context) Referrer {
	if referrer, ok := ctx.ValueAs[Referrer](referrerContextKey); ok {
		return referrer
	}

	// the underline net/http follows the https://tools.ietf.org/html/rfc7231#section-5.5.2,
	// so there is nothing special left to do.
	// https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Referrer-Policy
	refURL := ctx.Header("Referer")
	if refURL == "" {
		refURL = ctx.Header("Referrer")
		if refURL == "" {
			refURL = ctx.Query().Get("referer")
			if refURL == "" {
				refURL = ctx.Query().Get("referrer")
			}
		}
	}

	if refURL == "" {
		return emptyReferrer
	}

	if ref := goreferrer.DefaultRules.Parse(refURL); ref.Type > goreferrer.Invalid {
		referrer := Referrer{
			Raw:        refURL,
			Type:       ReferrerType(ref.Type),
			Label:      ref.Label,
			URL:        ref.URL,
			Subdomain:  ref.Subdomain,
			Domain:     ref.Domain,
			Tld:        ref.Tld,
			Path:       ref.Path,
			Query:      ref.Query,
			GoogleType: ReferrerGoogleSearchType(ref.GoogleType),
		}
		// ctx.Values().Set(referrerContextKey, referrer)
		return referrer
	}

	return emptyReferrer
}
