package hero_test

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/kataras/iris/v12"
	. "github.com/kataras/iris/v12/hero"
	"github.com/kataras/iris/v12/httptest"
)

var errTyp = reflect.TypeOf((*error)(nil)).Elem()

// isError returns true if "typ" is type of `error`.
func isError(typ reflect.Type) bool {
	return typ.Implements(errTyp)
}

type (
	testInput struct {
		Name string `json:"name"`
	}

	testOutput struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}
)

var (
	fn = func(id int, in testInput) testOutput {
		return testOutput{
			ID:   id,
			Name: in.Name,
		}
	}

	expectedOutput = testOutput{
		ID:   42,
		Name: "makis",
	}

	input = testInput{
		Name: "makis",
	}
)

func TestContainerHandler(t *testing.T) {
	app := iris.New()

	c := New()
	postHandler := c.Handler(fn)
	app.Post("/{id:int}", postHandler)

	e := httptest.New(t, app)
	path := fmt.Sprintf("/%d", expectedOutput.ID)
	e.POST(path).WithJSON(input).Expect().Status(httptest.StatusOK).JSON().IsEqual(expectedOutput)
}

func TestContainerInject(t *testing.T) {
	c := New()

	expected := testInput{Name: "test"}
	c.Register(expected)
	c.Register(&expected)

	// struct value.
	var got1 testInput
	if err := c.Inject(&got1); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(expected, got1) {
		t.Fatalf("[struct value] expected: %#+v but got: %#+v", expected, got1)
	}

	// ptr.
	var got2 *testInput
	if err := c.Inject(&got2); err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(&expected, got2) {
		t.Fatalf("[ptr] expected: %#+v but got: %#+v", &expected, got2)
	}

	// register implementation, expect interface.
	expected3 := &testServiceImpl{prefix: "prefix: "}
	c.Register(expected3)

	var got3 testService
	if err := c.Inject(&got3); err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(expected3, got3) {
		t.Fatalf("[service] expected: %#+v but got: %#+v", expected3, got3)
	}
}

func TestContainerUseResultHandler(t *testing.T) {
	c := New()
	resultLogger := func(next ResultHandler) ResultHandler {
		return func(ctx iris.Context, v interface{}) error {
			t.Logf("%#+v", v)
			return next(ctx, v)
		}
	}

	c.UseResultHandler(resultLogger)
	expectedResponse := map[string]interface{}{"injected": true}
	c.UseResultHandler(func(next ResultHandler) ResultHandler {
		return func(ctx iris.Context, v interface{}) error {
			return next(ctx, expectedResponse)
		}
	})
	c.UseResultHandler(resultLogger)

	handler := c.Handler(func(id int) testOutput {
		return testOutput{
			ID:   id,
			Name: "kataras",
		}
	})

	app := iris.New()
	app.Get("/{id:int}", handler)

	e := httptest.New(t, app)
	e.GET("/42").Expect().Status(httptest.StatusOK).JSON().IsEqual(expectedResponse)
}
