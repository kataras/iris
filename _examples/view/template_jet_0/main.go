// Package main shows how to use jet template parser with ease using the Iris built-in Jet view engine.
// This example is customized fork of https://github.com/CloudyKit/jet/tree/master/examples/todos, so you can
// notice the differences side by side.
package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/view"
)

type tTODO struct {
	Text string
	Done bool
}

type doneTODOs struct {
	list map[string]*tTODO
	keys []string
	len  int
	i    int
}

func (dt *doneTODOs) New(todos map[string]*tTODO) *doneTODOs {
	dt.len = len(todos)
	for k := range todos {
		dt.keys = append(dt.keys, k)
	}
	dt.list = todos
	return dt
}

// Range satisfies the jet.Ranger interface and only returns TODOs that are done,
// even when the list contains TODOs that are not done.
func (dt *doneTODOs) Range() (reflect.Value, reflect.Value, bool) {
	for dt.i < dt.len {
		key := dt.keys[dt.i]
		dt.i++
		if dt.list[key].Done {
			return reflect.ValueOf(key), reflect.ValueOf(dt.list[key]), false
		}
	}
	return reflect.Value{}, reflect.Value{}, true
}

// Note: jet version 4 requires this.
func (dt *doneTODOs) ProvidesIndex() bool { return true }

func (dt *doneTODOs) Render(r *view.JetRuntime) {
	r.Write([]byte("custom renderer"))
}

// Render implements jet.Renderer interface
func (t *tTODO) Render(r *view.JetRuntime) {
	done := "yes"
	if !t.Done {
		done = "no"
	}
	r.Write([]byte(fmt.Sprintf("TODO: %s (done: %s)", t.Text, done)))
}

func main() {
	//
	// Type aliases:
	// view.JetRuntimeVars = jet.VarMap
	// view.JetRuntime = jet.Runtime
	// view.JetArguments = jet.Arguments
	//
	// Iris also gives you the ability to put runtime variables
	// from middlewares as well, by:
	// view.AddJetRuntimeVars(ctx, vars)
	// or tmpl.AddRuntimeVars(ctx, vars)
	app := iris.New()
	tmpl := iris.Jet("./views", ".jet") // <--
	tmpl.Reload(true)                   // remove in production.
	tmpl.AddFunc("base64", func(a view.JetArguments) reflect.Value {
		a.RequireNumOfArguments("base64", 1, 1)

		buffer := bytes.NewBuffer(nil)
		fmt.Fprint(buffer, a.Get(0))

		return reflect.ValueOf(base64.URLEncoding.EncodeToString(buffer.Bytes()))
	})
	app.RegisterView(tmpl) // <--

	todos := map[string]*tTODO{
		"example-todo-1": {Text: "Add an show todo page to the example project", Done: true},
		"example-todo-2": {Text: "Add an add todo page to the example project"},
		"example-todo-3": {Text: "Add an update todo page to the example project"},
		"example-todo-4": {Text: "Add an delete todo page to the example project", Done: true},
	}

	app.Get("/", func(ctx iris.Context) {
		err := ctx.View("todos/index.jet", todos) // <--
		// Note that the `ctx.View` already logs the error if logger level is allowing it and returns the error.
		if err != nil {
			ctx.StopWithText(iris.StatusInternalServerError, "Templates not rendered!")
		}
	})

	app.Get("/todo", func(ctx iris.Context) {
		id := ctx.URLParam("id")
		todo, ok := todos[id]
		if !ok {
			ctx.Redirect("/")
			return
		}

		ctx.ViewData("title", "Show TODO")
		if err := ctx.View("todos/show.jet", todo); err != nil {
			ctx.HTML("<h3>%s</h3>", err.Error())
			return
		}
	})
	app.Get("/all-done", func(ctx iris.Context) {
		// vars := make(view.JetRuntimeVars)
		// vars.Set("showingAllDone", true)
		// vars.Set("title", "Todos - All Done")
		// view.AddJetRuntimeVars(ctx, vars)
		// ctx.View("todos/index.jet", (&doneTODOs{}).New(todos))
		//
		// OR
		ctx.ViewData("showingAllDone", true)
		ctx.ViewData("title", "Todos - All Done")

		// Use ctx.ViewData("_jet", jetData)
		// if using as middleware and you want
		// to pre-set the value or even change it later on from another next middleware.
		// ctx.ViewData("_jet", (&doneTODOs{}).New(todos))
		// and ctx.View("todos/index.jet")
		// OR
		if err := ctx.View("todos/index.jet", (&doneTODOs{}).New(todos)); err != nil {
			ctx.HTML("<h3>%s</h3>", err.Error())
			return
		}
	})

	port := os.Getenv("PORT")
	if len(port) == 0 {
		port = ":8080"
	} else if !strings.HasPrefix(":", port) {
		port = ":" + port
	}

	app.Listen(port)
}
