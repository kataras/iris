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

package iris

import (
	"encoding/base64"
	"time"

	"github.com/kataras/iris/utils"
	"github.com/valyala/fasthttp"
)

type (
	IContextStorage interface {
		Get(interface{}) interface{}
		GetString(interface{}) string
		GetInt(interface{}) int
		Set(interface{}, interface{})
		SetCookie(*fasthttp.Cookie)
		SetCookieKV(string, string)
		RemoveCookie(string)
		// Flash messages
		GetFlash(string) string
		GetFlashBytes(string) ([]byte, error)
		SetFlash(string, string)
		SetFlashBytes(string, []byte)
	}
)

// Get returns a value from a key
// if doesn't exists returns nil
func (ctx *Context) Get(key interface{}) interface{} {
	if ctx.values == nil {
		return nil
	}

	return ctx.values[key]
}

// GetFmt returns a value which has this format: func(format string, args ...interface{}) string
// if doesn't exists returns nil
func (ctx *Context) GetFmt(key interface{}) func(format string, args ...interface{}) string {
	if ctx.values == nil {
		return nil
	}

	return ctx.values[key].(func(format string, args ...interface{}) string)
}

// GetString same as Get but returns the value as string
func (ctx *Context) GetString(key interface{}) (value string) {
	if v := ctx.Get(key); v != nil {
		value = v.(string)
	}

	return
}

// GetInt same as Get but returns the value as int
func (ctx *Context) GetInt(key interface{}) (value int) {
	if v := ctx.Get(key); v != nil {
		value = v.(int)
	}

	return
}

// Set sets a value to a key in the values map
func (ctx *Context) Set(key interface{}, value interface{}) {
	if ctx.values == nil {
		ctx.values = make(map[interface{}]interface{})
	}

	ctx.values[key] = value
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
	c := &fasthttp.Cookie{}
	c.SetKey(key)
	c.SetValue(value)
	c.SetHTTPOnly(true)
	c.SetExpire(time.Now().Add(time.Duration(120) * time.Minute))
	ctx.SetCookie(c)
}

// RemoveCookie deletes a cookie by it's name/key
func (ctx *Context) RemoveCookie(name string) {
	cookie := &fasthttp.Cookie{}
	cookie.SetKey(name)
	cookie.SetValue("")
	cookie.SetPath("/")
	cookie.SetHTTPOnly(true)
	exp := time.Now().Add(-time.Duration(1) * time.Minute) //RFC says 1 second, but make sure 1 minute because we are using fasthttp
	cookie.SetExpire(exp)
	ctx.Response.Header.SetCookie(cookie)
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

// SetFlash sets a flash message, accepts 2 parameters the key(string) and the value([]byte)
func (ctx *Context) SetFlashBytes(key string, value []byte) {
	c := &fasthttp.Cookie{}
	c.SetKey(key)
	c.SetValue(base64.URLEncoding.EncodeToString(value))
	c.SetPath("/")
	c.SetHTTPOnly(true)
	ctx.RequestCtx.Response.Header.SetCookie(c)
}
