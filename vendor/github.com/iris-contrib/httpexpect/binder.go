package httpexpect

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
)

// Binder implements networkless http.RoundTripper attached directly to
// http.Handler.
//
// Binder emulates network communication by invoking given http.Handler
// directly. It passes httptest.ResponseRecorder as http.ResponseWriter
// to the handler, and then constructs http.Response from recorded data.
type Binder struct {
	// HTTP handler invoked for every request.
	Handler http.Handler
	// TLS connection state used for https:// requests.
	TLS *tls.ConnectionState
}

// NewBinder returns a new Binder given a http.Handler.
//
// Example:
//   client := &http.Client{
//       Transport: NewBinder(handler),
//   }
func NewBinder(handler http.Handler) Binder {
	return Binder{Handler: handler}
}

// RoundTrip implements http.RoundTripper.RoundTrip.
func (binder Binder) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Proto == "" {
		req.Proto = fmt.Sprintf("HTTP/%d.%d", req.ProtoMajor, req.ProtoMinor)
	}

	if req.Body != nil {
		if req.ContentLength == -1 {
			req.TransferEncoding = []string{"chunked"}
		}
	} else {
		req.Body = ioutil.NopCloser(bytes.NewReader(nil))
	}

	if req.URL != nil && req.URL.Scheme == "https" && binder.TLS != nil {
		req.TLS = binder.TLS
	}

	if req.RequestURI == "" {
		req.RequestURI = req.URL.RequestURI()
	}

	recorder := httptest.NewRecorder()

	binder.Handler.ServeHTTP(recorder, req)

	resp := http.Response{
		Request:    req,
		StatusCode: recorder.Code,
		Status:     http.StatusText(recorder.Code),
		Header:     recorder.HeaderMap,
	}

	if recorder.Flushed {
		resp.TransferEncoding = []string{"chunked"}
	}

	if recorder.Body != nil {
		resp.Body = ioutil.NopCloser(recorder.Body)
	}

	return &resp, nil
}

type connNonTLS struct {
	net.Conn
}

func (connNonTLS) RemoteAddr() net.Addr {
	return &net.TCPAddr{IP: net.IPv4zero}
}

func (connNonTLS) LocalAddr() net.Addr {
	return &net.TCPAddr{IP: net.IPv4zero}
}

type connTLS struct {
	connNonTLS
	state *tls.ConnectionState
}

func (c connTLS) ConnectionState() tls.ConnectionState {
	return *c.state
}
