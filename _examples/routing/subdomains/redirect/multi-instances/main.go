package main

import (
	"net/http"

	"github.com/kataras/iris/v12"
)

func main() {
	app := iris.New()

	hosts := map[string]*iris.Application{
		"mydomain.com":      createRoot("www.mydomain.com"), // redirects to www.
		"www.mydomain.com":  createWWW(),
		"test.mydomain.com": createTest(),
	}
	for _, r := range hosts {
		r.Build()
	}

	app.Downgrade(func(w http.ResponseWriter, r *http.Request) {
		host := r.Host
		if host == "" {
			host = r.URL.Host
		}

		if router, ok := hosts[host]; ok {
			router.ServeHTTP(w, r)
			return
		}

		http.NotFound(w, r)
	})

	app.Listen(":80")
}

func createRoot(redirectTo string) *iris.Application {
	app := iris.New()
	app.Downgrade(func(w http.ResponseWriter, r *http.Request) {
		fullScheme := "http://"
		if r.TLS != nil {
			fullScheme = "https://"
		}

		http.Redirect(w, r, fullScheme+redirectTo+r.URL.RequestURI(), iris.StatusMovedPermanently)
	})

	return app
}

func createWWW() *iris.Application {
	app := iris.New()
	app.Get("/", index)

	users := app.Party("/users")
	users.Get("/", usersIndex)
	users.Get("/login", getLogin)

	return app
}

func createTest() *iris.Application {
	app := iris.New()
	app.Get("/", func(ctx iris.Context) {
		ctx.WriteString("Test Index")
	})

	return app
}

func index(ctx iris.Context) {
	ctx.Writef("This is the www.mydomain.com endpoint.")
}

func usersIndex(ctx iris.Context) {
	ctx.Writef("This is the www.mydomain.com/users endpoint.")
}

func getLogin(ctx iris.Context) {
	ctx.Writef("This is the www.mydomain.com/users/login endpoint.")
}
