package context

import (
	"bufio"
	"io"
	"time"

	"github.com/valyala/fasthttp"
)

type (

	// IContext the interface for the iris/context
	// Used mostly inside packages which shouldn't be import ,directly, the kataras/iris.
	IContext interface {
		Param(string) string
		ParamInt(string) (int, error)
		ParamInt64(string) (int64, error)
		URLParam(string) string
		URLParamInt(string) (int, error)
		URLParamInt64(string) (int64, error)
		URLParams() map[string]string
		MethodString() string
		HostString() string
		Subdomain() string
		PathString() string
		RequestPath(bool) string
		RequestIP() string
		RemoteAddr() string
		RequestHeader(k string) string
		FormValueString(string) string
		FormValues(string) []string
		SetStatusCode(int)
		SetContentType(string)
		SetHeader(string, string)
		Redirect(string, ...int)
		RedirectTo(string, ...interface{})
		NotFound()
		Panic()
		EmitError(int)
		Write(string, ...interface{})
		HTML(int, string)
		Data(int, []byte) error
		RenderWithStatus(int, string, interface{}, ...map[string]interface{}) error
		Render(string, interface{}, ...map[string]interface{}) error
		MustRender(string, interface{}, ...map[string]interface{})
		TemplateString(string, interface{}, ...map[string]interface{}) string
		MarkdownString(string) string
		Markdown(int, string)
		JSON(int, interface{}) error
		JSONP(int, string, interface{}) error
		Text(int, string) error
		XML(int, interface{}) error
		ServeContent(io.ReadSeeker, string, time.Time, bool) error
		ServeFile(string, bool) error
		SendFile(string, string) error
		Stream(func(*bufio.Writer))
		StreamWriter(cb func(*bufio.Writer))
		StreamReader(io.Reader, int)
		ReadJSON(interface{}) error
		ReadXML(interface{}) error
		ReadForm(interface{}) error
		Get(string) interface{}
		GetString(string) string
		GetInt(string) int
		Set(string, interface{})
		VisitAllCookies(func(string, string))
		SetCookie(*fasthttp.Cookie)
		SetCookieKV(string, string)
		RemoveCookie(string)
		GetFlashes() map[string]string
		GetFlash(string) (string, error)
		SetFlash(string, string)
		Session() interface {
			ID() string
			Get(string) interface{}
			GetString(key string) string
			GetInt(key string) int
			GetAll() map[string]interface{}
			VisitAll(cb func(k string, v interface{}))
			Set(string, interface{})
			Delete(string)
			Clear()
		}
		SessionDestroy()
		Log(string, ...interface{})
		Reset(*fasthttp.RequestCtx)
		GetRequestCtx() *fasthttp.RequestCtx
		Clone() IContext
		Do()
		Next()
		StopExecution()
		IsStopped() bool
		GetHandlerName() string
	}
)
