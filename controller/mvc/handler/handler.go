package handler

import (
	stdContext "context"
	"unsafe"

	"github.com/kataras/iris/v14/context"
	"github.com/kataras/iris/v14/rest/errors"
)

type (
	// ContextType is a type which can be used as a context.Context or iris.Context.
	//
	// CONSTRAINT: instantiate this with EXACTLY one of two types:
	//
	//	*iris.Context        (github.com/kataras/iris/v14/context.Context)
	//	context.Context      (the standard library interface)
	//
	// The Go constraint cannot express that, so it permits more than the
	// implementation supports. `getCtx` builds the value with an unsafe
	// reinterpretation rather than a type assertion, which is only sound for
	// those two shapes:
	//
	//   - A DIFFERENT interface, e.g.
	//
	//	type TenantContext interface {
	//		context.Context
	//		TenantID() string
	//	}
	//
	//     compiles and is accepted here, but its itab has a different method set,
	//     so calling TenantID dispatches to the wrong slot.
	//
	//   - A concrete struct implementing context.Context is read as if it were a
	//     16-byte interface header.
	//
	// Neither is diagnosed at build time. Use one of the two supported types.
	ContextType interface {
		stdContext.Context
	}
	// RequestType is a type which can be used as a request payload.
	// It should be a struct.
	RequestType any
	// Note that this does NOT work, Go has no constraint for structures.
	// Read more about it at: https://groups.google.com/g/golang-nuts/c/UxVAj75L-rg/m/raCOyQRjAAAJ.
	//
	// RequestType interface {
	// 	~struct{}
	// }

	// ParamType is a type which can be used as an extra parameter for a service function,
	// e.g. path parameters and/or headers and/or url query parameters.
	ParamType interface {
		any | IDType // actually it's just an any type but we use IDType to show the usage.
	}

	// ResponseType is a type which can be used as a response payload.
	// It should be a struct.
	ResponseType any
	// FilterType is a type which can be used as a filter options for a list.
	// It should be a struct.
	FilterType any
	// CountType is a type that can be used as a total count of the items in a list.
	CountType = context.CountType
	// IDType is a type which can be used as an unique identifier for a resource.
	// It can be a string or int or int64 or uint64 or the standard library's uuid.UUID.
	IDType = context.IDType

	// Func is a function which takes a context and a generic type Request and returns a generic type Response and an error.
	// It is used to bind a request payload to a generic type Request and call a service function with it.
	Func[Context ContextType, Request RequestType, Response ResponseType] func(ctx Context, req Request) (Response, error)
	// JobFunc is a function which takes a context and a generic type Request and returns an error.
	// It is used to bind a request payload to a generic type Request and call a service function with it.
	// No response is sent to the client if error is nil.
	JobFunc[Context ContextType, Request RequestType] func(ctx Context, req Request) error
	// ActionFunc is a function which takes a context and returns a generic type Response and an error.
	// It is used to call a service function without a request payload.
	ActionFunc[Context ContextType, Response ResponseType] func(ctx Context) (Response, error)
	// ListFunc is a function which takes a context, a generic type Filter, limit and offset values and
	// returns a slice []Response, total count of the items and an error.
	//
	// It's used on the List function.
	ListFunc[Context ContextType, Filter FilterType, Response ResponseType, Total CountType] func(ctx Context, filter Filter, limit int, offset int) ([]Response, Total, error)

	// ParameterizedListFunc is a function which takes a context, a parameter, a generic type Filter, limit and offset values and returns a slice []Response, total count of the items and an error.
	// It is used for ListHandlerBy operations.
	ParameterizedListFunc[Context ContextType, Param ParamType, Filter FilterType, Response ResponseType, Total CountType] func(ctx Context, param Param, filter Filter, limit int, offset int) ([]Response, Total, error)

	// ParameterizedFunc is a function which takes a context, an ID and a generic type Request and returns a boolean and an error.
	// It is used to bind a request payload to a generic type Request and call a service function with it.
	// It is used for Update operations.
	ParameterizedFunc[Context ContextType, Param ParamType, Request RequestType, Response ResponseType] func(ctx Context, param Param, req Request) (Response, error)
)

func isIrisCtx[Context ContextType]() bool {
	var ctx Context
	_, isIrisContext := any(ctx).(*context.Context)
	return isIrisContext
}

func getCtx[Context ContextType](ctx *context.Context, isIrisContext bool) Context {
	if isIrisContext {
		// This is a trick to make it faster on stdContext.Context
		// Based on my benchmarks the type assertion of
		// any(ctx).(Context) has no actual performance cost if an iris.Context is passed
		// but it does when stdContext.Context is passed, the below
		// makes it entirely no-cost, so end-devs can use either iris.Context or context.Context
		// in their methods.
		return *(*Context)(unsafe.Pointer(&ctx))
	}

	// And this trick is to actually have a non-nil stdContext.Context
	// if the function's input argument was a stdContext/context.Context,
	// otherwise (if we just keep the above) it's nil.
	var stdContext stdContext.Context = ctx
	return *(*Context)(unsafe.Pointer(&stdContext))
}

// Handler returns a new Iris Handler which is ready to consume a request (by url and body) and send a success or fail response.
//
// The `fn` is the function which accepts a standard context.Context and a request model and returns a response model and an error.
// The `opts` are the handler options which can be used to configure the handler.
//
// If the error is not nil, it calls HandleError to send an appropriate error response to the client.
// If the response value is a string or int or int64 or uint64 or uuid.UUID it sends the IDPayload
// response as a JSON payload with the handler's status code, pass `WithStatusCode(201)`
// when a 201 (Created) is expected.
// If the response value is a boolean and true, it sets the status code to 204 (No Content).
// If the response value is a boolean and false, it sets the status code to 304 (Not Modified):
// the call succeeded but nothing changed.
// Otherwise it sets the status code to the handler's status code and sends the response as a JSON payload.
//
// Usage:
//
//	app.Post("/", Handler(handler, WithStatusCode(iris.StatusMovedPermanently))
//
//	func handler(ctx context.Context, req Request) (Response, error) {
//		// ...
//	}
func Handler[Context ContextType, Request RequestType, Response ResponseType](fn Func[Context, Request, Response], options ...Option) context.Handler {
	opts := makeOptions(options)

	decodeRequest := getRequestDecodeFunc[Request]()
	responseEncoder := getResponseEncoder[Response]()
	isIrisContext := isIrisCtx[Context]() // before serve-time, so this casting doesn't affect the performance. We could use a special .Value of Context but this will cause a bit (very little) performance cost.
	return func(ctx *context.Context) {
		if !opts.serveHTTP(ctx) {
			return
		}

		req, err := decodeRequest(ctx)
		if errors.HandleRequestError(ctx, err) {
			return
		}

		resp, err := fn(getCtx[Context](ctx, isIrisContext), req)
		if errors.HandleError(ctx, err) {
			return
		}

		errors.HandleError(ctx, responseEncoder(ctx, resp))
	}
}

// func CreateHandler[Context ContextType, Request RequestType, Response ResponseType](fn Func[Context, Request, Response], options ...Option) context.Handler {
// 	return Handler(fn, WithStatusCode(201))
// }
// Update/DeleteHandler...

// JobHandler returns a new Iris Handler which is ready to consume a request (by url and body) and
// do something but doesn't return a json response to the client.
func JobHandler[Context ContextType, Request RequestType](fn JobFunc[Context, Request], options ...Option) context.Handler {
	opts := makeOptions(options)

	decodeRequest := getRequestDecodeFunc[Request]()
	isIrisContext := isIrisCtx[Context]()
	return func(ctx *context.Context) {
		if !opts.serveHTTP(ctx) {
			return
		}

		req, err := decodeRequest(ctx)
		if errors.HandleRequestError(ctx, err) {
			return
		}

		err = fn(getCtx[Context](ctx, isIrisContext), req)
		errors.HandleError(ctx, err)
	}
}

// IndexHandler returns a new Iris Handler which just returns a response.
func IndexHandler[Context ContextType, Response ResponseType](fn ActionFunc[Context, Response], options ...Option) context.Handler {
	opts := makeOptions(options)

	responseEncoder := getResponseEncoder[Response]()

	isIrisContext := isIrisCtx[Context]()
	return func(ctx *context.Context) {
		if !opts.serveHTTP(ctx) {
			return
		}

		resp, err := fn(getCtx[Context](ctx, isIrisContext))
		if errors.HandleError(ctx, err) {
			return
		}
		errors.HandleError(ctx, responseEncoder(ctx, resp))
	}
}

func getRequestDecodeFunc[Request RequestType]() func(ctx *context.Context) (Request, error) {
	defaultDecoder := DefaultRequestDecoder[Request]()

	return func(ctx *context.Context) (Request, error) {
		request, gotRequest := getRequest[Request](ctx)
		if !gotRequest {
			if defaultDecoder == nil { // no decoder or struct with empty fields.
				// panic(fmt.Sprintf("no default decoder for the request type: %T, forgot to implement the 'DecodeRequest(iris.Context) error'?\n", request))
				return request, nil
			}

			err := defaultDecoder.DecodeRequest(ctx, &request)
			return request, err
		}
		return request, nil
	}
}

func getParamDecodeFunc[Param ParamType]() func(ctx *context.Context) (Param, error) {
	defaultDecoder := DefaultParamDecoder[Param]()

	return func(ctx *context.Context) (Param, error) {
		param, gotParam := getParam[Param](ctx)
		if !gotParam {
			if defaultDecoder == nil { // no decoder or struct with empty fields.
				// panic(fmt.Sprintf("no default decoder for the param type: %T, forgot to implement the 'DecodeRequest(iris.Context) error'?\n", param))
				return param, nil
			}

			err := defaultDecoder.DecodeRequest(ctx, &param)
			return param, err
		}
		return param, nil
	}
}
