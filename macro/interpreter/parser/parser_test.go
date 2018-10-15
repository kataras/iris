package parser

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/kataras/iris/macro/interpreter/ast"
)

type simpleParamType string

func (pt simpleParamType) Indent() string { return string(pt) }

type masterParamType simpleParamType

func (pt masterParamType) Indent() string { return string(pt) }
func (pt masterParamType) Master() bool   { return true }

type wildcardParamType string

func (pt wildcardParamType) Indent() string { return string(pt) }
func (pt wildcardParamType) Trailing() bool { return true }

type aliasedParamType []string

func (pt aliasedParamType) Indent() string { return string(pt[0]) }
func (pt aliasedParamType) Alias() string  { return pt[1] }

var (
	paramTypeString       = masterParamType("string")
	paramTypeNumber       = aliasedParamType{"number", "int"}
	paramTypeInt64        = aliasedParamType{"int64", "long"}
	paramTypeUint8        = simpleParamType("uint8")
	paramTypeUint64       = simpleParamType("uint64")
	paramTypeBool         = aliasedParamType{"bool", "boolean"}
	paramTypeAlphabetical = simpleParamType("alphabetical")
	paramTypeFile         = simpleParamType("file")
	paramTypePath         = wildcardParamType("path")
)

var testParamTypes = []ast.ParamType{
	paramTypeString,
	paramTypeNumber, paramTypeInt64, paramTypeUint8, paramTypeUint64,
	paramTypeBool,
	paramTypeAlphabetical, paramTypeFile, paramTypePath,
}

func TestParseParamError(t *testing.T) {
	// fail
	illegalChar := '$'

	input := "{id" + string(illegalChar) + "int range(1,5) else 404}"
	p := NewParamParser(input)

	_, err := p.Parse(testParamTypes)

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
	input2 := "{id:uint64 range(1,5) else 404}"
	p.Reset(input2)
	_, err = p.Parse(testParamTypes)

	if err != nil {
		t.Fatalf("expecting empty error on input '%s', but got: %s", input2, err.Error())
	}
	//
}

// mustLookupParamType same as `ast.LookupParamType` but it panics if "indent" does not match with a valid Param Type.
func mustLookupParamType(indent string) ast.ParamType {
	pt, found := ast.LookupParamType(indent, testParamTypes...)
	if !found {
		panic("param type '" + indent + "' is not part of the provided param types")
	}

	return pt
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
				Type: mustLookupParamType("number"),
				Funcs: []ast.ParamFunc{
					{
						Name: "min",
						Args: []string{"1"}},
					{
						Name: "max",
						Args: []string{"5"}},
				},
				ErrorCode: 404,
			}}, // 0

		{true,
			ast.ParamStatement{
				// test alias of int.
				Src:  "{id:number range(1,5)}",
				Name: "id",
				Type: mustLookupParamType("number"),
				Funcs: []ast.ParamFunc{
					{
						Name: "range",
						Args: []string{"1", "5"}},
				},
				ErrorCode: 404,
			}}, // 1
		{true,
			ast.ParamStatement{
				Src:  "{file:path contains(.)}",
				Name: "file",
				Type: mustLookupParamType("path"),
				Funcs: []ast.ParamFunc{
					{
						Name: "contains",
						Args: []string{"."}},
				},
				ErrorCode: 404,
			}}, // 2
		{true,
			ast.ParamStatement{
				Src:       "{username:alphabetical}",
				Name:      "username",
				Type:      mustLookupParamType("alphabetical"),
				ErrorCode: 404,
			}}, // 3
		{true,
			ast.ParamStatement{
				Src:       "{myparam}",
				Name:      "myparam",
				Type:      mustLookupParamType("string"),
				ErrorCode: 404,
			}}, // 4
		{false,
			ast.ParamStatement{
				Src:       "{myparam_:thisianunexpected}",
				Name:      "myparam_",
				Type:      nil,
				ErrorCode: 404,
			}}, // 5
		{true,
			ast.ParamStatement{
				Src:       "{myparam2}",
				Name:      "myparam2", // we now allow integers to the parameter names.
				Type:      ast.GetMasterParamType(testParamTypes...),
				ErrorCode: 404,
			}}, // 6
		{true,
			ast.ParamStatement{
				Src:  "{id:int even()}", // test param funcs without any arguments (LPAREN peek for RPAREN)
				Name: "id",
				Type: mustLookupParamType("number"),
				Funcs: []ast.ParamFunc{
					{
						Name: "even"},
				},
				ErrorCode: 404,
			}}, // 7
		{true,
			ast.ParamStatement{
				Src:       "{id:int64 else 404}",
				Name:      "id",
				Type:      mustLookupParamType("int64"),
				ErrorCode: 404,
			}}, // 8
		{true,
			ast.ParamStatement{
				Src:       "{id:long else 404}", // backwards-compatible test.
				Name:      "id",
				Type:      mustLookupParamType("int64"),
				ErrorCode: 404,
			}}, // 9
		{true,
			ast.ParamStatement{
				Src:       "{id:long else 404}",
				Name:      "id",
				Type:      mustLookupParamType("int64"), // backwards-compatible test of LookupParamType.
				ErrorCode: 404,
			}}, // 10
		{true,
			ast.ParamStatement{
				Src:       "{has:bool else 404}",
				Name:      "has",
				Type:      mustLookupParamType("bool"),
				ErrorCode: 404,
			}}, // 11
		{true,
			ast.ParamStatement{
				Src:       "{has:boolean else 404}", // backwards-compatible test.
				Name:      "has",
				Type:      mustLookupParamType("bool"),
				ErrorCode: 404,
			}}, // 12

	}

	p := new(ParamParser)
	for i, tt := range tests {
		p.Reset(tt.expectedStatement.Src)
		resultStmt, err := p.Parse(testParamTypes)

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
				Type: paramTypeNumber,
				Funcs: []ast.ParamFunc{
					{
						Name: "min",
						Args: []string{"1"}},
					{
						Name: "max",
						Args: []string{"5"}},
				},
				ErrorCode: 404,
			},
			}}, // 0
		{"/admin/{id:uint64 range(1,5)}", true,
			[]ast.ParamStatement{{
				Src:  "{id:uint64 range(1,5)}",
				Name: "id",
				Type: paramTypeUint64,
				Funcs: []ast.ParamFunc{
					{
						Name: "range",
						Args: []string{"1", "5"}},
				},
				ErrorCode: 404,
			},
			}}, // 1
		{"/files/{file:path contains(.)}", true,
			[]ast.ParamStatement{{
				Src:  "{file:path contains(.)}",
				Name: "file",
				Type: paramTypePath,
				Funcs: []ast.ParamFunc{
					{
						Name: "contains",
						Args: []string{"."}},
				},
				ErrorCode: 404,
			},
			}}, // 2
		{"/profile/{username:alphabetical}", true,
			[]ast.ParamStatement{{
				Src:       "{username:alphabetical}",
				Name:      "username",
				Type:      paramTypeAlphabetical,
				ErrorCode: 404,
			},
			}}, // 3
		{"/something/here/{myparam}", true,
			[]ast.ParamStatement{{
				Src:       "{myparam}",
				Name:      "myparam",
				Type:      paramTypeString,
				ErrorCode: 404,
			},
			}}, // 4
		{"/unexpected/{myparam_:thisianunexpected}", false,
			[]ast.ParamStatement{{
				Src:       "{myparam_:thisianunexpected}",
				Name:      "myparam_",
				Type:      nil,
				ErrorCode: 404,
			},
			}}, // 5
		{"/p2/{myparam2}", true,
			[]ast.ParamStatement{{
				Src:       "{myparam2}",
				Name:      "myparam2", // we now allow integers to the parameter names.
				Type:      paramTypeString,
				ErrorCode: 404,
			},
			}}, // 6
		{"/assets/{file:path}/invalid", false, // path should be in the end segment
			[]ast.ParamStatement{{
				Src:       "{file:path}",
				Name:      "file",
				Type:      paramTypePath,
				ErrorCode: 404,
			},
			}}, // 7
	}
	for i, tt := range tests {
		statements, err := Parse(tt.path, testParamTypes)

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
