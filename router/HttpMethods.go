package router

import (
	"reflect"
)

//type HttpMethod string

type HttpMethodType struct {
	GET    string
	POST   string
	PUT    string
	DELETE string
}

func (c *HttpMethodType) GetName(i int) string {
	return HttpMethodReflectType.Field(i).Name
}

var HttpMethods = HttpMethodType{"GET", "POST", "PUT", "DELETE"}
var HttpMethodReflectType = reflect.TypeOf(HttpMethods)
