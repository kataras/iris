package main

import (
	"fmt"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/mvc"
)

// https://github.com/kataras/iris/issues/1706
// When you need that type of logic behind a request input,
// e.g. set default values, the right way to do that is
// to register a request-scoped dependency for that type.
// We have the `Context.ReadQuery(pointer)`
// which you can use to bind a struct value, that value can have default values.
// Here is how you could do that:
func main() {
	app := iris.New()

	mvcApp := mvc.New(app)
	{
		mvcApp.Register(paramsDependency)

		mvcApp.Handle(new(controller))
	}

	// http://localhost:8080/records?phone=random&order_by=DESC
	// http://localhost:8080/records?phone=random
	app.Listen(":8080")
}

type params struct {
	CallID     string `url:"phone"`
	ComDir     int    `url:"dir"`
	CaseUserID string `url:"on"`
	StartYear  int    `url:"sy"`
	EndYear    int    `url:"ey"`
	OrderBy    string `url:"order_by"`
	Offset     int    `url:"offset"`
}

// As we've read in the previous examples, the paramsDependency
// describes a request-scoped dependency.
// It should accept the iris context (or any previously registered or builtin dependency)
// and it should return the value which will be binded to the
// controller's methods (or fields) - see `GetRecords`.
var paramsDependency = func(ctx iris.Context) params {
	p := params{
		OrderBy: "ASC", // default value.
	}
	// Bind the URL values to the "p":
	ctx.ReadQuery(&p)
	// Or bind a specific URL value by giving a default value:
	// p.OrderBy = ctx.URLParamDefault("order_by", "ASC")
	//
	// OR make checks for default values after ReadXXX,
	// e.g. if p.OrderBy == "" {...}

	/* More options to read a request:
	// Bind the JSON request body to the "p":
	ctx.ReadJSON(&p)
	// Bind the Form to the "p":
	ctx.ReadForm(&p)
	// Bind any, based on the client's content-type header:
	ctx.ReadBody(&p)
	// Bind the http requests to a struct value:
	ctx.ReadHeader(&h)
	*/

	return p
}

type controller struct{}

func (c *controller) GetRecords(stru params) string {
	return fmt.Sprintf("%#+v\n", stru)
}
