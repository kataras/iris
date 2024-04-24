package main

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/view"
)

func main() {
	tmpl := iris.Jet("./views", ".jet")
	tmpl.Reload(true)

	val := reflect.ValueOf(ViewBuiler{})
	fns := val.Type()
	for i := 0; i < fns.NumMethod(); i++ {
		method := fns.Method(i)
		tmpl.AddFunc(strings.ToLower(method.Name), val.Method(i).Interface())
	}

	app := iris.New()
	app.RegisterView(tmpl)

	app.Get("/", func(ctx iris.Context) {
		if err := ctx.View("index.jet"); err != nil {
			ctx.HTML("<h3>%s</h3>", err.Error())
			return
		}
	})

	app.Listen(":8080")
}

type ViewBuiler struct {
}

func (ViewBuiler) Asset(a view.JetArguments) reflect.Value {
	path := a.Get(0).String()
	// fmt.Println(os.Getenv("APP_URL"))
	return reflect.ValueOf(path)
}

func (ViewBuiler) Style(a view.JetArguments) reflect.Value {
	path := a.Get(0).String()
	s := fmt.Sprintf(`<link href="%v" rel="stylesheet">`, path)
	return reflect.ValueOf(s)
}

func (ViewBuiler) Script(a view.JetArguments) reflect.Value {
	path := a.Get(0).String()
	s := fmt.Sprintf(`<script src="%v"></script>`, path)
	return reflect.ValueOf(s)
}
