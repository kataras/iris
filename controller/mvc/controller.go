package mvc

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/kataras/iris/v14/context"
	"github.com/kataras/iris/v14/controller/mvc/handler"
	"github.com/kataras/iris/v14/core/pkg/reflex"
	"github.com/kataras/iris/v14/core/router"
	"github.com/kataras/iris/v14/rest/errors"
)

// Controller is the interface that describes the behavior of a controller.
// It's used to generate the routes for the controller.
// The controller may be registered as a dependency before calling `UseController`.
// If it is not registered, a new instance is created and its exported fields are
// bound from the dependency injection container.
//
// LIFETIME: a controller is a SINGLETON.
//
// It is resolved once, its exported fields are bound once, and its methods are
// captured into handlers at registration time. There is no per-request instance
// and no pooling, so every field is shared, unsynchronised, across all
// concurrent requests.
//
// Use fields for dependencies only:
//
//	type API struct {
//		Service *Service // fine: set once at registration
//		user    *User    // WRONG: one request's user, visible to every other
//	}
//
// Anything that belongs to a single request belongs on the Context or in a
// method parameter, never on the controller.
type Controller[Context handler.ContextType, Request any, Response any, ID context.IDType /* list */, Filter any, Total context.CountType] interface {
	// POST: /resource
	Create(Context, Request) (Response, error)
	// GET: /resource/{id}
	Get(Context, ID) (Response, error)
	// PUT: /resource/{id}?columns=column1&columns=column2
	Update(Context, ID, Request, ...string) (*Response, bool, error)
	// DELETE: /resource/{id}
	Delete(Context, ID) (bool, error)
	// POST: /resource/list
	List(ctx Context, filter Filter, limit int, offset int) ([]Response, Total, error)
}

// UseController registers a new controller to the router and sets up the necessary routes for CRUD operations.
// It accepts a generic type C which must implement the Controller interface.
//
// The function performs the following steps:
// 1. Creates a new path parameter for the controller's response type and ID type.
// 2. Registers the path parameter as a dependency to the router.
// 3. Resolves the controller from the dependency injection container or creates a new instance if not found.
// 4. Binds the controller's fields if necessary.
// 5. Calls the controller's custom Routing method if it implements the router.Controller interface, before the CRUD routes are registered, so that an r.Use inside it guards them too.
// 6. Registers the CRUD routes (Create, Get, Update, Delete, List) to the router.
//
// The optional Options customize the generated routes: `Guard` protects the
// write routes, `Without` drops routes, `IDParam` renames the path parameter,
// `BeforeWrite` stamps the decoded body of Create and Update, and
// `Pagination` declares the List route's sort policy.
//
// The function returns a DeployedController which can be used to add additional routes or controllers under the same path.
//
// Example usage:
//
//	app.Router("/stores", mvc.UseController[*StoreController]()).
//		With("/books", mvc.UseController[*BookController]())
//
// This will register the StoreController under the /stores path and the BookController under the /stores/{store_id}/books path.
func UseController[C Controller[Context, Request, Response, ID, Filter, Total], Context handler.ContextType, Request any, Response any, ID context.IDType, Filter any, Total context.CountType](opts ...Option) *DeployedController[C, Context, Request, Response, ID, Filter, Total] {
	o := makeOptions(opts)

	param := newRawPathParam[Response, ID]()
	if o.idParamName != "" {
		param = newNamedPathParam(o.idParamName, param.Type)
	}

	deployed := &DeployedController[C, Context, Request, Response, ID, Filter, Total]{
		ModelName: strings.ToLower(strings.ReplaceAll(reflex.ShortNameOf(new(Response)), "*", "")), // e.g. "customer".

		param: param,
	}

	var ctrl C
	routing := func(r router.Router) {
		o.checkBeforeWrite("UseController", r.GetRelPath(), reflect.TypeFor[*Request]())

		r.UseOnce(errors.Handler)

		// Register the path parameters as dependencies of their [Response] type.
		r.RegisterDependency(&PathParam[Response, ID]{RawPathParam: param})

		c, v, ok := constructController[C](r)
		if !ok {
			// Unreachable while C is constrained to a controller type: the
			// container resolves it by that exact type.
			panic(fmt.Sprintf("iris: mvc: UseController: %s: the resolved %T does not implement the Controller interface", r.GetRelPath(), v.Interface()))
		}
		ctrl = c
		// Publish the resolved controller on the returned value.
		//
		// Note(@kataras): this MUST be an assignment to the struct field, not to
		// the local. The struct literal used to capture the local "ctrl" while it
		// was still the zero C, so DeployedController.Controller stayed nil
		// forever. Extensions read that field - WithViews builds method values
		// from it - and a method value on a nil pointer receiver is legal Go that
		// only fails when it is invoked, i.e. on every request to a view route.
		//
		// appendController composes mainRouting before the extension's routing,
		// so the field is filled by the time an extension reads it.
		deployed.Controller = c

		typ := v.Type()

		// Store it as binded to the exact controller pointer value, so
		// it can be retrieved from its `Routing` method, if needed.
		storePathParam(v.Interface(), param)

		// Note(@kataras): Routing runs BEFORE the CRUD routes are registered.
		//
		// A route captures the router's middleware when it is created
		// (ConcatHandlerLists in the router builder deep-clones the list), so an
		// r.Use inside Routing cannot reach a route that already exists. With the
		// old ordering, a controller that guarded itself with
		//
		//	func (c *C) Routing(r iris.Router) { r.Use(auth.Require) ... }
		//
		// protected only the routes Routing itself added, and left the generated
		// Create/Get/Update/Delete/List unauthenticated - with no warning.
		if routing, ok := any(ctrl).(router.Controller); ok {
			routing.Routing(r)
		}

		// The BeforeWrite hooks, as a request decoder which decodes the body as
		// usual and then hands it to them; shared by Create and Update.
		beforeWrite := beforeWriteDecoder[Request](o)

		// Create Handler.
		if !o.skip(Create) {
			createHandler := handler.Handler(ctrl.Create, handler.WithStatusCode(201), beforeWrite)
			r.Post("/", o.handlers(Create, createHandler)...).SetSourceByMethodType(typ, "Create")
		}

		// Read/Get Handler.
		if !o.skip(Get) {
			readHandler := handler.Handler(ctrl.Get, handler.WithPathParameter[ID](param.Name))
			r.Get(param.RoutePath, o.handlers(Get, readHandler)...).SetSourceByMethodType(typ, "Get")
		}

		// Update Handler.
		if !o.skip(Update) {
			updateHandler := handler.HandlerByColumns(ctrl.Update, handler.WithPathParameter[ID](param.Name), beforeWrite)
			r.Put(param.RoutePath, o.handlers(Update, updateHandler)...).SetSourceByMethodType(typ, "Update")
		}

		// Delete Handler.
		if !o.skip(Delete) {
			deleteHandler := handler.Handler(ctrl.Delete, handler.WithPathParameter[ID](param.Name))
			r.Delete(param.RoutePath, o.handlers(Delete, deleteHandler)...).SetSourceByMethodType(typ, "Delete")
		}

		// List Handler.
		if !o.skip(List) {
			listHandler := handler.ListHandler(ctrl.List, handler.WithPagination(o.pagination...))
			// POST for existing clients, QUERY (RFC 10008) for the ones that can send it.
			r.Handle(router.MethodPost+" "+router.MethodQuery, "/list", o.handlers(List, listHandler)...).SetSourceByMethodType(typ, "List")
		}
	}

	deployed.routing = routing
	return deployed
}

func constructController[C any](r router.Router) (C, reflect.Value, bool) {
	// Diagnose the type parameter before reflect does it for us with an opaque
	// "reflect: Elem of invalid type" or a nil dereference.
	if typ := reflect.TypeFor[C](); typ.Kind() != reflect.Pointer || typ.Elem().Kind() != reflect.Struct {
		panic(fmt.Sprintf("iris: mvc: UseController: %s: C must be a pointer to a struct, got %s",
			r.GetRelPath(), reflect.TypeFor[C]().String()))
	}

	ctrl, ok := r.DI().Resolve[C]() // C is a pointer already, so it will be resolved as a pointer.
	if ok {
		return ctrl, reflect.ValueOf(ctrl), true
	}

	// If not resolved, create a new instance and bind its fields.
	// panic(fmt.Sprintf("UseController: %s:  the given controller must be already registered as a dependency. Use `Services` or `RegisterDependency` before calling `UseController`", r.GetRelPath()))
	v := reflect.New(reflect.TypeFor[C]().Elem())
	// The problem here is that the controller is already registered and resolved with its NewXXX func, so this is a bit late.
	// Another idea in order to not implement something like mvc.UseControllers(...) inside an app.RegisterDependency
	// is to make an exception for PathParam[T,V] to be defined as an exported field so we can resolve and bind them
	// manually inside the framework, see bindControllerPathParamFields.
	// bindControllerPathParamFields(v, r.DI().Dependencies())
	if err := r.DI().BindStructFields(v); err != nil {
		panic(fmt.Errorf("iris: mvc: UseController: %s: binding the controller's fields: %w", r.GetRelPath(), err))
	}

	if !v.CanInterface() {
		panic(fmt.Sprintf("iris: mvc: UseController: %s: the constructed controller is not interfaceable", r.GetRelPath()))
	}
	ctrl, ok = v.Interface().(C)
	return ctrl, v, ok
}

// DeployedController is a controller that has been deployed to a router through the `UseController` function to handle the CRUD routes.
type DeployedController[C Controller[Context, Request, Response, ID, Filter, Total], Context handler.ContextType, Request any, Response any, ID context.IDType, Filter any, Total context.CountType] struct {
	Controller C      `dep:"-"` // ignore for self-resolving.
	ModelName  string `dep:"-"`

	param   *RawPathParam
	routing router.Routing
}

// With registers additional controllers under the same path.
// It accepts a relative path without parameters and one or more controllers.
func (rc *DeployedController[C, Context, Request, Response, ID, Filter, Total]) With(relPathWithoutParam string, controllers ...any) *DeployedController[C, Context, Request, Response, ID, Filter, Total] {
	relPath := rc.param.RoutePath + relPathWithoutParam
	rc.appendController(relPath, controllers...)
	return rc
}

func (rc *DeployedController[C, Context, Request, Response, ID, Filter, Total]) appendController(relPath string, controllers ...any) {
	newRouting := func(r router.Router) {
		r.Router(relPath, controllers...)
	}
	mainRouting := rc.routing

	rc.routing = func(r router.Router) {
		mainRouting(r)
		newRouting(r)
	}
}

func (rc *DeployedController[C, Context, Request, Response, ID, Filter, Total]) Routing(r router.Router) {
	rc.routing(r)
}
