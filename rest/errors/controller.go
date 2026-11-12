// Package errors is the error handling system for the Iris web framework.
// This file holds the controller integration: it processes the errors returned by
// error-returning controller methods and routing functions.
package errors

import (
	"reflect"

	"github.com/kataras/iris/v14/context"
	"github.com/kataras/iris/v14/core/router"
)

func init() {
	router.RegisterControllerResolver(routingToController)
	router.RegisterControllerResolver(structToController)
}

// Routing is a function type that implements the router.Controller interface for error-aware routing.
// It accepts an error-aware Router and defines route handlers that can return errors.
// These errors are automatically processed by the error handling middleware.
//
// Routing functions provide a clean way to define error-returning routes:
//
//	app.Handle(errors.Routing(func(r *errors.Router) {
//	    r.Get("/users/{id}", func(ctx iris.Context) error {
//	        user, err := getUserByID(ctx.Params().Get("id"))
//	        if err != nil {
//	            return errors.NotFound.Wrap(err)
//	        }
//	        return ctx.JSON(user)
//	    })
//	}))
//
// The Routing type enables declarative error handling where any returned error
// is automatically converted to an appropriate HTTP response using the configured
// error handling options.
type Routing func(*Router)

var _ router.Controller = (*Routing)(nil)

// Routing implements the router.Controller interface, enabling automatic registration
// of error-aware routing functions with the Iris router system.
func (c Routing) Routing(r router.Router) {
	newRouter := NewRouter(r)
	c(newRouter)
}

// routingToController is a controller resolver that automatically converts Routing functions
// and structs with Routing methods to router.Controller implementations. This resolver is
// registered during package initialization, so error-aware routing works with the
// standard Iris controller system without further setup.
//
// The resolver handles two patterns:
//  1. Direct Routing functions: func(*errors.Router)
//  2. Structs implementing: Routing(*errors.Router) method
//
// This allows both functional and object-oriented approaches to error-aware routing:
//
//	// Functional approach
//	routing := func(r *errors.Router) {
//	    r.Get("/api/data", dataHandler)
//	}
//	app.Handle(routing)
//
//	// Object-oriented approach
//	type Controller struct{}
//	func (c *Controller) Routing(r *errors.Router) {
//	    r.Get("/api/users", c.getUsers)
//	}
//	app.Handle(&Controller{})
func routingToController(ctrl any) (router.Controller, bool) {
	if ctrl == nil {
		return nil, false
	}

	if fn, isRoutingFunc := ctrl.(func(*Router)); isRoutingFunc {
		c := router.Routing(func(r router.Router) {
			newRouter := NewRouter(r)
			fn(newRouter)
		})

		return c, true
	}

	if c, ok := ctrl.(interface{ Routing(*Router) }); ok {
		return Routing(c.Routing), true
	}

	return nil, false
}

type (
	// ctrlGetErrMethod defines a controller method that handles GET requests and returns an error.
	// Controllers implementing this interface can return errors that are automatically processed
	// by the error handling middleware, eliminating the need for manual error response handling.
	//
	// Example implementation:
	//
	//	type UserController struct{}
	//
	//	func (c *UserController) Get(ctx iris.Context) error {
	//	    users, err := c.userService.GetAll()
	//	    if err != nil {
	//	        return errors.InternalServerError.Wrap(err)
	//	    }
	//	    return ctx.JSON(users)
	//	}
	ctrlGetErrMethod interface {
		Get(*context.Context) error
	}

	// ctrlPostErrMethod defines a controller method that handles POST requests and returns an error.
	// Enables automatic error processing for creation and data submission endpoints.
	ctrlPostErrMethod interface {
		Post(*context.Context) error
	}

	// ctrlPutErrMethod defines a controller method that handles PUT requests and returns an error.
	// Suitable for update operations with automatic error handling.
	ctrlPutErrMethod interface {
		Put(*context.Context) error
	}

	// ctrlDeleteErrMethod defines a controller method that handles DELETE requests and returns an error.
	// Provides error-aware deletion operations with automatic response handling.
	ctrlDeleteErrMethod interface {
		Delete(*context.Context) error
	}

	// ctrlPatchErrMethod defines a controller method that handles PATCH requests and returns an error.
	// Enables partial update operations with automatic error processing.
	ctrlPatchErrMethod interface {
		Patch(*context.Context) error
	}

	// ctrlOptionsErrMethod defines a controller method that handles OPTIONS requests and returns an error.
	// Useful for CORS preflight and API capability discovery with error handling.
	ctrlOptionsErrMethod interface {
		Options(*context.Context) error
	}

	// ctrlHeadErrMethod defines a controller method that handles HEAD requests and returns an error.
	// Provides metadata retrieval operations with automatic error response handling.
	ctrlHeadErrMethod interface {
		Head(*context.Context) error
	}

	// ctrlConnectErrMethod defines a controller method that handles CONNECT requests and returns an error.
	// Supports tunnel establishment operations with error-aware processing.
	ctrlConnectErrMethod interface {
		Connect(*context.Context) error
	}

	// ctrlTraceErrMethod defines a controller method that handles TRACE requests and returns an error.
	// Enables diagnostic and debugging operations with automatic error handling.
	ctrlTraceErrMethod interface {
		Trace(*context.Context) error
	}

	// ctrlQueryErrMethod defines a controller method that handles QUERY requests and returns an error.
	// Suits list and search endpoints that read a filter from the request body.
	ctrlQueryErrMethod interface {
		Query(*context.Context) error
	}

	// ctrlErrorHandlerMethod defines a custom error handler method for controllers.
	// Controllers implementing this interface can provide custom error processing logic
	// that will be used as middleware for all routes defined by the controller.
	//
	// The HandleError method receives the context and the processed Error struct,
	// allowing for controller-specific error formatting, logging, or response customization.
	//
	// Example implementation:
	//
	//	type APIController struct{}
	//
	//	func (c *APIController) HandleError(ctx iris.Context, err *errors.Error) {
	//	    // Custom logging
	//	    log.Printf("API Error [%s]: %v", err.Code, err.Message)
	//
	//	    // Custom response format
	//	    ctx.JSON(iris.Map{
	//	        "error": err.Code,
	//	        "message": err.Message,
	//	        "timestamp": time.Now(),
	//	    })
	//	}
	ctrlErrorHandlerMethod interface {
		HandleError(*context.Context, *Error)
	}
)

// structToController is a controller resolver that automatically converts structs with
// error-returning HTTP method implementations into router.Controller instances.
// This resolver lets error-aware controllers work with the standard Iris routing
// system without further setup.
//
// The resolver automatically detects and registers HTTP methods that return errors:
//   - Get(iris.Context) error
//   - Post(iris.Context) error
//   - Put(iris.Context) error
//   - Delete(iris.Context) error
//   - Patch(iris.Context) error
//   - Options(iris.Context) error
//   - Head(iris.Context) error
//   - Connect(iris.Context) error
//   - Trace(iris.Context) error
//   - Query(iris.Context) error
//
// Additionally, if the controller implements HandleError(iris.Context, *Error),
// that method will be used as custom error handling middleware for all controller routes.
//
// Example usage:
//
//	type ProductController struct {
//	    service ProductService
//	}
//
//	func (c *ProductController) Get(ctx iris.Context) error {
//	    products, err := c.service.GetAll()
//	    if err != nil {
//	        return errors.InternalServerError.Wrap(err)
//	    }
//	    return ctx.JSON(products)
//	}
//
//	func (c *ProductController) Post(ctx iris.Context) error {
//	    var product Product
//	    if err := ctx.ReadJSON(&product); err != nil {
//	        return errors.BadRequest.Wrap(err)
//	    }
//
//	    created, err := c.service.Create(product)
//	    if err != nil {
//	        return errors.InternalServerError.Wrap(err)
//	    }
//	    return ctx.JSON(created)
//	}
//
//	func (c *ProductController) HandleError(ctx iris.Context, err *errors.Error) {
//	    // Custom error handling for this controller
//	    ctx.JSON(iris.Map{"error": err.Message, "code": err.Code})
//	}
//
//	// Register the controller
//	app.Handle(&ProductController{service: productService})
//
// The resolver automatically wraps each detected method with FuncHandler to ensure
// proper error processing and maintains method source information for debugging
// and routing introspection.
func structToController(ctrl any) (router.Controller, bool) {
	if ctrl == nil {
		return nil, false
	}

	getErrMethod, hasGetErrMethod := ctrl.(ctrlGetErrMethod)
	postErrMethod, hasPostErrMethod := ctrl.(ctrlPostErrMethod)
	putErrMethod, hasPutErrMethod := ctrl.(ctrlPutErrMethod)
	deleteErrMethod, hasDeleteErrMethod := ctrl.(ctrlDeleteErrMethod)
	patchErrMethod, hasPatchErrMethod := ctrl.(ctrlPatchErrMethod)
	optionsErrMethod, hasOptionsErrMethod := ctrl.(ctrlOptionsErrMethod)
	headErrMethod, hasHeadErrMethod := ctrl.(ctrlHeadErrMethod)
	connectErrMethod, hasConnectErrMethod := ctrl.(ctrlConnectErrMethod)
	traceErrMethod, hasTraceErrMethod := ctrl.(ctrlTraceErrMethod)
	queryErrMethod, hasQueryErrMethod := ctrl.(ctrlQueryErrMethod)

	errHandler, hasErrHandler := ctrl.(ctrlErrorHandlerMethod)

	if !hasGetErrMethod && !hasPostErrMethod && !hasPutErrMethod && !hasDeleteErrMethod && !hasPatchErrMethod && !hasOptionsErrMethod && !hasHeadErrMethod && !hasConnectErrMethod && !hasTraceErrMethod && !hasQueryErrMethod && !hasErrHandler {
		return nil, false
	}

	c, isControllerToo := ctrl.(router.Controller) // check for this afterwards, because there is already check for the Controller implementation.

	typ := reflect.TypeOf(ctrl)

	controller := router.Routing(func(r router.Router) {
		if hasErrHandler {
			r.Use(NewOptions().SetRenderer(errHandler.HandleError).Handler())
		}

		if hasGetErrMethod {
			route := r.Get("/", FuncHandler(getErrMethod.Get))
			route.SetSourceByMethodType(typ, "Get")
		}

		if hasPostErrMethod {
			route := r.Post("/", FuncHandler(postErrMethod.Post))
			route.SetSourceByMethodType(typ, "Post")
		}
		if hasPutErrMethod {
			route := r.Put("/", FuncHandler(putErrMethod.Put))
			route.SetSourceByMethodType(typ, "Put")
		}
		if hasDeleteErrMethod {
			route := r.Delete("/", FuncHandler(deleteErrMethod.Delete))
			route.SetSourceByMethodType(typ, "Delete")
		}
		if hasPatchErrMethod {
			route := r.Patch("/", FuncHandler(patchErrMethod.Patch))
			route.SetSourceByMethodType(typ, "Patch")
		}
		if hasOptionsErrMethod {
			route := r.Options("/", FuncHandler(optionsErrMethod.Options))
			route.SetSourceByMethodType(typ, "Options")
		}
		if hasHeadErrMethod {
			route := r.Head("/", FuncHandler(headErrMethod.Head))
			route.SetSourceByMethodType(typ, "Head")
		}
		if hasConnectErrMethod {
			route := r.Connect("/", FuncHandler(connectErrMethod.Connect))
			route.SetSourceByMethodType(typ, "Connect")
		}
		if hasTraceErrMethod {
			route := r.Trace("/", FuncHandler(traceErrMethod.Trace))
			route.SetSourceByMethodType(typ, "Trace")
		}
		if hasQueryErrMethod {
			route := r.Query("/", FuncHandler(queryErrMethod.Query))
			route.SetSourceByMethodType(typ, "Query")
		}

		if isControllerToo { // although this will not run on the current implementation becaues if ctrl was implemented Controller interface it will never come to this controller resolver.
			c.Routing(r)
		}
	})

	return controller, true
}
