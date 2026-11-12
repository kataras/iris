# Request and Response Recording

In [Go](https://go.dev), a request body is consumed by the first read and the response body is not buffered at all. That makes it awkward to log a payload, inspect a response, or let two middlewares both look at the same body. This middleware buffers both, so you can [read them more than once](https://stackoverflow.com/questions/43021058/golang-read-request-body-multiple-times) and rewrite the response before it goes out.

## Features

- **Request Body Recording**: Capture and read the request body multiple times.
- **Response Body Recording**: Capture and read the response body multiple times.
- **Middleware Integration**: Works alongside other Iris middlewares.
- **Custom Response Writer**: Use a custom response writer to capture response data.
- **Buffer Management**: Efficiently manage buffers for request and response bodies.

## Installation

To use the Iris Recorder middleware, you need to import it in your Iris application:

```go
import "github.com/kataras/iris/v14/middleware/recorder"
```

## Usage

### Request Recorder

The Request Recorder buffers the request body so it can be read more than once, which is what logging, debugging and any second middleware that inspects the payload need.

#### Example

```go
package main

import (
    "github.com/kataras/iris/v14"
    "github.com/kataras/iris/v14/middleware/recorder"
)

func main() {
    app := iris.New()
    app.Use(recorder.Request)

    app.Post("/path", func(ctx iris.Context) {
        body, _ := ctx.Body()
        // OR use recorder.ReadRequestBody to record
        // and retrieve the data at once,
        // without registering the recorder.Request handler:
        // body := recorder.ReadRequestBody(ctx)
        ctx.WriteString(string(body))

        // Other calls of ctx.Body or recorder.ReadRequestBody
        // inside the route's handlers will give the same, consumed, result.
        // bodyReadTwice, _ := ctx.Body()
    })

    app.Listen(":8080")
}
```

### Response Recorder

The Response Recorder buffers the response body so it can be read, and rewritten, before it goes out to the client.

#### Example

```go
package main

import (
    "github.com/kataras/iris/v14"
    "github.com/kataras/iris/v14/middleware/recorder"
)

func main() {
    app := iris.New()
    app.Use(recorder.Response)

    app.Get("/path", func(ctx iris.Context) {
        ctx.WriteString("Hello, World!")
    }, func(ctx iris.Context){
        // Get the response recorder from the context
        rr := recorder.GetResponseRecorder(ctx)
        // Get the response body
        body := rr.GetBody()
        // OR get the whole result as *http.Response:
        // resp := recorder.GetResponse(ctx)
        ctx.WriteString(string(body))
    })

    app.Listen(":8080")
}
```

## API Reference

### Request Recorder

#### `func Request(ctx iris.Context)`

Enables multiple request body reading by replacing the request's Body with a buffered one.

#### `func ReadRequestBody(ctx iris.Context) []byte`

Enables multiple request body reading by replacing the request's Body with a buffered one and returns the request body.

#### `func RecordRequestBody(ctx iris.Context) *BufferCloser`

Records the request body and returns a `BufferCloser`.

#### `func IsRequestBodyRecorded(ctx iris.Context) bool`

Reports whether the request body is buffered.

### Response Recorder

#### `func Response(ctx iris.Context)`

Captures the response of the request sent by the server.

#### `func GetResponseRecorder(ctx iris.Context) *ResponseRecorder`

Returns the `ResponseRecorder` from the Context.

#### `func IsRecordingResponse(ctx iris.Context) bool`

Returns true if the response is being recorded.

#### `func GetResponse(ctx iris.Context) *http.Response`

Returns the captured response as an `http.Response`.

#### `type ResponseRecorder struct`

A custom response writer that captures the response.

- `func (rr *ResponseRecorder) Clone() context.ResponseWriter`
- `func (rr *ResponseRecorder) Reset(w context.ResponseWriter)`
- `func (rr *ResponseRecorder) release()`
- `func (rr *ResponseRecorder) Written() int`
- `func (rr *ResponseRecorder) StatusCode() int`
- `func (rr *ResponseRecorder) Write(b []byte) (int, error)`
- `func (rr *ResponseRecorder) WriteString(s string) (int, error)`
- `func (rr *ResponseRecorder) WriteHeader(statusCode int)`
- `func (rr *ResponseRecorder) Header() http.Header`
- `func (rr *ResponseRecorder) SetBody(body []byte)`
- `func (rr *ResponseRecorder) GetBody() []byte`
- `func (rr *ResponseRecorder) Result() *http.Response`
- `func (rr *ResponseRecorder) Flush()`
- `func (rr *ResponseRecorder) Push(target string, opts *http.PushOptions) error`
