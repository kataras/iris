package context_test

import (
	"testing"
	"time"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/context"
	"github.com/kataras/iris/v14/httptest"
)

type streamItem struct {
	A int `json:"a"`
}

// readStreamSwallowing posts body to a handler that reads it with
// ReadJSONStream and swallows every per-item decode error, the pattern the
// DecodeFunc doc used to invite. It reports the error ReadJSONStream itself
// returned, or fails the test when the read does not finish in time.
func readStreamSwallowing(t *testing.T, body string, arrayStream bool) error {
	t.Helper()

	app := iris.New()
	var streamErr error
	app.Post("/stream", func(ctx iris.Context) {
		done := make(chan error, 1)
		go func() {
			done <- ctx.ReadJSONStream(func(decode context.DecodeFunc) error {
				var v streamItem
				_ = decode(&v) // swallowed on purpose.
				return nil
			}, context.JSONReader{ArrayStream: arrayStream})
		}()

		select {
		case streamErr = <-done:
		case <-time.After(2 * time.Second):
			t.Error("ReadJSONStream did not return: the loop spins after a swallowed decode error")
		}
		ctx.WriteString("done")
	})

	e := httptest.New(t, app)
	e.POST("/stream").WithHeader("Content-Type", "application/json").WithBytes([]byte(body)).
		Expect().Status(httptest.StatusOK)

	return streamErr
}

func TestReadJSONStreamSwallowedErrorsTerminate(t *testing.T) {
	cases := []struct {
		name        string
		body        string
		arrayStream bool
	}{
		{"array syntactic", `[{"a":1}, oops]`, true},
		{"array semantic", `[{"a":"x"}]`, true},
		{"array semantic mid-stream", `[{"a":1},{"a":"x"},{"a":3}]`, true},
		{"array non-array body", `{"a":1}`, true},
		{"ndjson syntactic", "{\"a\":1}\noops\n", false},
		{"ndjson semantic", "{\"a\":\"x\"}\n{\"a\":2}\n", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := readStreamSwallowing(t, tc.body, tc.arrayStream); err == nil {
				t.Fatal("expected the decode error to be returned")
			}
		})
	}
}

// A callback that never decodes cannot advance the stream; the loop stops
// with io.ErrNoProgress instead of spinning.
func TestReadJSONStreamCallbackWithoutDecode(t *testing.T) {
	app := iris.New()
	var streamErr error
	app.Post("/stream", func(ctx iris.Context) {
		done := make(chan error, 1)
		go func() {
			done <- ctx.ReadJSONStream(func(context.DecodeFunc) error { return nil })
		}()
		select {
		case streamErr = <-done:
		case <-time.After(2 * time.Second):
			t.Error("ReadJSONStream did not return")
		}
		ctx.WriteString("done")
	})

	e := httptest.New(t, app)
	e.POST("/stream").WithBytes([]byte(`{"a":1}`)).Expect().Status(httptest.StatusOK)
	if streamErr == nil {
		t.Fatal("expected io.ErrNoProgress")
	}
}

func TestReadJSONStreamArrayHappyPath(t *testing.T) {
	app := iris.New()
	var names []string
	var streamErr error
	app.Post("/stream", func(ctx iris.Context) {
		streamErr = ctx.ReadJSONStream(func(decode context.DecodeFunc) error {
			var v struct {
				Name string `json:"name"`
			}
			if err := decode(&v); err != nil {
				return err
			}
			names = append(names, v.Name)
			return nil
		}, context.JSONReader{ArrayStream: true})
		ctx.WriteString("done")
	})

	e := httptest.New(t, app)
	e.POST("/stream").WithBytes([]byte(`[{"name":"john"},{"name":"makis"}]`)).Expect().Status(httptest.StatusOK)

	if streamErr != nil {
		t.Fatal(streamErr)
	}
	if len(names) != 2 || names[0] != "john" || names[1] != "makis" {
		t.Fatalf("names = %q", names)
	}
}
