package main

import (
	"fmt"

	"github.com/kataras/iris/v12"

	"github.com/kataras/iris/v12/mvc"
)

func main() {
	app := iris.New()
	app.Logger().SetLevel("debug")

	mvcApp := mvc.New(app)
	// To all controllers, it can optionally be overridden per-controller
	// if the controller contains the `HandleError(ctx iris.Context, err error)` function.
	//
	mvcApp.HandleError(func(ctx iris.Context, err error) {
		ctx.HTML(fmt.Sprintf("<b>%s</b>", err.Error()))
	})
	//
	mvcApp.Handle(new(myController))

	// http://localhost:8080
	app.Listen(":8080")
}

type myController struct {
}

// overriddes the mvcApp.HandleError function.
func (c *myController) HandleError(ctx iris.Context, err error) {
	ctx.HTML(fmt.Sprintf("<i>%s</i>", err.Error()))
}

func (c *myController) Get() error {
	return fmt.Errorf("error here")
}
