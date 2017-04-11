package rule

import (
	"net/http"
)

// Conditional is a Rule witch adds a predicate in order to its methods to execute
type conditionalRule struct {
	claimPredicate func() bool
	validPredicate func() bool
}

var emptyConditionalPredicate = func() bool {
	return true
}

var _ Rule = &conditionalRule{}

// Conditional returns a new rule witch has conditionals
func Conditional(claimPredicate func() bool, validPredicate func() bool) Rule {
	if claimPredicate == nil {
		claimPredicate = emptyConditionalPredicate
	}

	if validPredicate == nil {
		validPredicate = emptyConditionalPredicate
	}

	return &conditionalRule{
		claimPredicate: claimPredicate,
		validPredicate: validPredicate,
	}
}

// Claim validator
func (c *conditionalRule) Claim(r *http.Request) bool {
	if !c.claimPredicate() {
		return false
	}
	return true
}

// Valid validator
func (c *conditionalRule) Valid(w http.ResponseWriter, r *http.Request) bool {
	if !c.validPredicate() {
		return false
	}
	return true
}
