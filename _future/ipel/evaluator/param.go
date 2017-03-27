package evaluator

import (
	"gopkg.in/kataras/iris.v6/_future/ipel/ast"
)

// exported to be able to change how param types are evaluating
var ParamTypeEvaluator = make(map[ast.ParamType]ParamEvaluator, 0)

func init() {

	// string type
	// anything.
	stringRegex, err := NewParamEvaluatorFromRegexp(".*")
	if err != nil {
		panic(err)
	}
	ParamTypeEvaluator[ast.ParamTypeString] = stringRegex

	// int type
	// only numbers (0-9)
	numRegex, err := NewParamEvaluatorFromRegexp("[0-9]+$")
	if err != nil {
		panic(err)
	}
	ParamTypeEvaluator[ast.ParamTypeInt] = numRegex

	// alphabetical/letter type
	// letters only (upper or lowercase)
	alphabeticRegex, err := NewParamEvaluatorFromRegexp("[a-zA-Z]+$")
	if err != nil {
		panic(err)
	}
	ParamTypeEvaluator[ast.ParamTypeAlphabetical] = alphabeticRegex

	// file type
	// letters (upper or lowercase)
	// numbers (0-9)
	// underscore (_)
	// dash (-)
	// point (.)
	// no spaces! or other character
	fileRegex, err := NewParamEvaluatorFromRegexp("[a-zA-Z0-9_.-]*$")
	if err != nil {
		panic(err)
	}
	ParamTypeEvaluator[ast.ParamTypeFile] = fileRegex

	// path type
	// file with slashes(anywhere)
	pathRegex, err := NewParamEvaluatorFromRegexp("[a-zA-Z0-9_.-/]*$")
	if err != nil {
		panic(err)
	}
	ParamTypeEvaluator[ast.ParamTypePath] = pathRegex
}
