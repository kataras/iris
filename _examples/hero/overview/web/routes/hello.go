// file: web/routes/hello.go

package routes

import (
	"errors"

	"github.com/kataras/iris/hero"
)

var helloView = hero.View{
	Name: "hello/index.html",
	Data: map[string]interface{}{
		"Title":     "Hello Page",
		"MyMessage": "Welcome to my awesome website",
	},
}

// Hello will return a predefined view with bind data.
//
// `hero.Result` is just an interface with a `Dispatch` function.
// `hero.Response` and `hero.View` are the builtin result type dispatchers
// you can even create custom response dispatchers by
// implementing the `github.com/kataras/iris/hero#Result` interface.
func Hello() hero.Result {
	return helloView
}

// you can define a standard error in order to re-use anywhere in your app.
var errBadName = errors.New("bad name")

// you can just return it as error or even better
// wrap this error with an hero.Response to make it an hero.Result compatible type.
var badName = hero.Response{Err: errBadName, Code: 400}

// HelloName returns a "Hello {name}" response.
// Demos:
// curl -i http://localhost:8080/hello/iris
// curl -i http://localhost:8080/hello/anything
func HelloName(name string) hero.Result {
	if name != "iris" {
		return badName
	}

	// return hero.Response{Text: "Hello " + name} OR:
	return hero.View{
		Name: "hello/name.html",
		Data: name,
	}
}
