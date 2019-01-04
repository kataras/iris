package hero

import (
	"reflect"
	"strings"

	"github.com/kataras/iris/context"
	"github.com/kataras/iris/hero/di"

	"github.com/fatih/structs"
)

// Result is a response dispatcher.
// All types that complete this interface
// can be returned as values from the method functions.
//
// Example at: https://github.com/kataras/iris/tree/master/_examples/hero/overview.
type Result interface {
	// Dispatch should sends the response to the context's response writer.
	Dispatch(ctx context.Context)
}

var defaultFailureResponse = Response{Code: DefaultErrStatusCode}

// Try will check if "fn" ran without any panics,
// using recovery,
// and return its result as the final response
// otherwise it returns the "failure" response if any,
// if not then a 400 bad request is being sent.
//
// Example usage at: https://github.com/kataras/iris/blob/master/hero/func_result_test.go.
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

// DefaultErrStatusCode is the default error status code (400)
// when the response contains an error which is not nil.
var DefaultErrStatusCode = 400

// DispatchErr writes the error to the response.
func DispatchErr(ctx context.Context, status int, err error) {
	if status < 400 {
		status = DefaultErrStatusCode
	}
	ctx.StatusCode(status)
	if text := err.Error(); text != "" {
		ctx.WriteString(text)
		ctx.StopExecution()
	}
}

// DispatchCommon is being used internally to send
// commonly used data to the response writer with a smart way.
func DispatchCommon(ctx context.Context,
	statusCode int, contentType string, content []byte, v interface{}, err error, found bool) {

	// if we have a false boolean as a return value
	// then skip everything and fire a not found,
	// we even don't care about the given status code or the object or the content.
	if !found {
		ctx.NotFound()
		return
	}

	status := statusCode
	if status == 0 {
		status = 200
	}

	if err != nil {
		DispatchErr(ctx, status, err)
		return
	}

	// write the status code, the rest will need that before any write ofc.
	ctx.StatusCode(status)
	if contentType == "" {
		// to respect any ctx.ContentType(...) call
		// especially if v is not nil.
		contentType = ctx.GetContentType()
	}

	if v != nil {
		if d, ok := v.(Result); ok {
			// write the content type now (internal check for empty value)
			ctx.ContentType(contentType)
			d.Dispatch(ctx)
			return
		}

		if strings.HasPrefix(contentType, context.ContentJavascriptHeaderValue) {
			_, err = ctx.JSONP(v)
		} else if strings.HasPrefix(contentType, context.ContentXMLHeaderValue) {
			_, err = ctx.XML(v, context.XML{Indent: " "})
		} else {
			// defaults to json if content type is missing or its application/json.
			_, err = ctx.JSON(v, context.JSON{Indent: " "})
		}

		if err != nil {
			DispatchErr(ctx, status, err)
		}

		return
	}

	ctx.ContentType(contentType)
	// .Write even len(content) == 0 , this should be called in order to call the internal tryWriteHeader,
	// it will not cost anything.
	ctx.Write(content)
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
func DispatchFuncResult(ctx context.Context, values []reflect.Value) {
	if len(values) == 0 {
		return
	}

	var (
		// if statusCode > 0 then send this status code.
		// Except when err != nil then check if status code is < 400 and
		// if it's set it as DefaultErrStatusCode.
		// Except when found == false, then the status code is 404.
		statusCode int
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
		// if not nil then check for its status code,
		// if not status code or < 400 then set it as DefaultErrStatusCode
		// and fire the error's text.
		err error
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
				content = []byte(value)
			}

		case []byte:
			// it's raw content, get the latest
			content = value
		case compatibleErr:
			if value != nil { // it's always not nil but keep it here.
				err = value
				if statusCode < 400 {
					statusCode = DefaultErrStatusCode
				}
				break // break on first error, error should be in the end but we
				// need to know break the dispatcher if any error.
				// at the end; we don't want to write anything to the response if error is not nil.
			}
		default:
			// else it's a custom struct or a dispatcher, we'll decide later
			// because content type and status code matters
			// do that check in order to be able to correctly dispatch:
			// (customStruct, error) -> customStruct filled and error is nil
			if custom == nil && f != nil {
				custom = f
			}
		}
	}

	DispatchCommon(ctx, statusCode, contentType, content, custom, err, found)
}

// Response completes the `methodfunc.Result` interface.
// It's being used as an alternative return value which
// wraps the status code, the content type, a content as bytes or as string
// and an error, it's smart enough to complete the request and send the correct response to the client.
type Response struct {
	Code        int
	ContentType string
	Content     []byte

	// if not empty then content type is the text/plain
	// and content is the text as []byte.
	Text string
	// If not nil then it will fire that as "application/json" or the
	// "ContentType" if not empty.
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
func (r Response) Dispatch(ctx context.Context) {
	if r.Path != "" && r.Err == nil {
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

	if s := r.Text; s != "" {
		r.Content = []byte(s)
	}

	DispatchCommon(ctx, r.Code, r.ContentType, r.Content, r.Object, r.Err, true)
}

// View completes the `hero.Result` interface.
// It's being used as an alternative return value which
// wraps the template file name, layout, (any) view data, status code and error.
// It's smart enough to complete the request and send the correct response to the client.
//
// Example at: https://github.com/kataras/iris/blob/master/_examples/hero/overview/web/controllers/hello_controller.go.
type View struct {
	Name   string
	Layout string
	Data   interface{} // map or a custom struct.
	Code   int
	Err    error
}

var _ Result = View{}

const dotB = byte('.')

// DefaultViewExt is the default extension if `view.Name `is missing,
// but note that it doesn't care about
// the app.RegisterView(iris.$VIEW_ENGINE("./$dir", "$ext"))'s $ext.
// so if you don't use the ".html" as extension for your files
// you have to append the extension manually into the `view.Name`
// or change this global variable.
var DefaultViewExt = ".html"

func ensureExt(s string) string {
	if len(s) == 0 {
		return "index" + DefaultViewExt
	}

	if strings.IndexByte(s, dotB) < 1 {
		s += DefaultViewExt
	}

	return s
}

// Dispatch writes the template filename, template layout and (any) data to the  client.
// Completes the `Result` interface.
func (r View) Dispatch(ctx context.Context) { // r as Response view.
	if r.Err != nil {
		if r.Code < 400 {
			r.Code = DefaultErrStatusCode
		}
		ctx.StatusCode(r.Code)
		ctx.WriteString(r.Err.Error())
		ctx.StopExecution()
		return
	}

	if r.Code > 0 {
		ctx.StatusCode(r.Code)
	}

	if r.Name != "" {
		r.Name = ensureExt(r.Name)

		if r.Layout != "" {
			r.Layout = ensureExt(r.Layout)
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
				if m, ok := r.Data.(map[string]interface{}); ok {
					setViewData(ctx, m)
				} else if m, ok := r.Data.(context.Map); ok {
					setViewData(ctx, m)
				} else if di.IndirectValue(reflect.ValueOf(r.Data)).Kind() == reflect.Struct {
					setViewData(ctx, structs.Map(r))
				}
			}
		}

		ctx.View(r.Name)
	}
}

func setViewData(ctx context.Context, data map[string]interface{}) {
	for k, v := range data {
		ctx.ViewData(k, v)
	}
}
