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

const (
	// ParameterStartByte is very used on the node, it's just contains the byte for the ':' rune/char
	ParameterStartByte = byte(':')
	// SlashByte is just a byte of '/' rune/char
	SlashByte = byte('/')
	// MatchEverythingByte is just a byte of '*" rune/char
	MatchEverythingByte = byte('*')
)

// IRouterMethods is the interface for method routing
type IRouterMethods interface {
	Get(path string, handlersFn ...HandlerFunc) IRoute
	Post(path string, handlersFn ...HandlerFunc) IRoute
	Put(path string, handlersFn ...HandlerFunc) IRoute
	Delete(path string, handlersFn ...HandlerFunc) IRoute
	Connect(path string, handlersFn ...HandlerFunc) IRoute
	Head(path string, handlersFn ...HandlerFunc) IRoute
	Options(path string, handlersFn ...HandlerFunc) IRoute
	Patch(path string, handlersFn ...HandlerFunc) IRoute
	Trace(path string, handlersFn ...HandlerFunc) IRoute
	Any(path string, handlersFn ...HandlerFunc) IRoute
}

// IRouter is the interface of which any Iris router must implement
type IRouter interface {
	IMiddlewareSupporter
	IRouterMethods
	IPartyHoster
	HandleAnnotated(Handler) (IRoute, error)
	Handle(string, string, ...Handler) IRoute
	HandleFunc(string, string, ...HandlerFunc) IRoute
	Errors() IHTTPErrors //at the main Router struct this is managed by the MiddlewareSupporter
	// ServeHTTP finds and serves a route by it's request
	// If no route found, it sends an http status 404
	ServeHTTP(http.ResponseWriter, *http.Request)
}
