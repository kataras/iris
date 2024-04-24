package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path"
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/kataras/golog"
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/httptest"
)

func TestJSONLogger(t *testing.T) {
	iters := 500

	out := new(bytes.Buffer)

	app := iris.New()
	app.Logger().SetTimeFormat("")     // disable timestamps.
	app.Logger().SetStacktraceLimit(1) // limit debug stacktrace to 1, show only the first caller.
	app.Logger().SetOutput(out)

	app.Logger().Handle(func(l *golog.Log) bool {
		enc := json.NewEncoder(l.Logger.Printer) // you can change the output to a file as well.
		err := enc.Encode(l)
		return err == nil
	})

	app.Get("/ping", ping)

	expectedSourceDir := getSourceDirPath()
	expectedLogStr := fmt.Sprintf(`{"level":"debug","message":"Request path: /ping","fields":{"request_id":null},"stacktrace":[{"function":"json-logger/ping","source":"%s/main.go:78"}]}`, expectedSourceDir) // gh actions-specific.
	e := httptest.New(t, app, httptest.LogLevel("debug"))
	wg := new(sync.WaitGroup)
	wg.Add(iters)
	for i := 0; i < iters; i++ {
		go func() {
			e.GET("/ping").Expect().Status(httptest.StatusOK).Body().IsEqual("pong")
			wg.Done()
		}()
	}

	wg.Wait()
	expected := ""
	for i := 0; i < iters; i++ {
		expected += expectedLogStr + "\n"
	}

	got := out.String()
	got = got[strings.Index(got, "{"):] // take only the json we care and after.
	if expected != got {
		if !strings.HasSuffix(got, expected) {
			// C:/mygopath vs /home/travis vs any file system,
			// pure check but it does the job.
			t.Fatalf("expected:\n%s\nbut got:\n%s", expected, got)
		}
	}
}

func getSourceDirPath() string {
	_, file, _, ok := runtime.Caller(1) // get the caller's file.
	if !ok {
		return "unknown source"
	}

	return path.Dir(file) // get the directory of the file (delimiter: /).
}
