package gorillamux_test

import (
	"testing"

	"gopkg.in/kataras/iris.v6"
	"gopkg.in/kataras/iris.v6/adaptors/gorillamux"
)

func TestRouteURLPath(t *testing.T) {
	app := iris.New()
	app.Adapt(gorillamux.New())

	app.None("/profile/{user_id}/{ref}/{anything:.*}", nil).ChangeName("profile")
	app.Boot()

	expected := "/profile/42/iris-go/something"

	if got := app.Path("profile", "user_id", 42, "ref", "iris-go", "anything", "something"); got != expected {
		t.Fatalf("gorillamux' reverse routing 'URLPath' error:  expected %s but got %s", expected, got)
	}
}
