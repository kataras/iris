package mvc

import (
	"github.com/kataras/iris/context"
	"github.com/kataras/iris/mvc/activator/methodfunc"
)

// Response completes the `methodfunc.Result` interface.
// It's being used as an alternative return value which
// wraps the status code, the content type, a content as bytes or as string
// and an error, it's smart enough to complete the request and send the correct response to the client.
type Response struct {
	Code        int
	ContentType string
	Content     []byte

	// if not empty then content type is the text/plain
	// and content is the text as []byte.
	Text string
	// If not nil then it will fire that as "application/json" or the
	// "ContentType" if not empty.
	Object interface{}

	// if not empty then fire a 400 bad request error
	// unless the Status is > 200, then fire that error code
	// with the Err.Error() string as its content.
	//
	// if Err.Error() is empty then it fires the custom error handler
	// if any otherwise the framework sends the default http error text based on the status.
	Err error
	Try func() int
}

var _ methodfunc.Result = Response{}

// Dispatch writes the response result to the context's response writer.
func (r Response) Dispatch(ctx context.Context) {
	if s := r.Text; s != "" {
		r.Content = []byte(s)
	}

	methodfunc.DispatchCommon(ctx, r.Code, r.ContentType, r.Content, r.Object, r.Err, true)
}
