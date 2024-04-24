package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

var defaultCtx = context.Background()

type testValue struct {
	Firstname string `json:"firstname"`
}

func TestClientJSON(t *testing.T) {
	expectedJSON := testValue{Firstname: "Makis"}

	app := http.NewServeMux()
	app.HandleFunc("/send", sendJSON(t, expectedJSON))

	var irisGotJSON testValue
	app.HandleFunc("/read", readJSON(t, &irisGotJSON, &expectedJSON))

	srv := httptest.NewServer(app)
	client := New(BaseURL(srv.URL))

	// Test ReadJSON (read from server).
	var got testValue
	if err := client.ReadJSON(defaultCtx, &got, http.MethodGet, "/send", nil); err != nil {
		t.Fatal(err)
	}

	// Test JSON (send to server).
	resp, err := client.JSON(defaultCtx, http.MethodPost, "/read", expectedJSON)
	if err != nil {
		t.Fatal(err)
	}
	client.DrainResponseBody(resp)
}

func sendJSON(t *testing.T, v interface{}) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		if err := json.NewEncoder(w).Encode(v); err != nil {
			t.Fatal(err)
		}
	}
}

func readJSON(t *testing.T, ptr interface{}, expected interface{}) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(ptr); err != nil {
			t.Fatal(err)
		}

		if !reflect.DeepEqual(ptr, expected) {
			t.Fatalf("expected to read json: %#+v but got: %#+v", ptr, expected)
		}
	}
}
