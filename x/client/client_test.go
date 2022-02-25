package client

import (
	stdContext "context"
	"reflect"
	"testing"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/httptest"
)

var defaultCtx = stdContext.Background()

type testValue struct {
	Firstname string `json:"firstname"`
}

func TestClientJSON(t *testing.T) {
	expectedJSON := testValue{Firstname: "Makis"}

	app := iris.New()
	app.Get("/", sendJSON(t, expectedJSON))

	var irisGotJSON testValue
	app.Post("/", readJSON(t, &irisGotJSON, &expectedJSON))

	srv := httptest.NewServer(t, app)
	client := New(BaseURL(srv.URL))

	// Test ReadJSON (read from server).
	var got testValue
	if err := client.ReadJSON(defaultCtx, &got, iris.MethodGet, "/", nil); err != nil {
		t.Fatal(err)
	}

	// Test JSON (send to server).
	resp, err := client.JSON(defaultCtx, iris.MethodPost, "/", expectedJSON)
	if err != nil {
		t.Fatal(err)
	}
	client.DrainResponseBody(resp)
}

func sendJSON(t *testing.T, v interface{}) iris.Handler {
	return func(ctx iris.Context) {
		if _, err := ctx.JSON(v); err != nil {
			t.Fatal(err)
		}
	}
}

func readJSON(t *testing.T, ptr interface{}, expected interface{}) iris.Handler {
	return func(ctx iris.Context) {
		if err := ctx.ReadJSON(ptr); err != nil {
			t.Fatal(err)
		}

		if !reflect.DeepEqual(ptr, expected) {
			t.Fatalf("expected to read json: %#+v but got: %#+v", ptr, expected)
		}
	}
}
