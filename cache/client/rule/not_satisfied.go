// Copyright 2017 Gerasimos Maropoulos, ΓΜ. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

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
