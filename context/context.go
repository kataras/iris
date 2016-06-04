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
		// MustRender same as .Render but returns 500 internal server http status (error) if rendering fail
		MustRender(name string, binding interface{}, layout ...string)
		// RenderString accepts a template filename, its context data and returns the result of the parsed template (string)
		RenderString(name string, binding interface{}, layout ...string) (result string, err error)
		// MarkdownString parses the (dynamic) markdown string and returns the converted html string
		MarkdownString(markdown string) string
		// Markdown parses and renders to the client a particular (dynamic) markdown string
		// accepts two parameters
		// first is the http status code
		// second is the markdown string
		Markdown(status int, markdown string)
		// JSON marshals the given interface object and writes the JSON response.
		JSON(status int, v interface{}) error
		// JSONP marshals the given interface object and writes the JSON response.
		JSONP(status int, callback string, v interface{}) error
		// Text writes out a string as plain text.
		Text(status int, v string) error
		// XML marshals the given interface object and writes the XML response.
		XML(status int, v interface{}) error

		ExecuteTemplate(*template.Template, interface{}) error
		ServeContent(io.ReadSeeker, string, time.Time, bool) error
		ServeFile(string, bool) error
		SendFile(filename string, destinationName string) error
		Stream(func(*bufio.Writer))
		StreamWriter(cb func(writer *bufio.Writer))
		StreamReader(io.Reader, int)
	}

	// IContextRequest is part of the IContext
	IContextRequest interface {
		Param(string) string
		ParamInt(string) (int, error)
		URLParam(string) string
		URLParamInt(string) (int, error)
		URLParams() map[string]string
		MethodString() string
		HostString() string
		PathString() string
		RequestIP() string
		RemoteAddr() string
		RequestHeader(k string) string
		PostFormValue(string) string
		GetSubdomain() string
	}

	// IContextResponse is part of the IContext
	IContextResponse interface {
		// SetStatusCode sets the http status code
		SetStatusCode(int)
		// SetContentType sets the "Content-Type" header, receives the value
		SetContentType(string)
		// SetHeader sets the response headers first parameter is the key, second is the value
		SetHeader(string, string)
		Redirect(string, ...int)
		RedirectTo(routeName string, args ...interface{})
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
