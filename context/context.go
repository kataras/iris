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
// DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER AND CONTRIBUTOR, GERASIMOS MAROPOULOS
// BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
// (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
// LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
// ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
// SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

package context

import (
	"bufio"
	"html/template"
	"io"
	"time"

	"github.com/kataras/iris/sessions/store"
	"github.com/valyala/fasthttp"
	"golang.org/x/net/context"
)

type (
	// IContext the interface for the Context
	IContext interface {
		context.Context
		IContextRenderer
		IContextStorage
		IContextBinder
		IContextRequest
		IContextResponse

		Reset(*fasthttp.RequestCtx)
		GetRequestCtx() *fasthttp.RequestCtx
		Clone() IContext
		Do()
		Next()
		StopExecution()
		IsStopped() bool
		GetHandlerName() string
	}

	// IContextBinder is part of the IContext
	IContextBinder interface {
		ReadJSON(interface{}) error
		ReadXML(interface{}) error
		ReadForm(formObject interface{}) error
	}

	// IContextRenderer is part of the IContext
	IContextRenderer interface {
		Write(string, ...interface{})
		WriteHTML(int, string)
		// Data writes out the raw bytes as binary data.
		Data(status int, v []byte) error
		// HTML builds up the response from the specified template and bindings.
		HTML(status int, name string, binding interface{}, layout ...string) error
		// Render same as .HTML but with status to iris.StatusOK (200)
		Render(name string, binding interface{}, layout ...string) error
		// JSON marshals the given interface object and writes the JSON response.
		JSON(status int, v interface{}) error
		// JSONP marshals the given interface object and writes the JSON response.
		JSONP(status int, callback string, v interface{}) error
		// Text writes out a string as plain text.
		Text(status int, v string) error
		// XML marshals the given interface object and writes the XML response.
		XML(status int, v interface{}) error

		ExecuteTemplate(*template.Template, interface{}) error
		ServeContent(io.ReadSeeker, string, time.Time) error
		ServeFile(string) error
		SendFile(filename string, destinationName string) error
		Stream(func(*bufio.Writer))
	}

	// IContextRequest is part of the IContext
	IContextRequest interface {
		Param(string) string
		ParamInt(string) (int, error)
		URLParam(string) string
		URLParamInt(string) (int, error)
		URLParams() map[string][]string
		MethodString() string
		HostString() string
		PathString() string
		RequestIP() string
		RemoteAddr() string
		RequestHeader(k string) string
		PostFormValue(string) string
	}

	// IContextResponse is part of the IContext
	IContextResponse interface {
		// SetContentType sets the "Content-Type" header, receives the values
		SetContentType([]string)
		// SetHeader sets the response headers first parameter is the key, second is the values
		SetHeader(string, []string)
		Redirect(string, ...int)
		// Errors
		NotFound()
		Panic()
		EmitError(int)
		//
	}

	// IContextStorage is part of the IContext
	IContextStorage interface {
		Get(string) interface{}
		GetString(string) string
		GetInt(string) int
		Set(string, interface{})
		SetCookie(*fasthttp.Cookie)
		SetCookieKV(string, string)
		RemoveCookie(string)
		// Flash messages
		GetFlash(string) string
		GetFlashBytes(string) ([]byte, error)
		SetFlash(string, string)
		SetFlashBytes(string, []byte)
		Session() store.IStore
		SessionDestroy()
	}
)
