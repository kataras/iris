package rule

import (
	"net/http"
)

type notSatisfiedRule struct{}

var _ Rule = &notSatisfiedRule{}

// NotSatisfied returns a rule which allows nothing
func NotSatisfied() Rule {
	return &notSatisfiedRule{}
}

func (n *notSatisfiedRule) Claim(*http.Request) bool {
	return false
}

func (n *notSatisfiedRule) Valid(http.ResponseWriter, *http.Request) bool {
	return false
}
