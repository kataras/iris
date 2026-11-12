package client

import (
	"net/http"

	"github.com/kataras/iris/v14/core/pkg/jsonutil"

	"github.com/kataras/httpclient"

	json "encoding/json/v2"
)

// unmarshalOptions returns the Iris decode policy, with the caller's options
// joined on top of it when there are any.
func unmarshalOptions(opts []json.Options) json.Options {
	if len(opts) == 0 {
		return jsonutil.DefaultUnmarshalOptions
	}

	return json.JoinOptions(append([]json.Options{jsonutil.DefaultUnmarshalOptions}, opts...)...)
}

// Bind decodes an already-received response into a new T, choosing the decoder
// from the response's Content-Type header (JSON, or plain text into a string or
// byte slice). Closing the response body is up to the caller.
//
// JSON bodies decode under the Iris policy; "opts" are joined on top of it.
func Bind[T any](resp *http.Response, opts ...json.Options) (T, error) {
	return httpclient.Bind[T](resp, unmarshalOptions(opts))
}

// BindError decodes the JSON body carried by the APIError inside "err" into a
// new T. When the remote server is an Iris application, that body is the
// rest/errors envelope:
//
//	remote, decodeErr := client.BindError[errors.Error](err)
//
// It returns the original "err" untouched when "err" does not carry an
// APIError, so "not an API error" stays distinguishable from "an API error I
// could not decode".
func BindError[T any](err error, opts ...json.Options) (T, error) {
	return httpclient.BindError[T](err, unmarshalOptions(opts))
}

// DecodeError binds the JSON body of an APIError to "destPtr". It returns the
// original "err" when it is not an APIError. Prefer the generic BindError.
func DecodeError(err error, destPtr any, opts ...json.Options) error {
	return httpclient.DecodeError(err, destPtr, unmarshalOptions(opts))
}

// IsErrEmptyJSON reports whether "err" comes from a BindJSON or ReadJSON call
// whose response body was empty, which several APIs answer to a successful
// write. It is a variable so that it can be replaced.
var IsErrEmptyJSON = httpclient.IsErrEmptyJSON
