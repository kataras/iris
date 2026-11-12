package router

import (
	"cmp"
	"fmt"
	"net/http"
	"reflect"
	"slices"
	"strings"

	"github.com/kataras/iris/v14/context"
	"github.com/kataras/iris/v14/core/router/macro/handler"
)

// RouteRegistry contains an unexported slice of *Route.
// It's used to store and manage the registered routes before `Application.Listen`.
type RouteRegistry struct {
	slice             []*Route
	routeRegisterRule RouteRegisterRule

	subdomains []string // a list of subdomains, this is empty until route repository's Build method.
}

// newRouteRegistry returns a new RouteRegistry.
func newRouteRegistry(rule RouteRegisterRule, n int) *RouteRegistry {
	return &RouteRegistry{
		slice:             make([]*Route, 0, n),
		routeRegisterRule: rule, // the default behavior is RouteOverride.
	}
}

// New returns a new empty RouteRegistry with the same route register rule as the current "repo".
// It panics if the receiver is nil.
func (repo *RouteRegistry) New(n int) *RouteRegistry {
	if repo == nil {
		panic("child: no route repository to make a child of")
	}

	return newRouteRegistry(repo.routeRegisterRule, n)
}

func (repo *RouteRegistry) appendRoutes(routes ...*Route) {
	if repo == nil {
		panic("appendRoutes: no route repository to add the route to")
	}

	repo.slice = append(repo.slice, routes...)
}

// CopyRoutes adds one or more routes to the list of routes of another repository.
// It does not check for anything (like Add does). It's only used by internal methods.
func (repo *RouteRegistry) CopyRoutes(otherRepo *RouteRegistry) {
	if repo == nil {
		panic("CopyRoutes: no route repository to add the route to")
	}

	if otherRepo == nil {
		return // nothing to copy from.
		// panic("CopyRoutes: other route repository is nil")
	}

	if otherRepo.routeRegisterRule != repo.routeRegisterRule {
		panic(fmt.Errorf("CopyRoutes: other route repository has different register rule, expected: %s but got: %s",
			repo.routeRegisterRule, otherRepo.routeRegisterRule))
	}

	if otherRepo.Len() == 0 {
		return // nothing to add.
	}

	repo.appendRoutes(otherRepo.slice...)
}

// registerRoutes registers one or more routes to the list of routes of another repository.
// unlike "CopyRoutes" and "appendRoutes" this method checks for this "repo"'s
// register rule and returns an error if such route already exists and the `rule` is `RouteError`,
// it calls the `Add` on each route of the "otherRepo". It may alter the "otherRepo" slice
// in order to set the validated, registered route.
func (repo *RouteRegistry) registerRoutes(otherRepo *RouteRegistry) (*RouteRegistry, error) {
	if repo == nil {
		panic("registerRoutes: no route repository to add the route to")
	}

	if otherRepo == nil {
		panic("registerRoutes: other route repository is nil")
	}

	if otherRepo.Len() == 0 {
		return nil, nil // nothing to add.
	}

	for i, route := range otherRepo.slice {
		registeredRoute, err := repo.addRoute(route, repo.routeRegisterRule)
		if err != nil {
			return nil, err
		}

		// replace the old route with the new one (although it should be the same pointer,
		// but just in case), we need to return only the new registered routes so "otherRepo.slice".
		otherRepo.slice[i] = registeredRoute
	}

	return otherRepo, nil
}

// RouteRegisterRule is a type of uint8.
// Defines the register rule for new routes that already exists.
// Available values are: RouteOverride, RouteSkip and RouteError.
//
// See `Router#SetRegisterRule`.
type RouteRegisterRule uint8

const (
	// RouteOverride replaces an existing route with the new one, the default rule.
	RouteOverride RouteRegisterRule = iota
	// RouteSkip keeps the original route and skips the new one.
	RouteSkip
	// RouteError log when a route already exists, shown after the `Build` state,
	// server never starts.
	RouteError
)

func (rule RouteRegisterRule) String() string {
	switch rule {
	case RouteOverride:
		return "ROUTE_OVERRIDE"
	case RouteSkip:
		return "ROUTE_SKIP"
	case RouteError:
		return "ROUTE_ERROR"
	default:
		return "ROUTE_UNKNOWN_RULE"
	}
}

// AddRouteError is the error type for the `Add` method of route repository when the register rule is `RouteError“
// and a route conflicts with an already registered one.
type AddRouteError struct {
	New *Route
	Old *Route
}

func (e *AddRouteError) Error() string {
	return "new route " + e.New.String() + " conflicts with the old route " + e.Old.String()
}

// SetRegisterRule sets a `RouteRegisterRule` for this Router and its children.
// Available values are:
// * RouteOverride (the default one)
// * RouteSkip
// * RouteError.
func (repo *RouteRegistry) SetRegisterRule(rule RouteRegisterRule) {
	repo.routeRegisterRule = rule
}

// SetSourceByMethodType sets the source caller of the routes by the method's type.
func (repo *RouteRegistry) SetSourceByMethodType(typ reflect.Type, methodName string) {
	if repo == nil {
		return
	}

	for _, r := range repo.slice {
		r.SetSourceByMethodType(typ, methodName)
	}
}

// Add registers a route to the list of routes before `Application.Listen`.
// Returns the route and an error if such route already exists and the `rule` is `RouteError`.
// If the `rule` is `RouteSkip` then it returns the existing route and nil error.
// If the `rule` is `RouteOverride` then it replaces the existing route with the new one and returns the new route and nil error.
func (repo *RouteRegistry) Add(route *Route) (*Route, error) {
	if repo == nil {
		return nil, fmt.Errorf("add: no repository to add the route to")
	}

	return repo.addRoute(route, repo.routeRegisterRule)
}

func (repo *RouteRegistry) addRoute(route *Route, rule RouteRegisterRule) (*Route, error) {
	route.topLink = repo.GetRelative(route) // global.

	for i, r := range repo.slice {
		// 14 August 2019 allow register same path pattern with different macro functions,
		// see #1058
		if route.DeepEqual(r) {
			if rule == RouteSkip {
				return r, nil
			} else if rule == RouteError {
				return nil, &AddRouteError{Old: r, New: route}
			} else {
				// remove the old entry,
				// and continue outside of the loop
				// to add the new one at the last position.
				repo.RemoveByIndex(i)
				// slice = append(slice[:i], slice[i+1:]...)
			}

			break
		}
	}

	// First time registering this route, or it's a new route.
	repo.slice = append(repo.slice, route)

	return route, nil
}

// RemoveByName deletes a registered route by its name before `Application.Listen`.
// The default naming for newly created routes is: method + subdomain + path.
// Reports whether a route with that name was found and removed successfully.
//
// Note that this method applies to all Parties (sub routers)
// even if each of the Parties have access to this method,
// as the route name is unique per Iris Application.
func (repo *RouteRegistry) RemoveByName(routeName string) bool {
	if repo == nil {
		return false
	}

	indexToDelete := repo.GetIndexByName(routeName)
	return repo.RemoveByIndex(indexToDelete)
}

// RemoveByIndex deletes a registered route by its index before `Application.Listen`.
func (repo *RouteRegistry) RemoveByIndex(indexToDelete int) bool {
	if repo == nil {
		return false
	}

	if indexToDelete == -1 {
		return false
	}

	if n := repo.Len(); n-1 == indexToDelete {
		repo.slice = repo.slice[:indexToDelete]
	} else if n == 1 {
		repo.slice = make([]*Route, 0)
	} else {
		repo.slice = append(repo.slice[:indexToDelete], repo.slice[indexToDelete+1:]...)
	}

	return true
}

// Len returns the length of the routes.
func (repo *RouteRegistry) Len() int {
	if repo == nil {
		return 0
	}

	return len(repo.slice)
}

// GetIndexByName returns the index of the route by its name.
func (repo *RouteRegistry) GetIndexByName(routeName string) int {
	if repo == nil {
		return -1
	}

	for i, r := range repo.slice {
		if r.Name == routeName {
			return i
		}
	}

	return -1
}

// GetByIndex returns the route by its index.
func (repo *RouteRegistry) GetByIndex(index int) *Route {
	if index == -1 {
		return nil
	}

	n := len(repo.slice)
	if n <= index {
		return nil
	}

	return repo.slice[index]
}

// GetByMethod returns the first route that its method is equal to the "method" argument.
func (repo *RouteRegistry) GetByMethod(method string) *Route {
	if repo == nil {
		return nil
	}

	for _, route := range repo.slice {
		if route.Method == method {
			return route
		}
	}

	return nil
}

// GetByName returns the first route that its name is equal to the "name" argument.
func (repo *RouteRegistry) GetByName(name string) *Route {
	if repo == nil {
		return nil
	}

	for _, route := range repo.slice {
		if route.Name == name {
			return route
		}
	}

	return nil
}

// GetByMethodAndPath returns the first route that its method and path are equal to the "method" and "path" arguments.
func (repo *RouteRegistry) GetByMethodAndPath(method, path string) *Route {
	if repo == nil {
		return nil
	}

	for _, route := range repo.slice {
		if route.Method == method && route.tmpl.Src == path {
			return route
		}
	}

	return nil
}

// GetByPath returns the first route that its path is equal to the "tmplPath" argument.
func (repo *RouteRegistry) GetByPath(tmplPath string) *Route {
	if repo == nil {
		return nil
	}

	for _, route := range repo.slice {
		if route.tmpl.Src == tmplPath {
			return route
		}
	}

	return nil
}

// GetRelative returns the first non-trailing route that its subdomain, method and
// formatted path is equal to the given one's, i.e. the same path registered with
// different parameter types. Routes with the identical path template are skipped.
// Returns nil if not found.
func (repo *RouteRegistry) GetRelative(r *Route) *Route {
	if repo == nil {
		return nil
	}

	if r.tmpl.IsTrailing() || !handler.CanMakeHandler(r.tmpl) {
		return nil
	}

	for _, route := range repo.slice {
		if r.tmpl.Src == route.tmpl.Src { // No topLink on the same route syntax.
			// Fixes #2008, because of RouterBuilder.handle, repo.getRelative and repo.register replacement but with a toplink of the old route.
			continue
		}

		if r.Subdomain == route.Subdomain &&
			r.Method == route.Method &&
			r.FormattedPath == route.FormattedPath &&
			!route.tmpl.IsTrailing() {
			return route
		}
	}

	return nil
}

// First returns the first route of the list.
// Returns nil if the list is empty.
func (repo *RouteRegistry) First() *Route {
	if repo == nil {
		return nil
	}

	if len(repo.slice) == 0 {
		return nil
	}

	return repo.slice[0]
}

// Last returns the last route of the list.
func (repo *RouteRegistry) Last() *Route {
	if repo == nil {
		return nil
	}

	n := repo.Len()
	if n == 0 {
		return nil
	}

	return repo.slice[n-1]
}

// Get returns the first route that its method is "GET".
func (repo *RouteRegistry) Get() *Route {
	if repo == nil {
		return nil
	}

	return repo.GetByMethod(http.MethodGet)
}

// Head returns the first route that its method is "HEAD".
func (repo *RouteRegistry) Head() *Route {
	if repo == nil {
		return nil
	}

	return repo.GetByMethod(http.MethodHead)
}

// Post returns the first route that its method is "POST".
func (repo *RouteRegistry) Post() *Route {
	if repo == nil {
		return nil
	}

	return repo.GetByMethod(http.MethodPost)
}

// Put returns the first route that its method is "PUT".
func (repo *RouteRegistry) Put() *Route {
	if repo == nil {
		return nil
	}

	return repo.GetByMethod(http.MethodPut)
}

// Patch returns the first route that its method is "PATCH".
func (repo *RouteRegistry) Patch() *Route {
	if repo == nil {
		return nil
	}

	return repo.GetByMethod(http.MethodPatch)
}

// Delete returns the first route that its method is "DELETE".
func (repo *RouteRegistry) Delete() *Route {
	if repo == nil {
		return nil
	}

	return repo.GetByMethod(http.MethodDelete)
}

// Connect returns the first route that its method is "CONNECT".
func (repo *RouteRegistry) Connect() *Route {
	if repo == nil {
		return nil
	}

	return repo.GetByMethod(http.MethodConnect)
}

// Options returns the first route that its method is "OPTIONS".
func (repo *RouteRegistry) Options() *Route {
	if repo == nil {
		return nil
	}

	return repo.GetByMethod(http.MethodOptions)
}

// Query returns the first route that its method is "QUERY".
func (repo *RouteRegistry) Query() *Route {
	if repo == nil {
		return nil
	}

	return repo.GetByMethod(MethodQuery)
}

// Trace returns the first route that its method is "TRACE".
func (repo *RouteRegistry) Trace() *Route {
	if repo == nil {
		return nil
	}

	return repo.GetByMethod(http.MethodTrace)
}

// List returns the slice of the registered routes as it is.
func (repo *RouteRegistry) List() []*Route {
	if repo == nil {
		return nil
	}

	return repo.slice
}

// Build sorts the underline slice of routes and returns it, it's a kind of a "build" method.
func (repo *RouteRegistry) Build() []*Route {
	for _, r := range repo.slice { // before sort.
		if r.topLink != nil {
			bindMultiParamTypesHandler(r)
		}

		// build subdomains list here, at the end of the registration, no need to loop through all routes again.
		if subdomain := r.Subdomain; subdomain != "" {
			if !slices.Contains(repo.subdomains, subdomain) {
				repo.subdomains = append(repo.subdomains, subdomain)
			}
		}
	}

	sortRoutes(repo.slice)
	return repo.slice
}

// ListSubdomains returns the list of subdomains that are registered before Build.
// It's empty until the Build method is called.
func (repo *RouteRegistry) ListSubdomains() []string {
	return repo.subdomains
}

// sortRoutes orders the registered routes so wildcard/longest-path/longer-subdomain
// routes come first. Order matters: bindMultiParamTypesHandler relies on
// {string} and {path} parameter types being seen before {id}/{uint}/etc., so
// when two routes have the same subdomain length and method we preserve
// registration order for parameterized routes (return 0 from the cmp).
func sortRoutes(registeredRoutes []*Route) {
	slices.SortStableFunc(registeredRoutes, func(first, second *Route) int {
		lsub1 := len(first.Subdomain)
		lsub2 := len(second.Subdomain)

		firstSlashLen := strings.Count(first.NodePath, "/")
		secondSlashLen := strings.Count(second.NodePath, "/")

		if lsub1 == lsub2 && first.Method == second.Method {
			if secondSlashLen < firstSlashLen {
				// fixes order when wildcard root is registered before other wildcard paths
				return -1
			}

			if secondSlashLen == firstSlashLen {
				// fixes order when static path with the same prefix with a wildcard path
				// is registered after the wildcard path, although this is managed
				// by the low-level node but it couldn't work if we registered a root level wildcard, this fixes it.
				if len(first.tmpl.Params) == 0 {
					return 1
				}
				if len(second.tmpl.Params) == 0 {
					return -1
				}

				// Equal: preserve registration order — {string}/{path} must be seen
				// before {id}/{uint} as documented above.
				return 0
			}
		}

		// the rest are handled inside the node — longer subdomain wins.
		return cmp.Compare(lsub2, lsub1)
	})
}

// ListByMethod returns a new RouteRegistry which contains
// the routes that their method is equal to the "method" argument.
func (repo *RouteRegistry) ListByMethod(method string) *RouteRegistry {
	if repo == nil {
		return nil
	}

	result := repo.New(0)

	for _, route := range repo.slice {
		if route.Method == method {
			result.slice = append(result.slice, route)
		}
	}

	return result
}

// ListByPattern returns the routes that their (relative) pattern is equal to the "pattern" argument.
func (repo *RouteRegistry) ListByPattern(pattern string) *RouteRegistry {
	if repo == nil {
		return nil
	}

	result := repo.New(0)

	for _, route := range repo.slice {
		if route.relativePattern == pattern {
			result.slice = append(result.slice, route)
		}
	}

	return result
}

// CountHandlers returns the size of the unique handlers that are registered.
func (repo *RouteRegistry) CountHandlers() int {
	if repo == nil {
		return -1
	}

	uniqueNames := make(map[string]struct{})

	for _, r := range repo.slice {
		for _, h := range r.Handlers {
			handlerName := context.HandlerName(h)
			if _, exists := uniqueNames[handlerName]; !exists {
				uniqueNames[handlerName] = struct{}{}
			}
		}
	}

	return len(uniqueNames)
}

//

/*
var defaultOverlapFilter = func(ctx *context.Context) bool {
	if ctx.IsStopped() {
		// It's stopped and the response can be overridden by a new handler.
		// An exception of compress writer, which does not implement Resetter (and it shouldn't):
		rs, ok := ctx.ResponseWriter().(context.ResponseWriterResetter)
		return ok && rs.Reset()
	}

	// It's not stopped, all OK no need to execute the alternative route.
	return false
}

func overlapRoute(r *Route, next *Route) {
	next.BuildHandlers()
	nextHandlers := next.Handlers[0:]

	decisionHandler := func(ctx *context.Context) {
		isErrorRoute := ctx.CurrentRoute().IsNoRoute() || ctx.CurrentRoute().IsMethodNotAllowed()
		ctx.Next()

		if isErrorRoute {
			// If it's an error we don't need to reset (see defaultOverlapFilter)
			// its status code(!) and its body, we just check if it was proceed or not.
			if !ctx.IsStopped() {
				return
			}
		} else {
			prevStatusCode := ctx.GetStatusCode()

			if !defaultOverlapFilter(ctx) {
				return
			}
			// set the status code that it was stopped with.
			// useful for dependencies with StopWithStatus(XXX)
			// instead of raw ctx.StopExecution().
			// The func_result now also catch the last registered status code
			// of the chain, unless the controller returns an integer.
			if prevStatusCode > 0 {
				// An exception when stored error
				// exists and it's type of ErrNotFound.
				// Example:
				// Version was not found:
				//	 we need to be able to send the status on the last not found version
				//   but reset the status code if a next available matched version was found.
				//	 see the versioning package.
				if !errors.Is(ctx.GetErr(), context.ErrNotFound) {
					ctx.StatusCode(prevStatusCode)
				}
			}
		}

		ctx.SetErr(nil) // clear any stored error.
		// Set the route to the next one and execute it.
		ctx.SetCurrentRoute(next)
		ctx.HandlerIndex(0)
		ctx.Do(nextHandlers)
	}

	r.builtinBeginHandlerList.Prepend(decisionHandler)
	// r.builtinBeginHandlers = append(context.Handlers{decisionHandler}, r.builtinBeginHandlers...)
	r.overlappedLink = next
}
*/
