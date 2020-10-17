package hero

import (
	stdContext "context"
	"errors"
	"net"
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/kataras/iris/v12/context"
	"github.com/kataras/iris/v12/sessions"

	"github.com/kataras/golog"
)

// Default is the default container value which can be used for dependencies share.
var Default = New().WithLogger(golog.Default)

// Container contains and delivers the Dependencies that will be binded
// to the controller(s) or handler(s) that can be created
// using the Container's `Handler` and `Struct` methods.
//
// This is not exported for being used by everyone, use it only when you want
// to share containers between multi mvc.go#Application
// or make custom hero handlers that can be used on the standard
// iris' APIBuilder.
//
// For a more high-level structure please take a look at the "mvc.go#Application".
type Container struct {
	// Optional Logger to report dependencies and matched bindings
	// per struct, function and method.
	// By default it is set by the Party creator of this Container.
	Logger *golog.Logger
	// Sorter specifies how the inputs should be sorted before binded.
	// Defaults to sort by "thinnest" target empty interface.
	Sorter Sorter
	// The dependencies entries.
	Dependencies []*Dependency
	// GetErrorHandler should return a valid `ErrorHandler` to handle bindings AND handler dispatch errors.
	// Defaults to a functon which returns the `DefaultErrorHandler`.
	GetErrorHandler func(*context.Context) ErrorHandler // cannot be nil.
	// Reports contains an ordered list of information about bindings for further analysys and testing.
	Reports []*Report

	// resultHandlers is a list of functions that serve the return struct value of a function handler.
	// Defaults to "defaultResultHandler" but it can be overridden.
	resultHandlers []func(next ResultHandler) ResultHandler
}

// A Report holds meta information about dependency sources and target values per package,
// struct, struct's fields, struct's method, package-level function or closure.
// E.g. main -> (*UserController) -> HandleHTTPError.
type Report struct {
	// The name is the last part of the name of a struct or its methods or a function.
	// Each name is splited by its package.struct.field or package.funcName or package.func.inlineFunc.
	Name string
	// If it's a struct or package or function
	// then it contains children reports of each one of its methods or input parameters
	// respectfully.
	Reports []*Report

	Parent  *Report
	Entries []ReportEntry
}

// A ReportEntry holds the information about a binding.
type ReportEntry struct {
	InputPosition   int          // struct field position or parameter position.
	InputFieldName  string       // if it's a struct field, then this is its type name (we can't get param names).
	InputFieldType  reflect.Type // the input's type.
	DependencyValue interface{}  // the dependency value binded to that InputPosition of Name.
	DependencyFile  string       // the file
	DependencyLine  int          // and line number of the dependency's value.
	Static          bool
}

func (r *Report) fill(bindings []*binding) {
	for _, b := range bindings {
		inputFieldName := b.Input.StructFieldName
		if inputFieldName == "" {
			// it's not a struct field, then type.
			inputFieldName = b.Input.Type.String()
		}
		// remove only the main one prefix.
		inputFieldName = strings.TrimPrefix(inputFieldName, "main.")

		fieldName := inputFieldName
		switch fieldName {
		case "*context.Context":
			inputFieldName = strings.Replace(inputFieldName, "*context", "iris", 1)
		case "hero.Code", "hero.Result", "hero.View", "hero.Response":
			inputFieldName = strings.Replace(inputFieldName, "hero", "mvc", 1)
		}

		entry := ReportEntry{
			InputPosition:  b.Input.Index,
			InputFieldName: inputFieldName,
			InputFieldType: b.Input.Type,

			DependencyValue: b.Dependency.OriginalValue,
			DependencyFile:  b.Dependency.Source.File,
			DependencyLine:  b.Dependency.Source.Line,
			Static:          b.Dependency.Static,
		}

		r.Entries = append(r.Entries, entry)
	}
}

// fillReport adds a report to the Reports field.
func (c *Container) fillReport(fullName string, bindings []*binding) {
	// r := c.getReport(fullName)

	r := &Report{
		Name: fullName,
	}
	r.fill(bindings)
	c.Reports = append(c.Reports, r)
}

// BuiltinDependencies is a list of builtin dependencies that are added on Container's initilization.
// Contains the iris context, standard context, iris sessions and time dependencies.
var BuiltinDependencies = []*Dependency{
	// iris context dependency.
	NewDependency(func(ctx *context.Context) *context.Context { return ctx }).Explicitly(),
	// standard context dependency.
	NewDependency(func(ctx *context.Context) stdContext.Context {
		return ctx.Request().Context()
	}).Explicitly(),
	// iris session dependency.
	NewDependency(func(ctx *context.Context) *sessions.Session {
		session := sessions.Get(ctx)
		if session == nil {
			ctx.Application().Logger().Debugf("binding: session is nil\nMaybe inside HandleHTTPError? Register it with app.UseRouter(sess.Handler()) to fix it")
			// let's don't panic here and let the application continue, now we support
			// not matched routes inside the controller through HandleHTTPError,
			// so each dependency can check if session was not nil or just use `UseRouter` instead of `Use`
			// to register the sessions middleware.
		}

		return session
	}).Explicitly(),
	// application's logger.
	NewDependency(func(ctx *context.Context) *golog.Logger {
		return ctx.Application().Logger()
	}).Explicitly(),
	// time.Time to time.Now dependency.
	NewDependency(func(ctx *context.Context) time.Time {
		return time.Now()
	}).Explicitly(),
	// standard http Request dependency.
	NewDependency(func(ctx *context.Context) *http.Request {
		return ctx.Request()
	}).Explicitly(),
	// standard http ResponseWriter dependency.
	NewDependency(func(ctx *context.Context) http.ResponseWriter {
		return ctx.ResponseWriter()
	}).Explicitly(),
	// http headers dependency.
	NewDependency(func(ctx *context.Context) http.Header {
		return ctx.Request().Header
	}).Explicitly(),
	// Client IP.
	NewDependency(func(ctx *context.Context) net.IP {
		return net.ParseIP(ctx.RemoteAddr())
	}).Explicitly(),
	// Status Code (special type for MVC HTTP Error handler to not conflict with path parameters)
	NewDependency(func(ctx *context.Context) Code {
		return Code(ctx.GetStatusCode())
	}).Explicitly(),
	// Context Error. May be nil
	NewDependency(func(ctx *context.Context) Err {
		err := ctx.GetErr()
		if err == nil {
			return nil
		}
		return err
	}).Explicitly(),
	// Context User, e.g. from basic authentication.
	NewDependency(func(ctx *context.Context) context.User {
		u := ctx.User()
		if u == nil {
			return nil
		}

		return u
	}),
	// payload and param bindings are dynamically allocated and declared at the end of the `binding` source file.
}

// New returns a new Container, a container for dependencies and a factory
// for handlers and controllers, this is used internally by the `mvc#Application` structure.
// Please take a look at the structure's documentation for more information.
func New(dependencies ...interface{}) *Container {
	deps := make([]*Dependency, len(BuiltinDependencies))
	copy(deps, BuiltinDependencies)

	c := &Container{
		Sorter:       sortByNumMethods,
		Dependencies: deps,
		GetErrorHandler: func(*context.Context) ErrorHandler {
			return DefaultErrorHandler
		},
	}

	for _, dependency := range dependencies {
		c.Register(dependency)
	}

	return c
}

// WithLogger injects a logger to use to debug dependencies and bindings.
func (c *Container) WithLogger(logger *golog.Logger) *Container {
	c.Logger = logger
	return c
}

// Clone returns a new cloned container.
// It copies the ErrorHandler, Dependencies and all Options from "c" receiver.
func (c *Container) Clone() *Container {
	cloned := New()
	cloned.Logger = c.Logger
	cloned.GetErrorHandler = c.GetErrorHandler
	cloned.Sorter = c.Sorter
	clonedDeps := make([]*Dependency, len(c.Dependencies))
	copy(clonedDeps, c.Dependencies)
	cloned.Dependencies = clonedDeps
	cloned.resultHandlers = c.resultHandlers
	// Reports are not cloned.
	return cloned
}

// Register adds a dependency.
// The value can be a single struct value-instance or a function
// which has one input and one output, that output type
// will be binded to the handler's input argument, if matching.
//
// Usage:
// - Register(loggerService{prefix: "dev"})
// - Register(func(ctx iris.Context) User {...})
// - Register(func(User) OtherResponse {...})
func Register(dependency interface{}) *Dependency {
	return Default.Register(dependency)
}

// Register adds a dependency.
// The value can be a single struct value or a function.
// Follow the rules:
// * <T>{structValue}
// * func(accepts <T>)                                 returns <D> or (<D>, error)
// * func(accepts iris.Context)                        returns <D> or (<D>, error)
// * func(accepts1 iris.Context, accepts2 *hero.Input) returns <D> or (<D>, error)
//
// A Dependency can accept a previous registered dependency and return a new one or the same updated.
// * func(accepts1 <D>, accepts2 <T>)                  returns <E> or (<E>, error) or error
// * func(acceptsPathParameter1 string, id uint64)     returns <T> or (<T>, error)
//
// Usage:
//
// - Register(loggerService{prefix: "dev"})
// - Register(func(ctx iris.Context) User {...})
// - Register(func(User) OtherResponse {...})
func (c *Container) Register(dependency interface{}) *Dependency {
	d := NewDependency(dependency, c.Dependencies...)
	if d.DestType == nil {
		// prepend the dynamic dependency so it will be tried at the end
		// (we don't care about performance here, design-time)
		c.Dependencies = append([]*Dependency{d}, c.Dependencies...)
	} else {
		c.Dependencies = append(c.Dependencies, d)
	}

	return d
}

// UseResultHandler adds a result handler to the Container.
// A result handler can be used to inject the returned struct value
// from a request handler or to replace the default renderer.
func (c *Container) UseResultHandler(handler func(next ResultHandler) ResultHandler) *Container {
	c.resultHandlers = append(c.resultHandlers, handler)
	return c
}

// Handler accepts a "handler" function which can accept any input arguments that match
// with the Container's `Dependencies` and any output result; like string, int (string,int),
// custom structs, Result(View | Response) and anything you can imagine.
// It returns a standard `iris/context.Handler` which can be used anywhere in an Iris Application,
// as middleware or as simple route handler or subdomain's handler.
func Handler(fn interface{}) context.Handler {
	return Default.Handler(fn)
}

// Handler accepts a handler "fn" function which can accept any input arguments that match
// with the Container's `Dependencies` and any output result; like string, int (string,int),
// custom structs, Result(View | Response) and more.
// It returns a standard `iris/context.Handler` which can be used anywhere in an Iris Application,
// as middleware or as simple route handler or subdomain's handler.
func (c *Container) Handler(fn interface{}) context.Handler {
	return c.HandlerWithParams(fn, 0)
}

// HandlerWithParams same as `Handler` but it can receive a total path parameters counts
// to resolve coblex path parameters input dependencies.
func (c *Container) HandlerWithParams(fn interface{}, paramsCount int) context.Handler {
	return makeHandler(fn, c, paramsCount)
}

// Struct accepts a pointer to a struct value and returns a structure which
// contains bindings for the struct's fields and a method to
// extract a Handler from this struct's method.
func (c *Container) Struct(ptrValue interface{}, partyParamsCount int) *Struct {
	return makeStruct(ptrValue, c, partyParamsCount)
}

// ErrMissingDependency may returned only from the `Container.Inject` method
// when not a matching dependency found for "toPtr".
var ErrMissingDependency = errors.New("missing dependency")

// Inject SHOULD only be used outside of HTTP handlers (performance is not priority for this method)
// as it does not pre-calculate the available list of bindings for the "toPtr" and the registered dependencies.
//
// It sets a static-only matching dependency to the value of "toPtr".
// The parameter "toPtr" SHOULD be a pointer to a value corresponding to a dependency,
// like input parameter of a handler or field of a struct.
//
// If no matching dependency found, the `Inject` method returns an `ErrMissingDependency` and
// "toPtr" keeps its original state (e.g. nil).
//
// Example Code:
// c.Register(&LocalDatabase{...})
// [...]
// var db Database
// err := c.Inject(&db)
func (c *Container) Inject(toPtr interface{}) error {
	val := reflect.Indirect(valueOf(toPtr))
	typ := val.Type()

	for _, d := range c.Dependencies {
		if d.Static && matchDependency(d, typ) {
			v, err := d.Handle(nil, &Input{Type: typ})
			if err != nil {
				if err == ErrSeeOther {
					continue
				}

				return err
			}

			val.Set(v)
			return nil
		}
	}

	return ErrMissingDependency
}
