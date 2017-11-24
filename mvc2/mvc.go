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

func (m *Mvc) Child() *Mvc {
	child := New()

	// copy the current parent's ctx func binders and services to this new child.
	if len(m.binders) > 0 {
		binders := make([]*InputBinder, len(m.binders), len(m.binders))
		for i, v := range m.binders {
			binders[i] = v
		}
		child.binders = binders
	}

	return child
}

func (m *Mvc) In(binders ...interface{}) {
	for _, binder := range binders {
		typ := resolveBinderType(binder)

		var (
			b   *InputBinder
			err error
		)

		if typ == functionType {
			b, err = MakeFuncInputBinder(binder)
		} else if typ == serviceType {
			b, err = MakeServiceInputBinder(binder)
		} else {
			err = errBad
		}

		if err != nil {
			continue
		}

		m.binders = append(m.binders, b)
	}
}

func (m *Mvc) Handler(handler interface{}) context.Handler {
	h, _ := MakeHandler(handler, m.binders) // it logs errors already, so on any error the "h" will be nil.
	return h
}
