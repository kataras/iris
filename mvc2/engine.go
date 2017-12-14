package mvc2

import (
	"errors"

	"github.com/kataras/di"
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
	dependencies *di.D
}

func New() *Engine {
	return &Engine{
		dependencies: di.New().Hijack(hijacker).GoodFunc(typeChecker),
	}
}

func (e *Engine) Bind(values ...interface{}) *Engine {
	e.dependencies.Bind(values...)
	return e
}

func (e *Engine) Child() *Engine {
	child := New()
	child.dependencies = e.dependencies.Clone()
	return child
}

func (e *Engine) Handler(handler interface{}) context.Handler {
	h, err := MakeHandler(handler, e.dependencies.Values...)
	if err != nil {
		golog.Errorf("mvc handler: %v", err)
	}
	return h
}

func (e *Engine) Controller(router router.Party, controller interface{}, onActivate ...func(*ControllerActivator)) {
	ca := newControllerActivator(router, controller, e.dependencies)

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
