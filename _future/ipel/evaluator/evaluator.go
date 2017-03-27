package evaluator

import (
	"fmt"
	"regexp"
)

// final evaluator signature for both param types and param funcs
type ParamEvaluator func(paramValue string) bool

func NewParamEvaluatorFromRegexp(expr string) (ParamEvaluator, error) {
	if expr == "" {
		return nil, fmt.Errorf("empty regex expression")
	}

	// add the last $ if missing (and not wildcard(?))
	if i := expr[len(expr)-1]; i != '$' && i != '*' {
		expr += "$"
	}

	r, err := regexp.Compile(expr)
	if err != nil {
		return nil, err
	}

	return r.MatchString, nil
}
