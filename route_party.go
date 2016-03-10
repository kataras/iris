package iris

import (
	"net/http"
	"strings"
)

// maybe at the future this will be by-default to all routes, and no need to handle it at different struct
// but this is unnecessary because both nodesprefix and tree are auto-sorted on the tree struct.
// so we will have the main router which is the router.go and all logic implementation is there,
// we have the router_memory which is just a IRouter and has underline router the Router also
// and the route_party which exists on both router and router_memory ofcourse

// RouteParty is used inside Router.Party method
type RouteParty struct {
	IRouteRegister
	MiddlewareSupporter
	_router   *Router
	_rootPath string
}

func newRouteParty(rootPath string, underlineMainRouter *Router) RouteParty {
	p := RouteParty{}
	p._router = underlineMainRouter

	//we don't want the root path ends with /
	lastSlashIndex := strings.LastIndexByte(rootPath, SlashByte)

	if lastSlashIndex == len(rootPath)-1 {
		rootPath = rootPath[0:lastSlashIndex]
	}

	p._rootPath = rootPath
	return p
}

func (p RouteParty) Get(path string, handler interface{}) *Route {
	return p._router.Get(p._rootPath+path, handler)
}
func (p RouteParty) Post(path string, handler interface{}) *Route {
	return p._router.Post(p._rootPath+path, handler)
}
func (p RouteParty) Put(path string, handler interface{}) *Route {
	return p._router.Put(p._rootPath+path, handler)
}
func (p RouteParty) Delete(path string, handler interface{}) *Route {
	return p._router.Delete(p._rootPath+path, handler)
}
func (p RouteParty) Connect(path string, handler interface{}) *Route {
	return p._router.Connect(p._rootPath+path, handler)
}
func (p RouteParty) Head(path string, handler interface{}) *Route {
	return p._router.Head(p._rootPath+path, handler)
}
func (p RouteParty) Options(path string, handler interface{}) *Route {
	return p._router.Options(p._rootPath+path, handler)
}
func (p RouteParty) Patch(path string, handler interface{}) *Route {
	return p._router.Patch(p._rootPath+path, handler)
}
func (p RouteParty) Trace(path string, handler interface{}) *Route {
	return p._router.Trace(p._rootPath+path, handler)
}
func (p RouteParty) Any(path string, handler interface{}) *Route {
	return p._router.Any(p._rootPath+path, handler)
}
func (p RouteParty) HandleAnnotated(irisHandler Annotated) (*Route, error) {
	return p._router.HandleAnnotated(irisHandler)
}
func (p RouteParty) Handle(params ...interface{}) *Route {
	return p._router.Handle(params)
}
func (p RouteParty) HandleFunc(path string, handler Handler, method string) *Route {
	return p._router.HandleFunc(p._rootPath+path, handler, method)
}

func (p RouteParty) Party(path string) IRouteRegister {
	return p._router.Party(p._rootPath + path)
}

// Use registers middleware for all routes which inside this party, which the node's prefix starts with the rootPath +"/" because all prefix ends with slash
func (p RouteParty) Use(handler MiddlewareHandler) {
	r := p._router
	for _, _nodes := range r.nodes {
		for _, v := range _nodes {
			if v.prefix == p._rootPath+"/" {
				for _, route := range v.routes {
					route.Use(handler)
				}
			}

		}

	}
}

func (p RouteParty) UseFunc(handlerFunc func(res http.ResponseWriter, req *http.Request, next http.HandlerFunc)) {
	p.Use(MiddlewareHandlerFunc(handlerFunc))
}

func (p RouteParty) UseHandler(handler http.Handler) {
	convertedMiddleware := MiddlewareHandlerFunc(func(res http.ResponseWriter, req *http.Request, next http.HandlerFunc) {
		handler.ServeHTTP(res, req)
		//run the next automatically after this handler finished
		next(res, req)

	})

	p.Use(convertedMiddleware)
}
