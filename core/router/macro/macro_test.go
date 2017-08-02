package macro

import (
	"reflect"
	"testing"
)

// Most important tests to look:
// ../parser/parser_test.go
// ../lexer/lexer_test.go

func TestGoodParamFunc(t *testing.T) {
	good1 := func(min int, max int) func(string) bool {
		return func(paramValue string) bool {
			return true
		}
	}

	good2 := func(min int, max int) func(string) bool {
		return func(paramValue string) bool {
			return true
		}
	}

	notgood1 := func(min int, max int) bool {
		return false
	}

	if !goodParamFunc(reflect.TypeOf(good1)) {
		t.Fatalf("expected good1 func to be good but it's not")
	}

	if !goodParamFunc(reflect.TypeOf(good2)) {
		t.Fatalf("expected good2 func to be good but it's not")
	}

	if goodParamFunc(reflect.TypeOf(notgood1)) {
		t.Fatalf("expected notgood1 func to be the worst")
	}
}

func TestGoodParamFuncName(t *testing.T) {
	tests := []struct {
		name string
		good bool
	}{
		{"range", true},
		{"_range", true},
		{"range_", true},
		{"r_ange", true},
		// numbers or other symbols are invalid.
		{"range1", false},
		{"2range", false},
		{"r@nge", false},
		{"rang3", false},
	}
	for i, tt := range tests {
		isGood := goodParamFuncName(tt.name)
		if tt.good && !isGood {
			t.Fatalf("tests[%d] - expecting valid name but got invalid for name %s", i, tt.name)
		} else if !tt.good && isGood {
			t.Fatalf("tests[%d] - expecting invalid name but got valid for name %s", i, tt.name)
		}
	}
}

func testEvaluatorRaw(macroEvaluator *Macro, input string, pass bool, i int, t *testing.T) {
	if got := macroEvaluator.Evaluator(input); pass != got {
		t.Fatalf("tests[%d] - expecting %v but got %v", i, pass, got)
	}
}

func TestStringEvaluatorRaw(t *testing.T) {
	f := NewMap()

	tests := []struct {
		pass  bool
		input string
	}{
		{true, "astring"},                         // 0
		{true, "astringwith_numb3rS_and_symbol$"}, // 1
		{true, "32321"},                           // 2
		{true, "main.css"},                        // 3
		{true, "/assets/main.css"},                // 4
		// false never
	} // 0

	for i, tt := range tests {
		testEvaluatorRaw(f.String, tt.input, tt.pass, i, t)
	}
}

func TestIntEvaluatorRaw(t *testing.T) {
	f := NewMap()

	tests := []struct {
		pass  bool
		input string
	}{
		{false, "astring"},                         // 0
		{false, "astringwith_numb3rS_and_symbol$"}, // 1
		{true, "32321"},                            // 2
		{false, "main.css"},                        // 3
		{false, "/assets/main.css"},                // 4
	}

	for i, tt := range tests {
		testEvaluatorRaw(f.Int, tt.input, tt.pass, i, t)
	}
}

func TestAlphabeticalEvaluatorRaw(t *testing.T) {
	f := NewMap()

	tests := []struct {
		pass  bool
		input string
	}{
		{true, "astring"},                          // 0
		{false, "astringwith_numb3rS_and_symbol$"}, // 1
		{false, "32321"},                           // 2
		{false, "main.css"},                        // 3
		{false, "/assets/main.css"},                // 4
	}

	for i, tt := range tests {
		testEvaluatorRaw(f.Alphabetical, tt.input, tt.pass, i, t)
	}
}

func TestFileEvaluatorRaw(t *testing.T) {
	f := NewMap()

	tests := []struct {
		pass  bool
		input string
	}{
		{true, "astring"},                          // 0
		{false, "astringwith_numb3rS_and_symbol$"}, // 1
		{true, "32321"},                            // 2
		{true, "main.css"},                         // 3
		{false, "/assets/main.css"},                // 4
	}

	for i, tt := range tests {
		testEvaluatorRaw(f.File, tt.input, tt.pass, i, t)
	}
}

func TestPathEvaluatorRaw(t *testing.T) {
	f := NewMap()

	pathTests := []struct {
		pass  bool
		input string
	}{
		{true, "astring"},                         // 0
		{true, "astringwith_numb3rS_and_symbol$"}, // 1
		{true, "32321"},                           // 2
		{true, "main.css"},                        // 3
		{true, "/assets/main.css"},                // 4
		{true, "disk/assets/main.css"},            // 5
	}

	for i, tt := range pathTests {
		testEvaluatorRaw(f.Path, tt.input, tt.pass, i, t)
	}
}

// func TestMapRegisterFunc(t *testing.T) {
// 	m := NewMap()
// 	m.String.RegisterFunc("prefix", func(prefix string) EvaluatorFunc {
// 		return func(paramValue string) bool {
// 			return strings.HasPrefix(paramValue, prefix)
// 		}
// 	})

// 	p, err := Parse("/user/@iris")
// 	if err != nil {
// 		t.Fatalf(err)
// 	}

// 	// 	p.Params = append(p.)

// 	testEvaluatorRaw(m.String, p.Src, false, 0, t)
// }
