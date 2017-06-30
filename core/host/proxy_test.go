// black-box testing
package host_test

import (
	"net"
	"net/url"
	"testing"
	"strconv"
	"github.com/cdren/iris"
	"github.com/cdren/iris/context"
	"github.com/cdren/iris/core/host"
	"github.com/cdren/iris/httptest"
)

func TestProxy(t *testing.T) {
	expectedIndex := "ok /"
	expectedAbout := "ok /about"
	expectedDemo := "ok /name/demo"
	expectedDemoId := "ok /id/1234"
	expectedDemoIdMin := "ok /min/1234"
	unexpectedRoute := "unexpected"

	// proxySrv := iris.New()
	u, err := url.Parse("https://localhost:4444")
	if err != nil {
		t.Fatalf("%v while parsing url", err)
	}

	// p := host.ProxyHandler(u)
	// transport := &http.Transport{
	// 	TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	// }
	// p.Transport = transport
	// proxySrv.Downgrade(p.ServeHTTP)
	// go proxySrv.Run(iris.Addr(":80"), iris.WithoutBanner, iris.WithoutInterruptHandler)

	go host.NewProxy("localhost:2017", u).ListenAndServe() // should be localhost/127.0.0.1:80 but travis throws permission denied.

	app := iris.New()

	app.Macros().Int.RegisterFunc("min", func(minValue int) func(string) bool {
		// do anything before serve here [...]
		// at this case we don't need to do anything
		return func(paramValue string) bool {
			n, err := strconv.Atoi(paramValue)
			if err != nil {
				return false
			}
			return n >= minValue
		}
	})

	app.Get("/", func(ctx context.Context) {
		ctx.WriteString(expectedIndex)
	})

	app.Get("/about", func(ctx context.Context) {
		ctx.WriteString(expectedAbout)
	})

	app.Get("/name/{name}", func(ctx context.Context) {
		name := ctx.Params().Get("name")
		ctx.WriteString("ok /name/" + name)
	})

	app.Get("/id/{id:int}", func(ctx context.Context) {
		id, _ := ctx.Params().GetInt("id")
		ctx.WriteString("ok /id/" + strconv.Itoa(id))
	})

	app.Get("/min/{id:int min(1000)}", func(ctx context.Context) {
		id, _ := ctx.Params().GetInt("id")
		ctx.WriteString("ok /min/" + strconv.Itoa(id))
	})

	app.OnErrorCode(iris.StatusNotFound, func(ctx context.Context) {
		ctx.WriteString(unexpectedRoute)
	})

	l, err := net.Listen("tcp", "localhost:4444") // should be localhost/127.0.0.1:443 but travis throws permission denied.
	if err != nil {
		t.Fatalf("%v while creating tcp4 listener for new tls local test listener", err)
	}
	// main server
	go app.Run(iris.Listener(httptest.NewLocalTLSListener(l)), iris.WithoutBanner)

	e := httptest.NewInsecure(t, httptest.URL("http://localhost:2017"))
	e.GET("/").Expect().Status(iris.StatusOK).Body().Equal(expectedIndex)
	e.GET("/about").Expect().Status(iris.StatusOK).Body().Equal(expectedAbout)
	e.GET("/name/demo").Expect().Status(iris.StatusOK).Body().Equal(expectedDemo)
	e.GET("/id/1234").Expect().Status(iris.StatusOK).Body().Equal(expectedDemoId)
	e.GET("/id/notfound").Expect().Status(iris.StatusNotFound).Body().Equal(unexpectedRoute)
	e.GET("/min/1234").Expect().Status(iris.StatusOK).Body().Equal(expectedDemoIdMin)
	e.GET("/min/123").Expect().Status(iris.StatusNotFound).Body().Equal(unexpectedRoute)
	e.GET("/min/notfound").Expect().Status(iris.StatusNotFound).Body().Equal(unexpectedRoute)
	e.GET("/notfound").Expect().Status(iris.StatusNotFound).Body().Equal(unexpectedRoute)
}
