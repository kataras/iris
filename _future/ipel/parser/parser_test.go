package parser

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"gopkg.in/kataras/iris.v6/_future/ipel/ast"
	"gopkg.in/kataras/iris.v6/_future/ipel/lexer"
)

func TestParseError(t *testing.T) {
	// fail
	illegalChar := '$'

	input := "{id" + string(illegalChar) + "int range(1,5) else 404}"
	l := lexer.New(input)
	p := New(l)

	_, err := p.Parse()

	if err == nil {
		t.Fatalf("expecting not empty error on input '%s'", input)
	}

	// println(input[8:9])
	// println(input[13:17])

	illIdx := strings.IndexRune(input, illegalChar)
	expectedErr := fmt.Sprintf("[%d:%d] illegal token: %s", illIdx, illIdx, "$")
	if got := err.Error(); got != expectedErr {
		t.Fatalf("expecting error to be '%s' but got: %s", expectedErr, got)
	}
	//

	// success
	input2 := "{id:int range(1,5) else 404}"
	l2 := lexer.New(input2)
	p2 := New(l2)

	_, err = p2.Parse()

	if err != nil {
		t.Fatalf("expecting empty error on input '%s', but got: %s", input2, err.Error())
	}
	//
}

func TestParse(t *testing.T) {
	tests := []struct {
		input             string
		valid             bool
		expectedStatement ast.ParamStatement
	}{
		{"{id:int min(1) max(5) else 404}", true,
			ast.ParamStatement{
				Name: "id",
				Type: ast.ParamTypeInt,
				Funcs: []ast.ParamFunc{
					ast.ParamFunc{
						Name: "min",
						Args: []ast.ParamFuncArg{1}},
					ast.ParamFunc{
						Name: "max",
						Args: []ast.ParamFuncArg{5}},
				},
				ErrorCode: 404,
			}}, // 0
		{"{id:int range(1,5)}", true,
			ast.ParamStatement{
				Name: "id",
				Type: ast.ParamTypeInt,
				Funcs: []ast.ParamFunc{
					ast.ParamFunc{
						Name: "range",
						Args: []ast.ParamFuncArg{1, 5}},
				},
				ErrorCode: 404,
			}}, // 1
		{"{file:path contains(.)}", true,
			ast.ParamStatement{
				Name: "file",
				Type: ast.ParamTypePath,
				Funcs: []ast.ParamFunc{
					ast.ParamFunc{
						Name: "contains",
						Args: []ast.ParamFuncArg{"."}},
				},
				ErrorCode: 404,
			}}, // 2
		{"{username:alphabetical", true,
			ast.ParamStatement{
				Name:      "username",
				Type:      ast.ParamTypeAlphabetical,
				ErrorCode: 404,
			}}, // 3
		{"{username:thisianunexpected", false,
			ast.ParamStatement{
				Name:      "username",
				Type:      ast.ParamTypeUnExpected,
				ErrorCode: 404,
			}}, // 4
	}

	for i, tt := range tests {
		l := lexer.New(tt.input)
		p := New(l)
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
