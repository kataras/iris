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
// DISCLAIMED. IN NO EVENT SHALL JULIEN SCHMIDT BE LIABLE FOR ANY
// DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
// (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
// LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
// ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
// SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
package iris

// The party holds memory to find the Root, I could make it with other design pattern but I choose this
// because I want to the future to be able to remove a Party and routes at the runtime
// this will be useful when I introduce the dynamic creation of subdomains parties ( the only one framework which will have this feature, as far as I know)
// this dynamic subdomains can created at the runtime and removed at runtime
// this is practial an example of create a user with a subdomain and when user deletes his account or his repo
// then delete the subdomain also without if else inside their handlers

import (
	"errors"
	"reflect"
	"strconv"
	"strings"
)

// IParty is the interface which implements the whole Party of routes
type IParty interface {
	IMiddlewareSupporter
	Handle(method string, registedPath string, handlers ...Handler)
	HandleFunc(method string, registedPath string, handlersFn ...HandlerFunc)
	HandleAnnotated(irisHandler Handler) error
	Get(path string, handlersFn ...HandlerFunc)
	Post(path string, handlersFn ...HandlerFunc)
	Put(path string, handlersFn ...HandlerFunc)
	Delete(path string, handlersFn ...HandlerFunc)
	Connect(path string, handlersFn ...HandlerFunc)
	Head(path string, handlersFn ...HandlerFunc)
	Options(path string, handlersFn ...HandlerFunc)
	Patch(path string, handlersFn ...HandlerFunc)
	Trace(path string, handlersFn ...HandlerFunc)
	Any(path string, handlersFn ...HandlerFunc)
	Party(path string) IParty // Each party can have a party too
	getRoot() IParty
	getPath() string
	isTheRoot() bool
	getMiddleware() Middleware
}

// GardenParty TODO: inline docs
type GardenParty struct {
	MiddlewareSupporter
	station  *Station // this station is where the party is happening, this station's Garden is the same for all Parties per Station & Router instance
	rootPath string
	hoster   *GardenParty
}

var _ IParty = &GardenParty{}

func NewParty(path string, station *Station, hoster *GardenParty) IParty {
	p := &GardenParty{}
	p.station = station

	//if this party is comes from other party
	if hoster != nil {
		p.hoster = hoster
		path = p.hoster.rootPath + path
		p.Middleware = p.hoster.Middleware
		lastSlashIndex := strings.LastIndexByte(path, SlashByte)

		if lastSlashIndex == len(path)-1 {
			path = path[0:lastSlashIndex]
		}
	}

	p.rootPath = path
	return p
}

// fixPath fix the double slashes, (because of root,I just do that before the .Handle no need for anything else special)
func fixPath(str string) string {
	return strings.Replace(str, "//", "/", -1)
}

// GetRoot find the root hoster of the parties, the root is this when the hoster is nil ( it's the  rootPath '/')
func (p *GardenParty) getRoot() IParty {
	if p.hoster != nil {
		return p.hoster.getRoot()
	} else {
		return p
	}

}

func (p *GardenParty) isTheRoot() bool {
	return p.hoster == nil
}

func (p *GardenParty) getMiddleware() Middleware {
	return p.Middleware
}

func (p GardenParty) getPath() string {
	return p.rootPath
}

// Handle registers a route to the server's router
func (p *GardenParty) Handle(method string, registedPath string, handlers ...Handler) {
	registedPath = p.rootPath + registedPath
	if registedPath == "" {
		registedPath = "/"
	}
	registedPath = fixPath(registedPath)

	if len(handlers) == 0 {
		panic("Iris.Handle: zero handler to " + method + ":" + registedPath)
	}

	rootParty := p.getRoot()

	tempHandlers := p.Middleware

	//println(registedPath, " party middleware len : ", len(tempHandlers))

	// from top to bottom -->||<--
	//check for root-global middleware WHEN THIS PARTY IS NOT THE ROOT, because if it's the Middleware already setted on the constructor NewParty)
	if rootParty.isTheRoot() == false && p.isTheRoot() == false && len(rootParty.getMiddleware()) > 0 {
		//println(registedPath, " is not the root and it's rootParty which is: ", rootParty.getPath(), "has ", len(rootParty.getMiddleware()), " handlers")
		//if global middlewares are registed then push them to this route.
		tempHandlers = append(rootParty.getMiddleware(), tempHandlers...)
	}
	//the party's middleware were setted on NewParty already, no need to check them.

	if len(tempHandlers) > 0 {
		handlers = append(tempHandlers, handlers...)
	}

	//println(" so the len of registed ", registedPath, " of handlers is: ", len(handlers))
	route := NewRoute(registedPath, handlers)

	p.station.GetPluginContainer().DoPreHandle(method, route)

	p.station.IRouter.setGarden(p.station.getGarden().Plant(method, route))

	p.station.GetPluginContainer().DoPostHandle(method, route)

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
				errMessage = errMessage + "\niris.HandleAnnotated: Error on getting path: " + unqerr.Error()
				continue
			}

			path = tagValue
			avalaibleMethodsStr := strings.Join(HTTPMethods.ANY, ",")

			if !strings.Contains(avalaibleMethodsStr, tagName) {
				//wrong method passed
				errMessage = errMessage + "\niris.HandleAnnotated: Wrong method passed to the anonymous property iris.Handler -> " + tagName
				continue
			}

			method = tagName

		} else {
			errMessage = "\nError on Iris.HandleAnnotated: Struct passed but it doesn't have an anonymous property of type iris.Hanndler, please refer to docs\n"
		}

	}

	if errMessage == "" {
		p.Handle(method, path, irisHandler)
	}

	var err error
	if errMessage != "" {
		err = errors.New(errMessage)
	}

	return err
}

///////////////////
//global middleware
///////////////////

// Party is just a group joiner of routes which have the same prefix and share same middleware(s) also.
// Party can also be named as 'Join' or 'Node' or 'Group' , Party chosen because it has more fun
func (p *GardenParty) Party(path string) IParty {
	return NewParty(path, p.station, p)
}

///////////////////////////////
//expose some methods as public
///////////////////////////////

// Get registers a route for the Get http method
func (p *GardenParty) Get(path string, handlersFn ...HandlerFunc) {
	p.HandleFunc(HTTPMethods.GET, path, handlersFn...)
}

// Post registers a route for the Post http method
func (p *GardenParty) Post(path string, handlersFn ...HandlerFunc) {
	p.HandleFunc(HTTPMethods.POST, path, handlersFn...)
}

// Put registers a route for the Put http method
func (p *GardenParty) Put(path string, handlersFn ...HandlerFunc) {
	p.HandleFunc(HTTPMethods.PUT, path, handlersFn...)
}

// Delete registers a route for the Delete http method
func (p *GardenParty) Delete(path string, handlersFn ...HandlerFunc) {
	p.HandleFunc(HTTPMethods.DELETE, path, handlersFn...)
}

// Connect registers a route for the Connect http method
func (p *GardenParty) Connect(path string, handlersFn ...HandlerFunc) {
	p.HandleFunc(HTTPMethods.CONNECT, path, handlersFn...)
}

// Head registers a route for the Head http method
func (p *GardenParty) Head(path string, handlersFn ...HandlerFunc) {
	p.HandleFunc(HTTPMethods.HEAD, path, handlersFn...)
}

// Options registers a route for the Options http method
func (p *GardenParty) Options(path string, handlersFn ...HandlerFunc) {
	p.HandleFunc(HTTPMethods.OPTIONS, path, handlersFn...)
}

// Patch registers a route for the Patch http method
func (p *GardenParty) Patch(path string, handlersFn ...HandlerFunc) {
	p.HandleFunc(HTTPMethods.PATCH, path, handlersFn...)
}

// Trace registers a route for the Trace http method
func (p *GardenParty) Trace(path string, handlersFn ...HandlerFunc) {
	p.HandleFunc(HTTPMethods.TRACE, path, handlersFn...)
}

// Any registers a route for ALL of the http methods (Get,Post,Put,Head,Patch,Options,Connect,Delete)
func (p *GardenParty) Any(path string, handlersFn ...HandlerFunc) {
	p.HandleFunc("", path, handlersFn...)
}
