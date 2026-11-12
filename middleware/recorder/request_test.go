package recorder_test

import (
	"testing"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/httptest"
	"github.com/kataras/iris/v14/middleware/recorder"
)

// ReadRequestBody must terminate: each full pass over the buffered body ends
// with EOF (so io.ReadAll returns), and the buffer rewinds afterwards so the
// body can be read again. An empty body must terminate the same way.
func TestReadRequestBodyTerminatesAndRepeats(t *testing.T) {
	app := iris.New()
	app.Post("/", func(ctx iris.Context) {
		first := recorder.ReadRequestBody(ctx)
		second := recorder.ReadRequestBody(ctx) // repeated read of the same body

		if string(first) != string(second) {
			ctx.StatusCode(httptest.StatusInternalServerError)
			ctx.WriteString("repeated read mismatch")
			return
		}

		ctx.Write(first)
	})

	e := httptest.New(t, app)

	e.POST("/").WithText("hello, recorder").Expect().
		Status(httptest.StatusOK).Body().IsEqual("hello, recorder")

	// An empty body must not spin forever.
	e.POST("/").Expect().Status(httptest.StatusOK).Body().IsEqual("")
}

// A json.Decoder consumer never issues the final read that returns io.EOF,
// so the buffer must rewind as soon as a pass drains it: the recorded body
// stays readable after ctx.ReadJSON.
func TestRecordedBodySurvivesJSONDecoder(t *testing.T) {
	app := iris.New()
	app.Use(recorder.Request)
	app.Post("/", func(ctx iris.Context) {
		var v map[string]any
		if err := ctx.ReadJSON(&v); err != nil {
			ctx.StopWithStatus(httptest.StatusBadRequest)
			return
		}

		ctx.Write(recorder.ReadRequestBody(ctx))
	})

	e := httptest.New(t, app)
	e.POST("/").WithJSON(map[string]int{"a": 1}).Expect().
		Status(httptest.StatusOK).Body().IsEqual(`{"a":1}`)
}
