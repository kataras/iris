package context_test

import (
	"errors"
	"net/http"
	"testing"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/context"
	"github.com/kataras/iris/v14/httptest"

	"google.golang.org/protobuf/types/known/wrapperspb"
)

var errWriteFailed = errors.New("write failed")

// failingWriter refuses every body write so the named-format writers have a
// write error to report.
type failingWriter struct{ http.ResponseWriter }

func (w *failingWriter) Write([]byte) (int, error) { return 0, errWriteFailed }

// The named-format writers (Binary, Text, HTML, Protobuf, ProtobufJSON,
// MsgPack, Negotiate) return only an error, like JSON, XML and the other
// format writers do; a failed write reaches the caller as that error.
// Write, Writef and WriteString keep the io.Writer (int, error) contract.
func TestNamedWritersReturnErrorOnly(t *testing.T) {
	cases := map[string]func(ctx iris.Context) error{
		"binary":       func(ctx iris.Context) error { return ctx.Binary([]byte("x")) },
		"text":         func(ctx iris.Context) error { return ctx.Text("x") },
		"html":         func(ctx iris.Context) error { return ctx.HTML("<p>x</p>") },
		"protobuf":     func(ctx iris.Context) error { return ctx.Protobuf(wrapperspb.String("x")) },
		"protobufjson": func(ctx iris.Context) error { return ctx.ProtobufJSON(wrapperspb.String("x")) },
		"msgpack":      func(ctx iris.Context) error { return ctx.MsgPack(iris.Map{"a": 1}) },
		"negotiate": func(ctx iris.Context) error {
			ctx.Negotiation().Text()
			return ctx.Negotiate("x")
		},
	}

	app := iris.New()
	app.Logger().SetLevel("disable")
	got := make(map[string]error, len(cases))
	for name, write := range cases {
		app.Get("/"+name, func(ctx iris.Context) {
			ctx.SetResponseWriter(context.AcquireResponseWriter(&failingWriter{ctx.ResponseWriter()}))
			got[name] = write(ctx)
		})
	}

	e := httptest.New(t, app)
	for name := range cases {
		e.GET("/" + name).Expect()
		if err := got[name]; !errors.Is(err, errWriteFailed) {
			t.Errorf("%s: got %v, want the write error", name, err)
		}
	}

	// The io.Writer trio keeps (int, error).
	var ctx *context.Context
	var (
		_ func([]byte) (int, error)         = ctx.Write
		_ func(string, ...any) (int, error) = ctx.Writef
		_ func(string) (int, error)         = ctx.WriteString
	)
}
