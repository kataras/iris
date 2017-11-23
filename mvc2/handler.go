package mvc2

import (
	"fmt"
	"reflect"

	"github.com/kataras/iris/context"
)

// checks if "handler" is context.Handler; func(context.Context).
func isContextHandler(handler interface{}) bool {
	_, is := handler.(context.Handler)
	return is
}

func validateHandler(handler interface{}) error {
	if typ := reflect.TypeOf(handler); !isFunc(typ) {
		return fmt.Errorf("handler expected to be a kind of func but got typeof(%s)", typ.String())
	}
	return nil
}
