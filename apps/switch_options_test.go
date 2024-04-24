package apps

import (
	"testing"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/httptest"
)

func TestSetHost(t *testing.T) {
	var (
		index = func(ctx iris.Context) {
			ctx.Header("Server", ctx.Application().String())
			ctx.WriteString(ctx.Host())
		}

		forceHost = "www.mydomain.com"
	)

	rootApp := iris.New().SetName("My Server")
	rootApp.Get("/", index)

	switcher := Switch(Hosts{
		{"^(www.)?mydomain.com$", rootApp},
	}, SetHost(forceHost))

	e := httptest.New(t, switcher)
	tests := []*httptest.Request{
		e.GET("/").WithURL("http://mydomain.com"),
		e.GET("/").WithURL("http://www.mydomain.com"),
	}

	for _, tt := range tests {
		ex := tt.Expect().Status(iris.StatusOK)
		ex.Header("Server").Equal(rootApp.String())
		ex.Body().IsEqual(forceHost)
	}
}
