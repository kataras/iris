package client

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
)

// See "Handler" client option.
type handlerTransport struct {
	handler http.Handler
}

// RoundTrip completes the http.RoundTripper interface.
// It can be used to test calls to a server's handler.
func (t *handlerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	reqCopy := *req

	if reqCopy.Proto == "" {
		reqCopy.Proto = fmt.Sprintf("HTTP/%d.%d", reqCopy.ProtoMajor, reqCopy.ProtoMinor)
	}

	if reqCopy.Body != nil {
		if reqCopy.ContentLength == -1 {
			reqCopy.TransferEncoding = []string{"chunked"}
		}
	} else {
		reqCopy.Body = io.NopCloser(bytes.NewReader(nil))
	}

	if reqCopy.RequestURI == "" {
		reqCopy.RequestURI = reqCopy.URL.RequestURI()
	}

	recorder := httptest.NewRecorder()

	t.handler.ServeHTTP(recorder, &reqCopy)

	resp := http.Response{
		Request:    &reqCopy,
		StatusCode: recorder.Code,
		Status:     http.StatusText(recorder.Code),
		Header:     recorder.Result().Header,
	}

	if recorder.Flushed {
		resp.TransferEncoding = []string{"chunked"}
	}

	if recorder.Body != nil {
		resp.Body = io.NopCloser(recorder.Body)
	}

	return &resp, nil
}
