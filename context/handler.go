package context

import (
	"regexp"

	"github.com/kataras/iris/v14/core/pkg/funcer"
)

// A Handler responds to an HTTP request.
// It writes reply headers and data to the Context.ResponseWriter() and then return.
// Returning signals that the request is finished;
// it is not valid to use the Context after or concurrently with the completion of the Handler call.
//
// Depending on the HTTP client software, HTTP protocol version,
// and any intermediaries between the client and the iris server,
// it may not be possible to read from the Context.Request().Body after writing to the Context.ResponseWriter().
// Cautious handlers should read the Context.Request().Body first, and then reply.
//
// Except for reading the body, handlers should not modify the provided Context.
//
// If Handler panics, the server (the caller of Handler) assumes that the effect of the panic was isolated to the active request.
// It recovers the panic, logs a stack trace to the server error log, and hangs up the connection.
type Handler = func(*Context)

// Handlers is just a type of slice of []Handler.
//
// See `Handler` for more.
type Handlers = []Handler

var (
	handlerNameResolver = funcer.NewNameResolver()

	// handlerPC is a type of funcer.ProgramCounter[Handler].
	// It resolves the file and line information and other information of a handler function.
	handlerPC = funcer.NewProgramCounter[Handler](handlerNameResolver)
)

// HandlerList is a type of funcer.List[Handler].
// It is an optimal type of list of handlers.
//
// Usage:
//
//	handlers := context.NewHandlerList()
//	handlers.Append/Prepend/AppendAfter/PrependBefore(...) and e.t.c.
type HandlerList = funcer.List[Handler]

// NewHandlerList returns a new HandlerList.
func NewHandlerList(handlers ...Handler) *HandlerList {
	return funcer.NewList(handlerPC, func(h Handler) bool { return h == nil }, handlers...)
}

// ConcatHandlerLists returns a new HandlerList which contains all the handlers of the given lists.
var ConcatHandlerLists = funcer.ConcatLists[Handler]

// HandlerInfo returns the file and line information of a handler.
func HandlerInfo(h Handler) *funcer.Func[Handler] {
	return handlerPC.Resolve(h)
}

// SetHandlerName sets a handler name that could be
// fetched through `HandlerName`. The "original" should be
// the Go's original regexp-featured (can be retrieved through a `HandlerName` call) function name.
// The "replacement" should be the custom, human-text of that function name.
//
// If the name starts with "iris" then it replaces that string with the
// full Iris module package name for convenient between Iris versions,
// see middleware/accesslog init function for example.
func SetHandlerName(original string, replacement string) {
	handlerNameResolver.SetName(original, replacement)
}

// HandlerName returns the handler's function name.
// See `Context.HandlerName` method to get function name of the current running handler in the chain.
// See `SetHandlerName` too.
func HandlerName(fn any) string {
	// return handlerNameResolver.GetName(fn)
	return handlerPC.GetName(fn)
}

// HandlerFileLine returns the handler's file and line information.
// See `Context.HandlerFileLine` method to get the file, line of the current running handler in the chain.
func HandlerFileLine(fn any) (file string, line int) {
	_, file, line = handlerNameResolver.GetNameAndFileAndLine(fn)
	return
	// f := handlerPC.Resolve(h)
	// return f.File, f.Line
}

var (
	ignoreHandlerNames = [...]string{
		"iris.macro",
		"iris/middleware/fileserver/fileserver.StripPrefix",
		"iris/middleware/fileserver/fileserver.PrefixDir",
		"iris/middleware/fileserver/fileserver.PrefixFS",
	}

	ignoreHandlerNamesRegex = [...]*regexp.Regexp{
		regexp.MustCompile(`.*\)\.Routing\.New\.func1`),
	}
)

// IgnoreHandlerName compares a static slice of Iris builtin
// internal methods that should be ignored from trace.
// Some internal methods are kept out of this list for actual debugging.
func IgnoreHandlerName(name string) bool {
	for _, ignore := range ignoreHandlerNames {
		if name == ignore {
			return true
		}
	}

	for _, ignoreRegex := range ignoreHandlerNamesRegex {
		if ignoreRegex.MatchString(name) {
			return true
		}
	}

	return false
}

// Filter is just a type of func(Context) bool which reports whether an action must be performed
// based on the incoming request.
//
// See `NewConditionalHandler` for more.
type Filter func(*Context) bool

// NewConditionalHandler returns a single Handler which can be registered
// as a middleware.
// Filter is just a type of Handler which returns a boolean.
// Handlers here should act like middleware, they should contain `ctx.Next` to proceed
// to the next handler of the chain. Those "handlers" are registered to the per-request context.
//
// It checks the "filter" and if passed then
// it, correctly, executes the "handlers".
//
// If passed, this function makes sure that the Context's information
// about its per-request handler chain based on the new "handlers" is always updated.
//
// If not passed, then simply the Next handler(if any) is executed and "handlers" are ignored.
//
// Example can be found at: _examples/06-middleware/conditional.
func NewConditionalHandler(filter Filter, handlers ...Handler) Handler {
	return func(ctx *Context) {
		if filter(ctx) {
			// Note that we don't want just to fire the incoming handlers, we must make sure
			// that it won't break any further handler chain
			// information that may be required for the next handlers.
			//
			// The below code makes sure that this conditional handler does not break
			// the ability that iris provides to its end-devs
			// to check and modify the per-request handlers chain at runtime.
			currIdx := ctx.HandlerIndex(-1)
			currHandlers := ctx.Handlers()

			if currIdx == len(currHandlers)-1 {
				// if this is the last handler of the chain
				// just add to the last the new handlers and call Next to fire those.
				ctx.AddHandler(handlers...)
				ctx.Next()
				return
			}
			// otherwise insert the new handlers in the middle of the current executed chain and the next chain.
			//
			// Build into a fresh slice. Appending onto currHandlers would write into
			// the route's registered Handlers array, and appending onto "handlers"
			// would write past the length of a slice shared by every request that
			// runs this middleware - two concurrent requests would then race over
			// the same scratch region and could execute each other's route tail.
			newHandlers := make(Handlers, 0, len(currHandlers)+len(handlers))
			newHandlers = append(newHandlers, currHandlers[:currIdx+1]...)
			newHandlers = append(newHandlers, handlers...)
			newHandlers = append(newHandlers, currHandlers[currIdx+1:]...)
			ctx.SetHandlers(newHandlers)
			ctx.Next()
			return
		}
		// if not pass, then just execute the next.
		ctx.Next()
	}
}

// HandlerExists reports whether a handler with the given name or function exists in the "handlers" slice.
func HandlerExists(handlers Handlers, nameOrFunc any) bool {
	switch v := nameOrFunc.(type) {
	case string:
		for _, h := range handlers {
			if HandlerName(h) == v {
				return true
			}
		}

		return false
	default:
		handlerName := HandlerName(nameOrFunc)
		return HandlerExists(handlers, handlerName)
	}
}

// ConcatHandlers returns a copy of given Handlers chains slice joined as one slice of Handlers.
func ConcatHandlers(handlerChains ...Handlers) Handlers {
	if len(handlerChains) == 0 {
		return nil
	}

	cleanHandlers := make(Handlers, 0)
	for _, handlerChain := range handlerChains {
		for _, handler := range handlerChain {
			if handler == nil {
				continue
			}

			cleanHandlers = append(cleanHandlers, handler)
		}
	}

	// No need:
	// newHandlers := make(Handlers, len(cleanHandlers))
	// copy(newHandlers, cleanHandlers)
	return cleanHandlers
}

// JoinHandlers returns a copy of given Handlers slice joined as one slice of Handlers.
func JoinHandlers(handlers ...Handler) Handlers {
	newHandlers := make(Handlers, 0, len(handlers))
	for _, handler := range handlers {
		if handler == nil {
			continue
		}

		newHandlers = append(newHandlers, handler)
	}

	return newHandlers
}

// JoinHandlersAsOneHandler converts a slice of Handlers to a single Handler.
func JoinHandlersAsOneHandler(handlers ...Handler) Handler {
	switch len(handlers) {
	case 0:
		return nil
	case 1:
		return handlers[0]
	default:
		return func(ctx *Context) {
			for _, handler := range handlers {
				handler(ctx)
				if ctx.IsStopped() {
					return
				}
			}
		}
	}
}

// JoinHandlersAsMiddleware converts a slice of Handlers to a single Handler which can be registered as a middleware.
// If all given handlers are nil, then it returns a nil Handler.
func JoinHandlersAsMiddleware(handlers ...Handler) Handler {
	handlersToJoin := make(Handlers, 0, len(handlers))
	for _, h := range handlers {
		if h == nil {
			continue
		}

		handlersToJoin = append(handlersToJoin, h)
	}

	if len(handlersToJoin) == 0 {
		return nil
	}

	return func(ctx *Context) {
		for _, handler := range handlersToJoin {
			handler(ctx)
			if ctx.IsStopped() {
				return
			}
		}

		ctx.Next()
	}
}

// UpsertHandlers returns a new slice holding "h1" with "h2" merged into it:
// a handler of "h2" whose name matches one in "h1" replaces it in place,
// otherwise it is appended.
//
// Only trusted names take the replace path: anonymous compiler-generated
// names (".funcN", "-fm") are not a safe identity since Go 1.27, where two
// distinct code-identical literals may share a code pointer — an unnamed
// closure is therefore always appended, never merged over an existing one.
// Name a middleware (SetHandlerName) to opt into replace semantics.
//
// Neither input slice is modified. That matters when "h1" is a route's
// registered Handlers, which every request for that route shares.
func UpsertHandlers(h1 Handlers, h2 Handlers) Handlers {
	if len(h2) == 0 || h2[0] == nil {
		return h1
	}

	result := make(Handlers, len(h1), len(h1)+len(h2))
	copy(result, h1)

reg:
	for _, handler := range h2 {
		info := HandlerInfo(handler).FuncInfo
		if info.TrustedFuncName() {
			for i, registeredHandler := range result {
				if info.Equal(HandlerInfo(registeredHandler).FuncInfo) {
					result[i] = handler // replace this handler with the new one.
					continue reg        // break and continue to the next handler.
				}
			}
		}

		result = append(result, handler) // or just insert it.
	}

	return result
}

// CopyHandlers returns a copy of "handlers" Handlers slice.
func CopyHandlers(handlers Handlers) Handlers {
	handlersCp := make(Handlers, 0, len(handlers))
	for _, handler := range handlers {
		if handler == nil {
			continue
		}

		handlersCp = append(handlersCp, handler)
	}

	return handlersCp
}

// UniqueHandlers returns a new slice of Handlers which does not contain
// any handler that exists in the "h2" Handlers slice.
//
// Every occurrence is removed, not just the first, and "h1" itself is left
// untouched. That matters when "h1" is a route's registered Handlers, which
// every request for that route shares.
//
// Only trusted names are excluded: an anonymous compiler-generated name is
// not a safe identity since Go 1.27 (distinct code-identical literals may
// share a code pointer), so matching on it could remove an unrelated handler.
func UniqueHandlers(h1 Handlers, h2 Handlers) Handlers {
	if len(h2) == 0 {
		return h1
	}

	exclude := make(map[string]struct{}, len(h2))
	for _, handler := range h2 {
		if info := HandlerInfo(handler).FuncInfo; info.TrustedFuncName() {
			exclude[info.Name] = struct{}{}
		}
	}

	result := make(Handlers, 0, len(h1))
	for _, handler := range h1 {
		if _, skip := exclude[HandlerName(handler)]; skip {
			continue
		}

		result = append(result, handler)
	}

	return result
}

/*
var (
	// MatchedRoutes is a special handler which indicates that the following handlers
	// should be executed only when a route is matched.
	// It's a special handler that should be used only by the `Router.Use` and `Router.UseOnce` methods.
	MatchedRoutes = func(ctx *Context) {
		panic("MatchedRoutes: Should NOT run. Report it as bug at: https://github.com/kataras/iris/issues/new")
	}
	matchedRoutesHandlerName = HandlerName(MatchedRoutes)

	// AnyRoutes is a special handler which indicates that the following handlers
	// should be executed on matched routes, not found and no method allowed requests.
	// It's a special handler that should be used only by the `Router.Use` and `Router.UseOnce` methods.
	AnyRoutes = func(ctx *Context) {
		panic("AnyRoutes: Should NOT run. Report it as bug at: https://github.com/kataras/iris/issues/new")
	}
	anyRoutesHandlerName = HandlerName(AnyRoutes)

	deleteFunc = func(h Handler) bool {
		if h == nil {
			return true
		}

		handlerName := HandlerName(h)
		return handlerName == matchedRoutesHandlerName || handlerName == anyRoutesHandlerName
	}
)

// ClearHandlers removes the nil, MatchedRoutes and AnyRoutes handlers from the given "handlers" slice.
func ClearHandlers(handlers Handlers) Handlers {
	return slices.DeleteFunc(handlers, deleteFunc)
}

// Use SplitHandlers splits the given "handlers" onto two slices, the first one should be executed on matched routes and
// the second one on matched routes, not found and no method allowed requests.
func SplitHandlers(handlers Handlers) (anyHandlers, matchedHandlers Handlers) {
	handlers = slices.DeleteFunc(handlers, func(h Handler) bool { return h == nil })
	if len(handlers) == 0 {
		return
	}

	if lastHandler := handlers[len(handlers)-1]; deleteFunc(lastHandler) {
		// If the last handler is MatchedRoutes or AnyRoutes then remove it,
		// fixes the case of Use(middleware1, MatchedRoutes) or Use(middleware2, AnyRoutes) without any following handler.
		return SplitHandlers(handlers[:len(handlers)-1]) // recursive call to fix Use(middleware1, MatchedRoutes, iris.MatchedRoutes, iris.MatchedRoutes, iris.AnyRoutes).
	}

	skipHandlers := make(map[string]struct{}, len(handlers))

	for i := 0; i < len(handlers); i++ {
		h := handlers[i]
		// if h == nil {
		// 	continue
		// }

		handlerName := HandlerName(h)
		if _, ok := skipHandlers[handlerName]; ok {
			continue
		}

		switch len(handlers) {
		case 1:
			// Use(MatchedRoutes) or Use(AnyRoutes), it's invalid.
			if handlerName == matchedRoutesHandlerName || handlerName == anyRoutesHandlerName {
				// panic("Use: invalid usage of MatchedRoutes or AnyRoutes, should follow by at least one handler")
				continue // let's just do nothing.
			}

			// Use(handler), the default behavior.
			anyHandlers = append(anyHandlers, h)
		default:
			// Use(MatchedRoutes, handler1, handler2, handler3) or
			// Use(AnyRoutes, handler1, handler2, handler3) or
			// Use(handler1, handler2, handler3) or
			// Use(handler1, (AnyRoutes?), handler2, handler3, MatchedRoutes, handler4, handler5)
			for j := i + 1; j < len(handlers); j++ {
				innerHandlerName := HandlerName(handlers[j])
				if innerHandlerName == matchedRoutesHandlerName || innerHandlerName == anyRoutesHandlerName {
					i = j - 1 // Update i to skip over the collected handlers.
					break
				}

				switch handlerName {
				case matchedRoutesHandlerName:
					matchedHandlers = append(matchedHandlers, handlers[j])
				default:
					anyHandlers = append(anyHandlers, handlers[j])
				}

				skipHandlers[innerHandlerName] = struct{}{}
			}
		}
	}

	// No need anymore:
	// Fix the case of Use(middleware1, MatchedRoutes) or Use(middleware2, AnyRoutes) without any following handler.
	// anyHandlers = slices.DeleteFunc(anyHandlers, deleteFunc)
	// matchedHandlers = slices.DeleteFunc(matchedHandlers, deleteFunc)

	return
}

*/
