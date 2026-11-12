package router

import (
	"reflect"

	"github.com/kataras/iris/v14/core/dep"
)

func init() {
	var requestDepTypes = []reflect.Type{
		reflect.TypeFor[Routing](), //.Elem()
		reflect.TypeOf(func(Router) {}),
		reflect.TypeOf(Controller(nil)),
		reflect.TypeOf(Router(nil)),
	}

	for _, t := range requestDepTypes {
		dep.RegisterRequestDependencyType(t)
	}
}
