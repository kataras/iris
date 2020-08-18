package router

import (
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/kataras/iris/v12/context"
	"github.com/kataras/iris/v12/macro"
	"github.com/kataras/iris/v12/macro/handler"

	"github.com/kataras/pio"
)

// Route contains the information about a registered Route.
// If any of the following fields are changed then the
// caller should Refresh the router.
type Route struct {
	Name        string         `json:"name"`        // "userRoute"
	Description string         `json:"description"` // "lists a user"
	Method      string         `json:"method"`      // "GET"
	StatusCode  int            `json:"statusCode"`  // 404 (only for HTTP error handlers).
	methodBckp  string         // if Method changed to something else (which is possible at runtime as well, via RefreshRouter) then this field will be filled with the old one.
	Subdomain   string         `json:"subdomain"` // "admin."
	tmpl        macro.Template // Tmpl().Src: "/api/user/{id:uint64}"
	// temp storage, they're appended to the Handlers on build.
	// Execution happens before Handlers, can be empty.
	beginHandlers context.Handlers
	// Handlers are the main route's handlers, executed by order.
	// Cannot be empty.
	Handlers         context.Handlers `json:"-"`
	MainHandlerName  string           `json:"mainHandlerName"`
	MainHandlerIndex int              `json:"mainHandlerIndex"`
	// temp storage, they're appended to the Handlers on build.
	// Execution happens after Begin and main Handler(s), can be empty.
	doneHandlers context.Handlers

	Path string `json:"path"` // the underline router's representation, i.e "/api/user/:id"
	// FormattedPath all dynamic named parameters (if any) replaced with %v,
	// used by Application to validate param values of a Route based on its name.
	FormattedPath string `json:"formattedPath"`

	// the source code's filename:filenumber that this route was created from.
	SourceFileName   string `json:"sourceFileName"`
	SourceLineNumber int    `json:"sourceLineNumber"`

	// where the route registered.
	RegisterFileName   string `json:"registerFileName"`
	RegisterLineNumber int    `json:"registerLineNumber"`

	topLink *Route

	// Sitemap properties: https://www.sitemaps.org/protocol.html
	LastMod    time.Time `json:"lastMod,omitempty"`
	ChangeFreq string    `json:"changeFreq,omitempty"`
	Priority   float32   `json:"priority,omitempty"`

	// ReadOnly is the read-only structure of the Route.
	ReadOnly context.RouteReadOnly

	// OnBuild runs right before BuildHandlers.
	OnBuild func(r *Route)
}

// NewRoute returns a new route based on its method,
// subdomain, the path (unparsed or original),
// handlers and the macro container which all routes should share.
// It parses the path based on the "macros",
// handlers are being changed to validate the macros at serve time, if needed.
func NewRoute(statusErrorCode int, method, subdomain, unparsedPath string,
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

	path = cleanPath(path) // maybe unnecessary here.
	defaultName := method + subdomain + tmpl.Src
	if statusErrorCode > 0 {
		defaultName = fmt.Sprintf("%d_%s", statusErrorCode, defaultName)
	}

	formattedPath := formatPath(path)

	route := &Route{
		StatusCode:    statusErrorCode,
		Name:          defaultName,
		Method:        method,
		methodBckp:    method,
		Subdomain:     subdomain,
		tmpl:          tmpl,
		Path:          path,
		Handlers:      handlers,
		FormattedPath: formattedPath,
	}

	route.ReadOnly = routeReadOnlyWrapper{route}
	return route, nil
}

// Use adds explicit begin handlers to this route.
// Alternatively the end-dev can prepend to the `Handlers` field.
// Should be used before the `BuildHandlers` which is
// called by the framework itself on `Application#Run` (build state).
//
// Used internally at  `APIBuilder#UseGlobal` -> `beginGlobalHandlers` -> `APIBuilder#Handle`.
func (r *Route) Use(handlers ...context.Handler) {
	if len(handlers) == 0 {
		return
	}
	r.beginHandlers = append(r.beginHandlers, handlers...)
}

// Done adds explicit finish handlers to this route.
// Alternatively the end-dev can append to the `Handlers` field.
// Should be used before the `BuildHandlers` which is
// called by the framework itself on `Application#Run` (build state).
//
// Used internally at  `APIBuilder#DoneGlobal` -> `doneGlobalHandlers` -> `APIBuilder#Handle`.
func (r *Route) Done(handlers ...context.Handler) {
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

// Describe sets the route's description
// that will be logged alongside with the route information
// in DEBUG log level.
// Returns the `Route` itself.
func (r *Route) Describe(description string) *Route {
	r.Description = description
	return r
}

// SetSourceLine sets the route's source caller, useful for debugging.
// Returns the `Route` itself.
func (r *Route) SetSourceLine(fileName string, lineNumber int) *Route {
	r.SourceFileName = fileName
	r.SourceLineNumber = lineNumber
	return r
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
	if r.OnBuild != nil {
		r.OnBuild(r)
	}

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
func (r *Route) String() string {
	start := r.Method
	if r.StatusCode > 0 {
		start = http.StatusText(r.StatusCode)
	}

	return fmt.Sprintf("%s %s%s",
		start, r.Subdomain, r.Tmpl().Src)
}

// Equal compares the method, subdomain and the
// underline representation of the route's path,
// instead of the `String` function which returns the front representation.
func (r *Route) Equal(other *Route) bool {
	return r.StatusCode == other.StatusCode && r.Method == other.Method && r.Subdomain == other.Subdomain && r.Path == other.Path
}

// DeepEqual compares the method, subdomain, the
// underline representation of the route's path,
// and the template source.
func (r *Route) DeepEqual(other *Route) bool {
	return r.Equal(other) && r.tmpl.Src == other.tmpl.Src
}

// SetLastMod sets the date of last modification of the file served by this static GET route.
func (r *Route) SetLastMod(t time.Time) *Route {
	r.LastMod = t
	return r
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

// SetPriority sets the priority of this static GET route's URL relative to other URLs on your site.
func (r *Route) SetPriority(prio float32) *Route {
	r.Priority = prio
	return r
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
	n := len(r.Handlers)
	if handler.CanMakeHandler(r.tmpl) {
		n--
	}

	return n
}

// IsOnline returns true if the route is marked as "online" (state).
func (r *Route) IsOnline() bool {
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

func traceHandlerFile(method, name, line string, number int) string {
	file := fmt.Sprintf("(%s:%d)", filepath.ToSlash(line), number)

	if context.IgnoreHandlerName(name) {
		return ""
	}

	space := strings.Repeat(" ", len(method)+1)
	return fmt.Sprintf("\n%s • %s %s", space, name, file)
}

var methodColors = map[string]int{
	http.MethodGet:     pio.Green,
	http.MethodPost:    pio.Magenta,
	http.MethodPut:     pio.Blue,
	http.MethodDelete:  pio.Red,
	http.MethodConnect: pio.Green,
	http.MethodHead:    23,
	http.MethodPatch:   pio.Blue,
	http.MethodOptions: pio.Gray,
	http.MethodTrace:   pio.Yellow,
	MethodNone:         203, // orange-red.
}

func traceMethodColor(method string) int {
	if color, ok := methodColors[method]; ok {
		return color
	}

	return 131 // for error handlers, of "ERROR [%STATUSCODE]"
}

// Trace prints some debug info about the Route to the "w".
// Should be called after `Build` state.
//
// It prints the @method: @path (@description) (@route_rel_location)
//               * @handler_name (@handler_rel_location)
//               * @second_handler ...
// If route and handler line:number locations are equal then the second is ignored.
func (r *Route) Trace(w io.Writer, stoppedIndex int) {
	method := r.Method
	if method == "" {
		method = fmt.Sprintf("%d", r.StatusCode)
	}

	// Color the method.
	color := traceMethodColor(method)

	// @method: @path
	// space := strings.Repeat(" ", len(http.MethodConnect)-len(method))
	// s := fmt.Sprintf("%s: %s", pio.Rich(method, color), path)
	pio.WriteRich(w, method, color)

	path := r.Tmpl().Src
	if path == "" {
		path = "/"
	}

	fmt.Fprintf(w, ": %s", path)

	// (@description)
	description := r.Description
	if description == "" {
		if method == MethodNone {
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
		// s += fmt.Sprintf(" %s", pio.Rich(description, pio.Cyan, pio.Underline))
		fmt.Fprint(w, " ")
		pio.WriteRich(w, description, pio.Cyan, pio.Underline)
	}

	// (@route_rel_location)
	// s += fmt.Sprintf(" (%s:%d)", r.RegisterFileName, r.RegisterLineNumber)
	fmt.Fprintf(w, " (%s:%d)", r.RegisterFileName, r.RegisterLineNumber)

	for i, h := range r.Handlers {
		var (
			name string
			file string
			line int
		)

		if i == r.MainHandlerIndex && r.MainHandlerName != "" {
			// Main handler info can be programmatically
			// changed to be more specific, respect these changes.
			name = r.MainHandlerName
			file = r.SourceFileName
			line = r.SourceLineNumber
		} else {
			name = context.HandlerName(h)
			file, line = context.HandlerFileLineRel(h)
			// If a middleware, e.g (macro) which changes the main handler index,
			// skip it.
			if file == r.SourceFileName && line == r.SourceLineNumber {
				continue
			}
		}

		// If a handler is an anonymous function then it was already
		// printed in the first line, skip it.
		if file == r.RegisterFileName && line == r.RegisterLineNumber {
			continue
		}

		// * @handler_name (@handler_rel_location)
		fmt.Fprint(w, traceHandlerFile(r.Method, name, file, line))
		if stoppedIndex != -1 && stoppedIndex <= len(r.Handlers) {
			if i <= stoppedIndex {
				pio.WriteRich(w, " ✓", pio.Green)
			} else {
				// pio.WriteRich(w, " ✕", pio.Red, pio.Underline)
			}
		}
	}

	fmt.Fprintln(w)
}

type routeReadOnlyWrapper struct {
	*Route
}

func (rd routeReadOnlyWrapper) StatusErrorCode() int {
	return rd.Route.StatusCode
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

func (rd routeReadOnlyWrapper) Trace(w io.Writer, stoppedIndex int) {
	rd.Route.Trace(w, stoppedIndex)
}

func (rd routeReadOnlyWrapper) Tmpl() macro.Template {
	return rd.Route.Tmpl()
}

func (rd routeReadOnlyWrapper) MainHandlerName() string {
	return rd.Route.MainHandlerName
}

func (rd routeReadOnlyWrapper) MainHandlerIndex() int {
	return rd.Route.MainHandlerIndex
}

func (rd routeReadOnlyWrapper) GetLastMod() time.Time {
	return rd.Route.LastMod
}

func (rd routeReadOnlyWrapper) GetChangeFreq() string {
	return rd.Route.ChangeFreq
}

func (rd routeReadOnlyWrapper) GetPriority() float32 {
	return rd.Route.Priority
}
