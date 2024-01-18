// black-box testing
package host_test

import (
	stdContext "context"
	"crypto/tls"
	"net"
	"net/url"
	"testing"
	"time"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/context"
	"github.com/kataras/iris/v12/core/host"
	"github.com/kataras/iris/v12/httptest"
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

	config := &tls.Config{
		InsecureSkipVerify: true,
		MinVersion:         tls.VersionTLS13,
	}
	proxy := host.NewProxy("", u, config)
	proxy.Configure(host.NonBlocking())

	addr := &net.TCPAddr{
		IP:   net.IPv4(127, 0, 0, 1),
		Port: 0,
	}

	listener, err := net.ListenTCP("tcp", addr)
	if err != nil {
		t.Fatalf("%v while creating listener", err)
	}

	// non-blocking (see above).
	proxy.Serve(listener)

	ctx, cancelFunc := stdContext.WithTimeout(stdContext.Background(), 15*time.Second)
	defer cancelFunc()

	// Wait for up to 15 seconds or until the proxy is ready to serve or a serve failure.
	proxy.Wait(ctx)

	t.Log(listener.Addr().String())

	app := iris.New()
	app.Get("/", func(ctx *context.Context) {
		ctx.WriteString(expectedIndex)
	})

	app.Get("/about", func(ctx *context.Context) {
		ctx.WriteString(expectedAbout)
	})

	app.OnErrorCode(iris.StatusNotFound, func(ctx *context.Context) {
		ctx.WriteString(unexpectedRoute)
	})

	l, err := net.Listen("tcp", "localhost:4444")
	if err != nil {
		t.Fatalf("%v while creating tcp4 listener for new tls local test listener", err)
	}
	// main server
	go app.Run(iris.Listener(httptest.NewLocalTLSListener(l)), iris.WithoutStartupLog) // nolint:errcheck

	e := httptest.NewInsecure(t, httptest.URL("http://"+listener.Addr().String()))
	e.GET("/").Expect().Status(iris.StatusOK).Body().IsEqual(expectedIndex)
	e.GET("/about").Expect().Status(iris.StatusOK).Body().IsEqual(expectedAbout)
	e.GET("/notfound").Expect().Status(iris.StatusNotFound).Body().IsEqual(unexpectedRoute)
}
