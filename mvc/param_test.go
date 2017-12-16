package mvc

import (
	"testing"

	"github.com/kataras/iris/context"
)

func TestPathParamsBinder(t *testing.T) {
	m := NewEngine()
	m.Dependencies.Add(PathParamsBinder)

	got := ""

	h := m.Handler(func(params PathParams) {
		got = params.Get("firstname") + params.Get("lastname")
	})

	ctx := context.NewContext(nil)
	ctx.Params().Set("firstname", "Gerasimos")
	ctx.Params().Set("lastname", "Maropoulos")
	h(ctx)
	expected := "GerasimosMaropoulos"
	if got != expected {
		t.Fatalf("expected the params 'firstname' + 'lastname' to be '%s' but got '%s'", expected, got)
	}
}
func TestPathParamBinder(t *testing.T) {
	m := NewEngine()
	m.Dependencies.Add(PathParamBinder("username"))

	got := ""
	executed := false
	h := m.Handler(func(username PathParam) {
		// this should not be fired at all if "username" param wasn't found at all.
		// although router is responsible for that but the `ParamBinder` makes that check as well because
		// the end-developer may put a param as input argument on her/his function but
		// on its route's path didn't describe the path parameter,
		// the handler fires a warning and stops the execution for the invalid handler to protect the user.
		executed = true
		got = username.String()
	})

	expectedUsername := "kataras"
	ctx := context.NewContext(nil)
	ctx.Params().Set("username", expectedUsername)
	h(ctx)

	if got != expectedUsername {
		t.Fatalf("expected the param 'username' to be '%s' but got '%s'", expectedUsername, got)
	}

	// test the non executed if param not found.
	executed = false
	got = ""

	ctx2 := context.NewContext(nil)
	h(ctx2)

	if got != "" {
		t.Fatalf("expected the param 'username' to be entirely empty but got '%s'", got)
	}
	if executed {
		t.Fatalf("expected the handler to not be executed")
	}
}
