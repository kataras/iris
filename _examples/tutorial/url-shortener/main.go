// Package main shows how you can create a simple URL Shortener.
//
// Article: https://medium.com/@kataras/a-url-shortener-service-using-go-iris-and-bolt-4182f0b00ae7
//
// $ go get github.com/etcd-io/bbolt
// $ go get github.com/satori/go.uuid
// $ cd $GOPATH/src/github.com/kataras/iris/_examples/tutorial/url-shortener
// $ go build
// $ ./url-shortener
package main

import (
	"html/template"

	"github.com/kataras/iris"
)

func main() {
	// assign a variable to the DB so we can use its features later.
	db := NewDB("shortener.db")
	// Pass that db to our app, in order to be able to test the whole app with a different database later on.
	app := newApp(db)

	// release the "db" connection when server goes off.
	iris.RegisterOnInterrupt(db.Close)

	app.Run(iris.Addr(":8080"))
}

func newApp(db *DB) *iris.Application {
	app := iris.Default() // or app := iris.New()

	// create our factory, which is the manager for the object creation.
	// between our web app and the db.
	factory := NewFactory(DefaultGenerator, db)

	// serve the "./templates" directory's "*.html" files with the HTML std view engine.
	tmpl := iris.HTML("./templates", ".html").Reload(true)
	// register any template func(s) here.
	//
	// Look ./templates/index.html#L16
	tmpl.AddFunc("IsPositive", func(n int) bool {
		if n > 0 {
			return true
		}
		return false
	})

	app.RegisterView(tmpl)

	// Serve static files (css)
	app.StaticWeb("/static", "./resources")

	indexHandler := func(ctx iris.Context) {
		ctx.ViewData("URL_COUNT", db.Len())
		ctx.View("index.html")
	}
	app.Get("/", indexHandler)

	// find and execute a short url by its key
	// used on http://localhost:8080/u/dsaoj41u321dsa
	execShortURL := func(ctx iris.Context, key string) {
		if key == "" {
			ctx.StatusCode(iris.StatusBadRequest)
			return
		}

		value := db.Get(key)
		if value == "" {
			ctx.StatusCode(iris.StatusNotFound)
			ctx.Writef("Short URL for key: '%s' not found", key)
			return
		}

		ctx.Redirect(value, iris.StatusTemporaryRedirect)
	}
	app.Get("/u/{shortkey}", func(ctx iris.Context) {
		execShortURL(ctx, ctx.Params().Get("shortkey"))
	})

	app.Post("/shorten", func(ctx iris.Context) {
		formValue := ctx.FormValue("url")
		if formValue == "" {
			ctx.ViewData("FORM_RESULT", "You need to a enter a URL")
			ctx.StatusCode(iris.StatusLengthRequired)
		} else {
			key, err := factory.Gen(formValue)
			if err != nil {
				ctx.ViewData("FORM_RESULT", "Invalid URL")
				ctx.StatusCode(iris.StatusBadRequest)
			} else {
				if err = db.Set(key, formValue); err != nil {
					ctx.ViewData("FORM_RESULT", "Internal error while saving the URL")
					app.Logger().Infof("while saving URL: " + err.Error())
					ctx.StatusCode(iris.StatusInternalServerError)
				} else {
					ctx.StatusCode(iris.StatusOK)
					shortenURL := "http://" + app.ConfigurationReadOnly().GetVHost() + "/u/" + key
					ctx.ViewData("FORM_RESULT",
						template.HTML("<pre><a target='_new' href='"+shortenURL+"'>"+shortenURL+" </a></pre>"))
				}

			}
		}

		indexHandler(ctx) // no redirect, we need the FORM_RESULT.
	})

	app.Post("/clear_cache", func(ctx iris.Context) {
		db.Clear()
		ctx.Redirect("/")
	})

	return app
}
