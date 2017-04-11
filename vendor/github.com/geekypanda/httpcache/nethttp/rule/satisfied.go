package rule

import (
	"net/http"
)

type satisfiedRule struct{}

var _ Rule = &satisfiedRule{}

// Satisfied returns a rule which allows anything,
// it's usualy the last rule on chained rules if no next rule is given,
// but it can be used outside of a chain too as a default allow-all rule.
func Satisfied() Rule {
	return &satisfiedRule{}
}

func (n *satisfiedRule) Claim(*http.Request) bool {
	return true
}

func (n *satisfiedRule) Valid(http.ResponseWriter, *http.Request) bool {
	return true
}
