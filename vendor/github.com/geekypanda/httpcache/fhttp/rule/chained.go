package rule

import (
	"github.com/valyala/fasthttp"
)

// chainedRule is a Rule with next Rule
type chainedRule struct {
	Rule
	next Rule
}

var _ Rule = &chainedRule{}

// chainedSingle returns a new rule witch has a next rule too
func chainedSingle(rule Rule, next Rule) Rule {
	if next == nil {
		next = Satisfied()
	}

	return &chainedRule{
		Rule: rule,
		next: next,
	}
}

// Chained returns a new rule which has more than one coming next ruleset
func Chained(rule Rule, next ...Rule) Rule {
	if len(next) == 0 {
		return chainedSingle(rule, nil)
	}
	c := chainedSingle(rule, next[0])

	for i := 1; i < len(next); i++ {
		c = chainedSingle(c, next[i])
	}

	return c

}

// Claim validator
func (c *chainedRule) Claim(reqCtx *fasthttp.RequestCtx) bool {
	if !c.Rule.Claim(reqCtx) {
		return false
	}
	return c.next.Claim(reqCtx)
}

// Valid validator
func (c *chainedRule) Valid(reqCtx *fasthttp.RequestCtx) bool {
	if !c.Rule.Valid(reqCtx) {
		return false
	}
	return c.next.Valid(reqCtx)
}
