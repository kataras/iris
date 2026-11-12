package context_test

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/httptest"
)

func TestContextURLParams(t *testing.T) {
	app := iris.New()
	app.Get("/", func(ctx iris.Context) {
		name := ctx.Query().Get("name")
		age := ctx.Query().GetInt("age")
		hobbies := ctx.Query().GetStringSlice("hobbies")
		ctx.Writef("name: %s:, age: %d, hobbies: %s", name, age, strings.Join(hobbies, ","))
	})

	e := httptest.New(t, app)
	e.GET("/").WithQueryString("name=kataras&age=30&hobbies=programming&hobbies=baseball").Expect().Status(httptest.StatusOK).
		Body().IsEqual("name: kataras:, age: 30, hobbies: programming,baseball")
}

// EscapedPath keeps the percent-encoding the client sent; Path is the
// decoded form.
func TestEscapedPath(t *testing.T) {
	app := iris.New()
	app.Get("/{p:path}", func(ctx iris.Context) {
		ctx.Writef("%s|%s", ctx.EscapedPath(), ctx.Path())
	})

	srv := serve(t, app)
	resp, err := http.Get(srv.URL + "/a%20b/c%2Fd")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if got, want := string(body), "/a%20b/c%2Fd|/a b/c/d"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}
