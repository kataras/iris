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

	// HTTPErrorHandler is just an object which stores a http status code and a handler
	HTTPErrorHandler struct {
		code    int
		handler HandlerFunc
	}

	// HTTPErrorContainer is the struct which contains the handlers which will execute if http error occurs
	// One struct per Server instance, the meaning of this is that the developer can change the default error message and replace them with his/her own completely custom handlers
	//
	// Example of usage:
	// iris.OnError(405, func (ctx *iris.Context){ c.SendStatus(405,"Method not allowed!!!")})
	// and inside the handler which you have access to the current Context:
	// ctx.EmitError(405)
	HTTPErrorContainer struct {
		// Errors contains all the httperrorhandlers
		Errors []*HTTPErrorHandler
	}
)

// HTTPErrorHandlerFunc creates a handler which is responsible to send a particular error to the client
func HTTPErrorHandlerFunc(statusCode int, message string) HandlerFunc {
	return func(ctx *Context) {
		ctx.SetStatusCode(statusCode)
		ctx.SetBodyString(message)
	}
}

// GetCode returns the http status code value
func (e *HTTPErrorHandler) GetCode() int {
	return e.code
}

// GetHandler returns the handler which is type of HandlerFunc
func (e *HTTPErrorHandler) GetHandler() HandlerFunc {
	return e.handler
}

// SetHandler sets the handler (type of HandlerFunc) to this particular ErrorHandler
func (e *HTTPErrorHandler) SetHandler(h HandlerFunc) {
	e.handler = h
}

// defaultHTTPErrors creates and returns an instance of HTTPErrorContainer with default handlers
func defaultHTTPErrors() *HTTPErrorContainer {
	httperrors := new(HTTPErrorContainer)
	httperrors.Errors = make([]*HTTPErrorHandler, 0)
	httperrors.OnError(StatusNotFound, HTTPErrorHandlerFunc(StatusNotFound, statusText[StatusNotFound]))
	httperrors.OnError(StatusInternalServerError, HTTPErrorHandlerFunc(StatusInternalServerError, statusText[StatusInternalServerError]))
	return httperrors
}

// GetByCode returns the error handler by it's http status code
func (he *HTTPErrorContainer) GetByCode(httpStatus int) *HTTPErrorHandler {
	if he != nil {
		for _, h := range he.Errors {
			if h.GetCode() == httpStatus {
				return h
			}
		}
	}

	return nil
}

// OnError Registers a handler for a specific http error status
func (he *HTTPErrorContainer) OnError(httpStatus int, handler HandlerFunc) {
	if httpStatus == StatusOK {
		return
	}

	if errH := he.GetByCode(httpStatus); errH != nil {

		errH.SetHandler(handler)
	} else {
		he.Errors = append(he.Errors, &HTTPErrorHandler{code: httpStatus, handler: handler})
	}

}

///TODO: the errors must have .Next too, as middlewares inside the Context, if I let it as it is then we have problem
// we cannot set a logger and a custom handler at one error because now the error handler takes only one handelrFunc and executes there from here...

// EmitError executes the handler of the given error http status code
func (he *HTTPErrorContainer) EmitError(errCode int, ctx *Context) {

	if errHandler := he.GetByCode(errCode); errHandler != nil {
		ctx.SetStatusCode(errCode) // for any case, user can change it after if want to
		errHandler.GetHandler().Serve(ctx)
	} else {
		//if no error is registed, then register it with the default http error text, and re-run the Emit
		he.OnError(errCode, func(c *Context) {
			c.SetStatusCode(errCode)
			c.SetBodyString(StatusText(errCode))
		})
		he.EmitError(errCode, ctx)
	}
}

// OnNotFound sets the handler for http status 404,
// default is a response with text: 'Not Found' and status: 404
func (he *HTTPErrorContainer) OnNotFound(handlerFunc HandlerFunc) {
	he.OnError(StatusNotFound, handlerFunc)
}

// OnPanic sets the handler for http status 500,
// default is a response with text: The server encountered an unexpected condition which prevented it from fulfilling the request. and status: 500
func (he *HTTPErrorContainer) OnPanic(handlerFunc HandlerFunc) {
	he.OnError(StatusInternalServerError, handlerFunc)
}
