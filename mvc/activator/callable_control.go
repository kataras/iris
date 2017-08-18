package activator

import (
	"reflect"

	"github.com/kataras/iris/context"
)

func getCustomFuncIndex(t *TController, funcNames ...string) (funcIndex int, has bool) {
	val := t.Value

	for _, funcName := range funcNames {
		if m, has := t.Type.MethodByName(funcName); has {
			if _, isRequestFunc := val.Method(m.Index).Interface().(func(ctx context.Context)); isRequestFunc {
				return m.Index, has
			}
		}
	}

	return -1, false
}

type callableControl struct {
	Functions []string
	index     int
}

func (cc *callableControl) Load(t *TController) error {
	funcIndex, has := getCustomFuncIndex(t, cc.Functions...)
	if !has {
		return ErrControlSkip
	}

	cc.index = funcIndex
	return nil
}

// the "c" is a new "c" instance
// which is being used at serve time, inside the Handler.
// it calls the custom function (can be "Init", "BeginRequest", "End" and "EndRequest"),
// the check of this function made at build time, so it's a safe a call.
func (cc *callableControl) Handle(ctx context.Context, c reflect.Value, methodFunc func()) {
	c.Method(cc.index).Interface().(func(ctx context.Context))(ctx)
}

// CallableControl is a generic-propose `TControl`
// which finds one function in the user's controller's struct
// based on the possible "funcName(s)" and executes
// that inside the handler, at serve-time, by passing
// the current request's `iris/context/#Context`.
func CallableControl(funcName ...string) TControl {
	return &callableControl{Functions: funcName}
}
