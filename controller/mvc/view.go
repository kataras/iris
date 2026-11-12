package mvc

import (
	"fmt"
	"reflect"

	"github.com/kataras/iris/v14/context"
	"github.com/kataras/iris/v14/controller/mvc/handler"
	"github.com/kataras/iris/v14/core/router"
)

// ViewModel is the view model.
// It is used as a Response type to render a view with `T` model.
type ViewModel[T any] struct {
	*handler.ViewBuilder
}

var _ handler.Renderer = (*ViewModel[any])(nil)

// View returns a new ViewModel.
// It can be used as a Response value.
func View[T any](status int, layout, name string, data T) ViewModel[T] {
	return ViewModel[T]{
		ViewBuilder: handler.View().StatusCode(status).Layout(layout).Name(name).Data(data),
	}
}

// ViewController is an interface that defines methods for rendering views.
// It extends the Controller interface with additional methods for view rendering.
type ViewController[Context handler.ContextType, Request any, Response any, ID context.IDType, Filter any, Total context.CountType, IndexViewData, CreateViewModel, ModelViewData, ListViewData any] interface {
	Controller[Context, Request, Response, ID, Filter, Total]

	// GET: /resource
	IndexView(Context) (IndexViewData, error)
	// GET: /resource/create
	CreateView(Context) (CreateViewModel, error)
	// GET: /resource/{id}/details
	//
	// Note that here we could use the same GET /resource/{id} based on the
	// content-type requested (HTML vs JSON) but we want to keep the separation of concerns.
	DetailsView(Context, ID) (ModelViewData, error)
	// GET: /resource/{id}/update
	UpdateView(Context, ID) (ModelViewData, error)
	// GET: /resource/{id}/delete
	DeleteView(Context, ID) (ModelViewData, error)
	// GET: /resource/list
	ListView(Context) (ListViewData, error)
}

// DefaultTemplateExtension is the default view file extension.
// Navigate through the `WithViews` method of a deployed controller for more.
var DefaultTemplateExtension = ".html"

// WithViews extends the deployed controller with view rendering capabilities.
// It registers routes for various view-related endpoints and binds them to the corresponding controller methods.
//
// The method is generic over the four view data types and requires the deployed
// controller to implement the ViewController interface for them; a mismatch
// panics at registration time. It does not check for a registered view engine,
// a missing one is only reported when a view is rendered.
//
// The method generates default template names based on the controller's model name and the
// package-level DefaultTemplateExtension (".html" unless modified).
// It then creates handlers for each view-related endpoint and registers them with the router.
//
// The following routes are registered:
// - GET /resource: Renders the index view.
// - GET /resource/create: Renders the create view.
// - GET /resource/{id}/details: Renders the details view for a specific resource.
// - GET /resource/{id}/update: Renders the update view for a specific resource.
// - GET /resource/{id}/delete: Renders the delete view for a specific resource.
// - GET /resource/list: Renders the list view.
//
// Example usage:
//
//	app.Router("/stores", mvc.UseController[*StoreController]().WithViews[IndexData, CreateData, ModelData, ListData]())
//
// It returns the deployed controller itself for further chaining.
func (rc *DeployedController[C, Context, Request, Response, ID, Filter, Total]) WithViews[IndexViewData, CreateViewData, ModelViewData, ListViewData any]() *DeployedController[C, Context, Request, Response, ID, Filter, Total] {
	routing := func(r router.Router) {
		// The routing below runs at registration time, after the main routing
		// has resolved and published the controller on rc.Controller,
		// see UseController and appendController for the ordering guarantee.
		ctrl, ok := any(rc.Controller).(ViewController[Context, Request, Response, ID, Filter, Total, IndexViewData, CreateViewData, ModelViewData, ListViewData])
		if !ok {
			panic(fmt.Sprintf("WithViews: %s: controller %T does not implement mvc.ViewController with view data types [%s, %s, %s, %s]",
				r.GetRelPath(), rc.Controller,
				reflect.TypeFor[IndexViewData]().String(), reflect.TypeFor[CreateViewData]().String(),
				reflect.TypeFor[ModelViewData]().String(), reflect.TypeFor[ListViewData]().String()))
		}

		viewExt := DefaultTemplateExtension
		var (
			defaultLayoutTmplName  = fmt.Sprintf("%s/layout%s", rc.ModelName, viewExt)
			defaultIndexTmplName   = fmt.Sprintf("%s/index%s", rc.ModelName, viewExt)
			defaultCreateTmplName  = fmt.Sprintf("%s/create%s", rc.ModelName, viewExt)
			defaultDetailsTmplName = fmt.Sprintf("%s/details%s", rc.ModelName, viewExt)
			defaultUpdateTmplName  = fmt.Sprintf("%s/update%s", rc.ModelName, viewExt)
			defaultDeleteTmplName  = fmt.Sprintf("%s/delete%s", rc.ModelName, viewExt)
			defaultListTmplName    = fmt.Sprintf("%s/list%s", rc.ModelName, viewExt)
		)

		indexHandler := handler.IndexHandler(makeViewHandler(defaultLayoutTmplName, defaultIndexTmplName, ctrl.IndexView))
		r.Get("/", indexHandler)

		createHandler := handler.IndexHandler(makeViewHandler(defaultLayoutTmplName, defaultCreateTmplName, ctrl.CreateView))
		r.Get("/create", createHandler)

		// Note(@kataras): reuse the controller's own path parameter instead of a
		// bare "{id}". An untyped parameter stores a plain string, so a controller
		// with a uuid.UUID or int ID answered 400 on every details/update/delete
		// view, and ctx.Params().Get("book_id") inside a view method came back
		// empty because the parameter was registered under a different name.
		idPath := rc.param.RoutePath

		detailsHandler := handler.Handler(makeViewHandlerWithID(defaultLayoutTmplName, defaultDetailsTmplName, ctrl.DetailsView))
		r.Get(idPath+"/details", detailsHandler)

		updateHandler := handler.Handler(makeViewHandlerWithID(defaultLayoutTmplName, defaultUpdateTmplName, ctrl.UpdateView))
		r.Get(idPath+"/update", updateHandler)

		deleteHandler := handler.Handler(makeViewHandlerWithID(defaultLayoutTmplName, defaultDeleteTmplName, ctrl.DeleteView))
		r.Get(idPath+"/delete", deleteHandler)

		listHandler := handler.IndexHandler(makeViewHandler(defaultLayoutTmplName, defaultListTmplName, ctrl.ListView))
		r.Get("/list", listHandler)
	}

	rc.appendController("/", router.Routing(routing))
	return rc
}

// makeViewHandler creates a handler function for rendering a view without an ID parameter.
// It takes the template layout, template name, and the controller method to be called.
// The handler function calls the controller method, gets the view data, and renders the view.
func makeViewHandler[Context handler.ContextType, T any](tmplLayout, tmplName string, fn func(Context) (T, error)) func(ctx Context) (ViewModel[T], error) {
	return func(ctx Context) (ViewModel[T], error) {
		resp, err := fn(ctx)
		if err != nil {
			return ViewModel[T]{}, err
		}

		viewModel := View(200, tmplLayout, tmplName, resp)
		return viewModel, nil
	}
}

// makeViewHandlerWithID creates a handler function for rendering a view with an ID parameter.
// It takes the template layout, template name, and the controller method to be called.
// The handler function calls the controller method with the ID, gets the view data, and renders the view.
func makeViewHandlerWithID[Context handler.ContextType, T any, ID handler.IDType](tmplLayout, tmplName string, fn func(Context, ID) (T, error)) func(Context, ID) (ViewModel[T], error) {
	return func(ctx Context, id ID) (ViewModel[T], error) {
		resp, err := fn(ctx, id)
		if err != nil {
			return ViewModel[T]{}, err
		}

		viewModel := View(200, tmplLayout, tmplName, resp)
		return viewModel, nil
	}
}
