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
// DISCLAIMED. IN NO EVENT SHALL JULIEN SCHMIDT BE LIABLE FOR ANY
// DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
// (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
// LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
// ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
// SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
package iris

import (
	"net/http"
)

type IErrorHandler interface {
	GetCode() int
	GetHandler() HandlerFunc
	SetHandler(h HandlerFunc)
}

type IHTTPErrors interface {
	GetByCode(httpStatus int) IErrorHandler
	On(httpStatus int, handler HandlerFunc)
	SetNotFound(handler HandlerFunc)
	Emit(errCode int, ctx *Context)
	NotFound(ctx *Context)
}

// ErrorHandler creates a handler which is responsible to send a particular error to the client
func ErrorHandlerFunc(statusCode int, message string) HandlerFunc {
	return func(ctx *Context) {
		ctx.SendStatus(statusCode, message)
	}
}

// ErrorHandlers just an array of struct{ code int, handler http.Handler}
type ErrorHandler struct {
	code    int
	handler HandlerFunc
}

func (e ErrorHandler) GetCode() int {
	return e.code
}

func (e ErrorHandler) GetHandler() HandlerFunc {
	return e.handler
}

func (e *ErrorHandler) SetHandler(h HandlerFunc) {
	e.handler = h
}

var _ IErrorHandler = &ErrorHandler{}

// HTTPErrors is the struct which contains the handlers which will execute if http error occurs
// One struct per Server instance, the meaning of this is that the developer can change the default error message and replace them with his/her own completely custom handlers
type HTTPErrors struct {
	//developer can do Errors.On(500, iris.Handler)
	ErrorHanders []IErrorHandler
}

// DefaultHTTPErrors creates and returns an instance of HTTPErrors with default handlers
func DefaultHTTPErrors() IHTTPErrors {
	httperrors := new(HTTPErrors)
	httperrors.ErrorHanders = make([]IErrorHandler, 0)
	httperrors.SetNotFound(ErrorHandlerFunc(http.StatusNotFound, "404 not found"))
	return httperrors
}

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

// On Registers a handler for a specific http error status ( overrides the NotFound and MethodNotAllowed)
func (he *HTTPErrors) On(httpStatus int, handler HandlerFunc) {
	if httpStatus == http.StatusOK {
		return
	}

	/*	httpHandlerOfficialType := reflect.TypeOf((*http.Handler)(nil)).Elem()
		if !reflect.TypeOf(handler).Implements(httpHandlerOfficialType) {
			//it is not a http.Handler
			//it is func(res,req) we will convert it to a handler using http.HandlerFunc
			handler = ToHandlerFunc(handler.(func(res http.ResponseWriter, req *http.Request)))
		}
	*/
	if errH := he.GetByCode(httpStatus); errH != nil {
		errH.SetHandler(handler)
	} else {
		he.ErrorHanders = append(he.ErrorHanders, &ErrorHandler{code: httpStatus, handler: handler})
	}

}

// SetNotFound this func could named it OnNotFound also, registers a custom StatusNotFound error 404 handler
// Possible parameter: iris.Handler or iris.HandlerFunc(func(ctx *Context){})
func (he *HTTPErrors) SetNotFound(handler HandlerFunc) {
	he.On(http.StatusNotFound, handler)
}

// Emit executes the handler of the given error http status code
func (he *HTTPErrors) Emit(errCode int, ctx *Context) {

	if errHandler := he.GetByCode(errCode); errHandler != nil {
		errHandler.GetHandler().Serve(ctx)

	}
}

// NotFound emits the registed NotFound (404) custom (or not) handler
func (he *HTTPErrors) NotFound(ctx *Context) {
	he.Emit(http.StatusNotFound, ctx)
}

var _ IHTTPErrors = &HTTPErrors{}
