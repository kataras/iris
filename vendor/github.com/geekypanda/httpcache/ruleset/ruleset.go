// Package ruleset provides the basics rules which are being extended bynethttp's and fhttp's rules.
package ruleset

// The shared header-mostly rules for both nethttp and fasthttp
var (
	AuthorizationRule = func(header GetHeader) bool {
		return header("Authorization") == "" &&
			header("Proxy-Authenticate") == ""
	}

	MustRevalidateRule = func(header GetHeader) bool {
		return header("Must-Revalidate") == ""
	}

	ZeroMaxAgeRule = func(header GetHeader) bool {
		return header("S-Maxage") != "0" &&
			header("Max-Age") != "0"
	}

	NoCacheRule = func(header GetHeader) bool {
		return header("No-Cache") != "true"
	}
)

// THESE ARE HERE BECAUSE THE GOLANG DOESN'T SUPPORTS THE F....  INTERFACE ALIAS, THIS SHOULD EXISTS ONLY ON /$package/rule
// or somehow move interface generic rules (such as conditional, header) here, because the code sharing is exactly THE SAME
// except the -end interface, this on other language can be designing very very nice but here no OOP so we stuck on this,
// it's better than before but not as I wanted to be.
// They will make me to forget my software design skills,
// they (the language's designers) rollback the feature of type alias, BLOG POSTING that is an UNUSEFUL feature.....

// GetHeader is a type of func which receives a func of string which returns a string
// used to get headers values, both request's and response's.
type GetHeader func(string) string

// HeaderPredicate is a type of func which receives a func of string which returns a string and a boolean,
// used to get headers values, both request's and response's.
type HeaderPredicate func(GetHeader) bool

// EmptyHeaderPredicate returns always true
var EmptyHeaderPredicate = func(GetHeader) bool {
	return true
}
