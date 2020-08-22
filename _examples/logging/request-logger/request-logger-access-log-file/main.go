package main // See https://github.com/kataras/iris/issues/1601

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/middleware/logger"
)

func main() {
	// Create or use the ./access.log file.
	f, err := os.OpenFile("access.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	iris.RegisterOnInterrupt(func() { f.Close() })
	//

	app := iris.New()

	// Init the request logger with
	// the LogFuncCtx field alone.
	reqLogger := logger.New(logger.Config{
		LogFuncCtx: requestLogFunc(f),
	})
	//

	// Wrap the request logger middleware
	// with a response recorder because
	// we want to record the response body
	// sent to the client.
	reqLoggerWithRecord := func(ctx iris.Context) {
		// Store the requested path just in case.
		ctx.Values().Set("path", ctx.Path())
		ctx.Record()
		reqLogger(ctx)
	}
	//

	// Register the middleware (UseRouter to catch http errors too).
	app.UseRouter(reqLoggerWithRecord)
	//

	// Register some routes...
	app.HandleDir("/", iris.Dir("./public"))

	app.Get("/user/{username}", userHandler)
	app.Post("/read_body", readBodyHandler)
	//

	// Start the server with `WithoutBodyConsumptionOnUnmarshal`
	// option so the request body can be readen twice:
	// one for our handlers and one from inside our request logger middleware.
	app.Listen(":8080", iris.WithoutBodyConsumptionOnUnmarshal)
}

func readBodyHandler(ctx iris.Context) {
	var request interface{}
	if err := ctx.ReadBody(&request); err != nil {
		ctx.StopWithPlainError(iris.StatusBadRequest, err)
		return
	}

	ctx.JSON(iris.Map{"message": "OK"})
}

func userHandler(ctx iris.Context) {
	ctx.Writef("Hello, %s!", ctx.Params().Get("username"))
}

func jsonToString(src []byte) string {
	buf := new(bytes.Buffer)
	if err := json.Compact(buf, src); err != nil {
		return err.Error()
	}

	return buf.String()
}

func requestLogFunc(w io.Writer) func(ctx iris.Context, lat time.Duration) {
	return func(ctx iris.Context, lat time.Duration) {
		var (
			method = ctx.Method() // request method.
			// Use a stored value instead of ctx.Path()
			// because some handlers may change the relative path
			// to perform some action.
			path = ctx.Values().GetString("path")
			code = ctx.GetStatusCode() // response status code
			// request and response data or error reading them.
			requestBody  string
			responseBody string

			// url parameters and path parameters separated by space,
			// key=value key2=value2.
			requestValues string
		)

		// any error handler stored ( ctx.SetErr or StopWith(Plain)Error )
		errHandler := ctx.GetErr()
		// check if not error and client sent a response with a content-type set-ed.
		if errHandler == nil {
			if ctx.GetContentTypeRequested() == "application/json" {
				// Read and log request body the client sent to the server:
				//
				// You can use ctx.ReadBody(&body)
				// which will decode any body (json, xml, msgpack, protobuf...)
				// and use %v inside the fmt.Fprintf to print something like:
				// map[age:22 id:10 name:Tim]
				//
				// But if you want specific to json string,
				// then do that:
				var tmp json.RawMessage
				if err := ctx.ReadJSON(&tmp); err != nil {
					requestBody = err.Error()
				} else {
					requestBody = jsonToString(tmp)
				}
				//
			} else {
				// left for exercise.
			}
		} else {
			requestBody = fmt.Sprintf("error(%s)", errHandler.Error())
		}

		responseData := ctx.Recorder().Body()
		// check if the server sent any response with content type,
		// note that this will return the ;charset too
		// so we check for its prefix instead.
		if strings.HasPrefix(ctx.GetContentType(), "application/json") {
			responseBody = jsonToString(responseData)
		} else {
			responseBody = string(responseData)
		}

		var buf strings.Builder

		ctx.Params().Visit(func(key, value string) {
			buf.WriteString(key)
			buf.WriteByte('=')
			buf.WriteString(value)
			buf.WriteByte(' ')
		})

		for _, entry := range ctx.URLParamsSorted() {
			buf.WriteString(entry.Key)
			buf.WriteByte('=')
			buf.WriteString(entry.Value)
			buf.WriteByte(' ')
		}

		if n := buf.Len(); n > 1 {
			requestValues = buf.String()[0 : n-1] // remove last space.
		}

		fmt.Fprintf(w, "%s|%s|%s|%s|%s|%d|%s|%s|\n",
			time.Now().Format("2006-01-02 15:04:05"),
			lat,
			method,
			path,
			requestValues,
			code,
			requestBody,
			responseBody,
		)
	}
}
