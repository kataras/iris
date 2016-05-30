package bindings

import "github.com/kataras/iris/errors"

var (
	// ErrNoForm returns an error with message: 'Request has no any valid form'
	ErrNoForm = errors.New("Request has no any valid form")
	// ErrWriteJSON returns an error with message: 'Before JSON be written to the body, JSON Encoder returned an error. Trace: +specific error'
	ErrWriteJSON = errors.New("Before JSON be written to the body, JSON Encoder returned an error. Trace: %s")
	// ErrRenderMarshalled returns an error with message: 'Before +type Rendering, MarshalIndent retured an error. Trace: +specific error'
	ErrRenderMarshalled = errors.New("Before +type Rendering, MarshalIndent returned an error. Trace: %s")
	// ErrReadBody returns an error with message: 'While trying to read +type from the request body. Trace +specific error'
	ErrReadBody = errors.New("While trying to read %s from the request body. Trace %s")
)
