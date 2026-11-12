package handler

import (
	"github.com/kataras/iris/v14/context"
	"github.com/kataras/iris/v14/rest/errors"
)

// HandlerBy returns a new Iris Handler which is ready to consume two request objects and send a success or fail response.
// Usually the first is an object which contains the path parameters and the second one is the request body model itself.
// It works like `Handler` but it accepts an extra request model (Params) which can be used to bind the url path parameters.
//
// Usage:
//
//	app.Post("/{company_id:string}/lists/{list_id:string}/objects", handler.HandlerBy(handleCreateByParams))
//
//	type Params struct {
//		CompanyID string `param:"company_id"`
//		ListID    string `param:"list_id"`
//	}
//
//	func handleCreateByParams(ctx context.Context, params Params, request Request) (string, error) {
//		fmt.Printf("creating by params: %+v and request: %+v\n", params, request)
//		return "generated-id", nil
//	}
func HandlerBy[Context ContextType, Param ParamType, Request RequestType, Response ResponseType](fn ParameterizedFunc[Context, Param, Request, Response], options ...Option) context.Handler {
	opts := makeOptions(options)

	// pre-resolve request decoder.
	decodeRequest := getRequestDecodeFunc[Request]()
	// pre-resolve param decoder.
	decodeParam := getParamDecodeFunc[Param]() // default param decoder if no other specified.
	// pre-resolve response encoder.
	responseEncoder := getResponseEncoder[Response]()
	// pre-resolve Iris or std Context.
	isIrisContext := isIrisCtx[Context]()
	return func(ctx *context.Context) {
		if !opts.serveHTTP(ctx) {
			return
		}

		param, err := decodeParam(ctx) // decode the path parameters.
		if errors.HandleRequestError(ctx, err) {
			return
		}

		// No need to cast any(param).type() there because we know the type of the param.
		// param, err := bindParam[Param](ctx)
		// if HandleRequestError(ctx, err) {
		// 	return
		// }

		req, err := decodeRequest(ctx)
		if errors.HandleRequestError(ctx, err) {
			return
		}

		resp, err := fn(getCtx[Context](ctx, isIrisContext), param, req)
		if errors.HandleError(ctx, err) {
			return
		}

		// err = validateResponse(ctx, req, &resp)
		// if HandleError(ctx, err) {
		// 	return false
		// }

		errors.HandleError(ctx, responseEncoder(ctx, resp))
	}
}

// HandlerByColumns returns a new Iris Handler which is ready to consume two request objects (one Parameter and one the Request body)
// and send a success or fail response.
// It's like the `HandlerBy` but accepts a variadic list of strings which are the columns that the client requested to be updated.
// The "fn" returns the response to render with the handler's status code, or nil to let the
// boolean decide: true renders a 204 No Content, false a 304 Not Modified, both with an empty body.
// A non-nil error renders through the rest/errors pipeline.
func HandlerByColumns[Context ContextType, Param ParamType, Request RequestType, Response ResponseType](fn func(Context, Param, Request, ...string) (*Response, bool, error), options ...Option) context.Handler {
	opts := makeOptions(options)

	// pre-resolve request decoder.
	decodeRequest := getRequestDecodeFunc[Request]()
	// pre-resolve param decoder.
	decodeParam := getParamDecodeFunc[Param]() // default param decoder if no other specified.
	// pre-resolve response encoder.
	responseEncoder := getResponseEncoder[Response]()
	// pre-resolve Iris or std Context.
	isIrisContext := isIrisCtx[Context]()
	return func(ctx *context.Context) {
		if !opts.serveHTTP(ctx) {
			return
		}

		param, err := decodeParam(ctx) // decode the path parameters.
		if errors.HandleRequestError(ctx, err) {
			return
		}

		req, err := decodeRequest(ctx)
		if errors.HandleRequestError(ctx, err) {
			return
		}

		onlyColumns := readUpdateColumns(ctx)

		resp, updated, err := fn(getCtx[Context](ctx, isIrisContext), param, req, onlyColumns...)
		if resp != nil {
			errors.HandleError(ctx, responseEncoder(ctx, *resp))
			return
		}

		ctx.NoContent(updated, err)
	}
}
