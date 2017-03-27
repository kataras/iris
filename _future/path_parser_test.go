package router

import (
	"fmt"
	"reflect"
	"testing"
)

func testPathParser(source string, t *PathTmpl) error {
	result, err := ParsePath(source)
	if err != nil {
		return err
	}

	if expected, got := t.SegmentsLength, result.SegmentsLength; expected != got {
		return fmt.Errorf("expecting SegmentsLength to be %d but got %d", expected, got)
	}

	if expected, got := t.Params, result.Params; len(expected) != len(got) {
		return fmt.Errorf("expecting Params length to be %d but got %d", expected, got)
	}

	if !reflect.DeepEqual(*t, *result) {
		return fmt.Errorf("Expected and Result don't match. Details:\n%#v\nvs\n%#v\n", *t, *result)
	}

	return nil
}

func TestPathParser(t *testing.T) {
	// /users/{id:int}
	expected := &PathTmpl{
		SegmentsLength: 2,
		Params: []PathParamTmpl{
			PathParamTmpl{
				SegmentIndex: 1,
				Param: ParamTmpl{
					Name:           "id",
					Expression:     "int",
					FailStatusCode: 404,
					Macro:          MacroTmpl{Name: "int"},
				},
			},
		},
	}

	if err := testPathParser("/users/{id:int}", expected); err != nil {
		t.Fatal(err)
		return
	}

	// /api/users/{id:int range(1,5) !404}/other/{username:string contains(s) min(10) !402}
	expected = &PathTmpl{
		SegmentsLength: 5,
		Params: []PathParamTmpl{
			PathParamTmpl{
				SegmentIndex: 2,
				Param: ParamTmpl{
					Name:           "id",
					Expression:     "int range(1,5) !404",
					FailStatusCode: 404,
					Macro: MacroTmpl{Name: "int",
						Funcs: []MacroFuncTmpl{
							MacroFuncTmpl{Name: "range", Params: []string{"1", "5"}},
						},
					},
				},
			},
			PathParamTmpl{
				SegmentIndex: 4,
				Param: ParamTmpl{
					Name:           "username",
					Expression:     "string contains(s) min(10) !402",
					FailStatusCode: 402,
					Macro: MacroTmpl{Name: "string",
						Funcs: []MacroFuncTmpl{
							MacroFuncTmpl{Name: "contains", Params: []string{"s"}},
							MacroFuncTmpl{Name: "min", Params: []string{"10"}},
						},
					},
				},
			},
		},
	}
	if err := testPathParser("/api/users/{id:int range(1,5) !404}/other/{username:string contains(s) min(10) !402}", expected); err != nil {
		t.Fatal(err)
		return
	}
}
