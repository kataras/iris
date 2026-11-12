package handler_test

import (
	"testing"

	"github.com/kataras/iris/v14/controller/mvc/handler"

	"uuid"
)

// The "uuid" macro stores a uuid.UUID. A handler that declares a string id
// (the default id type, which MVC maps to the uuid macro) receives the
// canonical text, and one that declares uuid.UUID receives the value as is.
func TestConvertPathParamValueUUID(t *testing.T) {
	id := uuid.MustParse("a6ce7330-d584-4675-af9e-2c2d2c40de07")

	if got, ok := handler.ConvertPathParamValue[string](id); !ok || got != id.String() {
		t.Fatalf("string from uuid.UUID: expected (%q, true), got (%q, %v)", id.String(), got, ok)
	}
	if got, ok := handler.ConvertPathParamValue[uuid.UUID](id); !ok || got != id {
		t.Fatalf("uuid.UUID from uuid.UUID: expected (%s, true), got (%s, %v)", id, got, ok)
	}
	// An untyped "{id}" stores the text; the typed handler still gets a value.
	if got, ok := handler.ConvertPathParamValue[uuid.UUID](id.String()); !ok || got != id {
		t.Fatalf("uuid.UUID from string: expected (%s, true), got (%s, %v)", id, got, ok)
	}
	if _, ok := handler.ConvertPathParamValue[uuid.UUID]("garbage"); ok {
		t.Fatal("uuid.UUID from garbage: expected false")
	}
	if _, ok := handler.ConvertPathParamValue[int64](id); ok {
		t.Fatal("int64 from uuid.UUID: expected false")
	}
}
