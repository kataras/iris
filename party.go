// Copyright (c) 2016, Gerasimos Maropoulos
// All rights reserved.
//
// Redistribution and use in source and binary forms, with or without modification,
// are permitted provided that the following conditions are met:
//
// 1. Redistributions of source code must retain the above copyright notice,
//    this list of conditions and the following disclaimep.
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
// DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER AND CONTRIBUTOR, GERASIMOS MAROPOULOS BE LIABLE FOR ANY
// DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
// (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
// LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
// ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
// SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

package iris

import (
	"path"
	"reflect"
	"strconv"
	"strings"

	"github.com/valyala/fasthttp"
)

type (
	// IParty is the interface which implements the whole Party of routes
	IParty interface {
		Handle(string, string, ...Handler)
		HandleFunc(string, string, ...HandlerFunc)
		HandleAnnotated(Handler) error
		Get(string, ...HandlerFunc)
		Post(string, ...HandlerFunc)
		Put(string, ...HandlerFunc)
		Delete(string, ...HandlerFunc)
		Connect(string, ...HandlerFunc)
		Head(string, ...HandlerFunc)
		Options(string, ...HandlerFunc)
		Patch(string, ...HandlerFunc)
		Trace(string, ...HandlerFunc)
		Any(string, ...HandlerFunc)
		Use(...Handler)
		UseFunc(...HandlerFunc)
		// Static serves a directory
		// accepts three parameters
		// first parameter is the request url path (string)
		// second parameter is the system directory (string)
		// third parameter is the level (int) of stripSlashes
		// * stripSlashes = 0, original path: "/foo/bar", result: "/foo/bar"
		// * stripSlashes = 1, original path: "/foo/bar", result: "/bar"
		// * stripSlashes = 2, original path: "/foo/bar", result: ""
		Static(string, string, int)
		Party(string, ...HandlerFunc) IParty // Each party can have a party too
		IsRoot() bool
	}

	GardenParty struct {
		relativePath string
		station      *Iris // this station is where the party is happening, this station's Garden is the same for all Parties per Station & Router instance
		middleware   Middleware
		root         bool
	}
)

var _ IParty = &GardenParty{}

// IsRoot returns true if this is the root party ("/")
func (p *GardenParty) IsRoot() bool {
	return p.root
}

// Handle registers a route to the server's router
func (p *GardenParty) Handle(method string, registedPath string, handlers ...Handler) {
	path := fixPath(absPath(p.relativePath, registedPath))
	middleware := JoinMiddleware(p.middleware, handlers)
	route := NewRoute(method, path, middleware)
	p.station.Plugins.DoPreHandle(route)
	p.station.addRoute(route)
	p.station.Plugins.DoPostHandle(route)
}

// HandleFunc registers and returns a route with a method string, path string and a handler
// registedPath is the relative url path
// handler is the iris.Handler which you can pass anything you want via iris.ToHandlerFunc(func(res,req){})... or just use func(c *iris.Context)
func (p *GardenParty) HandleFunc(method string, registedPath string, handlersFn ...HandlerFunc) {
	p.Handle(method, registedPath, ConvertToHandlers(handlersFn)...)
}

// HandleAnnotated registers a route handler using a Struct implements iris.Handler (as anonymous property)
// which it's metadata has the form of
// `method:"path"` and returns the route and an error if any occurs
// handler is passed by func(urstruct MyStruct) Serve(ctx *Context) {}
func (p *GardenParty) HandleAnnotated(irisHandler Handler) error {
	var method string
	var path string
	var errMessage = ""
	val := reflect.ValueOf(irisHandler).Elem()

	for i := 0; i < val.NumField(); i++ {
		typeField := val.Type().Field(i)

		if typeField.Anonymous && typeField.Name == "Handler" {
			tags := strings.Split(strings.TrimSpace(string(typeField.Tag)), " ")
			firstTag := tags[0]

			idx := strings.Index(string(firstTag), ":")

			tagName := strings.ToUpper(string(firstTag[:idx]))
			tagValue, unqerr := strconv.Unquote(string(firstTag[idx+1:]))

			if unqerr != nil {
				errMessage = errMessage + "\non getting path: " + unqerr.Error()
				continue
			}

			path = tagValue
			avalaibleMethodsStr := strings.Join(AllMethods[0:], ",")

			if !strings.Contains(avalaibleMethodsStr, tagName) {
				//wrong method passed
				errMessage = errMessage + "\nWrong method passed to the anonymous property iris.Handler -> " + tagName
				continue
			}

			method = tagName

		} else {
			errMessage = "\nStruct passed but it doesn't have an anonymous property of type iris.Hanndler, please refer to docs\n"
		}

	}

	if errMessage == "" {
		p.Handle(method, path, irisHandler)
	}

	var err error
	if errMessage != "" {
		err = ErrHandleAnnotated.Format(errMessage)
	}

	return err
}

// Get registers a route for the Get http method
func (p *GardenParty) Get(path string, handlersFn ...HandlerFunc) {
	p.HandleFunc(MethodGet, path, handlersFn...)
}

// Post registers a route for the Post http method
func (p *GardenParty) Post(path string, handlersFn ...HandlerFunc) {
	p.HandleFunc(MethodPost, path, handlersFn...)
}

// Put registers a route for the Put http method
func (p *GardenParty) Put(path string, handlersFn ...HandlerFunc) {
	p.HandleFunc(MethodPut, path, handlersFn...)
}

// Delete registers a route for the Delete http method
func (p *GardenParty) Delete(path string, handlersFn ...HandlerFunc) {
	p.HandleFunc(MethodDelete, path, handlersFn...)
}

// Connect registers a route for the Connect http method
func (p *GardenParty) Connect(path string, handlersFn ...HandlerFunc) {
	p.HandleFunc(MethodConnect, path, handlersFn...)
}

// Head registers a route for the Head http method
func (p *GardenParty) Head(path string, handlersFn ...HandlerFunc) {
	p.HandleFunc(MethodHead, path, handlersFn...)
}

// Options registers a route for the Options http method
func (p *GardenParty) Options(path string, handlersFn ...HandlerFunc) {
	p.HandleFunc(MethodOptions, path, handlersFn...)
}

// Patch registers a route for the Patch http method
func (p *GardenParty) Patch(path string, handlersFn ...HandlerFunc) {
	p.HandleFunc(MethodPatch, path, handlersFn...)
}

// Trace registers a route for the Trace http method
func (p *GardenParty) Trace(path string, handlersFn ...HandlerFunc) {
	p.HandleFunc(MethodTrace, path, handlersFn...)
}

// Any registers a route for ALL of the http methods (Get,Post,Put,Head,Patch,Options,Connect,Delete)
func (p *GardenParty) Any(path string, handlersFn ...HandlerFunc) {
	for _, k := range AllMethods {
		p.HandleFunc(k, path, handlersFn...)
	}

}

// Use registers a Handler middleware
func (p *GardenParty) Use(handlers ...Handler) {
	p.middleware = append(p.middleware, handlers...)
}

// UseFunc registers a HandlerFunc middleware
func (p *GardenParty) UseFunc(handlersFn ...HandlerFunc) {
	p.Use(ConvertToHandlers(handlersFn)...)
}

// Static registers a route which serves a system directory
func (p *GardenParty) Static(relative string, systemPath string, stripSlashes int) {
	h := fasthttp.FSHandler(systemPath, stripSlashes)
	p.Get(relative+"/*filepath", func(c *Context) {
		h(c.RequestCtx)
	})
}

// Party is just a group joiner of routes which have the same prefix and share same middleware(s) also.
// Party can also be named as 'Join' or 'Node' or 'Group' , Party chosen because it has more fun
func (p *GardenParty) Party(path string, handlersFn ...HandlerFunc) IParty {
	middleware := ConvertToHandlers(handlersFn)
	if path[0] != SlashByte && strings.Contains(path, ".") {
		//it's domain so no handlers share (even the global ) or path, nothing.
	} else {
		// set path to parent+child
		path = absPath(p.relativePath, path)
		// append the parent's +child's handlers
		middleware = JoinMiddleware(p.middleware, middleware)
	}

	return &GardenParty{relativePath: path, station: p.station, middleware: middleware}
}

func absPath(rootPath string, relativePath string) (absPath string) {

	if relativePath == "" {
		absPath = rootPath
	} else {
		absPath = path.Join(rootPath, relativePath)
	}

	return
}

// fixPath fix the double slashes, (because of root,I just do that before the .Handle no need for anything else special)
func fixPath(str string) string {

	strafter := strings.Replace(str, "//", Slash, -1)

	if strafter[0] == SlashByte && strings.Count(strafter, ".") >= 2 {
		//it's domain, remove the first slash
		strafter = strafter[1:]
	}

	return strafter
}
