package handler

import (
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/kataras/iris/v14/context"
	"github.com/kataras/iris/v14/context/encoding"
	"github.com/kataras/iris/v14/core/pkg/reflex"
	"github.com/kataras/iris/v14/core/router"
	"github.com/kataras/iris/v14/core/router/macro"
	"github.com/kataras/iris/v14/rest/errors"
)

// Method registers a handler for the specified HTTP method and path.
// The path parameter is resolved by the last field with the `param` tag.
// The path parameter is used to create the route path.
// The handler is a function that accepts a context, a request, and a response.
// The handler function must return a response and an optional error.
// The response is written to the client with a status code of 200.
// The error is written to the client with a call of the HandleError function using the errors subpackage.
// The handler function can return a custom success status code by using the `WithStatusCode` option.
func Method[Context ContextType, Request RequestType | ParamType, Response ResponseType](method string, fn Func[Context, Request, Response], options ...Option) router.Controller {
	handler := Handler(fn, options...)

	return router.Routing(func(r router.Router) {
		path := "/"

		id, ok := ResolveRouteParam[Request](r.GetRelPath())
		if ok { // if has a `param` struct field tag. Check if this parameter is already registered as a path parameter. If not, then use it.
			if !macro.ContainsParam(*r.Macros(), r.GetRelPath(), id.Name) {
				path = id.Path()
			}
		}
		// if !ok {... panic("CreateBy: struct param must contain at least one field with the 'param' tag in order to resolve the relative path")
		// no ^ let's allow parameters without the param tag, it's useful for the user
		// to define the path manually or the parameter is not actual a route parameter but a request header or body parameter (e.g authentication).

		// fmt.Printf("Get: ID: type = %s, name = %s, path = %s\n", id.Type, id.Name, id.Path())

		r.Handle(method, path, handler)
	})
}

// MethodBy registers a handler for the specified HTTP method and path just like Method but it accepts an extra parameter.
// The handler is a function that accepts a context, a parameter, a request, and a response.
// The handler function must return a response and an optional error.
func MethodBy[Context ContextType, Param ParamType, Request RequestType, Response ResponseType](method string, fn ParameterizedFunc[Context, Param, Request, Response], options ...Option) router.Controller {
	// Note(@kataras): no forced status code here. This used to append
	// WithStatusCode(201) unconditionally, which made a successful GetBy answer
	// "201 Created". DeleteBy and UpdateBy only escaped it because their bool
	// response makes the encoder overwrite the code with 204/304. The 201 belongs
	// to CreateBy, which sets it itself - see Create/CreateBy below.
	handler := HandlerBy(fn, options...)

	return router.Routing(func(r router.Router) {
		path := "/"

		id, ok := ResolveRouteParam[Param](r.GetRelPath())
		if ok { // if has a `param` struct field tag. Check if this parameter is already registered as a path parameter. If not, then use it.
			if !macro.ContainsParam(*r.Macros(), r.GetRelPath(), id.Name) {
				path = id.Path()
			}
		}

		r.Handle(method, path, handler)
	})
}

// ListMethod registers a handler for the specified HTTP method(s), space separated, on "/list".
// The handler is a function that accepts a context, a filter, a response, and a total count.
func ListMethod[Context ContextType, Filter FilterType, Response ResponseType, Total CountType](method string, fn ListFunc[Context, Filter, Response, Total], options ...Option) router.Controller {
	handler := ListHandler(fn, options...)

	return router.Routing(func(r router.Router) {
		path := "/list"
		r.Handle(method, path, handler)
	})
}

// ListMethodBy registers a handler for the specified HTTP method and path just like ListMethod but it accepts an extra parameter.
// The handler is a function that accepts a context, a parameter, a filter, a response, and a total count.
func ListMethodBy[Context ContextType, Param ParamType, Filter FilterType, Response ResponseType, Total CountType](method string, fn ParameterizedListFunc[Context, Param, Filter, Response, Total], options ...Option) router.Controller {
	handler := ListByHandler(fn, options...)

	return router.Routing(func(r router.Router) {
		var path string

		id, ok := ResolveRouteParam[Param](r.GetRelPath())
		if ok { // if has a `param` struct field tag. Check if this parameter is already registered as a path parameter. If not, then use it.
			if !macro.ContainsParam(*r.Macros(), r.GetRelPath(), id.Name) {
				path = id.Path()
			}
		}
		path += "/list"

		r.Handle(method, path, handler)
	})
}

// Index registers a handler for the root path.
// The handler is a function that accepts a context and returns a response.
// The handler function must return a response and an optional error.
func Index[Context ContextType, Response ResponseType](fn ActionFunc[Context, Response], options ...Option) router.Controller {
	handler := IndexHandler(fn, options...)

	return router.Routing(func(r router.Router) {
		r.Get("/", handler)
	})
}

// Job registers a handler for the root path.
// The handler is a function that accepts a context and a request.
// The handler function may return an error.
func Job[Context ContextType, Request RequestType](fn JobFunc[Context, Request], options ...Option) router.Controller {
	handler := JobHandler(fn, options...)

	return router.Routing(func(r router.Router) {
		r.Post("/", handler)
	})
}

// Get registers a GET request handler.
// The handler is a function that accepts a context and returns a response.
func Get[Context ContextType, Param ParamType, Response ResponseType](fn Func[Context, Param, Response], options ...Option) router.Controller {
	return Method("GET", fn, options...)
}

// GetBy registers a GET request handler with a path parameter.
// The path parameter is resolved by the last field with the `param` tag.
// The path parameter is used to create the route path.
// The handler is a function that accepts a context, a parameter, and returns a response.
func GetBy[Context ContextType, Param ParamType, Request RequestType, Response ResponseType](fn ParameterizedFunc[Context, Param, Request, Response], options ...Option) router.Controller {
	return MethodBy("GET", fn, options...)
}

// Create registers a POST request handler.
// The handler is a function that accepts a context and a request.
// The handler function must return a response and an optional error.
// The response is written to the client with a status code of 201.
// The error is written to the client with a call of the HandleError function using the errors subpackage.
func Create[Context ContextType, Request RequestType, Response ResponseType](fn Func[Context, Request, Response], options ...Option) router.Controller {
	// slices.Clone: appending to the caller's variadic slice writes into their
	// backing array whenever cap > len, silently mutating a slice they may reuse.
	opts := append(slices.Clone(options), WithStatusCode(201))
	return Method("POST", fn, opts...)
}

// CreateBy registers a POST request handler with a path parameter.
// The path parameter is resolved by the last field with the `param` tag.
// The path parameter is used to create the route path.
func CreateBy[Context ContextType, Param ParamType, Request RequestType, Response ResponseType](fn ParameterizedFunc[Context, Param, Request, Response], options ...Option) router.Controller {
	opts := append(slices.Clone(options), WithStatusCode(201))
	return MethodBy("POST", fn, opts...)
}

func Delete[Context ContextType, RequestOrParam RequestType | ParamType](fn Func[Context, RequestOrParam, bool], options ...Option) router.Controller {
	return Method("DELETE", fn, options...)
}

func DeleteBy[Context ContextType, Param ParamType, Request RequestType](fn ParameterizedFunc[Context, Param, Request, bool], options ...Option) router.Controller {
	return MethodBy("DELETE", fn, options...)
}

func Update[Context ContextType, RequestOrParam RequestType | ParamType](fn Func[Context, RequestOrParam, bool], options ...Option) router.Controller {
	return Method("PUT", fn, options...)
}

func UpdateBy[Context ContextType, Param ParamType, Request RequestType](fn ParameterizedFunc[Context, Param, Request, bool], options ...Option) router.Controller {
	return MethodBy("PUT", fn, options...)
}

func UpdatePartial[Context ContextType, ID IDType, Request RequestType, Response ResponseType](fn func(Context, ID, Request, ...string) (*Response, bool, error), options ...Option) router.Controller {
	opts := makeOptions(options)

	decodeRequest := getRequestDecodeFunc[Request]()
	isIrisContext := isIrisCtx[Context]() // before serve-time, so this casting doesn't affect the performance. We could use a special .Value of Context but this will cause a bit (very little) performance cost.
	handler := func(ctx *context.Context) {
		if !opts.serveHTTP(ctx) {
			return
		}

		req, err := decodeRequest(ctx)
		if errors.HandleRequestError(ctx, err) {
			return
		}

		onlyColumns := readUpdateColumns(ctx)

		// Note(@kataras): both the presence check and the conversion are required.
		// GetLastEntry reports (zero, false) when the route carries no path
		// parameter, and the macro evaluator's stored type routinely differs from
		// ID (the "uuid" macro stores a uuid.UUID, ID may be a string). A bare
		// idParam.Value.(ID) panicked on a value the client controls - an
		// unrecovered panic per request, i.e. a remote DoS.
		idParam, ok := ctx.Params().GetLastEntry()
		if !ok {
			errors.HandleRequestError(ctx, errors.InvalidArgument)
			return
		}

		id, ok := ConvertPathParamValue[ID](idParam.Value)
		if !ok {
			errors.HandleRequestError(ctx, errors.InvalidArgument)
			return
		}

		resp, updated, err := fn(getCtx[Context](ctx, isIrisContext), id, req, onlyColumns...)
		if resp != nil {
			ctx.OK(resp, err)
			return
		}

		ctx.NoContent(updated, err)
	}

	return router.Routing(func(r router.Router) {
		routeIDParam, _ := ResolveRouteParam[ID](r.GetRelPath())
		// Path() already returns a leading slash; prefixing another produced "//{...}".
		r.Put(routeIDParam.Path(), handler)
	})
}

// listMethods registers List and ListBy for POST (existing clients) and QUERY (RFC 10008).
const listMethods = router.MethodPost + " " + router.MethodQuery

// List registers a list handler on POST and QUERY /list.
func List[Context ContextType, Filter FilterType, Response ResponseType, Total CountType](fn ListFunc[Context, Filter, Response, Total], options ...Option) router.Controller {
	return ListMethod(listMethods, fn, options...)
}

func ListBy[Context ContextType, Param ParamType, Filter FilterType, Response ResponseType, Total CountType](fn ParameterizedListFunc[Context, Param, Filter, Response, Total], options ...Option) router.Controller {
	return ListMethodBy(listMethods, fn, options...)
}

// ResolveRouteParam resolves the route parameter of T for the given relative path,
// reporting whether one was found.
//
// The only good idea I thought to resolve the parameter single or multi by struct,
// is to have a rule that the last field with param tag is the current one that should be used here,
// its name and its type. If it's just a single IDType, then the name is generated.
func ResolveRouteParam[T ParamType](relPath string) (*RouteParam[T], bool) {
	typ := reflex.IndirectType(reflect.TypeFor[*T]())

	isStruct := typ.Kind() == reflect.Struct
	if isStruct { // lookup by last field.
		structFields := reflex.LookupFields(typ, encoding.DefaultPathParamTag)
		if len(structFields) == 0 {
			return nil, false
		}

		lastParamField := structFields[len(structFields)-1]

		paramOptions := strings.Split(lastParamField.Tag.Get(encoding.DefaultPathParamTag), ",")
		paramName := paramOptions[0]
		paramType := getPathParameterType(lastParamField.Type)
		if len(paramOptions) > 1 { // this is optional as the framework tries to auto-resolves its type by go field type above.
			paramType = paramOptions[1]
		}

		id := &RouteParam[T]{
			Name: paramName, // E.g `param:"store_id,uuid"`.
			Type: paramType,
		}
		return id, true
	} // else lookup by type (string, int, ...)
	id := &RouteParam[T]{
		// Name is set on the Configure or WithPathParamName method.
		Type: getPathParameterType(reflect.TypeFor[T]()),
		Name: getRouterPathParameterNameByRelPath((*T)(nil), relPath),
	}

	return id, true
}

// RouteParam is a helper struct that contains the name and type of a path parameter.
type RouteParam[T any] struct {
	Name string
	Type string
}

// Path returns the route path of the path parameter as a string.
func (id *RouteParam[T]) Path() string {
	return fmt.Sprintf("/{%s:%s}", id.Name, id.Type)
}

// Value returns the request value of the path parameter from the request.
func (id *RouteParam[T]) Value(ctx *context.Context) T {
	value, _ := ctx.ValueAs[T](requestParamContextKey)
	return value
}

func getRouterPathParameterNameByRelPath(p any, relPath string) string {
	// Try to make it unique for each router, if the same router is used for different types then it will be a problem.
	// E.g. id_errors_router_main_createrequest_main_createresponse_main_listfiltermain_myservice
	idPathParameterName := fmt.Sprintf("%s_%s",
		"id",
		trimTypeString(reflect.TypeOf(p)),
	)
	if relPath != "" {
		idPathParameterName += reflex.PathParamReplacer.Replace(relPath)
	}

	return idPathParameterName
}

// getPathParameterType maps a Go type to the route macro that should validate it.
//
// Note(@kataras): it takes a reflect.Type, never a value. The previous
// value-based form could not work from either call site: reflect.Indirect of a
// nil pointer yields the zero Value (Kind Invalid), and a reflect.Type passed as
// an "any" reflects as a *rtype pointer (Kind Struct after Indirect). Both fell
// through to the "uuid" default, so an int/int64/uint64 id registered as
// "{id:uuid}" and every request to it 404'd, with no error at boot.
func getPathParameterType(typ reflect.Type) string {
	switch reflex.IndirectType(typ).Kind() {
	case reflect.Int:
		return "int"
	case reflect.Int64:
		return "int64"
	case reflect.Uint64:
		return "uint64"
	case reflect.String:
		// return "string"
		fallthrough
	default:
		return "uuid"
	}
}

var typeReplacer = strings.NewReplacer(".", "_", "*", "", "[", "_", "]", "_", ",", "_")

func trimTypeString(t reflect.Type) string {
	typeString := strings.ReplaceAll(t.String(), "*", "")
	return strings.ToLower(strings.Trim(typeReplacer.Replace(typeString), "_"))
}
