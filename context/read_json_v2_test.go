package context_test

import (
	"errors"
	"io"
	"net/http"
	nethttptest "net/http/httptest"
	"strings"
	"testing"

	iris "github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/context"
	"github.com/kataras/iris/v14/httptest"
)

// readJSONResult drives ctx.ReadJSON through a real request and reports the
// decode error classification and the decoded value.
func readJSONApp(t *testing.T, out any, opts ...context.JSONReader) (*httptest.Expect, *error) {
	t.Helper()

	var lastErr error
	app := iris.New()
	app.Post("/read", func(ctx iris.Context) {
		lastErr = ctx.ReadJSON(out, opts...)
		if lastErr != nil {
			ctx.StopWithStatus(httptest.StatusBadRequest)
			return
		}
		ctx.WriteString("ok")
	})

	return httptest.New(t, app), &lastErr
}

// TestReadJSONBodyClassification pins the structural (never error-text)
// classification of the encoding/json/v2-based request body reader.
func TestReadJSONBodyClassification(t *testing.T) {
	cases := []struct {
		name      string
		body      string
		wantEmpty bool // IsErrEmptyJSON
		wantErr   bool
	}{
		{"empty body", "", true, true},
		{"whitespace only", "  \n\t ", true, true},
		{"garbage", "xyz", false, true},
		{"truncated value", `{"a":`, false, true},
		{"duplicate members", `{"a":1,"a":2}`, false, true}, // v2 strictness: smuggling defense.
		{"invalid utf-8", "{\"a\":\"\xff\xfe\"}", false, true},
		{"valid object", `{"a":1}`, false, false},
		{"trailing data tolerated", `{"a":1}{"b":2}`, false, false}, // one value per call, like the old json.Decoder.
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var v map[string]any
			e, lastErr := readJSONApp(t, &v)

			req := e.POST("/read").WithHeader("Content-Type", "application/json")
			if tc.body != "" {
				req = req.WithBytes([]byte(tc.body))
			}

			status := httptest.StatusOK
			if tc.wantErr {
				status = httptest.StatusBadRequest
			}
			req.Expect().Status(status)

			if got := context.IsErrEmptyJSON(*lastErr); got != tc.wantEmpty {
				t.Fatalf("IsErrEmptyJSON = %v, want %v (err: %v)", got, tc.wantEmpty, *lastErr)
			}
			if tc.wantEmpty && !errors.Is(*lastErr, io.EOF) {
				t.Fatalf("an empty-body error must keep matching errors.Is(err, io.EOF), got: %v", *lastErr)
			}
		})
	}
}

// TestReadJSONCaseSensitivity pins the compatibility default (old
// encoding/json matched member names case-insensitively) and its opt-out.
func TestReadJSONCaseSensitivity(t *testing.T) {
	type payload struct {
		Name string `json:"name"`
	}

	body := `{"NAME":"iris"}`

	t.Run("default is case-insensitive", func(t *testing.T) {
		var v payload
		e, _ := readJSONApp(t, &v)
		e.POST("/read").WithHeader("Content-Type", "application/json").
			WithBytes([]byte(body)).Expect().Status(httptest.StatusOK)
		if v.Name != "iris" {
			t.Fatalf("expected the wrong-case member to bind, got %+v", v)
		}
	})

	t.Run("CaseSensitive opts out", func(t *testing.T) {
		var v payload
		e, _ := readJSONApp(t, &v, context.JSONReader{CaseSensitive: true})
		e.POST("/read").WithHeader("Content-Type", "application/json").
			WithBytes([]byte(body)).Expect().Status(httptest.StatusOK) // unknown member is ignored, not an error.
		if v.Name != "" {
			t.Fatalf("expected the wrong-case member NOT to bind, got %+v", v)
		}
	})

	// v1 EqualFold parity: '_' still counts, so "vO2PEAK" binds VO2PEAK only
	// instead of failing as ambiguous with VO2_PEAK. Covers the default reader
	// and the non-default option path (DisallowUnknownFields builds its own set).
	type delimited struct {
		Peak         float64 `json:"VO2PEAK"`
		RecoveryPeak float64 `json:"VO2_PEAK"`
	}
	delimitedBody := `{"vO2PEAK":27,"vO2_PEAK":1829.5}`

	for name, opts := range map[string][]context.JSONReader{
		"default keeps delimiters":               nil,
		"DisallowUnknownFields keeps delimiters": {{DisallowUnknownFields: true}},
	} {
		t.Run(name, func(t *testing.T) {
			var v delimited
			e, lastErr := readJSONApp(t, &v, opts...)
			e.POST("/read").WithHeader("Content-Type", "application/json").
				WithBytes([]byte(delimitedBody)).Expect().Status(httptest.StatusOK)
			if *lastErr != nil || v.Peak != 27 || v.RecoveryPeak != 1829.5 {
				t.Fatalf("got %+v, err: %v", v, *lastErr)
			}
		})
	}
}

// TestReadJSONStreamReaderErrors ensures a failing body reader surfaces as an
// error instead of ending the stream loop silently.
func TestReadJSONStreamReaderErrors(t *testing.T) {
	app := iris.New()
	app.Use(func(ctx iris.Context) {
		ctx.Options().BodyMaxMemory = 8
		ctx.Next()
	})

	var streamErr error
	app.Post("/stream", func(ctx iris.Context) {
		streamErr = ctx.ReadJSONStream(func(decode context.DecodeFunc) error {
			var v map[string]any
			return decode(&v)
		})
		if streamErr != nil {
			ctx.StopWithStatus(httptest.StatusRequestEntityTooLarge)
			return
		}
		ctx.WriteString("ok")
	})

	e := httptest.New(t, app)
	e.POST("/stream").WithBytes([]byte(`{"a":"` + strings.Repeat("x", 4096) + `"}`)).Expect().
		Status(httptest.StatusRequestEntityTooLarge)

	if !errors.Is(streamErr, context.ErrRequestEntityTooLarge) {
		t.Fatalf("expected ErrRequestEntityTooLarge through the stream reader, got: %v", streamErr)
	}
}

// A hand-built request - http.NewRequest(method, url, nil), what an in-process
// caller and most unit tests produce - carries a nil Body, which the net/http
// server never does. jsontext.NewDecoder reads from it immediately, so the
// readers have to see the nil first; the answer is the same one an empty body
// gets, so BindJSONOptional still yields the zero value and BindJSON still
// answers 400.
func TestReadJSONNilRequestBody(t *testing.T) {
	type payload struct {
		Name string `json:"name"`
	}

	var (
		lastErr  error
		bound    payload
		boundOK  bool
		streamed int
	)

	app := iris.New()
	app.Post("/read", func(ctx iris.Context) {
		var v payload
		lastErr = ctx.ReadJSON(&v)
	})
	app.Post("/stream", func(ctx iris.Context) {
		lastErr = ctx.ReadJSONStream(func(decode context.DecodeFunc) error {
			streamed++
			var v payload
			return decode(&v)
		})
	})
	app.Post("/optional", func(ctx iris.Context) {
		bound, boundOK = ctx.BindJSONOptional[payload]()
	})
	app.Post("/required", func(ctx iris.Context) {
		_, boundOK = ctx.BindJSON[payload]()
	})

	if err := app.Build(); err != nil {
		t.Fatal(err)
	}

	post := func(path string) *nethttptest.ResponseRecorder {
		t.Helper()

		req, err := http.NewRequest(http.MethodPost, "http://iris.test"+path, nil)
		if err != nil {
			t.Fatal(err)
		}
		if req.Body != nil {
			t.Fatal("setup: http.NewRequest with a nil body no longer leaves Body nil")
		}
		req.Header.Set("Content-Type", "application/json")

		rec := nethttptest.NewRecorder()
		app.ServeHTTP(rec, req) // must not panic.
		return rec
	}

	t.Run("ReadJSON", func(t *testing.T) {
		post("/read")
		if !context.IsErrEmptyJSON(lastErr) {
			t.Fatalf("ReadJSON error = %v, want the empty-body error", lastErr)
		}
		if !errors.Is(lastErr, io.EOF) {
			t.Fatalf("the empty-body error must keep matching errors.Is(err, io.EOF), got: %v", lastErr)
		}
	})

	t.Run("ReadJSONStream", func(t *testing.T) {
		post("/stream")
		if !context.IsErrEmptyJSON(lastErr) {
			t.Fatalf("ReadJSONStream error = %v, want the empty-body error", lastErr)
		}
		if streamed != 0 {
			t.Fatalf("the decode callback ran %d time(s) on a nil body", streamed)
		}
	})

	t.Run("BindJSONOptional", func(t *testing.T) {
		bound, boundOK = payload{Name: "sentinel"}, false
		post("/optional")
		if !boundOK || bound != (payload{}) {
			t.Fatalf("BindJSONOptional = (%+v, %v), want the zero value and true", bound, boundOK)
		}
	})

	t.Run("BindJSON", func(t *testing.T) {
		boundOK = true
		rec := post("/required")
		if boundOK {
			t.Fatal("BindJSON reported success for a request with no body")
		}
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
		}
	})
}
