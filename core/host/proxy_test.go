// black-box testing
package host_test

import (
	"net"
	"net/url"
	"testing"
	"time"

	"github.com/kataras/iris"
	"github.com/kataras/iris/context"
	"github.com/kataras/iris/core/host"
	"github.com/kataras/iris/httptest"
)

func TestProxy(t *testing.T) {
	expectedIndex := "ok /"
	expectedAbout := "ok /about"
	unexpectedRoute := "unexpected"

	// proxySrv := iris.New()
	u, err := url.Parse("https://localhost:4444")
	if err != nil {
		t.Fatalf("%v while parsing url", err)
	}

	proxy := host.NewProxy("", u)

	addr := &net.TCPAddr{
		IP:   net.IPv4(127, 0, 0, 1),
		Port: 0,
	}

	listener, err := net.ListenTCP("tcp", addr)
	if err != nil {
		t.Fatalf("%v while creating listener", err)
	}

	go proxy.Serve(listener) // should be localhost/127.0.0.1:80 but travis throws permission denied.

	t.Log(listener.Addr().String())
	<-time.After(time.Second)
	t.Log(listener.Addr().String())

	app := iris.New()
	app.Get("/", func(ctx context.Context) {
		ctx.WriteString(expectedIndex)
	})

	app.Get("/about", func(ctx context.Context) {
		ctx.WriteString(expectedAbout)
	})

	app.OnErrorCode(iris.StatusNotFound, func(ctx context.Context) {
		ctx.WriteString(unexpectedRoute)
	})

	l, err := net.Listen("tcp", "localhost:4444") // should be localhost/127.0.0.1:443 but travis throws permission denied.
	if err != nil {
		t.Fatalf("%v while creating tcp4 listener for new tls local test listener", err)
	}
	// main server
	go app.Run(iris.Listener(httptest.NewLocalTLSListener(l)), iris.WithoutStartupLog)

	e := httptest.NewInsecure(t, httptest.URL("http://"+listener.Addr().String()))
	e.GET("/").Expect().Status(iris.StatusOK).Body().Equal(expectedIndex)
	e.GET("/about").Expect().Status(iris.StatusOK).Body().Equal(expectedAbout)
	e.GET("/notfound").Expect().Status(iris.StatusNotFound).Body().Equal(unexpectedRoute)
}
