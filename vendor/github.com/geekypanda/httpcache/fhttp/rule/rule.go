package rule

import (
	"github.com/valyala/fasthttp"
)

// Rule a superset of validators
type Rule interface {
	Claim(*fasthttp.RequestCtx) bool
	Valid(*fasthttp.RequestCtx) bool
}
