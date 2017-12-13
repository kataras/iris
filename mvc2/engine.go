package mvc2

import (
	"errors"
	"reflect"

	"github.com/kataras/golog"
	"github.com/kataras/iris/context"
	"github.com/kataras/iris/core/router"
)

var (
	errNil           = errors.New("nil")
	errBad           = errors.New("bad")
	errAlreadyExists = errors.New("already exists")
)

type Engine struct {
	Input []reflect.Value
}

func New() *Engine {
	return new(Engine)
}

func (e *Engine) Bind(values ...interface{}) *Engine {
	for _, val := range values {
		if v := reflect.ValueOf(val); goodVal(v) {
			e.Input = append(e.Input, v)
		}
	}

	return e
}

func (e *Engine) Child() *Engine {
	child := New()

	// copy the current parent's ctx func binders and services to this new child.
	if n := len(e.Input); n > 0 {
		input := make([]reflect.Value, n, n)
		copy(input, e.Input)
		child.Input = input
	}

	return child
}

func (e *Engine) Handler(handler interface{}) context.Handler {
	h, err := MakeHandler(handler, e.Input...)
	if err != nil {
		golog.Errorf("mvc handler: %v", err)
	}
	return h
}

func (e *Engine) Controller(router router.Party, controller interface{}, onActivate ...func(*ControllerActivator)) {
	ca := newControllerActivator(router, controller, e.Input...)

	// give a priority to the "onActivate"
	// callbacks, if any.
	for _, cb := range onActivate {
		cb(ca)
	}

	// check if controller has an "OnActivate" function
	// which accepts the controller activator and call it.
	if activateListener, ok := controller.(interface {
		OnActivate(*ControllerActivator)
	}); ok {
		activateListener.OnActivate(ca)
	}

	ca.activate()
}
