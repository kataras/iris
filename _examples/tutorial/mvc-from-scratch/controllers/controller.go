package controllers

import (
	"reflect"
	"strings"

	"github.com/kataras/iris"
	"github.com/kataras/iris/context"
	"github.com/kataras/iris/core/router"
)

type Controller struct {
	// path params.
	Params *context.RequestParams

	// view properties.
	Layout string
	Tmpl   string
	Data   map[string]interface{}

	// give access to the request context itself.
	Ctx context.Context
}

// all lowercase, so user can see only the fields
// that are necessary to him/her.
func (b *Controller) init(ctx context.Context) {
	b.Ctx = ctx
	b.Params = ctx.Params()
	b.Data = make(map[string]interface{}, 0)
}

func (b *Controller) exec() {
	if v := b.Tmpl; v != "" {
		if l := b.Layout; l != "" {
			b.Ctx.ViewLayout(l)
		}
		if d := b.Data; d != nil {
			for key, value := range d {
				b.Ctx.ViewData(key, value)
			}
		}
		b.Ctx.View(v)
	}
}

// get the field name at compile-time,
// will help us to catch any unexpected results on future versions.
var baseControllerName = reflect.TypeOf(Controller{}).Name()

func RegisterController(app *iris.Application, path string, c interface{}) {
	typ := reflect.TypeOf(c)

	if typ.Kind() != reflect.Ptr {
		typ = reflect.PtrTo(typ)
	}

	elem := typ.Elem()

	// check if "c" has the "Controller" typeof `Controller` field.
	b, has := elem.FieldByName(baseControllerName)
	if !has {
		panic("controller should have a field of Controller type")
	}

	baseControllerFieldIndex := b.Index[0]
	persistenceFields := make(map[int]reflect.Value, 0)

	if numField := elem.NumField(); numField > 1 {
		val := reflect.Indirect(reflect.ValueOf(c))

		for i := 0; i < numField; i++ {
			f := elem.Field(i)
			valF := val.Field(i)
			// catch persistence data by tags, i.e:
			// MyData string `iris:"persistence"`
			// DB     *DB	 `iris:"persistence"`
			if t, ok := f.Tag.Lookup("iris"); ok {
				if t == "persistence" {
					persistenceFields[i] = reflect.ValueOf(valF.Interface())
					continue
				}
			}

		}
	}

	// check if has Any() or All()
	// if yes, then register all http methods and
	// exit.
	m, has := typ.MethodByName("Any")
	if !has {
		m, has = typ.MethodByName("All")
	}
	if has {
		app.Any(path,
			controllerToHandler(elem, persistenceFields,
				baseControllerFieldIndex, m.Index))
		return
	}

	// else search the entire controller
	// for any compatible method function
	// and register that.
	for _, method := range router.AllMethods {
		httpMethodFuncName := strings.Title(strings.ToLower(method))

		m, has := typ.MethodByName(httpMethodFuncName)
		if !has {
			continue
		}

		httpMethodIndex := m.Index

		app.Handle(method, path,
			controllerToHandler(elem, persistenceFields,
				baseControllerFieldIndex, httpMethodIndex))
	}

}

func controllerToHandler(elem reflect.Type, persistenceFields map[int]reflect.Value,
	baseControllerFieldIndex, httpMethodIndex int) context.Handler {
	return func(ctx context.Context) {
		// create a new controller instance of that type(>ptr).
		c := reflect.New(elem)

		// get the responsible method.
		// Remember:
		// To improve the performance
		// we don't compare the ctx.Method()[HTTP Method]
		// to the instance's Method, each handler is registered
		// to a specific http method.
		methodFunc := c.Method(httpMethodIndex)

		// get the Controller embedded field.
		b, _ := c.Elem().Field(baseControllerFieldIndex).Addr().Interface().(*Controller)

		if len(persistenceFields) > 0 {
			elem := c.Elem()
			for index, value := range persistenceFields {
				elem.Field(index).Set(value)
			}
		}

		// init the new controller instance.
		b.init(ctx)

		// execute the responsible method for that handler.
		methodFunc.Interface().(func())()

		// finally, execute the controller.
		b.exec()
	}
}
