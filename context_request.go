package iris

import (
	"net"
	"strconv"
	"strings"

	"github.com/kataras/iris/bindings"
	"github.com/kataras/iris/utils"
	"github.com/valyala/fasthttp"
)

// Param returns the string representation of the key's path named parameter's value
func (ctx *Context) Param(key string) string {
	return ctx.Params.Get(key)
}

// ParamInt returns the int representation of the key's path named parameter's value
func (ctx *Context) ParamInt(key string) (int, error) {
	val, err := strconv.Atoi(ctx.Param(key))
	return val, err
}

// URLParam returns the get parameter from a request , if any
func (ctx *Context) URLParam(key string) string {
	return string(ctx.RequestCtx.Request.URI().QueryArgs().Peek(key))
}

// URLParams returns a map of a list of each url(query) parameter
func (ctx *Context) URLParams() map[string]string {
	urlparams := make(map[string]string)
	ctx.RequestCtx.Request.URI().QueryArgs().VisitAll(func(key, value []byte) {
		urlparams[string(key)] = string(value)
	})
	return urlparams
}

// URLParamInt returns the get parameter int value from a request , if any
func (ctx *Context) URLParamInt(key string) (int, error) {
	return strconv.Atoi(ctx.URLParam(key))
}

// MethodString returns the HTTP Method
func (ctx *Context) MethodString() string {
	return utils.BytesToString(ctx.Method())
}

// HostString returns the Host of the request( the url as string )
func (ctx *Context) HostString() string {
	return utils.BytesToString(ctx.Host())
}

// PathString returns the full path as string
func (ctx *Context) PathString() string {
	return utils.BytesToString(ctx.Path())
}

// RequestIP gets just the Remote Address from the client.
func (ctx *Context) RequestIP() string {
	if ip, _, err := net.SplitHostPort(strings.TrimSpace(ctx.RequestCtx.RemoteAddr().String())); err == nil {
		return ip
	}
	return ""
}

// RemoteAddr is like RequestIP but it checks for proxy servers also, tries to get the real client's request IP
func (ctx *Context) RemoteAddr() string {
	header := string(ctx.RequestCtx.Request.Header.Peek("X-Real-Ip"))
	realIP := strings.TrimSpace(header)
	if realIP != "" {
		return realIP
	}
	realIP = string(ctx.RequestCtx.Request.Header.Peek("X-Forwarded-For"))
	idx := strings.IndexByte(realIP, ',')
	if idx >= 0 {
		realIP = realIP[0:idx]
	}
	realIP = strings.TrimSpace(realIP)
	if realIP != "" {
		return realIP
	}
	return ctx.RequestIP()

}

// RequestHeader returns the request header's value
// accepts one parameter, the key of the header (string)
// returns string
func (ctx *Context) RequestHeader(k string) string {
	return utils.BytesToString(ctx.RequestCtx.Request.Header.Peek(k))
}

// PostFormValue returns a single value from post request's data
func (ctx *Context) PostFormValue(name string) string {
	return string(ctx.RequestCtx.PostArgs().Peek(name))
}

/* Credits to Manish Singh @kryptodev for URLEncode */
// URLEncode returns the path encoded as url
// useful when you want to pass something to a database and be valid to retrieve it via context.Param
// use it only for special cases, when the default behavior doesn't suits you.
//
// http://www.blooberry.com/indexdot/html/topics/urlencoding.htm
func URLEncode(path string) string {
	if path == "" {
		return ""
	}
	u := fasthttp.AcquireURI()
	u.SetPath(path)
	encodedPath := u.String()[8:]
	fasthttp.ReleaseURI(u)
	return encodedPath
}

// ReadJSON reads JSON from request's body
func (ctx *Context) ReadJSON(jsonObject interface{}) error {
	return bindings.BindJSON(ctx, jsonObject)
}

// ReadXML reads XML from request's body
func (ctx *Context) ReadXML(xmlObject interface{}) error {
	return bindings.BindXML(ctx, xmlObject)
}

// ReadForm binds the formObject  with the form data
// it supports any kind of struct
func (ctx *Context) ReadForm(formObject interface{}) error {
	return bindings.BindForm(ctx, formObject)
}
