package rule

import (
	"github.com/kataras/iris/context"
)

type satisfiedRule struct{}

var _ Rule = &satisfiedRule{}

// Satisfied returns a rule which allows anything,
// it's usually the last rule on chained rules if no next rule is given,
// but it can be used outside of a chain too as a default allow-all rule.
func Satisfied() Rule {
	return &satisfiedRule{}
}

func (n *satisfiedRule) Claim(context.Context) bool {
	return true
}

func (n *satisfiedRule) Valid(context.Context) bool {
	return true
}
