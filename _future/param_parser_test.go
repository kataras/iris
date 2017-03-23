package router

import (
	"fmt"
	"reflect"
	"testing"
)

func testParamParser(source string, t *ParamTmpl) error {
	result, err := ParseParam(source)
	if err != nil {
		return err
	}

	// first check of param name
	if expected, got := t.Name, result.Name; expected != got {
		return fmt.Errorf("Expecting Name to be '%s' but got '%s'", expected, got)
	}

	// first check on macro name
	if expected, got := t.Macro.Name, result.Macro.Name; expected != got {
		return fmt.Errorf("Expecting Macro.Name to be '%s' but got '%s'", expected, got)
	}

	// first check of length of the macro's funcs
	if expected, got := len(t.Macro.Funcs), len(result.Macro.Funcs); expected != got {
		return fmt.Errorf("Expecting Macro.Funs Len to be '%d' but got '%d'", expected, got)
	}

	// first check of the functions
	if len(t.Macro.Funcs) > 0 {
		if expected, got := t.Macro.Funcs[0].Name, result.Macro.Funcs[0].Name; expected != got {
			return fmt.Errorf("Expecting Macro.Funcs[0].Name to be '%s' but got '%s'", expected, got)
		}

		if expected, got := t.Macro.Funcs[0].Params, result.Macro.Funcs[0].Params; expected[0] != got[0] {
			return fmt.Errorf("Expecting Macro.Funcs[0].Params to be '%s' but got '%s'", expected, got)
		}
	}

	// and the final test for all, to be sure
	// here the details are more
	if !reflect.DeepEqual(*t, *result) {
		return fmt.Errorf("Expected and Result don't match. Details:\n%#v\n%#v", *t, *result)
	}
	return nil

}

func TestParamParser(t *testing.T) {

	// id:int
	expected := &ParamTmpl{
		Name:           "id",
		Expression:     "int",
		FailStatusCode: 404,
		Macro:          MacroTmpl{Name: "int"},
	}
	source := expected.Name + string(ParamNameSeperator) + expected.Expression
	if err := testParamParser(expected.Name+":"+expected.Expression, expected); err != nil {
		t.Fatal(err)
		return
	}

	// id:int range(1,5)
	expected = &ParamTmpl{
		Name:           "id",
		Expression:     "int range(1,5)",
		FailStatusCode: 404,
		Macro: MacroTmpl{Name: "int",
			Funcs: []MacroFuncTmpl{
				MacroFuncTmpl{Name: "range", Params: []string{"1", "5"}},
			},
		},
	}
	source = expected.Name + string(ParamNameSeperator) + expected.Expression
	if err := testParamParser(expected.Name+":"+expected.Expression, expected); err != nil {
		t.Fatal(err)
		return
	}

	// id:int min(1) max(5)
	expected = &ParamTmpl{
		Name:           "id",
		Expression:     "int min(1) max(5)",
		FailStatusCode: 404,
		Macro: MacroTmpl{Name: "int",
			Funcs: []MacroFuncTmpl{
				MacroFuncTmpl{Name: "min", Params: []string{"1"}},
				MacroFuncTmpl{Name: "max", Params: []string{"5"}},
			},
		},
	}
	source = expected.Name + string(ParamNameSeperator) + expected.Expression
	if err := testParamParser(expected.Name+":"+expected.Expression, expected); err != nil {
		t.Fatal(err)
		return
	}

	// username:string contains('blabla') max(20) !402
	expected = &ParamTmpl{
		Name:           "username",
		Expression:     "string contains(blabla) max(20) !402",
		FailStatusCode: 402,
		Macro: MacroTmpl{Name: "string",
			Funcs: []MacroFuncTmpl{
				MacroFuncTmpl{Name: "contains", Params: []string{"blabla"}},
				MacroFuncTmpl{Name: "max", Params: []string{"20"}},
			},
		},
	}
	source = expected.Name + string(ParamNameSeperator) + expected.Expression
	if err := testParamParser(source, expected); err != nil {
		t.Fatal(err)
		return
	}

}
