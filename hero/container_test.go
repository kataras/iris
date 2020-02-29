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

func TestHeroHandler(t *testing.T) {
	app := iris.New()

	b := New()
	postHandler := b.Handler(fn)
	app.Post("/{id:int}", postHandler)

	e := httptest.New(t, app)
	path := fmt.Sprintf("/%d", expectedOutput.ID)
	e.POST(path).WithJSON(input).Expect().Status(httptest.StatusOK).JSON().Equal(expectedOutput)
}
