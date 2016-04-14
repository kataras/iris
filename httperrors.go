// Copyright (c) 2016, Gerasimos Maropoulos
// All rights reserved.
//
// Redistribution and use in source and binary forms, with or without modification,
// are permitted provided that the following conditions are met:
//
// 1. Redistributions of source code must retain the above copyright notice,
//    this list of conditions and the following disclaimer.
//
// 2. Redistributions in binary form must reproduce the above copyright notice,
//	  this list of conditions and the following disclaimer
//    in the documentation and/or other materials provided with the distribution.
//
// 3. Neither the name of the copyright holder nor the names of its contributors may be used to endorse
//    or promote products derived from this software without specific prior written permission.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
// ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
// WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
// DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER AND CONTRIBUTOR, GERASIMOS MAROPOULOS BE LIABLE FOR ANY
// DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
// (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
// LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
// ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
// SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
package iris

//taken from net/http
const (
	StatusContinue           = 100
	StatusSwitchingProtocols = 101

	StatusOK                   = 200
	StatusCreated              = 201
	StatusAccepted             = 202
	StatusNonAuthoritativeInfo = 203
	StatusNoContent            = 204
	StatusResetContent         = 205
	StatusPartialContent       = 206

	StatusMultipleChoices   = 300
	StatusMovedPermanently  = 301
	StatusFound             = 302
	StatusSeeOther          = 303
	StatusNotModified       = 304
	StatusUseProxy          = 305
	StatusTemporaryRedirect = 307

	StatusBadRequest                   = 400
	StatusUnauthorized                 = 401
	StatusPaymentRequired              = 402
	StatusForbidden                    = 403
	StatusNotFound                     = 404
	StatusMethodNotAllowed             = 405
	StatusNotAcceptable                = 406
	StatusProxyAuthRequired            = 407
	StatusRequestTimeout               = 408
	StatusConflict                     = 409
	StatusGone                         = 410
	StatusLengthRequired               = 411
	StatusPreconditionFailed           = 412
	StatusRequestEntityTooLarge        = 413
	StatusRequestURITooLong            = 414
	StatusUnsupportedMediaType         = 415
	StatusRequestedRangeNotSatisfiable = 416
	StatusExpectationFailed            = 417
	StatusTeapot                       = 418
	StatusPreconditionRequired         = 428
	StatusTooManyRequests              = 429
	StatusRequestHeaderFieldsTooLarge  = 431
	StatusUnavailableForLegalReasons   = 451

	StatusInternalServerError           = 500
	StatusNotImplemented                = 501
	StatusBadGateway                    = 502
	StatusServiceUnavailable            = 503
	StatusGatewayTimeout                = 504
	StatusHTTPVersionNotSupported       = 505
	StatusNetworkAuthenticationRequired = 511
)

var statusText = map[int]string{
	StatusContinue:           "Continue",
	StatusSwitchingProtocols: "Switching Protocols",

	StatusOK:                   "OK",
	StatusCreated:              "Created",
	StatusAccepted:             "Accepted",
	StatusNonAuthoritativeInfo: "Non-Authoritative Information",
	StatusNoContent:            "No Content",
	StatusResetContent:         "Reset Content",
	StatusPartialContent:       "Partial Content",

	StatusMultipleChoices:   "Multiple Choices",
	StatusMovedPermanently:  "Moved Permanently",
	StatusFound:             "Found",
	StatusSeeOther:          "See Other",
	StatusNotModified:       "Not Modified",
	StatusUseProxy:          "Use Proxy",
	StatusTemporaryRedirect: "Temporary Redirect",

	StatusBadRequest:                   "Bad Request",
	StatusUnauthorized:                 "Unauthorized",
	StatusPaymentRequired:              "Payment Required",
	StatusForbidden:                    "Forbidden",
	StatusNotFound:                     "Not Found",
	StatusMethodNotAllowed:             "Method Not Allowed",
	StatusNotAcceptable:                "Not Acceptable",
	StatusProxyAuthRequired:            "Proxy Authentication Required",
	StatusRequestTimeout:               "Request Timeout",
	StatusConflict:                     "Conflict",
	StatusGone:                         "Gone",
	StatusLengthRequired:               "Length Required",
	StatusPreconditionFailed:           "Precondition Failed",
	StatusRequestEntityTooLarge:        "Request Entity Too Large",
	StatusRequestURITooLong:            "Request URI Too Long",
	StatusUnsupportedMediaType:         "Unsupported Media Type",
	StatusRequestedRangeNotSatisfiable: "Requested Range Not Satisfiable",
	StatusExpectationFailed:            "Expectation Failed",
	StatusTeapot:                       "I'm a teapot",
	StatusPreconditionRequired:         "Precondition Required",
	StatusTooManyRequests:              "Too Many Requests",
	StatusRequestHeaderFieldsTooLarge:  "Request Header Fields Too Large",
	StatusUnavailableForLegalReasons:   "Unavailable For Legal Reasons",

	StatusInternalServerError:           "Internal Server Error",
	StatusNotImplemented:                "Not Implemented",
	StatusBadGateway:                    "Bad Gateway",
	StatusServiceUnavailable:            "Service Unavailable",
	StatusGatewayTimeout:                "Gateway Timeout",
	StatusHTTPVersionNotSupported:       "HTTP Version Not Supported",
	StatusNetworkAuthenticationRequired: "Network Authentication Required",
}

// StatusText returns a text for the HTTP status code. It returns the empty
// string if the code is unknown.
func StatusText(code int) string {
	return statusText[code]
}

//

type (
	// IErrorHandler is the interface which an http error handler should implement
	IErrorHandler interface {
		GetCode() int
		GetHandler() HandlerFunc
		SetHandler(h HandlerFunc)
	}

	// IHTTPErrors is the interface which the HTTPErrors using
	IHTTPErrors interface {
		GetByCode(httpStatus int) IErrorHandler
		On(httpStatus int, handler HandlerFunc)
		Emit(errCode int, ctx *Context)
	}

	// ErrorHandler is just an object which stores a http status code and a handler
	ErrorHandler struct {
		code    int
		handler HandlerFunc
	}

	// HTTPErrors is the struct which contains the handlers which will execute if http error occurs
	// One struct per Server instance, the meaning of this is that the developer can change the default error message and replace them with his/her own completely custom handlers
	//
	// Example of usage:
	// iris.OnError(405, func (ctx *iris.Context){ c.SendStatus(405,"Method not allowed!!!")})
	// and inside the handler which you have access to the current Context:
	// ctx.EmitError(405)
	// that is the circle, the httpErrors variable stays at the Station(via it's Router), sets from there and emits from a context,
	// but you can also emit directly from iris.Errors().Emit(405,ctx) if that's necessary
	HTTPErrors struct {
		//developer can do Errors.On(500, iris.Handler)
		ErrorHanders []IErrorHandler
	}
)

var _ IErrorHandler = &ErrorHandler{}
var _ IHTTPErrors = &HTTPErrors{}

// ErrorHandlerFunc creates a handler which is responsible to send a particular error to the client
func ErrorHandlerFunc(statusCode int, message string) HandlerFunc {
	return func(ctx *Context) {
		ctx.WriteText(statusCode, message)
	}
}

// GetCode returns the http status code value
func (e ErrorHandler) GetCode() int {
	return e.code
}

// GetHandler returns the handler which is type of HandlerFunc
func (e ErrorHandler) GetHandler() HandlerFunc {
	return e.handler
}

// SetHandler sets the handler (type of HandlerFunc) to this particular ErrorHandler
func (e *ErrorHandler) SetHandler(h HandlerFunc) {
	e.handler = h
}

// defaultHTTPErrors creates and returns an instance of HTTPErrors with default handlers
func defaultHTTPErrors() *HTTPErrors {
	httperrors := new(HTTPErrors)
	httperrors.ErrorHanders = make([]IErrorHandler, 0)
	httperrors.On(StatusNotFound, ErrorHandlerFunc(StatusNotFound, "404 not found"))
	httperrors.On(StatusInternalServerError, ErrorHandlerFunc(StatusInternalServerError, "The server encountered an unexpected condition which prevented it from fulfilling the request."))
	return httperrors
}

// GetByCode returns the error handler by it's http status code
func (he *HTTPErrors) GetByCode(httpStatus int) IErrorHandler {
	if he == nil {
		return nil
	}
	for _, h := range he.ErrorHanders {
		if h.GetCode() == httpStatus {
			return h
		}
	}
	return nil
}

// On Registers a handler for a specific http error status
func (he *HTTPErrors) On(httpStatus int, handler HandlerFunc) {
	if httpStatus == StatusOK {
		return
	}

	if errH := he.GetByCode(httpStatus); errH != nil {
		errH.SetHandler(handler)
	} else {
		he.ErrorHanders = append(he.ErrorHanders, &ErrorHandler{code: httpStatus, handler: handler})
	}

}

///TODO: the errors must have .Next too, as middlewares inside the Context, if I let it as it is then we have problem
// we cannot set a logger and a custom handler at one error because now the error handler takes only one handelrFunc and executes there from here...

// Emit executes the handler of the given error http status code
func (he *HTTPErrors) Emit(errCode int, ctx *Context) {

	if errHandler := he.GetByCode(errCode); errHandler != nil {
		// I do that because before the user should to: ctx.WriteStatus(404) in order to send the actual error code, maybe this is ok but I changed my mind let's do this here automatically
		ctx.SetStatusCode(errCode)
		errHandler.GetHandler().Serve(ctx)
	} else {
		//if no error is registed, then register it with the default http error text, and re-run the Emit
		he.On(errCode, func(c *Context) {
			c.WriteText(errCode, StatusText(errCode))
		})
		he.Emit(errCode, ctx)
	}
}
