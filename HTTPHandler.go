package gapi

import (
	"reflect"
)


//type HTTPHandler func(http.ResponseWriter, *http.Request)
type HTTPHandler interface{} //func(...interface{}) //[]reflect.Value

//check if the handler needs a Context , has the first parameter as type of *Context
//use in NewHTTPRoute inside HTTPRoute.go
func hasContextParam(handlerType reflect.Type) bool {
    //if the handler doesn't take arguments, false
    if handlerType.NumIn() == 0 {
        return false
    }

    //if the first argument is not a pointer, false
    a0 := handlerType.In(0)
    if a0.Kind() != reflect.Ptr {
        return false
    }
    //but if the first argument is a context, true
    if a0.Elem() == contextType {
        return true
    }

    return false
}