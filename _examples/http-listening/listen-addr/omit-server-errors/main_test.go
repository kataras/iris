package main

import (
	"bytes"
	stdContext "context"
	"strings"
	"testing"
	"time"

	"github.com/kataras/iris"
)

func logger(app *iris.Application) *bytes.Buffer {
	buf := &bytes.Buffer{}

	app.Logger().SetOutput(buf)

	// disable the "Now running at...." in order to have a clean log of the error.
	// we could attach that on `Run` but better to keep things simple here.
	app.Configure(iris.WithoutStartupLog)
	return buf
}

func TestListenAddr(t *testing.T) {
	app := iris.New()
	// we keep the logger running as well but in a controlled way.
	log := logger(app)

	// close the server at 3-6 seconds
	go func() {
		time.Sleep(3 * time.Second)
		ctx, cancel := stdContext.WithTimeout(stdContext.TODO(), 3*time.Second)
		defer cancel()
		app.Shutdown(ctx)
	}()

	err := app.Run(iris.Addr(":9829"))
	// in this case the error should be logged and return as well.
	if err != iris.ErrServerClosed {
		t.Fatalf("expecting err to be `iris.ErrServerClosed` but got: %v", err)
	}

	expectedMessage := iris.ErrServerClosed.Error()

	if got := log.String(); !strings.Contains(got, expectedMessage) {
		t.Fatalf("expecting to log to contains the:\n'%s'\ninstead of:\n'%s'", expectedMessage, got)
	}

}

func TestListenAddrWithoutServerErr(t *testing.T) {
	app := iris.New()
	// we keep the logger running as well but in a controlled way.
	log := logger(app)

	// close the server at 3-6 seconds
	go func() {
		time.Sleep(3 * time.Second)
		ctx, cancel := stdContext.WithTimeout(stdContext.TODO(), 3*time.Second)
		defer cancel()
		app.Shutdown(ctx)
	}()

	// we disable the ErrServerClosed, so the error should be nil when server is closed by `app.Shutdown`.

	// so in this case the iris/http.ErrServerClosed should be NOT logged and NOT return.
	err := app.Run(iris.Addr(":9827"), iris.WithoutServerError(iris.ErrServerClosed))
	if err != nil {
		t.Fatalf("expecting err to be nil but got: %v", err)
	}

	if got := log.String(); got != "" {
		t.Fatalf("expecting to log nothing but logged: '%s'", got)
	}
}
