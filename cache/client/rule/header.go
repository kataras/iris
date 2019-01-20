package rule

import (
	"github.com/kataras/iris/context"

	"github.com/kataras/iris/cache/ruleset"
)

// The HeaderPredicate should be alived on each of $package/rule BUT GOLANG DOESN'T SUPPORT type alias and I don't want to have so many copies around
// read more at ../../ruleset.go

// headerRule is a Rule witch receives and checks for a header predicates
// request headers on Claim and response headers on Valid.
type headerRule struct {
	claim ruleset.HeaderPredicate
	valid ruleset.HeaderPredicate
}

var _ Rule = &headerRule{}

// Header returns a new rule witch claims and execute the post validations trough headers
func Header(claim ruleset.HeaderPredicate, valid ruleset.HeaderPredicate) Rule {
	if claim == nil {
		claim = ruleset.EmptyHeaderPredicate
	}

	if valid == nil {
		valid = ruleset.EmptyHeaderPredicate
	}

	return &headerRule{
		claim: claim,
		valid: valid,
	}
}

// HeaderClaim returns a header rule which cares only about claiming (pre-validation)
func HeaderClaim(claim ruleset.HeaderPredicate) Rule {
	return Header(claim, nil)
}

// HeaderValid returns a header rule which cares only about valid (post-validation)
func HeaderValid(valid ruleset.HeaderPredicate) Rule {
	return Header(nil, valid)
}

// Claim validator
func (h *headerRule) Claim(ctx context.Context) bool {
	return h.claim(ctx.Request().Header.Get)
}

// Valid validator
func (h *headerRule) Valid(ctx context.Context) bool {
	return h.valid(ctx.ResponseWriter().Header().Get)
}
