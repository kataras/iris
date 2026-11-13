package router

import (
	"fmt"
	"io"
	"maps"
	"net/http"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/kataras/iris/v14/context"
	"github.com/kataras/iris/v14/core/router/macro"
	"github.com/kataras/iris/v14/core/router/macro/handler"

	"github.com/kataras/golog/printer"
)

// Route contains the information about a registered Route.
// If any of the following fields are changed then the
// caller should Refresh the router.
type Route struct {
	router      *RouterBuilder // the router that this route belongs to.
	Properties  context.Map    // Properties (map) coming from the Router this route belongs to.
	Title       string         `json:"title"`       // custom name to replace the method on debug logging.
	Name        string         `json:"name"`        // "userRoute"
	Description string         `json:"description"` // "lists a user"
	Method      string         `json:"method"`      // "GET"
	Subdomain   string         `json:"subdomain"`   // "admin."
	tmpl        macro.Template // Tmpl().Src: "/api/user/{id:uint64}"
	// temp storage, they're appended to the Handlers on build.
	// Execution happens before Handlers, can be empty.
	// They run right after any builtinBeginHandlers.
	relativePattern string // it's the relative pattern-path, i.e "/api/user/{id}".

	// temp storage, these are always registered first as Handlers on Build.
	// There are the handlers may be added by the framework and
	// can NOT be modified by the end-developer (i.e overlapRoute & bindMultiParamTypesHandler),
	// even if a function like Use is used.
	builtinBeginHandlerList *context.HandlerList `json:"-"`

	// MiddlewareList contains the middleware handlers as they given by the end-developer (router builder).
	MiddlewareList *context.HandlerList `json:"middleware_list"`
	// MainHandlerList contains the main handlers as they given by the end-developer (router builder).
	MainHandlerList *context.HandlerList `json:"main_handler_list"`

	// HandlerList contains the middleware and main handlers.
	// They are built after BuildHandlers method (except newEmptyRoute).
	HandlerList *context.HandlerList `json:"handler_list"`

	// Handlers are the main route's handlers, executed by order.
	// Cannot be empty. Empty until built by the BuildHandlers method.
	Handlers context.Handlers `json:"-"`

	// Path is the registered path part (without subdomain), i.e "/api/user/{id:uint64}".
	// Shortcut of Tmpl().Src.
	Path string `json:"path"` // the registered path, i.e "/api/user/{id:uuid}"

	// NodePath is the underline router's representation, i.e "/api/user/:id".
	NodePath string `json:"node_path"` // the underline router's representation, i.e "/api/user/:id"
	// FormattedPath all dynamic named parameters (if any) replaced with %v,
	// used by Application to validate param values of a Route based on its name.
	FormattedPath string `json:"formatted_path"`

	// Initialized on buildHandlers and used for the trace.
	MainHandlerName  string `json:"main_handler_name"`
	MainHandlerIndex int    `json:"main_handler_index"`

	// the source code's filename:filenumber that this route was registered on.
	RegisterFileName   string `json:"register_filename"`
	RegisterLineNumber int    `json:"register_line_number"`

	// see RouterBuilder.handle, routerHandler.bindMultiParamTypesHandler and routerHandler.Build,
	// it's the parent route of the last registered of the same path parameter. Specifically for path parameters.
	topLink *Route
	// overlappedLink specifically for overlapRoute feature.
	// keeps the second route of the same path pattern registered.
	// It's used ONLY for logging.
	overlappedLink *Route
	// allowedOnly reports that the route was created for a method of the
	// Router's AllowMethods and not for a method the caller registered.
	allowedOnly bool

	// Sitemap properties: https://www.sitemaps.org/protocol.html
	NoSitemap  bool      // when this route should be hidden from sitemap.
	LastMod    time.Time `json:"last_mod,omitempty"`
	ChangeFreq string    `json:"change_freq,omitempty"`
	Priority   float32   `json:"priority,omitzero"`

	// OnBuild runs right before BuildHandlers.
	OnBuild func(r *Route)
	NoLog   bool // disables debug logging for this route.
}

var _ context.RouteReadOnly = &Route{}

// NewRoute returns a new route based on its method,
// subdomain, the path (unparsed or original),
// handlers and the macro container which all routes should share.
// It parses the path based on the "macros",
// handlers are being changed to validate the macros at serve time, if needed.
func NewRoute(router *RouterBuilder,
	method, relativePath, pattern string,
	middlewareList, mainHandlersList *context.HandlerList /* they are clones */) (*Route, error) {

	fullpath := relativePath + pattern // for now, keep the trailing "/" if any, it will be clenaned by the cleanPath below.
	// separate the subdomain and relative nodePath.
	subdomain, nodePath := SplitSubdomainAndPath(fullpath)
	// required. Before macro template parse as the cleanPath does not modify the dynamic path route parts.
	nodePath = CleanPath(nodePath)

	tmpl, err := macro.Parse(nodePath, *router.Macros())
	if err != nil {
		return nil, err
	}

	nodePath = convertMacroTmplToNodePath(tmpl)
	// prepend the macro handler to the route, now,
	// right before the register to the tree, so RouterBuilder#Use will work as expected.
	if handler.CanMakeHandler(tmpl) {
		macroEvaluatorHandler := handler.MakeHandler(tmpl)
		middlewareList.Prepend(macroEvaluatorHandler)
	}

	defaultName := method + subdomain + tmpl.Src
	formattedPath := formatPath(nodePath)

	route := &Route{
		router:                  router,
		Properties:              maps.Clone(router.Properties()), // Clone in order to make sure this isn't modified by us at serve-time.
		Name:                    defaultName,
		Method:                  method,
		Subdomain:               subdomain,
		tmpl:                    tmpl,
		Path:                    tmpl.Src,
		NodePath:                nodePath,
		builtinBeginHandlerList: context.NewHandlerList(),
		MiddlewareList:          middlewareList,
		MainHandlerList:         mainHandlersList,
		FormattedPath:           formattedPath,
		relativePattern:         pattern,
	}

	return route, nil
}

func newEmptyRoute(router *RouterBuilder, name string, handlerList *context.HandlerList) *Route {
	return &Route{
		router:                  router,
		Name:                    name,
		Method:                  MethodNone,
		Subdomain:               "",
		tmpl:                    macro.Template{},
		relativePattern:         "",
		builtinBeginHandlerList: context.NewHandlerList(),
		topLink:                 nil,
		overlappedLink:          nil,
		Properties:              nil,
		Title:                   "",
		Description:             "",
		MiddlewareList:          nil,
		MainHandlerList:         handlerList,
		HandlerList:             handlerList,
		// The newEmptyRoute need to call the BuildHandlers() too, so we don't set the final handlers chain here.
		// Handlers:                handlerList.Values(),
		// MainHandlerName:         handlerList.Last().Name,
		// MainHandlerIndex:        handlerList.Count() - 1,
		Path:     "/",
		NodePath: "/",
		NoLog:    true,
	}
}

// IsNoRoute reports whether this route is a no-route (404, fired by the serve mux).
// Access of the not found path is gain by `ctx.Path()`.
func (r *Route) IsNoRoute() bool {
	return r.Name == NoRouteName
}

// IsMethodNotAllowed reports whether this route is a no-method (405, method not allowed, fired by the serve mux).
// Access of the forbidden request method is gain by `ctx.Method()` and allowed methods through "Allow" response header.
func (r *Route) IsMethodNotAllowed() bool {
	return r.Name == NoMethodName
}

// IsPreRouting reports whether this route is running before the main serve mux.
// Means that this Route's Handlers are running under `UseRouter`.
func (r *Route) IsPreRouting() bool {
	return r.Name == PreRoutingName
}

// IsRoute reports whether this route is a registered, end-developer route
// (not a built-in 404 or 405 one, and not nil).
// Note that the pre-routing slot reports true as well, see `IsPreRouting`.
func (r *Route) IsRoute() bool {
	return r != nil && !r.IsNoRoute() && !r.IsMethodNotAllowed()
}

// SetSourceByMethodType sets the route's source caller based on the method's type and name.
func (r *Route) SetSourceByMethodType(typ reflect.Type, methodName string) {
	if r == nil {
		return
	}

	if last := r.MainHandlerList.Last(); last != nil {
		last.SetNameFileAndLineByMethod(typ, methodName)
	}
}

// SetCaller sets the route's source caller, useful for debugging.
func (r *Route) SetCaller(fileName string, fileLine int) {
	r.RegisterFileName = fileName
	r.RegisterLineNumber = fileLine
}

// Describe sets the route's description
// that will be logged alongside with the route information
// in DEBUG log level.
// Returns the `Route` itself.
func (r *Route) Describe(description string) *Route {
	r.Description = description
	return r
}

// BuildHandlers is executed automatically by the router handler
// at the `Application#Build` state. Do not call it manually, unless
// you were defined your own request mux handler.
func (r *Route) BuildHandlers() {
	if r.OnBuild != nil {
		r.OnBuild(r)
	}

	if r.MainHandlerName == "" {
		if last := r.MainHandlerList.Last(); last != nil {
			r.MainHandlerName = last.Name
			r.MainHandlerIndex = r.MainHandlerList.Count() - 1 // last index.
		}
	}

	allRouteHandlers := context.ConcatHandlerLists(r.builtinBeginHandlerList.Clone(), r.MiddlewareList, r.MainHandlerList)
	r.HandlerList = allRouteHandlers
	r.Handlers = allRouteHandlers.Values()
	// reset the temp storage of builtin begin handlers, so a second call of
	// BuildHandlers will not re-add them (it shouldn't happen).
	r.builtinBeginHandlerList.Reset()
}

// String returns the form of METHOD, SUBDOMAIN, TMPL PATH.
func (r *Route) String() string {
	start := r.GetTitle()
	// if r.StatusCode > 0 {
	// 	start = fmt.Sprintf("%d (%s)", r.StatusCode, http.StatusText(r.StatusCode))
	// }

	return fmt.Sprintf("%s %s%s",
		start, r.Subdomain, r.Tmpl().Src)
}

// Equal compares the method, subdomain and the
// underline representation of the route's path,
// instead of the `String` function which returns the front representation.
func (r *Route) Equal(other *Route) bool {
	return r.Method == other.Method && r.Subdomain == other.Subdomain && r.NodePath == other.NodePath
}

// DeepEqual compares the method, subdomain, the
// underline representation of the route's path,
// and the template source.
func (r *Route) DeepEqual(other *Route) bool {
	return r.Equal(other) && r.tmpl.Src == other.tmpl.Src
}

// GetMainHandlerName returns the name of the route's main handler,
// that is the last handler registered in the MainHandlerList.
func (r *Route) GetMainHandlerName() string {
	return r.MainHandlerName
}

// GetMainHandlerIndex returns the index of the route's main handler
// inside the MainHandlerList, that is the index of its last handler.
func (r *Route) GetMainHandlerIndex() int {
	return r.MainHandlerIndex
}

// GetHandlerList returns the route's handler list.
// It contains all handlers, middleware and main handlers.
// It's available only after BuildHandlers method.
func (r *Route) GetHandlerList() *context.HandlerList {
	return r.HandlerList
}

// GetRegisterFileName returns the file name that this route was registered on.
func (r *Route) GetRegisterFileName() string {
	return r.RegisterFileName
}

// GetRegisterLineNumber returns the line number that this route was registered on.
func (r *Route) GetRegisterLineNumber() int {
	return r.RegisterLineNumber
}

// GetProperty returns a specific property based on its "key"
// of this route's Router owner.
func (r *Route) GetProperty(key string) (any, bool) {
	if len(r.Properties) == 0 {
		return nil, false
	}

	property, ok := r.Properties[key]
	return property, ok
}

// GetMethod returns the route's method.
func (r *Route) GetMethod() string {
	return r.Method
}

// GetSubdomain returns the route's subdomain.
func (r *Route) GetSubdomain() string {
	return r.Subdomain
}

// GetPath returns the route's original registered path.
func (r *Route) GetPath() string {
	return r.Path
}

// SetName overrides the default route name which defaults to
// method + subdomain + path and
// statusErrorCode_method+subdomain+path for error routes.
//
// Note that the route name MUST BE unique per Iris Application.
func (r *Route) SetName(newRouteName string) *Route {
	r.Name = newRouteName
	return r
}

// GetName returns the route's name.
func (r *Route) GetName() string {
	return r.Name
}

// Router returns the read-only router that this route belongs to.
func (r *Route) Router() context.Router {
	return r.router
}

// ExcludeSitemap excludes this route page from sitemap generator.
// It sets the NoSitemap field to true.
//
// See `SetLastMod`, `SetChangeFreq`, `SetPriority` methods
// and the `sitemap.New` controller of the
// github.com/kataras/iris/v14/controller/sitemap package too.
func (r *Route) ExcludeSitemap() *Route {
	r.NoSitemap = true
	return r
}

// SetLastMod sets the date of last modification of the file served by this static GET route.
func (r *Route) SetLastMod(t time.Time) *Route {
	r.LastMod = t
	return r
}

// GetLastMod returns the date of last modification of the file served by this static GET route.
func (r *Route) GetLastMod() time.Time {
	return r.LastMod
}

// SetChangeFreq sets how frequently this static GET route's page is likely to change,
// possible values:
// - "always"
// - "hourly"
// - "daily"
// - "weekly"
// - "monthly"
// - "yearly"
// - "never"
func (r *Route) SetChangeFreq(freq string) *Route {
	r.ChangeFreq = freq
	return r
}

// GetChangeFreq returns the the page frequently is likely to change.
func (r *Route) GetChangeFreq() string {
	return r.ChangeFreq
}

// SetPriority sets the priority of this static GET route's URL relative to other URLs on your site.
func (r *Route) SetPriority(prio float32) *Route {
	r.Priority = prio
	return r
}

// GetPriority returns the priority of this route's URL relative to other URLs on your site.
func (r *Route) GetPriority() float32 {
	return r.Priority
}

// Tmpl returns the path template,
// it contains the parsed template
// for the route's path.
// May contain zero named parameters.
//
// Developer can get his registered path
// via Tmpl().Src, Route.Path is the path
// converted to match the underline router's specs.
func (r *Route) Tmpl() macro.Template {
	return r.tmpl
}

// RegisteredHandlersLen returns the end-developer's registered handlers, all except the macro evaluator handler
// if was required by the build process.
func (r *Route) RegisteredHandlersLen() int {
	n := r.MiddlewareList.Count() + r.HandlerList.Count()
	if handler.CanMakeHandler(r.tmpl) {
		n--
	}

	return n
}

// IsOnline returns true if the route is marked as "online" (state).
func (r *Route) IsOnline() bool {
	return r.Method != MethodNone
}

// IsSitemapDisabled reports whether this route is excluded from the sitemap.
func (r *Route) IsSitemapDisabled() bool {
	return r.NoSitemap
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
			startRune         = paramStartCharacter
			wildcardStartRune = wildcardParamStartCharacter
		)

		var formattedParts []string
		parts := strings.SplitSeq(path, "/")
		for part := range parts {
			if part == "" {
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

// IsStatic reports whether this route is a static route.
// Does not contain dynamic path parameters,
// is online and registered on GET HTTP Method.
func (r *Route) IsStatic() bool {
	return r.IsOnline() && len(r.Tmpl().Params) == 0 && r.Method == "GET"
}

// StaticPath returns the static part of the original, registered route path.
// if /user/{id} it will return /user
// if /user/{id}/friend/{friendid:uint64} it will return /user too
// if /assets/{filepath:path} it will return /assets.
func (r *Route) StaticPath() string {
	src := r.tmpl.Src
	return staticPath(src)
}

// ResolvePath returns the formatted path's %v replaced with the args.
func (r *Route) ResolvePath(args ...string) string {
	rpath, formattedPath := r.NodePath, r.FormattedPath
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

func traceHandlerFile(title, name, line string, number int) string {
	file := fmt.Sprintf("(%s:%d)", filepath.ToSlash(line), number)

	if context.IgnoreHandlerName(name) {
		return ""
	}

	space := strings.Repeat(" ", len(title)+1)
	return fmt.Sprintf("\n%s • %s %s", space, name, file)
}

var methodColors = map[string]int{
	http.MethodGet:     32,
	http.MethodPost:    35,
	http.MethodPut:     34,
	http.MethodDelete:  31,
	http.MethodConnect: 32,
	http.MethodHead:    23,
	http.MethodPatch:   34,
	http.MethodOptions: 37,
	http.MethodTrace:   33,
	MethodQuery:        36,
	MethodNone:         203, // orange-red.
}

// TraceTitleColorCode returns the color code depending on the method or the status.
func TraceTitleColorCode(method string) int {
	if color, ok := methodColors[method]; ok {
		return color
	}

	return 131 // for error handlers, of "ERROR [%STATUSCODE]"
}

// GetTitle returns the custom Title or the method.
func (r *Route) GetTitle() string {
	title := r.Title
	if title == "" {
		title = r.Method // else is its method, e.g. GET
	}

	return title
}

// Trace prints some debug info about the Route to the "w".
// Should be called after `Build` state.
//
// It prints the @method: @path (@description) (@route_rel_location)
//   - @handler_name (@handler_rel_location)
//   - @second_handler ...
//
// If route and handler line:number locations are equal then the second is ignored.
func (r *Route) Trace(w io.Writer, stoppedIndex int) {
	title := r.GetTitle()

	// Color the method.
	color := TraceTitleColorCode(title)

	// @method: @path
	// space := strings.Repeat(" ", len(http.MethodConnect)-len(method))
	// s := fmt.Sprintf("%s: %s", printer.Rich(title, color), path)
	printer.WriteRich(w, title, color)

	path := r.tmpl.Src
	if path == "" {
		path = "/"
	}

	fmt.Fprintf(w, ": %s", path)

	// (@description)
	description := r.Description
	if description == "" {
		if title == MethodNone {
			description = "offline"
		}

		if subdomain := r.Subdomain; subdomain != "" {
			if subdomain == "*." { // wildcard.
				subdomain = "subdomain"
			}

			if description == "offline" {
				description += ", "
			}

			description += subdomain
		}
	}

	if description != "" {
		// s += fmt.Sprintf(" %s", printer.Rich(description, printer.Cyan, printer.Underline))
		fmt.Fprint(w, " ")
		printer.WriteRich(w, description, printer.Cyan, printer.Underline)
	}

	// (@route_rel_location)
	// s += fmt.Sprintf(" (%s:%d)", r.RegisterFileName, r.RegisterLineNumber)
	fmt.Fprintf(w, " (%s:%d)", r.RegisterFileName, r.RegisterLineNumber)

	infos := r.HandlerList.List() // this is why it must be called after Build, otherwise concat the MiddlewareList and MainHandlerList, we must separate main handler list to modify the handler info (e.g. errors/router).
	for i, info := range infos {
		// * @handler_name (@handler_rel_location)
		fmt.Fprint(w, traceHandlerFile(title, info.Name, info.File, info.Line))
		if stoppedIndex != -1 && stoppedIndex <= len(infos) {
			if i <= stoppedIndex {
				printer.WriteRich(w, " ✓", printer.Green)
				// } else {
				// printer.WriteRich(w, " ✕", printer.Red, printer.Underline)
			}
		}
	}

	fmt.Fprintln(w)

	if r.overlappedLink != nil {
		bckpDesc := r.overlappedLink.Description
		r.overlappedLink.Description += " (overlapped)"
		r.overlappedLink.Trace(w, -1)
		r.overlappedLink.Description = bckpDesc
	}
}
