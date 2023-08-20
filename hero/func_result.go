package hero

import (
	"reflect"
	"strings"

	"github.com/kataras/iris/v12/context"

	"github.com/fatih/structs"
	"google.golang.org/protobuf/proto"
)

// ResultHandler describes the function type which should serve the "v" struct value.
type ResultHandler func(ctx *context.Context, v interface{}) error

func defaultResultHandler(ctx *context.Context, v interface{}) error {
	if p, ok := v.(PreflightResult); ok {
		if err := p.Preflight(ctx); err != nil {
			return err
		}
	}

	if d, ok := v.(Result); ok {
		d.Dispatch(ctx)
		return nil
	}

	switch context.TrimHeaderValue(ctx.GetContentType()) {
	case context.ContentXMLHeaderValue, context.ContentXMLUnreadableHeaderValue:
		return ctx.XML(v)
	case context.ContentYAMLHeaderValue:
		return ctx.YAML(v)
	case context.ContentProtobufHeaderValue:
		msg, ok := v.(proto.Message)
		if !ok {
			return context.ErrContentNotSupported
		}

		_, err := ctx.Protobuf(msg)
		return err
	case context.ContentMsgPackHeaderValue, context.ContentMsgPack2HeaderValue:
		_, err := ctx.MsgPack(v)
		return err
	default:
		// otherwise default to JSON.
		return ctx.JSON(v)
	}
}

// Result is a response dispatcher.
// All types that complete this interface
// can be returned as values from the method functions.
//
// Example at: https://github.com/kataras/iris/tree/main/_examples/dependency-injection/overview.
type Result interface {
	// Dispatch should send a response to the client.
	Dispatch(*context.Context)
}

// PreflightResult is an interface which implementers
// should be responsible to perform preflight checks of a <T> resource (or Result) before sent to the client.
//
// If a non-nil error returned from the `Preflight` method then the JSON result
// will be not sent to the client and an ErrorHandler will be responsible to render the error.
//
// Usage: a custom struct value will be a JSON body response (by-default) but it contains
// "Code int" and `ID string` fields, the "Code" should be the status code of the response
// and the "ID" should be sent as a Header of "X-Request-ID: $ID".
//
// The caller can manage it at the handler itself. However,
// to reduce thoese type of duplications it's preferable to use such a standard interface instead.
//
// The Preflight method can return `iris.ErrStopExecution` to render
// and override any interface that the structure value may implement, e.g. mvc.Result.
type PreflightResult interface {
	Preflight(*context.Context) error
}

var defaultFailureResponse = Response{Code: DefaultErrStatusCode}

// Try will check if "fn" ran without any panics,
// using recovery,
// and return its result as the final response
// otherwise it returns the "failure" response if any,
// if not then a 400 bad request is being sent.
//
// Example usage at: https://github.com/kataras/iris/blob/main/hero/func_result_test.go.
func Try(fn func() Result, failure ...Result) Result {
	var failed bool
	var actionResponse Result

	func() {
		defer func() {
			if rec := recover(); rec != nil {
				failed = true
			}
		}()
		actionResponse = fn()
	}()

	if failed {
		if len(failure) > 0 {
			return failure[0]
		}
		return defaultFailureResponse
	}

	return actionResponse
}

const slashB byte = '/'

type compatibleErr interface {
	Error() string
}

// dispatchErr sets the error status code
// and the error value to the context.
// The APIBuilder's On(Any)ErrorCode is responsible to render this error code.
func dispatchErr(ctx *context.Context, status int, err error) bool {
	if err == nil {
		return false
	}

	if err != ErrStopExecution {
		if status == 0 || !context.StatusCodeNotSuccessful(status) {
			status = DefaultErrStatusCode
		}

		ctx.StatusCode(status)
	}

	ctx.SetErr(err)
	return true
}

// DispatchFuncResult is being used internally to resolve
// and send the method function's output values to the
// context's response writer using a smart way which
// respects status code, content type, content, custom struct
// and an error type.
// Supports for:
// func(c *ExampleController) Get() string |
// (string, string) |
// (string, int) |
// ...
// int |
// (int, string |
// (string, error) |
// ...
// error |
// (int, error) |
// (customStruct, error) |
// ...
// bool |
// (int, bool) |
// (string, bool) |
// (customStruct, bool) |
// ...
// customStruct |
// (customStruct, int) |
// (customStruct, string) |
// Result or (Result, error) and so on...
//
// where Get is an HTTP METHOD.
func dispatchFuncResult(ctx *context.Context, values []reflect.Value, handler ResultHandler) error {
	if len(values) == 0 {
		return nil
	}

	var (
		// if statusCode > 0 then send this status code.
		// Except when err != nil then check if status code is < 400 and
		// if it's set it as DefaultErrStatusCode.
		// Except when found == false, then the status code is 404.
		statusCode = ctx.GetStatusCode() // Get the current status code given by any previous middleware.
		// if not empty then use that as content type,
		// if empty and custom != nil then set it to application/json.
		contentType string
		// if len > 0 then write that to the response writer as raw bytes,
		// except when found == false or err != nil or custom != nil.
		content []byte
		// if not nil then check
		// for content type (or json default) and send the custom data object
		// except when found == false or err != nil.
		custom interface{}
		// if false then skip everything and fire 404.
		found = true // defaults to true of course, otherwise will break :)
	)

	for _, v := range values {
		// order of these checks matters
		// for example, first  we need to check for status code,
		// secondly the string (for content type and content)...
		// if !v.IsValid() || !v.CanInterface() {
		// 	continue
		// }
		if !v.IsValid() {
			continue
		}

		f := v.Interface()
		/*
				if b, ok := f.(bool); ok {
					found = b
					if !found {
						// skip everything, we don't care about other return values,
						// this boolean is the higher in order.
						break
					}
					continue
				}

				if i, ok := f.(int); ok {
					statusCode = i
					continue
				}

				if s, ok := f.(string); ok {
					// a string is content type when it contains a slash and
					// content or custom struct is being calculated already;
					// (string -> content, string-> content type)
					// (customStruct, string -> content type)
					if (len(content) > 0 || custom != nil) && strings.IndexByte(s, slashB) > 0 {
						contentType = s
					} else {
						// otherwise is content
						content = []byte(s)
					}

					continue
				}

				if b, ok := f.([]byte); ok {
					// it's raw content, get the latest
					content = b
					continue
				}

				if e, ok := f.(compatibleErr); ok {
					if e != nil { // it's always not nil but keep it here.
						err = e
						if statusCode < 400 {
							statusCode = DefaultErrStatusCode
						}
						break // break on first error, error should be in the end but we
						// need to know break the dispatcher if any error.
						// at the end; we don't want to write anything to the response if error is not nil.
					}
					continue
				}

				// else it's a custom struct or a dispatcher, we'll decide later
				// because content type and status code matters
				// do that check in order to be able to correctly dispatch:
				// (customStruct, error) -> customStruct filled and error is nil
				if custom == nil && f != nil {
					custom = f
				}

			}

		*/
		switch value := f.(type) {
		case bool:
			found = value
			if !found {
				// skip everything, skip other values, we don't care about other return values,
				// this boolean is the higher in order.
				break
			}
		case int:
			statusCode = value
		case string:
			// a string is content type when it contains a slash and
			// content or custom struct is being calculated already;
			// (string -> content, string-> content type)
			// (customStruct, string -> content type)
			if (len(content) > 0 || custom != nil) && strings.IndexByte(value, slashB) > 0 {
				contentType = value
			} else {
				// otherwise is content
				contentType = context.ContentTextHeaderValue
				content = []byte(value)
			}

		case []byte:
			// it's raw content, get the latest
			content = value
		case compatibleErr:
			if value == nil || isNil(v) {
				continue
			}

			if statusCode < 400 && value != ErrStopExecution {
				statusCode = DefaultErrStatusCode
			}

			ctx.StatusCode(statusCode)
			return value
		default:
			// else it's a custom struct or a dispatcher, we'll decide later
			// because content type and status code matters
			// do that check in order to be able to correctly dispatch:
			// (customStruct, error) -> customStruct filled and error is nil
			if custom == nil {
				// if it's a pointer to struct/map.

				if isNil(v) {
					// if just a ptr to struct with no content type given
					// then try to get the previous response writer's content type,
					// and if that is empty too then force-it to application/json
					// as the default content type we use for structs/maps.
					if contentType == "" {
						contentType = ctx.GetContentType()
						if contentType == "" {
							contentType = context.ContentJSONHeaderValue
						}
					}

					continue
				}

				if value != nil {
					custom = value // content type will be take care later on.
				}
			}
		}
	}

	return dispatchCommon(ctx, statusCode, contentType, content, custom, handler, found)
}

// dispatchCommon is being used internally to send
// commonly used data to the response writer with a smart way.
func dispatchCommon(ctx *context.Context,
	statusCode int, contentType string, content []byte, v interface{}, handler ResultHandler, found bool) error {
	// if we have a false boolean as a return value
	// then skip everything and fire a not found,
	// we even don't care about the given status code or the object or the content.
	if !found {
		ctx.NotFound()
		return nil
	}

	status := statusCode
	if status == 0 {
		status = 200
	}

	// write the status code, the rest will need that before any write ofc.
	ctx.StatusCode(status)
	if contentType == "" {
		// to respect any ctx.ContentType(...) call
		// especially if v is not nil.
		if contentType = ctx.GetContentType(); contentType == "" {
			// if it's still empty set to JSON. (useful for dynamic middlewares that returns an int status code and the next handler dispatches the JSON,
			// see dependency-injection/basic/middleware example)
			contentType = context.ContentJSONHeaderValue
		}
	}

	// write the content type now (internal check for empty value)
	ctx.ContentType(contentType)

	if v != nil {
		return handler(ctx, v)
	}

	// .Write even len(content) == 0 , this should be called in order to call the internal tryWriteHeader,
	// it will not cost anything.
	_, err := ctx.Write(content)
	return err
}

// Response completes the `methodfunc.Result` interface.
// It's being used as an alternative return value which
// wraps the status code, the content type, a content as bytes or as string
// and an error, it's smart enough to complete the request and send the correct response to the client.
type Response struct {
	Code        int
	ContentType string
	Content     []byte

	// If not empty then content type is the "text/plain"
	// and content is the text as []byte. If not empty and
	// the "Lang" field is not empty then this "Text" field
	// becomes the current locale file's key.
	Text string
	// If not empty then "Text" field becomes the locale file's key that should point
	// to a translation file's unique key. See `Object` for locale template data.
	// The "Lang" field is the language code
	// that should render the text inside the locale file's key.
	Lang string
	// If not nil then it will fire that as "application/json" or any
	// previously set "ContentType". If "Lang" and "Text" are not empty
	// then this "Object" field becomes the template data that the
	// locale text should use to be rendered.
	Object interface{}

	// If Path is not empty then it will redirect
	// the client to this Path, if Code is >= 300 and < 400
	// then it will use that Code to do the redirection, otherwise
	// StatusFound(302) or StatusSeeOther(303) for post methods will be used.
	// Except when err != nil.
	Path string

	// if not empty then fire a 400 bad request error
	// unless the Status is > 200, then fire that error code
	// with the Err.Error() string as its content.
	//
	// if Err.Error() is empty then it fires the custom error handler
	// if any otherwise the framework sends the default http error text based on the status.
	Err error
	Try func() int

	// if true then it skips everything else and it throws a 404 not found error.
	// Can be named as Failure but NotFound is more precise name in order
	// to be visible that it's different than the `Err`
	// because it throws a 404 not found instead of a 400 bad request.
	// NotFound bool
	// let's don't add this yet, it has its dangerous of missuse.
}

var _ Result = Response{}

// Dispatch writes the response result to the context's response writer.
func (r Response) Dispatch(ctx *context.Context) {
	if dispatchErr(ctx, r.Code, r.Err) {
		return
	}

	if r.Path != "" {
		// it's not a redirect valid status
		if r.Code < 300 || r.Code >= 400 {
			if ctx.Method() == "POST" {
				r.Code = 303 // StatusSeeOther
			}
			r.Code = 302 // StatusFound
		}
		ctx.Redirect(r.Path, r.Code)
		return
	}

	if r.Text != "" {
		if r.Lang != "" {
			if r.Code > 0 {
				ctx.StatusCode(r.Code)
			}
			ctx.ContentType(r.ContentType)

			ctx.SetLanguage(r.Lang)
			r.Content = []byte(ctx.Tr(r.Text, r.Object))
		} else {
			r.Content = []byte(r.Text)
		}
	}

	err := dispatchCommon(ctx, r.Code, r.ContentType, r.Content, r.Object, defaultResultHandler, true)
	dispatchErr(ctx, r.Code, err)
}

// View completes the `hero.Result` interface.
// It's being used as an alternative return value which
// wraps the template file name, layout, (any) view data, status code and error.
// It's smart enough to complete the request and send the correct response to the client.
//
// Example at: https://github.com/kataras/iris/blob/main/_examples/dependency-injection/overview/web/routes/hello.go.
type View struct {
	Name   string
	Layout string
	Data   interface{} // map or a custom struct.
	Code   int
	Err    error
}

var _ Result = View{}

// Dispatch writes the template filename, template layout and (any) data to the  client.
// Completes the `Result` interface.
func (r View) Dispatch(ctx *context.Context) { // r as Response view.
	if dispatchErr(ctx, r.Code, r.Err) {
		return
	}

	if r.Code > 0 {
		ctx.StatusCode(r.Code)
	}

	if r.Name != "" {
		if r.Layout != "" {
			ctx.ViewLayout(r.Layout)
		}

		if r.Data != nil {
			// In order to respect any c.Ctx.ViewData that may called manually before;
			dataKey := ctx.Application().ConfigurationReadOnly().GetViewDataContextKey()
			if ctx.Values().Get(dataKey) == nil {
				// if no c.Ctx.ViewData set-ed before (the most common scenario) then do a
				// simple set, it's faster.
				ctx.Values().Set(dataKey, r.Data)
			} else {
				// else check if r.Data is map or struct, if struct convert it to map,
				// do a range loop and modify the data one by one.
				// context.Map is actually a map[string]interface{} but we have to make that check:
				if m, ok := r.Data.(context.Map); ok {
					setViewData(ctx, m)
				} else if reflect.Indirect(reflect.ValueOf(r.Data)).Kind() == reflect.Struct {
					setViewData(ctx, structs.Map(r))
				}
			}
		}

		_ = ctx.View(r.Name)
	}
}

func setViewData(ctx *context.Context, data map[string]interface{}) {
	for k, v := range data {
		ctx.ViewData(k, v)
	}
}
