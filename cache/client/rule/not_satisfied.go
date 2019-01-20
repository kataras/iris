package rule

import (
	"github.com/kataras/iris/context"
)

type notSatisfiedRule struct{}

var _ Rule = &notSatisfiedRule{}

// NotSatisfied returns a rule which allows nothing
func NotSatisfied() Rule {
	return &notSatisfiedRule{}
}

func (n *notSatisfiedRule) Claim(context.Context) bool {
	return false
}

func (n *notSatisfiedRule) Valid(context.Context) bool {
	return false
}
