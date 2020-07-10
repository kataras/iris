package rule

import "github.com/kataras/iris/v12/context"

// Rule a superset of validators
type Rule interface {
	Claim(ctx *context.Context) bool
	Valid(ctx *context.Context) bool
}
