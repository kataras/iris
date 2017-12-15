package mvc2

import (
	"github.com/kataras/iris/mvc2/di"
	"github.com/kataras/golog"

	"github.com/kataras/iris/context"
	"github.com/kataras/iris/core/router"
)

type Engine struct {
	Dependencies *di.D
}

func NewEngine() *Engine {
	return &Engine{
		Dependencies: di.New().Hijack(hijacker).GoodFunc(typeChecker),
	}
}

func (e *Engine) Clone() *Engine {
	child := NewEngine()
	child.Dependencies = e.Dependencies.Clone()
	return child
}

func (e *Engine) Handler(handler interface{}) context.Handler {
	h, err := MakeHandler(handler, e.Dependencies.Values...)
	if err != nil {
		golog.Errorf("mvc handler: %v", err)
	}
	return h
}

func (e *Engine) Controller(router router.Party, controller interface{}, beforeActivate ...func(*ControllerActivator)) {
	ca := newControllerActivator(router, controller, e.Dependencies)

	// give a priority to the "beforeActivate"
	// callbacks, if any.
	for _, cb := range beforeActivate {
		cb(ca)
	}

	// check if controller has an "BeforeActivate" function
	// which accepts the controller activator and call it.
	if activateListener, ok := controller.(interface {
		BeforeActivate(*ControllerActivator)
	}); ok {
		activateListener.BeforeActivate(ca)
	}

	ca.activate()
}
