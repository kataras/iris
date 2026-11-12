// This file holds the Context's typed binding methods: the generic-method
// counterparts of the former rest package's ctx-first generic functions.
// Their error responses render through the RestErrorRenderer seam
// (see rest_error_renderer.go): the canonical renderer lives in rest/errors
// and installs itself at package-init time.
package context

import (
	json "encoding/json/v2"
)

// firstJSONByte returns the first non-whitespace byte of a JSON document,
// or 0 for an empty (or all-whitespace) body. Whitespace per RFC 8259.
func firstJSONByte(body []byte) byte {
	for _, b := range body {
		switch b {
		case ' ', '\t', '\n', '\r':
			continue
		}
		return b
	}

	return 0
}

// Validatable is implemented by request types that validate themselves after
// decoding. BindJSON, BindJSONOptional and the BindJSONOneOrMany variants
// call Validate on the decoded value and reject the request with a client
// error response when it returns a non-nil error. BindUpdate calls it only
// for a full update: a partial update (the "columns" query key is present)
// skips it, see BindUpdate.
type Validatable interface {
	Validate() error
}

// validateDecoded runs the app-level ContextOptions.RequestValidator and then
// the payload's own Validate method on a value decoded outside ReadJSON
// (which runs the app validator itself). It writes a client error response
// and returns false on the first failure.
func validateDecoded(ctx *Context, payloadPtr any) bool {
	if err := ctx.OptionsReadOnly().Validate(payloadPtr); err != nil {
		restErrorRenderer.HandleValidationError(ctx, err)
		return false
	}

	return validatePayload(ctx, payloadPtr)
}

// validatePayload runs the payload's Validate method, if implemented.
// It writes a client error response and returns false on validation failure.
func validatePayload(ctx *Context, payloadPtr any) bool {
	v, ok := payloadPtr.(Validatable)
	if !ok {
		return true
	}

	if err := v.Validate(); err != nil {
		restErrorRenderer.HandleValidationError(ctx, err)
		return false
	}

	return true
}

// BindJSON reads a JSON payload and returns it as a generic type Request.
// It also returns a boolean value indicating whether the read was successful or not.
// If the read fails, it sends an appropriate error response to the client.
// If the Request type implements the Validatable interface, its Validate
// method runs after decoding and a non-nil result rejects the request.
func (ctx *Context) BindJSON[Request any]() (Request, bool) {
	return bindJSON[Request](ctx, true, true)
}

// BindJSONOptional reads a JSON payload like BindJSON but treats an empty
// request body as success: it returns the zero value of Request with true
// and skips validation. Use it when the request body is optional.
func (ctx *Context) BindJSONOptional[Request any]() (Request, bool) {
	return bindJSON[Request](ctx, false, true)
}

// bindJSON decodes the JSON body into Request. "required" rejects an empty
// body (BindJSONOptional passes false and returns the zero value instead).
// "validate" runs the payload's own Validate method after decoding; the
// app-level ContextOptions.RequestValidator runs inside ReadJSON regardless.
func bindJSON[Request any](ctx *Context, required, validate bool) (Request, bool) {
	var payload Request
	err := ctx.ReadJSON(&payload)
	if !required && IsErrEmptyJSON(err) { // empty json body but it was optionally, so return its zero value with true.
		return payload, true
	}

	if restErrorRenderer.HandleRequestJSONError(ctx, err) {
		return payload, false
	}

	if err != nil {
		// Not a JSON-decoder error: e.g. the app-level
		// ContextOptions.RequestValidator rejected the decoded payload.
		restErrorRenderer.HandleValidationError(ctx, err)
		return payload, false
	}

	if validate && !validatePayload(ctx, &payload) {
		return payload, false
	}

	return payload, true
}

// BindJSONOneOrMany reads the request body from the context and attempts to unmarshal it
// into either a single object or an array of objects of type Request.
// It returns a pointer to the single object, a slice of objects, and a boolean indicating whether the unmarshalling was successful.
// If the unmarshalling fails, it handles the JSON error and returns nil values along with a boolean indicating failure.
//
// Returns:
// - A pointer to a single object of type Request if the body contains a single object.
// - A slice of objects of type Request if the body contains an array of objects.
// - A boolean indicating whether the unmarshalling was successful.
func (ctx *Context) BindJSONOneOrMany[Request any]() (*Request, []Request, bool) {
	body, err := ctx.Body()
	if err != nil {
		// A body-read failure is not a decoder error: the request error
		// mapper knows ErrRequestEntityTooLarge (413) and the rest render
		// as 400. The JSON error mapper would not recognize it and the
		// caller would continue with a nil payload and a 200.
		restErrorRenderer.HandleRequestError(ctx, err)
		return nil, nil, false
	}

	// One source of truth for the decode defaults: the same option set
	// Context.ReadJSON uses (case-insensitive member matching and
	// time.Duration from int64 nanoseconds, see JSONReader.unmarshalOptions).
	unmarshalOptions := DefaultJSONReader.unmarshalOptions()

	// The first significant byte decides the shape structurally; no
	// error-text sniffing (Go no longer guarantees error text) and no
	// wasted unmarshal attempt of the whole body into the wrong shape.
	// A plain byte scan, not a jsontext.Decoder: the body is already in
	// memory and a decoder costs three allocations just to peek one token.
	if firstJSONByte(body) == '[' {
		var many []Request
		if err := json.Unmarshal(body, &many, unmarshalOptions...); err != nil {
			return nil, nil, !restErrorRenderer.HandleRequestJSONError(ctx, err)
		}

		for i := range many {
			if !validateDecoded(ctx, &many[i]) {
				return nil, nil, false
			}
		}

		return nil, many, true
	}

	// Object, scalar, empty or garbage: the object arm's error mapping
	// covers every failure.
	var single Request
	if err := json.Unmarshal(body, &single, unmarshalOptions...); err != nil {
		return nil, nil, !restErrorRenderer.HandleRequestJSONError(ctx, err)
	}

	if !validateDecoded(ctx, &single) {
		return nil, nil, false
	}

	return &single, nil, true
}

// BindJSONList reads the request body from the context and attempts to unmarshal it
// into either a single object or an array of objects of type Request.
// It returns a slice of objects of type Request and a boolean indicating whether the unmarshalling was successful.
// If the unmarshalling fails, it handles the JSON error and returns nil values along with a boolean indicating failure.
// If the body contains a single object, it is converted into a slice of objects.
func (ctx *Context) BindJSONList[Request any]() ([]Request, bool) {
	obj, arr, ok := ctx.BindJSONOneOrMany[Request]()
	if !ok {
		return nil, false
	}

	if obj != nil {
		arr = []Request{*obj}
	}

	return arr, true
}

// BindParams returns the path parameters bound to a struct of type Struct
// (field tag `param`) via `ctx.ReadParams`.
//
// It returns false and sends a client error if the path parameters binder failed.
func (ctx *Context) BindParams[Struct any]() (Struct, bool) {
	var params Struct

	if err := ctx.ReadParams(&params); err != nil {
		restErrorRenderer.HandleRequestError(ctx, err)
		return params, false
	}

	return params, true
}

// BindParam returns a single path parameter of type T.
//
// It returns false if the parameter is not found or if the parameter is not of the expected type.
// It sends a 404 Not Found or 400 Bad Request error to the client.
func (ctx *Context) BindParam[T any](name string) (T, bool) {
	paramValue, ok := ctx.Params().Lookup(name)
	if !ok {
		restErrorRenderer.HandleNotFound(ctx, "param %s not found", name)
		var t T
		return t, false
	}

	value, ok := paramValue.(T)
	if !ok {
		// The Go type name stays server-side: the client sees only which
		// parameter was wrong.
		restErrorRenderer.HandleInvalidArgument(ctx, "param %s has an unexpected type", name)
		return value, false
	}

	return value, true
}

// RequestValueContextKey is the key used to store the request value in the Iris Context.
const RequestValueContextKey = "iris.rest.request"

// RequestValue returns any cached value of the "iris.rest.request" key from the Iris Context.
func (ctx *Context) RequestValue[Request any]() (Request, bool) {
	return ctx.ValueAs[Request](RequestValueContextKey)
}
