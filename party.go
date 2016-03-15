package iris

import (
	"strings"
)

/*Usage
admin := api.Party("/admin")
{
	admin.Get("/", func(c iris.Context) {
		c.Write("Hello from /admin/")
	})
	admin.Get("/hello", func(c iris.Context) {
		c.Write("Hello from /admin/hello")
	})

}

adminSettings := admin.Party("/settings")
{
	adminSettings.Get("/security", func(c iris.Context) {
		c.Write("Hello to /settings/security")
	})
}

admin.UseFunc(func(res http.ResponseWriter, req *http.Request, next http.HandlerFunc) {
	println("[/admin] This is the middleware for: ", req.URL.Path)
	next(res, req)
})

adminSettings.UseFunc(func(res http.ResponseWriter, req *http.Request, next http.HandlerFunc) {
	println("[/admin/settings] This is the middleware for: ", req.URL.Path)
	next(res, req)
})

*/
type IPartyHoster interface {
	Party(path string) IParty
}
type IParty interface {
	IMiddlewareSupporter
	IRouterMethods
	IPartyHoster
	// Each party can have a party too
}

// maybe at the future this will be by-default to all routes, and no need to handle it at different struct
// but this is unnecessary because both nodesprefix and tree are auto-sorted on the tree struct.
// so we will have the main router which is the router.go and all logic implementation is there,
// we have the router_memory which is just a IRouter and has underline router the Router also
// and the route_party which exists on both router and router_memory ofcourse

// party is used inside Router.Party method
type party struct {
	IParty
	_router   *Router
	_rootPath string
}

func newParty(rootPath string, underlineMainRouter *Router) IParty {
	p := party{}
	p._router = underlineMainRouter

	//we don't want the root path ends with /
	lastSlashIndex := strings.LastIndexByte(rootPath, SlashByte)

	if lastSlashIndex == len(rootPath)-1 {
		rootPath = rootPath[0:lastSlashIndex]
	}

	p._rootPath = rootPath
	return p
}

func (p party) Get(path string, handlerFn HandlerFunc) *Route {
	return p._router.Get(p._rootPath+path, handlerFn)
}
func (p party) Post(path string, handlerFn HandlerFunc) *Route {
	return p._router.Post(p._rootPath+path, handlerFn)
}
func (p party) Put(path string, handlerFn HandlerFunc) *Route {
	return p._router.Put(p._rootPath+path, handlerFn)
}
func (p party) Delete(path string, handlerFn HandlerFunc) *Route {
	return p._router.Delete(p._rootPath+path, handlerFn)
}
func (p party) Connect(path string, handlerFn HandlerFunc) *Route {
	return p._router.Connect(p._rootPath+path, handlerFn)
}
func (p party) Head(path string, handlerFn HandlerFunc) *Route {
	return p._router.Head(p._rootPath+path, handlerFn)
}
func (p party) Options(path string, handlerFn HandlerFunc) *Route {
	return p._router.Options(p._rootPath+path, handlerFn)
}
func (p party) Patch(path string, handlerFn HandlerFunc) *Route {
	return p._router.Patch(p._rootPath+path, handlerFn)
}
func (p party) Trace(path string, handlerFn HandlerFunc) *Route {
	return p._router.Trace(p._rootPath+path, handlerFn)
}
func (p party) Any(path string, handlerFn HandlerFunc) *Route {
	return p._router.Any(p._rootPath+path, handlerFn)
}
func (p party) HandleAnnotated(irisHandler Handler) (*Route, error) {
	return p._router.HandleAnnotated(irisHandler)
}
func (p party) Handle(method string, registedPath string, handler Handler) *Route {
	return p._router.Handle(method, registedPath, handler)
}
func (p party) HandleFunc(method string, path string, handlerFn HandlerFunc) *Route {
	return p._router.HandleFunc(method, p._rootPath+path, handlerFn)
}

func (p party) Party(path string) IParty {
	return p._router.Party(p._rootPath + path)
}

// Use registers middleware for all routes which inside this party, which the node's prefix starts with the rootPath +"/" because all prefix ends with slash
func (p party) Use(handler MiddlewareHandler) {
	for _, _tree := range p._router.tempTrees {
		for _, _route := range _tree {
			if _route.PathPrefix == p._rootPath+"/" {
				_route.Use(handler)
			}

		}

	}
}

func (p party) UseFunc(handlerFunc func(ctx *Context, next Handler)) {
	p.Use(MiddlewareHandlerFunc(handlerFunc))
}
