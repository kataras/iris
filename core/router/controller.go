package router

import (
	"reflect"
	"strings"

	"github.com/kataras/iris/context"
	"github.com/kataras/iris/core/errors"
)

// Controller is the base controller for the high level controllers instances.
//
// This base controller is used as an alternative way of building
// APIs, the controller can register all type of http methods.
//
// Keep note that controllers are bit slow
// because of the reflection use however it's as fast as possible because
// it does preparation before the serve-time handler but still
// remains slower than the low-level handlers
// such as `Handle, Get, Post, Put, Delete, Connect, Head, Trace, Patch`.
//
//
// All fields that are tagged with iris:"persistence"`
// are being persistence and kept between the different requests,
// meaning that these data will not be reset-ed on each new request,
// they will be the same for all requests.
//
// An Example Controller can be:
//
// type IndexController struct {
// 	Controller
// }
//
// func (c *IndexController) Get() {
// 	c.Tmpl = "index.html"
// 	c.Data["title"] = "Index page"
// 	c.Data["message"] = "Hello world!"
// }
//
// Usage: app.Controller("/", new(IndexController))
//
//
// Another example with persistence data:
//
// type UserController struct {
// 	Controller
//
// 	CreatedAt time.Time `iris:"persistence"`
// 	Title     string    `iris:"persistence"`
// 	DB        *DB		`iris:"persistence"`
// }
//
// // Get serves using the User controller when HTTP Method is "GET".
// func (c *UserController) Get() {
// 	c.Tmpl = "user/index.html"
// 	c.Data["title"] = c.Title
// 	c.Data["username"] = "kataras " + c.Params.Get("userid")
// 	c.Data["connstring"] = c.DB.Connstring
// 	c.Data["uptime"] = time.Now().Sub(c.CreatedAt).Seconds()
// }
//
// Usage: app.Controller("/user/{id:int}", &UserController{
// 	CreatedAt: time.Now(),
// 	Title: "User page",
// 	DB: yourDB,
// })
//
// Look `router#APIBuilder#Controller` method too.
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
// that are necessary to him/her, do not confuse that
// with the optional custom `Init` of the higher-level Controller.
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

var (
	// ErrInvalidControllerType is a static error which fired from `Controller` when
	// the passed "c" instnace is not a valid type of `Controller`.
	ErrInvalidControllerType = errors.New("controller should have a field of Controller type")
)

// get the field name at compile-time,
// will help us to catch any unexpected results on future versions.
var baseControllerName = reflect.TypeOf(Controller{}).Name()

// registers a controller to a specific `Party`.
// Consumed by `APIBuilder#Controller` function.
func registerController(p Party, path string, c interface{}) ([]*Route, error) {
	typ := reflect.TypeOf(c)

	if typ.Kind() != reflect.Ptr {
		typ = reflect.PtrTo(typ)
	}

	elem := typ.Elem()

	// check if "c" has the "Controller" typeof `Controller` field.
	b, has := elem.FieldByName(baseControllerName)
	if !has {
		return nil, ErrInvalidControllerType
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
			if t, ok := f.Tag.Lookup("iris"); ok {
				if t == "persistence" {
					persistenceFields[i] = reflect.ValueOf(valF.Interface())
					continue
				}
			}

			// no: , lets have only the tag
			// even for pointers, this will make
			// things clear
			// so a *Session can be declared
			// without having to introduce
			// a new tag such as `iris:"omit_persistence"`
			// old:
			// catch persistence data by pointer, i.e:
			// DB *Database
			// if f.Type.Kind() == reflect.Ptr {
			// 	if !valF.IsNil() {
			// 		persistenceFields[i] = reflect.ValueOf(valF.Interface())
			// 	}
			// }
		}
	}

	customInitFuncIndex, _ := getCustomInitFuncIndex(typ)

	// check if has Any() or All()
	// if yes, then register all http methods and
	// exit.
	m, has := typ.MethodByName("Any")
	if !has {
		m, has = typ.MethodByName("All")
	}
	if has {
		routes := p.Any(path,
			controllerToHandler(elem, persistenceFields,
				baseControllerFieldIndex, m.Index, customInitFuncIndex))
		return routes, nil
	}

	var routes []*Route
	// else search the entire controller
	// for any compatible method function
	// and register that.
	for _, method := range AllMethods {
		httpMethodFuncName := strings.Title(strings.ToLower(method))

		m, has := typ.MethodByName(httpMethodFuncName)
		if !has {
			continue
		}

		httpMethodIndex := m.Index

		r := p.Handle(method, path,
			controllerToHandler(elem, persistenceFields,
				baseControllerFieldIndex, httpMethodIndex, customInitFuncIndex))
		routes = append(routes, r)
	}
	return routes, nil
}

func controllerToHandler(elem reflect.Type, persistenceFields map[int]reflect.Value,
	baseControllerFieldIndex, httpMethodIndex int, customInitFuncIndex int) context.Handler {
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

		// calls the higher "Init(ctx context.Context)",
		// if exists.
		if customInitFuncIndex > 0 {
			callCustomInit(ctx, c, customInitFuncIndex)
		}

		// if custom Init didn't stop the execution of the
		// context
		if !ctx.IsStopped() {
			// execute the responsible method for that handler.
			methodFunc.Interface().(func())()
		}

		// finally, execute the controller.
		b.exec()
	}
}

// Init can be used as a custom function
// to init the new instance of controller
// that is created on each new request.
//
// Useful when more than one methods are using the same
// request data.
const customInitFuncName = "Init"

func getCustomInitFuncIndex(typ reflect.Type) (initFuncIndex int, has bool) {
	if m, has := typ.MethodByName(customInitFuncName); has {
		return m.Index, has
	}

	return -1, false
}

// the "cServeTime" is a new "c" instance
// which is being used at serve time, inside the Handler.
// it calls the custom "Init", the check of this
// function made at build time, so it's a safe a call.
func callCustomInit(ctx context.Context, cServeTime reflect.Value, initFuncIndex int) {
	cServeTime.Method(initFuncIndex).Interface().(func(ctx context.Context))(ctx)
}
