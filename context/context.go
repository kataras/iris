package context

import (
	"bufio"
	"html/template"
	"io"
	"time"

	"github.com/kataras/iris/sessions/store"
	"github.com/valyala/fasthttp"
)

type (
	// IContext the interface for the iris/context
	// Used mostly inside packages which shouldn't be import ,directly, the kataras/iris.
	IContext interface {
		Param(string) string
		ParamInt(string) (int, error)
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
		PostFormValue(string) string
		PostFormMulti(string) []string
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
		RenderWithStatus(int, string, interface{}, ...string) error
		Render(string, interface{}, ...string) error
		MustRender(string, interface{}, ...string)
		TemplateString(string, interface{}, ...string) string
		MarkdownString(string) string
		Markdown(int, string)
		JSON(int, interface{}) error
		JSONP(int, string, interface{}) error
		Text(int, string) error
		XML(int, interface{}) error
		ExecuteTemplate(*template.Template, interface{}) error
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
		SetCookie(*fasthttp.Cookie)
		SetCookieKV(string, string)
		RemoveCookie(string)
		GetFlash(string) string
		GetFlashBytes(string) ([]byte, error)
		SetFlash(string, string)
		SetFlashBytes(string, []byte)
		Session() store.IStore
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
