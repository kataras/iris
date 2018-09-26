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

	good2 := func(min uint64, max uint64) func(string) bool {
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

func testEvaluatorRaw(t *testing.T, macroEvaluator *Macro, input string, pass bool, i int) {
	if got := macroEvaluator.Evaluator(input); pass != got {
		t.Fatalf("%s - tests[%d] - expecting %v but got %v", t.Name(), i, pass, got)
	}
}

func TestStringEvaluatorRaw(t *testing.T) {
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
		testEvaluatorRaw(t, String, tt.input, tt.pass, i)
	}
}

func TestNumberEvaluatorRaw(t *testing.T) {
	tests := []struct {
		pass  bool
		input string
	}{
		{false, "astring"},                                // 0
		{false, "astringwith_numb3rS_and_symbol$"},        // 1
		{true, "32321"},                                   // 2
		{true, "18446744073709551615"},                    // 3
		{true, "-18446744073709551615"},                   // 4
		{true, "-18446744073709553213213213213213121615"}, // 5
		{false, "42 18446744073709551615"},                // 6
		{false, "--42"},                                   // 7
		{false, "+42"},                                    // 8
		{false, "main.css"},                               // 9
		{false, "/assets/main.css"},                       // 10
	}

	for i, tt := range tests {
		testEvaluatorRaw(t, Number, tt.input, tt.pass, i)
	}
}

func TestInt64EvaluatorRaw(t *testing.T) {
	tests := []struct {
		pass  bool
		input string
	}{
		{false, "astring"},                                 // 0
		{false, "astringwith_numb3rS_and_symbol$"},         // 1
		{false, "18446744073709551615"},                    // 2
		{false, "92233720368547758079223372036854775807"},  // 3
		{false, "9223372036854775808 9223372036854775808"}, // 4
		{false, "main.css"},                                // 5
		{false, "/assets/main.css"},                        // 6
		{true, "9223372036854775807"},                      // 7
		{true, "-9223372036854775808"},                     // 8
		{true, "-0"},                                       // 9
		{true, "1"},                                        // 10
		{true, "-042"},                                     // 11
		{true, "142"},                                      // 12
	}

	for i, tt := range tests {
		testEvaluatorRaw(t, Int64, tt.input, tt.pass, i)
	}
}

func TestUint8EvaluatorRaw(t *testing.T) {
	tests := []struct {
		pass  bool
		input string
	}{
		{false, "astring"},                                 // 0
		{false, "astringwith_numb3rS_and_symbol$"},         // 1
		{false, "-9223372036854775808"},                    // 2
		{false, "main.css"},                                // 3
		{false, "/assets/main.css"},                        // 4
		{false, "92233720368547758079223372036854775807"},  // 5
		{false, "9223372036854775808 9223372036854775808"}, // 6
		{false, "-1"},                                      // 7
		{false, "-0"},                                      // 8
		{false, "+1"},                                      // 9
		{false, "18446744073709551615"},                    // 10
		{false, "9223372036854775807"},                     // 11
		{false, "021"},                                     // 12 - no leading zeroes are allowed.
		{false, "300"},                                     // 13
		{true, "0"},                                        // 14
		{true, "255"},                                      // 15
		{true, "21"},                                       // 16
	}

	for i, tt := range tests {
		testEvaluatorRaw(t, Uint8, tt.input, tt.pass, i)
	}
}

func TestUint64EvaluatorRaw(t *testing.T) {
	tests := []struct {
		pass  bool
		input string
	}{
		{false, "astring"},                                 // 0
		{false, "astringwith_numb3rS_and_symbol$"},         // 1
		{false, "-9223372036854775808"},                    // 2
		{false, "main.css"},                                // 3
		{false, "/assets/main.css"},                        // 4
		{false, "92233720368547758079223372036854775807"},  // 5
		{false, "9223372036854775808 9223372036854775808"}, // 6
		{false, "-1"},                                      // 7
		{false, "-0"},                                      // 8
		{false, "+1"},                                      // 9
		{true, "18446744073709551615"},                     // 10
		{true, "9223372036854775807"},                      // 11
		{true, "0"},                                        // 12
	}

	for i, tt := range tests {
		testEvaluatorRaw(t, Uint64, tt.input, tt.pass, i)
	}
}

func TestAlphabeticalEvaluatorRaw(t *testing.T) {
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
		testEvaluatorRaw(t, Alphabetical, tt.input, tt.pass, i)
	}
}

func TestFileEvaluatorRaw(t *testing.T) {
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
		testEvaluatorRaw(t, File, tt.input, tt.pass, i)
	}
}

func TestPathEvaluatorRaw(t *testing.T) {
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
		testEvaluatorRaw(t, Path, tt.input, tt.pass, i)
	}
}

func TestConvertBuilderFunc(t *testing.T) {
	fn := func(min uint64, slice []string) func(string) bool {
		return func(paramValue string) bool {
			if expected, got := "ok", paramValue; expected != got {
				t.Fatalf("paramValue is not the expected one: %s vs %s", expected, got)
			}

			if expected, got := uint64(1), min; expected != got {
				t.Fatalf("min argument is not the expected one: %d vs %d", expected, got)
			}

			if expected, got := []string{"name1", "name2"}, slice; len(expected) == len(got) {
				if expected, got := "name1", slice[0]; expected != got {
					t.Fatalf("slice argument[%d] does not contain the expected value: %s vs %s", 0, expected, got)
				}

				if expected, got := "name2", slice[1]; expected != got {
					t.Fatalf("slice argument[%d] does not contain the expected value: %s vs %s", 1, expected, got)
				}
			} else {
				t.Fatalf("slice argument is not the expected one, the length is difference: %d vs %d", len(expected), len(got))
			}

			return true
		}
	}

	evalFunc := convertBuilderFunc(fn)

	if !evalFunc([]string{"1", "[name1,name2]"})("ok") {
		t.Fatalf("failed, it should fail already")
	}
}
