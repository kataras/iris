// package main contains an example on how to use the ReadQuery,
// same way you can do the ReadJSON & ReadProtobuf and e.t.c.
package main

import (
	"github.com/kataras/iris/v12"
)

type MyType struct {
	Name string `url:"name,required"`
	Age  int    `url:"age"`
}

func main() {
	app := iris.New()

	app.Get("/", func(ctx iris.Context) {
		var t MyType
		err := ctx.ReadQuery(&t)

		// To ignore errors of "required" or when unexpected values are passed to the query,
		// use the iris.IsErrPath.
		// It can be ignored, e.g:
		// if err!=nil && !iris.IsErrPath(err) { ... return }
		//
		// To receive an error on EMPTY query when ReadQuery is called
		// you should enable the `FireEmptyFormError/WithEmptyFormError` ( see below).
		// To check for the empty error you simple compare the error with the ErrEmptyForm, e.g.:
		// err == iris.ErrEmptyForm, so, to ignore both path and empty errors, you do:
		// if err!=nil && err != iris.ErrEmptyForm && !iris.IsErrPath(err) { ctx.StopWithError(...); return }
		if err != nil {
			ctx.StopWithError(iris.StatusInternalServerError, err)
			return
		}

		ctx.Writef("MyType: %#v", t)
	})

	app.Get("/simple", func(ctx iris.Context) {
		names := ctx.URLParamSlice("name")
		ctx.Writef("names: %v", names)
	})

	// http://localhost:8080?name=iris&age=3
	// http://localhost:8080/simple?name=john&name=doe&name=kataras
	//
	// Note: this `WithEmptyFormError` will give an error if the query was empty.
	app.Listen(":8080", iris.WithEmptyFormError)
}
