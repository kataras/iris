package router

import (
	"fmt"
	"strings"

	"github.com/kataras/iris/context"
	"github.com/kataras/iris/macro"
	"github.com/kataras/iris/macro/handler"
)

// Route contains the information about a registered Route.
// If any of the following fields are changed then the
// caller should Refresh the router.
type Route struct {
	Name       string         `json:"name"`   // "userRoute"
	Method     string         `json:"method"` // "GET"
	methodBckp string         // if Method changed to something else (which is possible at runtime as well, via RefreshRouter) then this field will be filled with the old one.
	Subdomain  string         `json:"subdomain"` // "admin."
	tmpl       macro.Template // Tmpl().Src: "/api/user/{id:uint64}"
	// temp storage, they're appended to the Handlers on build.
	// Execution happens before Handlers, can be empty.
	beginHandlers context.Handlers
	// Handlers are the main route's handlers, executed by order.
	// Cannot be empty.
	Handlers        context.Handlers `json:"-"`
	MainHandlerName string           `json:"mainHandlerName"`
	// temp storage, they're appended to the Handlers on build.
	// Execution happens after Begin and main Handler(s), can be empty.
	doneHandlers context.Handlers
	Path         string `json:"path"` // "/api/user/:id"
	// FormattedPath all dynamic named parameters (if any) replaced with %v,
	// used by Application to validate param values of a Route based on its name.
	FormattedPath string `json:"formattedPath"`
}

// NewRoute returns a new route based on its method,
// subdomain, the path (unparsed or original),
// handlers and the macro container which all routes should share.
// It parses the path based on the "macros",
// handlers are being changed to validate the macros at serve time, if needed.
func NewRoute(method, subdomain, unparsedPath, mainHandlerName string,
	handlers context.Handlers, macros macro.Macros) (*Route, error) {

	tmpl, err := macro.Parse(unparsedPath, macros)
	if err != nil {
		return nil, err
	}

	path := convertMacroTmplToNodePath(tmpl)
	// prepend the macro handler to the route, now,
	// right before the register to the tree, so APIBuilder#UseGlobal will work as expected.
	if handler.CanMakeHandler(tmpl) {
		macroEvaluatorHandler := handler.MakeHandler(tmpl)
		handlers = append(context.Handlers{macroEvaluatorHandler}, handlers...)
	}

	path = cleanPath(path) // maybe unnecessary here but who cares in this moment
	defaultName := method + subdomain + tmpl.Src
	formattedPath := formatPath(path)

	route := &Route{
		Name:            defaultName,
		Method:          method,
		methodBckp:      method,
		Subdomain:       subdomain,
		tmpl:            tmpl,
		Path:            path,
		Handlers:        handlers,
		MainHandlerName: mainHandlerName,
		FormattedPath:   formattedPath,
	}
	return route, nil
}

// use adds explicit begin handlers(middleware) to this route,
// It's being called internally, it's useless for outsiders
// because `Handlers` field is exported.
// The callers of this function are: `APIBuilder#UseGlobal` and `APIBuilder#Done`.
//
// BuildHandlers should be called to build the route's `Handlers`.
func (r *Route) use(handlers context.Handlers) {
	if len(handlers) == 0 {
		return
	}
	r.beginHandlers = append(r.beginHandlers, handlers...)
}

// use adds explicit done handlers to this route.
// It's being called internally, it's useless for outsiders
// because `Handlers` field is exported.
// The callers of this function are: `APIBuilder#UseGlobal` and `APIBuilder#Done`.
//
// BuildHandlers should be called to build the route's `Handlers`.
func (r *Route) done(handlers context.Handlers) {
	if len(handlers) == 0 {
		return
	}
	r.doneHandlers = append(r.doneHandlers, handlers...)
}

// ChangeMethod will try to change the HTTP Method of this route instance.
// A call of `RefreshRouter` is required after this type of change in order to change to be really applied.
func (r *Route) ChangeMethod(newMethod string) bool {
	if newMethod != r.Method {
		r.methodBckp = r.Method
		r.Method = newMethod
		return true
	}

	return false
}

// SetStatusOffline will try make this route unavailable.
// A call of `RefreshRouter` is required after this type of change in order to change to be really applied.
func (r *Route) SetStatusOffline() bool {
	return r.ChangeMethod(MethodNone)
}

// RestoreStatus will try to restore the status of this route instance, i.e if `SetStatusOffline` called on a "GET" route,
// then this function will make this route available with "GET" HTTP Method.
// Note if that you want to set status online for an offline registered route then you should call the `ChangeMethod` instead.
// It will return true if the status restored, otherwise false.
// A call of `RefreshRouter` is required after this type of change in order to change to be really applied.
func (r *Route) RestoreStatus() bool {
	return r.ChangeMethod(r.methodBckp)
}

// BuildHandlers is executed automatically by the router handler
// at the `Application#Build` state. Do not call it manually, unless
// you were defined your own request mux handler.
func (r *Route) BuildHandlers() {
	if len(r.beginHandlers) > 0 {
		r.Handlers = append(r.beginHandlers, r.Handlers...)
		r.beginHandlers = r.beginHandlers[0:0]
	}

	if len(r.doneHandlers) > 0 {
		r.Handlers = append(r.Handlers, r.doneHandlers...)
		r.doneHandlers = r.doneHandlers[0:0]
	} // note: no mutex needed, this should be called in-sync when server is not running of course.
}

// String returns the form of METHOD, SUBDOMAIN, TMPL PATH.
func (r Route) String() string {
	return fmt.Sprintf("%s %s%s",
		r.Method, r.Subdomain, r.Tmpl().Src)
}

// Tmpl returns the path template,
// it contains the parsed template
// for the route's path.
// May contain zero named parameters.
//
// Developer can get his registered path
// via Tmpl().Src, Route.Path is the path
// converted to match the underline router's specs.
func (r Route) Tmpl() macro.Template {
	return r.tmpl
}

// RegisteredHandlersLen returns the end-developer's registered handlers, all except the macro evaluator handler
// if was required by the build process.
func (r Route) RegisteredHandlersLen() int {
	n := len(r.Handlers)
	if handler.CanMakeHandler(r.tmpl) {
		n--
	}

	return n
}

// IsOnline returns true if the route is marked as "online" (state).
func (r Route) IsOnline() bool {
	return r.Method != MethodNone
}

// formats the parsed to the underline path syntax.
// path = "/api/users/:id"
// return "/api/users/%v"
//
// path = "/files/*file"
// return /files/%v
//
// path = "/:username/messages/:messageid"
// return "/%v/messages/%v"
// we don't care about performance here, it's prelisten.
func formatPath(path string) string {
	if strings.Contains(path, ParamStart) || strings.Contains(path, WildcardParamStart) {
		var (
			startRune         = ParamStart[0]
			wildcardStartRune = WildcardParamStart[0]
		)

		var formattedParts []string
		parts := strings.Split(path, "/")
		for _, part := range parts {
			if len(part) == 0 {
				continue
			}
			if part[0] == startRune || part[0] == wildcardStartRune {
				// is param or wildcard param
				part = "%v"
			}
			formattedParts = append(formattedParts, part)
		}

		return "/" + strings.Join(formattedParts, "/")
	}
	// the whole path is static just return it
	return path
}

// StaticPath returns the static part of the original, registered route path.
// if /user/{id} it will return /user
// if /user/{id}/friend/{friendid:uint64} it will return /user too
// if /assets/{filepath:path} it will return /assets.
func (r Route) StaticPath() string {
	src := r.tmpl.Src
	bidx := strings.IndexByte(src, '{')
	if bidx == -1 || len(src) <= bidx {
		return src // no dynamic part found
	}
	if bidx == 0 { // found at first index,
		// but never happens because of the prepended slash
		return "/"
	}

	return src[:bidx]
}

// ResolvePath returns the formatted path's %v replaced with the args.
func (r Route) ResolvePath(args ...string) string {
	rpath, formattedPath := r.Path, r.FormattedPath
	if rpath == formattedPath {
		// static, no need to pass args
		return rpath
	}
	// check if we have /*, if yes then join all arguments to one as path and pass that as parameter
	if rpath[len(rpath)-1] == WildcardParamStart[0] {
		parameter := strings.Join(args, "/")
		return fmt.Sprintf(formattedPath, parameter)
	}
	// else return the formattedPath with its args,
	// the order matters.
	for _, s := range args {
		formattedPath = strings.Replace(formattedPath, "%v", s, 1)
	}
	return formattedPath
}

// Trace returns some debug infos as a string sentence.
// Should be called after Build.
func (r Route) Trace() string {
	printfmt := fmt.Sprintf("%s:", r.Method)
	if r.Subdomain != "" {
		printfmt += fmt.Sprintf(" %s", r.Subdomain)
	}
	printfmt += fmt.Sprintf(" %s ", r.Tmpl().Src)

	if l := r.RegisteredHandlersLen(); l > 1 {
		printfmt += fmt.Sprintf("-> %s() and %d more", r.MainHandlerName, l-1)
	} else {
		printfmt += fmt.Sprintf("-> %s()", r.MainHandlerName)
	}

	// printfmt := fmt.Sprintf("%s: %s >> %s", r.Method, r.Subdomain+r.Tmpl().Src, r.MainHandlerName)
	// if l := len(r.Handlers); l > 0 {
	// 	printfmt += fmt.Sprintf(" and %d more", l)
	// }
	return printfmt // without new line.
}

type routeReadOnlyWrapper struct {
	*Route
}

func (rd routeReadOnlyWrapper) Method() string {
	return rd.Route.Method
}

func (rd routeReadOnlyWrapper) Name() string {
	return rd.Route.Name
}

func (rd routeReadOnlyWrapper) Subdomain() string {
	return rd.Route.Subdomain
}

func (rd routeReadOnlyWrapper) Path() string {
	return rd.Route.tmpl.Src
}

func (rd routeReadOnlyWrapper) Trace() string {
	return rd.Route.Trace()
}

func (rd routeReadOnlyWrapper) Tmpl() macro.Template {
	return rd.Route.Tmpl()
}

func (rd routeReadOnlyWrapper) MainHandlerName() string {
	return rd.Route.MainHandlerName
}
