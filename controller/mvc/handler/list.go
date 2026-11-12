package handler

import (
	"github.com/kataras/iris/v14/context"
	"github.com/kataras/iris/v14/rest/errors"
)

// ListHandler returns a handler which handles a generic response and error from a service paginated call and sends a JSON response to the client.
// If the error is not nil, it calls errors.HandleError to send an appropriate error response to the client.
// It reads the PageOptions from the URL Query and any filter options of generic T from the request body.
// It sends a *context.Page[Response] response as a JSON payload with the current (200 OK by default) status code.
//
// The Filter type is the type of the filter options that the client can send to the server.
// The Response type is the type of the response that the server will send to the client.
// The TotalCount type is the type of the total count of the items.
//
// Usage:
//
//	app.Post("/list", handler.ListHandler(handleList))
//
//	func handleList(ctx context.Context, filter Filter, limit int, offset int) ([]Response, int, error) {
//		fmt.Printf("listing with limit: %d offset: %d and filter: %+v\n", limit, offset, filter)
//		return []Response{{Firstname: "Makis", Lastname: "Maropoulos"}}, 1, nil
//	}
func ListHandler[Context ContextType, Filter FilterType, Response ResponseType, Total CountType](fn ListFunc[Context, Filter, Response, Total], options ...Option) context.Handler {
	opts := makeOptions(options)

	responseEncoder := getResponseEncoder[*context.Page[Response]]()

	isIrisContext := isIrisCtx[Context]()
	return func(ctx *context.Context) {
		if !opts.serveHTTP(ctx) {
			return
		}

		pageOpts, filter, ok := ctx.BindPagination[Filter](opts.pagination...)
		if !ok {
			return
		}

		items, totalCount, err := fn(getCtx[Context](ctx, isIrisContext), filter, pageOpts.GetLimit(), pageOpts.GetOffset())
		if errors.HandleError(ctx, err) {
			return
		}

		resp := context.NewPage(pageOpts, filter, items, context.CountToInt64(totalCount))
		errors.HandleError(ctx, responseEncoder(ctx, resp))
	}
}

// ListByHandler same as `ListHandler` but it accepts an extra parameter which can be used to bind the extra parameters such us dynamic path or url parameters.
func ListByHandler[Context ContextType, Param ParamType, Filter FilterType, Response ResponseType, Total CountType](fn ParameterizedListFunc[Context, Param, Filter, Response, Total], options ...Option) context.Handler {
	opts := makeOptions(options)

	// pre-resolve request decoder.
	// requestDecoder := getRequestDecoder[Filter]()
	// pre-resolve param decoder.
	decodeParam := getParamDecodeFunc[Param]()
	// pre-resolve response encoder.
	responseEncoder := getResponseEncoder[*context.Page[Response]]()
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

		pageOpts, filter, ok := ctx.BindPagination[Filter](opts.pagination...)
		if !ok {
			return
		}

		items, totalCount, err := fn(getCtx[Context](ctx, isIrisContext), param, filter, pageOpts.GetLimit(), pageOpts.GetOffset())
		if errors.HandleError(ctx, err) {
			return
		}

		resp := context.NewPage(pageOpts, filter, items, context.CountToInt64(totalCount))
		errors.HandleError(ctx, responseEncoder(ctx, resp))
	}
}
