package parser

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/kataras/iris/core/router/macro/interpreter/ast"
)

func TestParseParamError(t *testing.T) {
	// fail
	illegalChar := '$'

	input := "{id" + string(illegalChar) + "int range(1,5) else 404}"
	p := NewParamParser(input)

	_, err := p.Parse()

	if err == nil {
		t.Fatalf("expecting not empty error on input '%s'", input)
	}

	illIdx := strings.IndexRune(input, illegalChar)
	expectedErr := fmt.Sprintf("[%d:%d] illegal token: %s", illIdx, illIdx, "$")
	if got := err.Error(); got != expectedErr {
		t.Fatalf("expecting error to be '%s' but got: %s", expectedErr, got)
	}
	//

	// success
	input2 := "{id:int range(1,5) else 404}"
	p.Reset(input2)
	_, err = p.Parse()

	if err != nil {
		t.Fatalf("expecting empty error on input '%s', but got: %s", input2, err.Error())
	}
	//
}

func TestParseParam(t *testing.T) {
	tests := []struct {
		valid             bool
		expectedStatement ast.ParamStatement
	}{
		{true,
			ast.ParamStatement{
				Src:  "{id:int min(1) max(5) else 404}",
				Name: "id",
				Type: ast.ParamTypeInt,
				Funcs: []ast.ParamFunc{
					{
						Name: "min",
						Args: []ast.ParamFuncArg{1}},
					{
						Name: "max",
						Args: []ast.ParamFuncArg{5}},
				},
				ErrorCode: 404,
			}}, // 0

		{true,
			ast.ParamStatement{
				Src:  "{id:int range(1,5)}",
				Name: "id",
				Type: ast.ParamTypeInt,
				Funcs: []ast.ParamFunc{
					{
						Name: "range",
						Args: []ast.ParamFuncArg{1, 5}},
				},
				ErrorCode: 404,
			}}, // 1
		{true,
			ast.ParamStatement{
				Src:  "{file:path contains(.)}",
				Name: "file",
				Type: ast.ParamTypePath,
				Funcs: []ast.ParamFunc{
					{
						Name: "contains",
						Args: []ast.ParamFuncArg{"."}},
				},
				ErrorCode: 404,
			}}, // 2
		{true,
			ast.ParamStatement{
				Src:       "{username:alphabetical}",
				Name:      "username",
				Type:      ast.ParamTypeAlphabetical,
				ErrorCode: 404,
			}}, // 3
		{true,
			ast.ParamStatement{
				Src:       "{myparam}",
				Name:      "myparam",
				Type:      ast.ParamTypeString,
				ErrorCode: 404,
			}}, // 4
		{false,
			ast.ParamStatement{
				Src:       "{myparam_:thisianunexpected}",
				Name:      "myparam_",
				Type:      ast.ParamTypeUnExpected,
				ErrorCode: 404,
			}}, // 5
		{false, // false because it will give an error of unexpeced token type with value 2
			ast.ParamStatement{
				Src:       "{myparam2}",
				Name:      "myparam", // expected "myparam" because we don't allow integers to the parameter names.
				Type:      ast.ParamTypeString,
				ErrorCode: 404,
			}}, // 6
		{true,
			ast.ParamStatement{
				Src:  "{id:int even()}", // test param funcs without any arguments (LPAREN peek for RPAREN)
				Name: "id",
				Type: ast.ParamTypeInt,
				Funcs: []ast.ParamFunc{
					{
						Name: "even"},
				},
				ErrorCode: 404,
			}}, // 7
		{true,
			ast.ParamStatement{
				Src:       "{id:long else 404}",
				Name:      "id",
				Type:      ast.ParamTypeLong,
				ErrorCode: 404,
			}}, // 8
		{true,
			ast.ParamStatement{
				Src:       "{has:boolean else 404}",
				Name:      "has",
				Type:      ast.ParamTypeBoolean,
				ErrorCode: 404,
			}}, // 9

	}

	p := new(ParamParser)
	for i, tt := range tests {
		p.Reset(tt.expectedStatement.Src)
		resultStmt, err := p.Parse()

		if tt.valid && err != nil {
			t.Fatalf("tests[%d] - error %s", i, err.Error())
		} else if !tt.valid && err == nil {
			t.Fatalf("tests[%d] - expected to be a failure", i)
		}

		if resultStmt != nil { // is valid here
			if !reflect.DeepEqual(tt.expectedStatement, *resultStmt) {
				t.Fatalf("tests[%d] - wrong statement, expected and result differs. Details:\n%#v\n%#v", i, tt.expectedStatement, *resultStmt)
			}
		}

	}
}

func TestParse(t *testing.T) {
	tests := []struct {
		path               string
		valid              bool
		expectedStatements []ast.ParamStatement
	}{
		{"/api/users/{id:int min(1) max(5) else 404}", true,
			[]ast.ParamStatement{{
				Src:  "{id:int min(1) max(5) else 404}",
				Name: "id",
				Type: ast.ParamTypeInt,
				Funcs: []ast.ParamFunc{
					{
						Name: "min",
						Args: []ast.ParamFuncArg{1}},
					{
						Name: "max",
						Args: []ast.ParamFuncArg{5}},
				},
				ErrorCode: 404,
			},
			}}, // 0
		{"/admin/{id:int range(1,5)}", true,
			[]ast.ParamStatement{{
				Src:  "{id:int range(1,5)}",
				Name: "id",
				Type: ast.ParamTypeInt,
				Funcs: []ast.ParamFunc{
					{
						Name: "range",
						Args: []ast.ParamFuncArg{1, 5}},
				},
				ErrorCode: 404,
			},
			}}, // 1
		{"/files/{file:path contains(.)}", true,
			[]ast.ParamStatement{{
				Src:  "{file:path contains(.)}",
				Name: "file",
				Type: ast.ParamTypePath,
				Funcs: []ast.ParamFunc{
					{
						Name: "contains",
						Args: []ast.ParamFuncArg{"."}},
				},
				ErrorCode: 404,
			},
			}}, // 2
		{"/profile/{username:alphabetical}", true,
			[]ast.ParamStatement{{
				Src:       "{username:alphabetical}",
				Name:      "username",
				Type:      ast.ParamTypeAlphabetical,
				ErrorCode: 404,
			},
			}}, // 3
		{"/something/here/{myparam}", true,
			[]ast.ParamStatement{{
				Src:       "{myparam}",
				Name:      "myparam",
				Type:      ast.ParamTypeString,
				ErrorCode: 404,
			},
			}}, // 4
		{"/unexpected/{myparam_:thisianunexpected}", false,
			[]ast.ParamStatement{{
				Src:       "{myparam_:thisianunexpected}",
				Name:      "myparam_",
				Type:      ast.ParamTypeUnExpected,
				ErrorCode: 404,
			},
			}}, // 5
		{"/p2/{myparam2}", false, // false because it will give an error of unexpeced token type with value 2
			[]ast.ParamStatement{{
				Src:       "{myparam2}",
				Name:      "myparam", // expected "myparam" because we don't allow integers to the parameter names.
				Type:      ast.ParamTypeString,
				ErrorCode: 404,
			},
			}}, // 6
		{"/assets/{file:path}/invalid", false, // path should be in the end segment
			[]ast.ParamStatement{{
				Src:       "{file:path}",
				Name:      "file",
				Type:      ast.ParamTypePath,
				ErrorCode: 404,
			},
			}}, // 7
	}
	for i, tt := range tests {
		statements, err := Parse(tt.path)

		if tt.valid && err != nil {
			t.Fatalf("tests[%d] - error %s", i, err.Error())
		} else if !tt.valid && err == nil {
			t.Fatalf("tests[%d] - expected to be a failure", i)
		}
		for j := range statements {
			for l := range tt.expectedStatements {
				if !reflect.DeepEqual(tt.expectedStatements[l], *statements[j]) {
					t.Fatalf("tests[%d] - wrong statements, expected and result differs. Details:\n%#v\n%#v", i, tt.expectedStatements[l], *statements[j])
				}
			}
		}

	}
}
