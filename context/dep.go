package context

import (
	"reflect"

	"github.com/kataras/iris/v14/core/dep"
)

func init() {
	contextType := reflect.TypeFor[*Context]() // .Elem()
	dep.RegisterRequestDependencyType(contextType)
}
