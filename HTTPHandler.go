package gapi

import (
	"reflect"
)

//type HTTPHandler func(http.ResponseWriter, *http.Request)
type HTTPHandler interface{} //func(...interface{}) //[]reflect.Value

//4 possibilities
//1. http.ResponseWriter, *http.Request
//2. *gapi.Context
//3. *gapi.Renderer
//4. *gapi.Context, *gapi.Renderer

//check the first parameter only for Context
//check if the handler needs a Context , has the first parameter as type of *Context
//use in NewHTTPRoute inside HTTPRoute.go
func hasContextParam(handlerType reflect.Type) bool {
	//if the handler doesn't take arguments, false
	if handlerType.NumIn() == 0 {
		return false
	}

	//if the first argument is not a pointer, false
	p1 := handlerType.In(0)
	if p1.Kind() != reflect.Ptr {
		return false
	}
	//but if the first argument is a context, true
	if p1.Elem() == contextType {
		return true
	}

	return false
}

//check the first parameter only for Renderer
func hasRendererParam(handlerType reflect.Type) bool {
	//if the handler doesn't take arguments, false
	if handlerType.NumIn() == 0 {
		return false
	}

	//if the first argument is not a pointer, false
	p1 := handlerType.In(0)
	if p1.Kind() != reflect.Ptr {
		return false
	}
	//but if the first argument is a renderer, true
	if p1.Elem() == rendererType {
		return true
	}

	return false
}

func hasContextAndRenderer(handlerType reflect.Type) bool {

	//first check if we have pass 2 arguments
	if handlerType.NumIn() < 2 {
		return false
	}

	
	firstParamIsContext := hasContextParam(handlerType)
	
	//the first argument/parameter is always context if exists otherwise it's only Renderer or ResponseWriter,Request.
	if firstParamIsContext == false {
		return false
	}
	
	p2 := handlerType.In(1)
	if p2.Kind() != reflect.Ptr {
		return false
	}
	//but if the first argument is a context, true
	if p2.Elem() == rendererType {
		return true
	}
	
	return false
}
