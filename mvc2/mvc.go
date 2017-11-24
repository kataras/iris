package mvc2

import (
	"errors"

	"github.com/kataras/iris/context"
)

var (
	errNil           = errors.New("nil")
	errBad           = errors.New("bad")
	errAlreadyExists = errors.New("already exists")
)

type Mvc struct {
	binders []*InputBinder
}

func New() *Mvc {
	return new(Mvc)
}

func (m *Mvc) RegisterBinder(binders ...interface{}) error {
	for _, binder := range binders {
		b, err := MakeFuncInputBinder(binder)
		if err != nil {
			return err
		}
		m.binders = append(m.binders, b)
	}

	return nil
}

func (m *Mvc) RegisterService(services ...interface{}) error {
	for _, service := range services {
		b, err := MakeServiceInputBinder(service)
		if err != nil {
			return err
		}
		m.binders = append(m.binders, b)
	}

	return nil
}

func (m *Mvc) Handler(handler interface{}) context.Handler {
	h, _ := MakeHandler(handler, m.binders) // it logs errors already, so on any error the "h" will be nil.
	return h
}
