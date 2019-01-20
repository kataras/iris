package macro

import (
	"reflect"
	"strconv"
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

func testEvaluatorRaw(t *testing.T, macroEvaluator *Macro, input string, expectedType reflect.Kind, pass bool, i int) {
	if macroEvaluator.Evaluator == nil && pass {
		return // if not evaluator defined then it should allow everything.
	}
	value, passed := macroEvaluator.Evaluator(input)
	if pass != passed {
		t.Fatalf("%s - tests[%d] - expecting[pass] %v but got %v", t.Name(), i, pass, passed)
	}

	if !passed {
		return
	}

	if value == nil && expectedType != reflect.Invalid {
		t.Fatalf("%s - tests[%d] - expecting[value] to not be nil", t.Name(), i)
	}

	if v := reflect.ValueOf(value); v.Kind() != expectedType {
		t.Fatalf("%s - tests[%d] - expecting[value.Kind] %v but got %v", t.Name(), i, expectedType, v.Kind())
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
		testEvaluatorRaw(t, String, tt.input, reflect.String, tt.pass, i)
	}
}

func TestIntEvaluatorRaw(t *testing.T) {
	x64 := strconv.IntSize == 64

	tests := []struct {
		pass  bool
		input string
	}{
		{false, "astring"},                                 // 0
		{false, "astringwith_numb3rS_and_symbol$"},         // 1
		{true, "32321"},                                    // 2
		{x64, "9223372036854775807" /* max int64 */},       // 3
		{x64, "-9223372036854775808" /* min int64 */},      // 4
		{false, "-18446744073709553213213213213213121615"}, // 5
		{false, "42 18446744073709551615"},                 // 6
		{false, "--42"},                                    // 7
		{false, "+42"},                                     // 8
		{false, "main.css"},                                // 9
		{false, "/assets/main.css"},                        // 10
	}

	for i, tt := range tests {
		testEvaluatorRaw(t, Int, tt.input, reflect.Int, tt.pass, i)
	}
}

func TestInt8EvaluatorRaw(t *testing.T) {
	tests := []struct {
		pass  bool
		input string
	}{
		{false, "astring"},                         // 0
		{false, "astringwith_numb3rS_and_symbol$"}, // 1
		{false, "32321"},                           // 2
		{true, "127" /* max int8 */},               // 3
		{true, "-128" /* min int8 */},              // 4
		{false, "128"},                             // 5
		{false, "-129"},                            // 6
		{false, "-18446744073709553213213213213213121615"}, // 7
		{false, "42 18446744073709551615"},                 // 8
		{false, "--42"},                                    // 9
		{false, "+42"},                                     // 10
		{false, "main.css"},                                // 11
		{false, "/assets/main.css"},                        // 12
	}

	for i, tt := range tests {
		testEvaluatorRaw(t, Int8, tt.input, reflect.Int8, tt.pass, i)
	}
}

func TestInt16EvaluatorRaw(t *testing.T) {
	tests := []struct {
		pass  bool
		input string
	}{
		{false, "astring"},                         // 0
		{false, "astringwith_numb3rS_and_symbol$"}, // 1
		{true, "32321"},                            // 2
		{true, "32767" /* max int16 */},            // 3
		{true, "-32768" /* min int16 */},           // 4
		{false, "-32769"},                          // 5
		{false, "32768"},                           // 6
		{false, "-18446744073709553213213213213213121615"}, // 7
		{false, "42 18446744073709551615"},                 // 8
		{false, "--42"},                                    // 9
		{false, "+42"},                                     // 10
		{false, "main.css"},                                // 11
		{false, "/assets/main.css"},                        // 12
	}

	for i, tt := range tests {
		testEvaluatorRaw(t, Int16, tt.input, reflect.Int16, tt.pass, i)
	}
}

func TestInt32EvaluatorRaw(t *testing.T) {
	tests := []struct {
		pass  bool
		input string
	}{
		{false, "astring"},                         // 0
		{false, "astringwith_numb3rS_and_symbol$"}, // 1
		{true, "32321"},                            // 2
		{true, "1"},                                // 3
		{true, "42"},                               // 4
		{true, "2147483647" /* max int32 */},       // 5
		{true, "-2147483648" /* min int32 */},      // 6
		{false, "-2147483649"},                     // 7
		{false, "2147483648"},                      // 8
		{false, "-18446744073709553213213213213213121615"}, // 9
		{false, "42 18446744073709551615"},                 // 10
		{false, "--42"},                                    // 11
		{false, "+42"},                                     // 12
		{false, "main.css"},                                // 13
		{false, "/assets/main.css"},                        // 14
	}

	for i, tt := range tests {
		testEvaluatorRaw(t, Int32, tt.input, reflect.Int32, tt.pass, i)
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
		testEvaluatorRaw(t, Int64, tt.input, reflect.Int64, tt.pass, i)
	}
}

func TestUintEvaluatorRaw(t *testing.T) {
	x64 := strconv.IntSize == 64

	tests := []struct {
		pass  bool
		input string
	}{
		{false, "astring"},                             // 0
		{false, "astringwith_numb3rS_and_symbol$"},     // 1
		{true, "32321"},                                // 2
		{true, "1"},                                    // 3
		{true, "42"},                                   // 4
		{x64, "18446744073709551615" /* max uint64 */}, // 5
		{true, "4294967295" /* max uint32 */},          // 6
		{false, "-2147483649"},                         // 7
		{true, "2147483648"},                           // 8
		{false, "-18446744073709553213213213213213121615"}, // 9
		{false, "42 18446744073709551615"},                 // 10
		{false, "--42"},                                    // 11
		{false, "+42"},                                     // 12
		{false, "main.css"},                                // 13
		{false, "/assets/main.css"},                        // 14
	}

	for i, tt := range tests {
		testEvaluatorRaw(t, Uint, tt.input, reflect.Uint, tt.pass, i)
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
		testEvaluatorRaw(t, Uint8, tt.input, reflect.Uint8, tt.pass, i)
	}
}

func TestUint16EvaluatorRaw(t *testing.T) {
	tests := []struct {
		pass  bool
		input string
	}{
		{false, "astring"},                         // 0
		{false, "astringwith_numb3rS_and_symbol$"}, // 1
		{true, "32321"},                            // 2
		{true, "65535" /* max uint16 */},           // 3
		{true, "0" /* min uint16 */},               // 4
		{false, "-32769"},                          // 5
		{true, "32768"},                            // 6
		{false, "-18446744073709553213213213213213121615"}, // 7
		{false, "42 18446744073709551615"},                 // 8
		{false, "--42"},                                    // 9
		{false, "+42"},                                     // 10
		{false, "main.css"},                                // 11
		{false, "/assets/main.css"},                        // 12
	}

	for i, tt := range tests {
		testEvaluatorRaw(t, Uint16, tt.input, reflect.Uint16, tt.pass, i)
	}
}

func TestUint32EvaluatorRaw(t *testing.T) {
	tests := []struct {
		pass  bool
		input string
	}{
		{false, "astring"},                         // 0
		{false, "astringwith_numb3rS_and_symbol$"}, // 1
		{true, "32321"},                            // 2
		{true, "1"},                                // 3
		{true, "42"},                               // 4
		{true, "4294967295" /* max uint32*/},       // 5
		{true, "0" /* min uint32 */},               // 6
		{false, "-2147483649"},                     // 7
		{true, "2147483648"},                       // 8
		{false, "-18446744073709553213213213213213121615"}, // 9
		{false, "42 18446744073709551615"},                 // 10
		{false, "--42"},                                    // 11
		{false, "+42"},                                     // 12
		{false, "main.css"},                                // 13
		{false, "/assets/main.css"},                        // 14
	}

	for i, tt := range tests {
		testEvaluatorRaw(t, Uint32, tt.input, reflect.Uint32, tt.pass, i)
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
		testEvaluatorRaw(t, Uint64, tt.input, reflect.Uint64, tt.pass, i)
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
		testEvaluatorRaw(t, Alphabetical, tt.input, reflect.String, tt.pass, i)
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
		testEvaluatorRaw(t, File, tt.input, reflect.String, tt.pass, i)
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
		testEvaluatorRaw(t, Path, tt.input, reflect.String, tt.pass, i)
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
	if !evalFunc([]string{"1", "[name1,name2]"}).Call([]reflect.Value{reflect.ValueOf("ok")})[0].Interface().(bool) {
		t.Fatalf("failed, it should fail already")
	}
}
