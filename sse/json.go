package sse

import (
	"bytes"
	json "encoding/json/v2"

	"github.com/kataras/iris/v14/core/pkg/jsonutil"
)

// marshalJSONTo encodes v for an event payload into a pooled buffer, using
// the framework's marshal defaults (jsonutil). Release the returned buffer
// with jsonutil.ReleaseBuffer once its bytes have been consumed — safe right
// after Send, because sendLocked copies Data into the stream's own frame
// buffer synchronously.
//
// It uses the json encoder directly rather than the Context writers, because
// the payload of one event is not the response body: a Context writer would
// set a Content-Type and a status code, both of which belong to the stream as
// a whole and were decided when it opened.
func marshalJSONTo(v any) (*bytes.Buffer, error) {
	buf := jsonutil.AcquireBuffer()

	if err := json.MarshalWrite(buf, v, jsonutil.DefaultMarshalOptionsSlice...); err != nil {
		jsonutil.ReleaseBuffer(buf)
		return nil, err
	}

	return buf, nil
}
