package mvc

import (
	"testing"

	"github.com/kataras/iris/v14/context"
	"github.com/kataras/iris/v14/context/encoding"
)

func TestResolvePathParamName(t *testing.T) {
	type Customer struct {
		Name string `json:"name"`
	}

	got := resolvePathParamName[Customer]() // should return "customer_id".
	expected := "customer_id"
	if got != expected {
		t.Fatalf("expected %s but got %s", expected, got)
	}

	got = resolvePathParamName[context.Context]()
	expected = "context_id"
	if got != expected {
		t.Fatalf("expected %s but got %s", expected, got)
	}

	got = resolvePathParamName[encoding.EntriesDecoderOptions[string]]()
	expected = "entries_decoder_options_id"
	if got != expected {
		t.Fatalf("expected %s but got %s", expected, got)
	}
}
