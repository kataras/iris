package mvc2

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/kataras/iris/context"
)

type testUserStruct struct {
	ID       int64
	Username string
}

func testBinderFunc(ctx context.Context) testUserStruct {
	id, _ := ctx.Params().GetInt64("id")
	username := ctx.Params().Get("username")
	return testUserStruct{
		ID:       id,
		Username: username,
	}
}

type testBinderStruct struct{}

func (t *testBinderStruct) Bind(ctx context.Context) testUserStruct {
	return testBinderFunc(ctx)
}

func TestMakeFuncInputBinder(t *testing.T) {
	testMakeFuncInputBinder(t, testBinderFunc)
	testMakeFuncInputBinder(t, new(testBinderStruct))
}

func testMakeFuncInputBinder(t *testing.T, binder interface{}) {
	b, err := MakeFuncInputBinder(binder)
	if err != nil {
		t.Fatalf("failed to make binder: %v", err)
	}

	if b == nil {
		t.Fatalf("excepted non-nil *InputBinder but got nil")
	}

	if expected, got := reflect.TypeOf(testUserStruct{}), b.BindType; expected != got {
		t.Fatalf("expected type of the binder's return value to be: %T but got: %T", expected, got)
	}

	expected := testUserStruct{
		ID:       42,
		Username: "kataras",
	}
	ctx := context.NewContext(nil)
	ctx.Params().Set("id", fmt.Sprintf("%v", expected.ID))
	ctx.Params().Set("username", expected.Username)
	ctxValue := []reflect.Value{reflect.ValueOf(ctx)}
	v := b.BindFunc(ctxValue)
	if !v.CanInterface() {
		t.Fatalf("result of binder func cannot be interfaced: %#+v", v)
	}

	got, ok := v.Interface().(testUserStruct)
	if !ok {
		t.Fatalf("result of binder func should be a type of 'testUserStruct' but got: %#+v", v.Interface())
	}

	if got != expected {
		t.Fatalf("invalid result of binder func, expected: %v but got: %v", expected, got)
	}
}

func testCheck(t *testing.T, testName string, shouldPass bool, errString string) {
	if shouldPass && errString != "" {
		t.Fatalf("[%s] %s", testName, errString)
	}
	if !shouldPass && errString == "" {
		t.Fatalf("[%s] expected not to pass", testName)
	}
}

// TestGetBindersForInput will test two available binders, one for int
// and other for a string,
// the first input will contains both of them in the same order,
// the second will contain both of them as well but with a different order,
// the third will contain only the int input and should fail,
// the forth one will contain only the string input and should fail,
// the fifth one will contain two integers and should fail,
// the last one will contain a struct and should fail,
// that no of othe available binders will support it,
// so no len of the result should be zero there.
func TestGetBindersForInput(t *testing.T) {
	// binders
	var (
		stringBinder = MustMakeFuncInputBinder(func(ctx context.Context) string {
			return "a string"
		})
		intBinder = MustMakeFuncInputBinder(func(ctx context.Context) int {
			return 42
		})
	)
	// in
	var (
		stringType = reflect.TypeOf("string")
		intType    = reflect.TypeOf(1)
	)

	// 1
	testCheck(t, "test1", true, testGetBindersForInput(t, []*InputBinder{intBinder, stringBinder},
		[]interface{}{"a string", 42}, stringType, intType))
	availableBinders := []*InputBinder{stringBinder, intBinder} // different order than the fist test.
	// 2
	testCheck(t, "test2", true, testGetBindersForInput(t, availableBinders,
		[]interface{}{"a string", 42}, stringType, intType))
	// 3
	testCheck(t, "test-3-fail", false, testGetBindersForInput(t, availableBinders,
		[]interface{}{42}, stringType, intType))
	// 4
	testCheck(t, "test-4-fail", false, testGetBindersForInput(t, availableBinders,
		[]interface{}{"a string"}, stringType, intType))
	// 5
	testCheck(t, "test-5-fail", false, testGetBindersForInput(t, availableBinders,
		[]interface{}{42, 42}, stringType, intType))
	// 6
	testCheck(t, "test-6-fail", false, testGetBindersForInput(t, availableBinders,
		[]interface{}{testUserStruct{}}, stringType, intType))

}

func testGetBindersForInput(t *testing.T, binders []*InputBinder, expectingResults []interface{}, in ...reflect.Type) (errString string) {
	m := getBindersForInput(binders, in...)

	if expected, got := len(expectingResults), len(m); expected != got {
		return fmt.Sprintf("expected results length(%d) and valid binders length(%d) to be equal, so each input has one binder", expected, got)
	}

	ctxValue := []reflect.Value{reflect.ValueOf(context.NewContext(nil))}
	for idx, expected := range expectingResults {
		if m[idx] != nil {
			v := m[idx].BindFunc(ctxValue)
			if got := v.Interface(); got != expected {
				return fmt.Sprintf("expected result[%d] to be: %v but got: %v", idx, expected, got)
			}
		} else {
			t.Logf("m[%d] = nil on input = %v\n", idx, expected)
		}
	}

	return ""
}
