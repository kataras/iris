package router

import (
	"net/http"
)

//The router is handled on BOTTOM of the default mux.

//type Handler func(http.Handler) http.Handler
type Handler func(http.ResponseWriter,*http.Request)

type HttpRoute struct {
	Method   string
	Handler Handler
}

type HttpRouter struct {
	//Routes map[string]http.Handler
	Routes map[string]*HttpRoute
}

func NewHttpRouter() *HttpRouter {

	return &HttpRouter{Routes: make(map[string]*HttpRoute)}
}

func (this *HttpRouter) Unroute(urlPath string) *HttpRouter {
	delete(this.Routes, urlPath)
	return this
}

func (this *HttpRouter) Route(urlPath string, handler Handler) *HttpRouter {
	if urlPath == "" {
		urlPath = "/"
	}
	this.Routes[urlPath] = &HttpRoute{Method: HttpMethods.GET, Handler: handler}
	return this
}

func (this *HttpRouter) Middleware() http.Handler {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		_route := this.Routes[req.URL.Path]
		if _route == nil {
			NotFoundRoute().ServeHTTP(res, req)
			return
		}
		//http.ServeFile(res,req,"edw vazw to directory as poume kia to kanei serve mazi me ta checks gia conten types ktlp prostoparwn omws edw den 9elw auto")

		//	_route.ServeHTTP(res, req)
		//	_route.Handler.ServeHTTP(res, req)
	})
}
