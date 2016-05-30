package iris

import (
	"encoding/base64"
	"time"

	"github.com/kataras/iris/sessions/store"
	"github.com/kataras/iris/utils"
	"github.com/valyala/fasthttp"
)

// After v2.2.3 Get/GetFmt/GetString/GetInt/Set are all return values from the RequestCtx.userValues they are reseting on each connection.

// Get returns the user's value from a key
// if doesn't exists returns nil
func (ctx *Context) Get(key string) interface{} {
	return ctx.RequestCtx.UserValue(key)
}

// GetFmt returns a value which has this format: func(format string, args ...interface{}) string
// if doesn't exists returns nil
func (ctx *Context) GetFmt(key string) func(format string, args ...interface{}) string {
	if v, ok := ctx.Get(key).(func(format string, args ...interface{}) string); ok {
		return v
	}
	return func(format string, args ...interface{}) string { return "" }

}

// GetString same as Get but returns the value as string
// if nothing founds returns empty string ""
func (ctx *Context) GetString(key string) string {
	if v, ok := ctx.Get(key).(string); ok {
		return v
	}

	return ""
}

// GetInt same as Get but returns the value as int
// if nothing founds returns -1
func (ctx *Context) GetInt(key string) int {
	if v, ok := ctx.Get(key).(int); ok {
		return v
	}

	return -1
}

// Set sets a value to a key in the values map
func (ctx *Context) Set(key string, value interface{}) {
	ctx.RequestCtx.SetUserValue(key, value)
}

// GetCookie returns cookie's value by it's name
// returns empty string if nothing was found
func (ctx *Context) GetCookie(name string) (val string) {
	bcookie := ctx.RequestCtx.Request.Header.Cookie(name)
	if bcookie != nil {
		val = string(bcookie)
	}
	return
}

// SetCookie adds a cookie
func (ctx *Context) SetCookie(cookie *fasthttp.Cookie) {
	ctx.RequestCtx.Response.Header.SetCookie(cookie)
}

// SetCookieKV adds a cookie, receives just a key(string) and a value(string)
func (ctx *Context) SetCookieKV(key, value string) {
	c := fasthttp.AcquireCookie() // &fasthttp.Cookie{}
	c.SetKey(key)
	c.SetValue(value)
	c.SetHTTPOnly(true)
	c.SetExpire(time.Now().Add(time.Duration(120) * time.Minute))
	ctx.SetCookie(c)
	fasthttp.ReleaseCookie(c)
}

// RemoveCookie deletes a cookie by it's name/key
func (ctx *Context) RemoveCookie(name string) {
	cookie := fasthttp.AcquireCookie()
	cookie.SetKey(name)
	cookie.SetValue("")
	cookie.SetPath("/")
	cookie.SetHTTPOnly(true)
	exp := time.Now().Add(-time.Duration(1) * time.Minute) //RFC says 1 second, but make sure 1 minute because we are using fasthttp
	cookie.SetExpire(exp)
	ctx.Response.Header.SetCookie(cookie)
	fasthttp.ReleaseCookie(cookie)
}

// GetFlash get a flash message by it's key
// after this action the messages is removed
// returns string, if the cookie doesn't exists the string is empty
func (ctx *Context) GetFlash(key string) string {
	val, err := ctx.GetFlashBytes(key)
	if err != nil {
		return ""
	}
	return string(val)
}

// GetFlashBytes get a flash message by it's key
// after this action the messages is removed
// returns []byte along with an error if the cookie doesn't exists or decode fails
func (ctx *Context) GetFlashBytes(key string) (value []byte, err error) {
	cookieValue := string(ctx.RequestCtx.Request.Header.Cookie(key))
	if cookieValue == "" {
		err = ErrFlashNotFound.Return()
	} else {
		value, err = base64.URLEncoding.DecodeString(cookieValue)
		//remove the message
		ctx.RemoveCookie(key)
		//it should'b be removed until the next reload, so we don't do that: ctx.Request.Header.SetCookie(key, "")
	}
	return
}

// SetFlash sets a flash message, accepts 2 parameters the key(string) and the value(string)
func (ctx *Context) SetFlash(key string, value string) {
	ctx.SetFlashBytes(key, utils.StringToBytes(value))
}

// SetFlashBytes sets a flash message, accepts 2 parameters the key(string) and the value([]byte)
func (ctx *Context) SetFlashBytes(key string, value []byte) {
	c := fasthttp.AcquireCookie()
	c.SetKey(key)
	c.SetValue(base64.URLEncoding.EncodeToString(value))
	c.SetPath("/")
	c.SetHTTPOnly(true)
	ctx.RequestCtx.Response.Header.SetCookie(c)
	fasthttp.ReleaseCookie(c)
}

// Sessionreturns the current session store, returns nil if provider is ""
func (ctx *Context) Session() store.IStore {
	if ctx.station.sessionManager == nil || ctx.station.config.Sessions.Provider == "" { //the second check can be changed on runtime, users are able to  turn off the sessions by setting provider to  ""
		return nil
	}

	if ctx.sessionStore == nil {
		ctx.sessionStore = ctx.station.sessionManager.Start(ctx)
	}
	return ctx.sessionStore
}

// SessionDestroy destroys the whole session, calls the provider's destory and remove the cookie
func (ctx *Context) SessionDestroy() {
	if ctx.station.sessionManager != nil {
		if store := ctx.Session(); store != nil {
			ctx.station.sessionManager.Destroy(ctx)
		}
	}

}
