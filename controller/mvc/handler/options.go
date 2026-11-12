package handler

import (
	"fmt"
	"uuid"

	"github.com/kataras/iris/v14/context"
	"github.com/kataras/iris/v14/rest/errors"
)

type Options struct {
	middleware []context.Handler
	request    []RequestOption // prior idea: to have only request options valid to be passed, but it's better to have a structure for future optional settings.
	// pagination is the sort policy of the list handlers, passed through to
	// Context.BindPagination.
	pagination []context.PaginationOption
	// skipDefaults is set by NoOption. It is a typed sentinel: the previous
	// implementation recognized NoOption by comparing function code pointers,
	// which is unsound since Go 1.27 (distinct code-identical literals may
	// share a code pointer, so a user option could be mistaken for NoOption).
	skipDefaults bool
}

func makeOptions(options []Option) *Options {
	opts := &Options{}
	for _, opt := range options {
		if opt == nil {
			continue
		}

		opt(opts)
	}
	return opts
}

func (opts *Options) appendRequestOptions(options ...RequestOption) {
	for _, opt := range options {
		if opt == nil {
			continue
		}

		opts.request = append(opts.request, opt)
	}
}

func (opts *Options) prependRequestOptions(options ...RequestOption) {
	notEmptyOptions := make([]RequestOption, 0, len(options))
	for _, opt := range options {
		if opt == nil {
			continue
		}

		notEmptyOptions = append(notEmptyOptions, opt)
	}

	opts.request = append(notEmptyOptions, opts.request...)
}

// SetDefaultRequestOptions sets the default request options for the handler.
// It does nothing when NoOption was applied.
func (opts *Options) SetDefaultRequestOptions(options ...RequestOption) {
	if !opts.skipDefaults {
		// opts = append(opts, WithRequestDecoder(decoder))
		//
		// Sometimes (like WithRequestHandler)
		// we need to read the request and pass it to the function
		// but options ran before we read and bind the request to the context values.
		// That's why we prepend the request decoder in the Handlers function.
		opts.prependRequestOptions(options...)
	}
}

func (opts *Options) serveHTTP(ctx *context.Context) bool {
	if !ctx.Proceed(opts.middleware...) { // empty middleware is handled inside.
		ctx.StopExecution()
		return false
	}

	return applyRequestOptions(ctx, opts.request)
}

type Option = func(*Options)

// NewOptions returns a new, empty Options value, ready to be filled through
// the chainable With* methods. Pass the prepared result to a handler
// through the `WithOptions` option:
//
//	opts := handler.NewOptions().WithStatusCode(201).WithPathParameter[string]("id")
//	handler.Handler(fn, handler.WithOptions(opts))
func NewOptions() *Options {
	return &Options{}
}

// WithOptions returns an option which applies a prepared Options value,
// built through `NewOptions` and its chainable methods, onto the target options.
// It appends the prepared middleware and request options and carries the
// skip-defaults sentinel (see `NoOption`) over.
func WithOptions(prepared *Options) Option {
	return func(opts *Options) {
		if prepared == nil {
			return
		}

		opts.middleware = append(opts.middleware, prepared.middleware...)
		opts.appendRequestOptions(prepared.request...)
		opts.pagination = append(opts.pagination, prepared.pagination...)
		if prepared.skipDefaults {
			opts.skipDefaults = true
		}
	}
}

// apply runs the given option, built by the package-level constructors,
// against the options value itself; the chainable methods below delegate
// through it so their behavior cannot drift from the constructors'.
func (opts *Options) apply(opt Option) *Options {
	if opt != nil {
		opt(opts)
	}

	return opts
}

// WithPagination is the chainable form of the package-level `WithPagination` option.
func (opts *Options) WithPagination(paginationOptions ...context.PaginationOption) *Options {
	return opts.apply(WithPagination(paginationOptions...))
}

// WithMiddleware is the chainable form of the package-level `WithMiddleware` option.
func (opts *Options) WithMiddleware(handlers ...context.Handler) *Options {
	return opts.apply(WithMiddleware(handlers...))
}

// WithRequestDecoder is the chainable form of the package-level `WithRequestDecoder` option.
func (opts *Options) WithRequestDecoder[Request any](requestDecoder RequestDecoder[Request]) *Options {
	return opts.apply(WithRequestDecoder(requestDecoder))
}

// WithRequestDecoderFunc is the chainable form of the package-level `WithRequestDecoderFunc` option.
func (opts *Options) WithRequestDecoderFunc[Request RequestType](fn RequestDecoderFunc[Request]) *Options {
	return opts.apply(WithRequestDecoderFunc(fn))
}

// WithParamDecoder is the chainable form of the package-level `WithParamDecoder` option.
func (opts *Options) WithParamDecoder[Param any](paramDecoder RequestDecoder[Param]) *Options {
	return opts.apply(WithParamDecoder(paramDecoder))
}

// WithParamDecoderFunc is the chainable form of the package-level `WithParamDecoderFunc` option.
func (opts *Options) WithParamDecoderFunc[Param ParamType](fn RequestDecoderFunc[Param]) *Options {
	return opts.apply(WithParamDecoderFunc(fn))
}

// WithPathParameter is the chainable form of the package-level `WithPathParameter` option.
func (opts *Options) WithPathParameter[Param any](paramName string) *Options {
	return opts.apply(WithPathParameter[Param](paramName))
}

// WithPathDecoder is the chainable form of the package-level `WithPathDecoder` option.
func (opts *Options) WithPathDecoder[Param any](paramName string) *Options {
	return opts.apply(WithPathDecoder[Param](paramName))
}

// WithStatusCode is the chainable form of the package-level `WithStatusCode` option.
func (opts *Options) WithStatusCode(statusCode int) *Options {
	return opts.apply(WithStatusCode(statusCode))
}

// WithMiddleware returns an option which registers the given handlers as middleware.
func WithMiddleware(handlers ...context.Handler) Option {
	return func(opts *Options) {
		opts.middleware = append(opts.middleware, handlers...)
	}
	/* this would be a request option for middleware:
	return func(ctx *context.Context) error {
		if !ctx.Proceed(handlers...) {
			ctx.StopExecution()
		}

		return nil
	}
	*/
}

// WithPagination returns an option which sets the sort policy of a list
// handler, the options `Context.BindPagination` takes (see context.AllowSort
// and context.DenySort). A sort the policy does not allow answers 400 before
// the handler's function runs.
func WithPagination(paginationOptions ...context.PaginationOption) Option {
	if len(paginationOptions) == 0 {
		return nil
	}

	return func(opts *Options) {
		opts.pagination = append(opts.pagination, paginationOptions...)
	}
}

// Pagination returns the sort policy options registered through
// `WithPagination`. The list handlers pass them to `Context.BindPagination`.
func (opts *Options) Pagination() []context.PaginationOption {
	return opts.pagination
}

func WithRequestOptions(options ...RequestOption) Option {
	return func(opts *Options) {
		opts.appendRequestOptions(options...)
	}
}

// RequestOption is a function which can be used to modify the request context.
// It can be used to set the request, path parameters, status code, etc.
// It can also be used to skip the default options.
// The function should return an error if the option fails to apply.
// If the error is not nil, the request will be stopped and the error will be handled.
type RequestOption = func(*context.Context) error

const (
	requestContextKey      = context.RequestValueContextKey
	requestParamContextKey = "iris.rest.request.param"
)

func setContextValue[Request RequestType](ctx *context.Context, key string, req Request) {
	ctx.Values().Set(key, req)
}

func getRequest[Request RequestType](ctx *context.Context) (Request, bool) {
	return ctx.RequestValue[Request]()
}

func getParam[Param ParamType](ctx *context.Context) (Param, bool) {
	return ctx.ValueAs[Param](requestParamContextKey)
}

/*
responseContextKey = "iris.rest.response"
responseHandlerKey = "iris.rest.response.handler"

func getResponse[Response ResponseType](ctx *context.Context) (Response, bool) {
	return ctx.ValueAs[Response](responseContextKey)
}
// WithResponseHandler returns an option which sets the response handler for the given response type.
func WithResponseHandler[Request RequestType, Response ResponseType](responseHandler ResponsePayloadHandler[Request, Response]) Option {
	return func(ctx *context.Context) error {
		// TODO: Register it on the iris.rest.response.handler and make it run after response is got from funcs, but before the response is sent.
		// However, this is extreme. Let's keep it simple for now.
		req, ok := getRequest[Request](ctx)
		if !ok {
			return nil
		}

		resp, ok := getResponse[Response](ctx)
		if !ok {
			return nil
		}

		return responseHandler.HandleResponse(ctx, req, &resp)
	}
}
*/

func setRequestDecoder[Request any](ctx *context.Context, key string, requestDecoder RequestDecoder[Request]) error {
	var request Request
	err := requestDecoder.DecodeRequest(ctx, &request)
	if err != nil {
		return err
	}

	setContextValue(ctx, key, request)
	return nil
}

// SetRequestDecoder sets the request decoder for the given request type.
func SetRequestDecoder[Request any](ctx *context.Context, requestDecoder RequestDecoder[Request]) error {
	return setRequestDecoder(ctx, requestContextKey, requestDecoder)
}

// SetRequest sets the request for the given request type.
func SetRequest[Request any](ctx *context.Context, request Request) {
	setContextValue(ctx, requestContextKey, request)
}

// SetParamDecoder sets the request decoder for the given param type.
func SetParamDecoder[Param ParamType](ctx *context.Context, paramDecoder RequestDecoder[Param]) error {
	return setRequestDecoder(ctx, requestParamContextKey, paramDecoder)
}

// SetParam sets the request for the given param type.
func SetParam[Param any](ctx *context.Context, request Param) {
	setContextValue(ctx, requestParamContextKey, request)
}

// WithRequestDecoder returns an option which sets the request decoder for the given request type.
func WithRequestDecoder[Request any](requestDecoder RequestDecoder[Request]) Option {
	if requestDecoder == nil {
		return nil
	}

	return WithRequestOptions(withRequestDecoder(requestDecoder))
}

// WithRequestDecoderFunc returns an option which sets the request decoder for the given request type.
func WithRequestDecoderFunc[Request RequestType](fn RequestDecoderFunc[Request]) Option {
	return WithRequestDecoder(fn)
}

func withRequestDecoder[Request any](requestDecoder RequestDecoder[Request]) RequestOption {
	if requestDecoder == nil {
		return nil
	}

	return func(ctx *context.Context) error {
		return SetRequestDecoder(ctx, requestDecoder)
	}
}

// WithParamDecoder returns an option which sets the request decoder for the given param type.
func WithParamDecoder[Param any](paramDecoder RequestDecoder[Param]) Option {
	return WithRequestOptions(withParamDecoder(paramDecoder))
}

// WithParamDecoderFunc returns an option which sets the request decoder for the given param type.
func WithParamDecoderFunc[Param ParamType](fn RequestDecoderFunc[Param]) Option {
	return WithParamDecoder(fn)
}

func withParamDecoder[Param any](paramDecoder RequestDecoder[Param]) RequestOption {
	return func(ctx *context.Context) error {
		return SetParamDecoder(ctx, paramDecoder)
	}
}

// WithPathParameter returns an option which sets the path parameter decoder for the given param type.
func WithPathParameter[Param any](paramName string) Option {
	return WithRequestOptions(withPathParameter[Param](paramName))
}

func withPathParameter[Param any](paramName string) RequestOption {
	return withParamDecoder(NewPathParamDecoder[Param](paramName))
}

// WithPathDecoder returns an option which sets the path parameter decoder for the given param type.
// If the Param type is a struct, it will decode the path parameters into the struct fields.
// If the Param type is a single parameter, of string or int or int64 or uint64 or uuid.UUID, it will decode the path parameter into the parameter.
func WithPathDecoder[Param any](paramName string) Option {
	var decoder RequestDecoder[Param]

	var param Param
	switch any(param).(type) {
	case string, int, int64, uuid.UUID: // it's a single parameter.
		decoder = NewPathParamDecoder[Param](paramName)
	default:
		decoder = NewPathParamsDecoder[Param]() // it's a struct with path parameters as fields.
	}

	return WithParamDecoder(decoder)
}

// WithStatusCode returns an option which sets the status code of the response.
func WithStatusCode(statusCode int) Option {
	return WithRequestOptions(withStatusCode(statusCode))
}

func withStatusCode(statusCode int) RequestOption {
	return func(ctx *context.Context) error {
		ctx.StatusCode(statusCode)
		return nil
	}
}

func applyRequestOptions(ctx *context.Context, options []RequestOption) bool {
	for _, opt := range options {
		if err := opt(ctx); err != nil {
			if err == errSkipOptions {
				return true
			}
			errors.HandleRequestError(ctx, err)
			return false
		}

		if ctx.IsStopped() {
			return false
		}
	}

	return true
}

var (
	// NoOption is a special option which can be used to skip the default options.
	NoOption Option = func(opts *Options) {
		opts.skipDefaults = true
		opts.appendRequestOptions(noOption)
	}
	// noOption short-circuits the remaining request options at serve time;
	// applyRequestOptions recognizes it by the errSkipOptions error VALUE
	// (never by function pointer, see Options.skipDefaults).
	noOption RequestOption = func(ctx *context.Context) error { return errSkipOptions }

	errSkipOptions = fmt.Errorf("skip next options")
)
