package hero

import (
	"testing"

	"github.com/kataras/iris/context"
)

func TestPathParams(t *testing.T) {
	got := ""
	h := New()
	handler := h.Handler(func(firstname string, lastname string) {
		got = firstname + lastname
	})

	h.Register(func(ctx context.Context) func() string { return func() string { return "" } })
	handlerWithOther := h.Handler(func(f func() string, firstname string, lastname string) {
		got = f() + firstname + lastname
	})

	handlerWithOtherBetweenThem := h.Handler(func(firstname string, f func() string, lastname string) {
		got = f() + firstname + lastname
	})

	ctx := context.NewContext(nil)
	ctx.Params().Set("firstname", "Gerasimos")
	ctx.Params().Set("lastname", "Maropoulos")
	handler(ctx)
	expected := "GerasimosMaropoulos"
	if got != expected {
		t.Fatalf("expected the params 'firstname' + 'lastname' to be '%s' but got '%s'", expected, got)
	}

	got = ""
	handlerWithOther(ctx)
	expected = "GerasimosMaropoulos"
	if got != expected {
		t.Fatalf("expected the params 'firstname' + 'lastname' to be '%s' but got '%s'", expected, got)
	}

	got = ""
	handlerWithOtherBetweenThem(ctx)
	expected = "GerasimosMaropoulos"
	if got != expected {
		t.Fatalf("expected the params 'firstname' + 'lastname' to be '%s' but got '%s'", expected, got)
	}

}
