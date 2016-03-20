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
package domain

import (
	"net/http"
)

type IContext interface {
	New()
	Do()
	Redo(req *http.Request, res http.ResponseWriter)
	Next()
	GetResponseWriter() IMemoryWriter
	GetRequest() *http.Request
	Param(key string) string
	ParamInt(key string) (int, error)
	URLParam(key string) string
	URLParamInt(key string) (int, error)
	Get(key string) interface{}
	GetString(key string) (value string)
	GetInt(key string) (value int)
	Set(key string, value interface{})
	Write(format string, a ...interface{})
	ServeFile(path string)
	GetCookie(name string) string
	SetCookie(name string, value string)
	NotFound()
	SendStatus(statusCode int, message string)
	Panic()
	RequestIP() string
	Close()
	End()
	IsStopped() bool
	Clone() IContext
	RenderFile(file string, pageContext interface{}) error
	Render(pageContext interface{}) error
	WriteHTML(httpStatus int, htmlContents string)
	HTML(htmlContents string)
	WriteData(httpStatus int, binaryData []byte)
	Data(binaryData []byte)
	WriteText(httpStatus int, text string)
	Text(text string)
	RenderJSON(httpStatus int, jsonStructs ...interface{}) error
	WriteJSON(httpStatus int, jsonObjectOrArray interface{}) error
	JSON(jsonObjectOrArray interface{}) error
	WriteXML(httpStatus int, xmlStructs ...interface{}) error
	XML(xmlStructs ...interface{}) error
}
