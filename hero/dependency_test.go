package hero_test

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/kataras/iris/v12/context"
	. "github.com/kataras/iris/v12/hero"
)

type testDependencyTest struct {
	Dependency interface{}
	Expected   interface{}
}

func TestDependency(t *testing.T) {
	var tests = []testDependencyTest{
		{
			Dependency: "myValue",
			Expected:   "myValue",
		},
		{
			Dependency: struct{ Name string }{"name"},
			Expected:   struct{ Name string }{"name"},
		},
		{
			Dependency: func(*context.Context, *Input) (reflect.Value, error) {
				return reflect.ValueOf(42), nil
			},
			Expected: 42,
		},
		{
			Dependency: DependencyHandler(func(*context.Context, *Input) (reflect.Value, error) {
				return reflect.ValueOf(255), nil
			}),
			Expected: 255,
		},
		{
			Dependency: func(*context.Context) (reflect.Value, error) {
				return reflect.ValueOf("OK without Input"), nil
			},
			Expected: "OK without Input",
		},
		{

			Dependency: func(*context.Context, ...string) (reflect.Value, error) {
				return reflect.ValueOf("OK variadic ignored"), nil
			},
			Expected: "OK variadic ignored",
		},
		{

			Dependency: func(*context.Context) reflect.Value {
				return reflect.ValueOf("OK without Input and error")
			},
			Expected: "OK without Input and error",
		},
		{

			Dependency: func(*context.Context, ...int) reflect.Value {
				return reflect.ValueOf("OK without error and variadic ignored")
			},
			Expected: "OK without error and variadic ignored",
		},
		{

			Dependency: func(*context.Context) interface{} {
				return "1"
			},
			Expected: "1",
		},
		{

			Dependency: func(*context.Context) interface{} {
				return false
			},
			Expected: false,
		},
	}

	testDependencies(t, tests)
}

// Test dependencies that depend on previous one(s).
func TestDependentDependency(t *testing.T) {
	msgBody := "prefix: it is a deep dependency"
	newMsgBody := msgBody + " new"
	var tests = []testDependencyTest{
		// test three level depth and error.
		{ // 0
			Dependency: &testServiceImpl{prefix: "prefix:"},
			Expected:   &testServiceImpl{prefix: "prefix:"},
		},
		{ // 1
			Dependency: func(service testService) testMessage {
				return testMessage{Body: service.Say("it is a deep") + " dependency"}
			},
			Expected: testMessage{Body: msgBody},
		},
		{ // 2
			Dependency: func(msg testMessage) string {
				return msg.Body
			},
			Expected: msgBody,
		},
		{ // 3
			Dependency: func(msg testMessage) error {
				return fmt.Errorf(msg.Body)
			},
			Expected: fmt.Errorf(msgBody),
		},
		// Test depend on more than one previous registered dependencies and require a before-previous one.
		{ // 4
			Dependency: func(body string, msg testMessage) string {
				if body != msg.Body {
					t.Fatalf("body[%s] != msg.Body[%s]", body, msg.Body)
				}

				return body + " new"
			},
			Expected: newMsgBody,
		},
		// Test dependency order by expecting the first <string> returning value and not the later-on registered dependency(#4).
		// 5
		{
			Dependency: func(body string) string {
				return body
			},
			Expected: newMsgBody,
		},
	}

	testDependencies(t, tests)
}

func testDependencies(t *testing.T, tests []testDependencyTest) {
	t.Helper()

	c := New()
	for i, tt := range tests {
		d := c.Register(tt.Dependency)

		if d == nil {
			t.Fatalf("[%d] expected %#+v to be converted to a valid dependency", i, tt)
		}

		val, err := d.Handle(context.NewContext(nil), &Input{})

		if expectError := isError(reflect.TypeOf(tt.Expected)); expectError {
			val = reflect.ValueOf(err)
			err = nil
		}

		if err != nil {
			t.Fatalf("[%d] expected a nil error but got: %v", i, err)
		}

		if !val.CanInterface() {
			t.Fatalf("[%d] expected output value to be accessible: %T", i, val)
		}

		if expected, got := fmt.Sprintf("%#+v", tt.Expected), fmt.Sprintf("%#+v", val.Interface()); expected != got {
			t.Fatalf("[%d] expected return value to be:\n%s\nbut got:\n%s", i, expected, got)
		}

		// t.Logf("[%d] %s", i, d)
		// t.Logf("[%d] output: %#+v", i, val.Interface())
	}
}
