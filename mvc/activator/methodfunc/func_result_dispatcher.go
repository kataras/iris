package methodfunc

import (
	"reflect"
	"strings"

	"github.com/kataras/iris/context"
)

// Result is a response dispatcher.
// All types that complete this interface
// can be returned as values from the method functions.
type Result interface {
	// Dispatch should sends the response to the context's response writer.
	Dispatch(ctx context.Context)
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
	numOut := len(values)
	if numOut == 0 {
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
		if !v.IsValid() {
			continue
		}

		f := v.Interface()

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

	DispatchCommon(ctx, statusCode, contentType, content, custom, err, found)
}
