package iris

import (
	"strings"
)

// RouteParty is used inside Router.Party method
// maybe at the future this will be by-default to all routes, and no need to handle it at different struct
type RouteParty struct {
	IRouteRegister
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
