package router

import (

)

type HttpRouterBuilder struct {
	//This can be nil and it is when gapi.New() or gapi.NewRouter() .
	router *HttpRouter
}

func NewHttpRouterBuilder(startRouter *HttpRouter) *HttpRouterBuilder {
	if startRouter == nil {
		startRouter = NewHttpRouter()
	}
	builder := &HttpRouterBuilder{router: startRouter}
	return builder
}

type RouterThenBuilder struct {
	builder *HttpRouterBuilder
	paths   []string
}

//IF

//paths one or more paths that the router should be listening to.
func (this *HttpRouterBuilder) If(paths ...string) *RouterThenBuilder {
	return &RouterThenBuilder{builder: this, paths: paths}
}

//THEN

func (this RouterThenBuilder) Then(handler Handler) *HttpRouterBuilder {
	for _, path := range this.paths {
		///TODO: /home to /home,/home/ but if not already exists, cuz dev can do it manually and then we will have multiple unnessecary handles
		//as t oaknw edw oxi sto HttpRouter.Route.
		this.builder.router.Route(path, handler)
	}
	return this.builder
}

//BUILD

func (this *HttpRouterBuilder) Build() *HttpRouter {
	//maybe copy this.router, delete it and after return its copy?
	return this.router
}
