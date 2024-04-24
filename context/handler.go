package context

import (
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"strings"
	"sync"
)

var (
	// PackageName is the Iris Go module package name.
	PackageName = strings.TrimSuffix(reflect.TypeOf(Context{}).PkgPath(), "/context")

	// WorkingDir is the (initial) current directory.
	WorkingDir, _ = os.Getwd()
)

var (
	handlerNames   = make(map[*NameExpr]string)
	handlerNamesMu sync.RWMutex

	ignoreMainHandlerNames = [...]string{
		"iris.cache",
		"iris.basicauth",
		"iris.hCaptcha",
		"iris.reCAPTCHA",
		"iris.profiling",
		"iris.recover",
		"iris.accesslog",
		"iris.grpc",
		"iris.requestid",
		"iris.rewrite",
		"iris.cors",
		"iris.jwt",
		"iris.logger",
		"iris.rate",
		"iris.methodoverride",
		"iris.errors.recover",
	}
)

// SetHandlerName sets a handler name that could be
// fetched through `HandlerName`. The "original" should be
// the Go's original regexp-featured (can be retrieved through a `HandlerName` call) function name.
// The "replacement" should be the custom, human-text of that function name.
//
// If the name starts with "iris" then it replaces that string with the
// full Iris module package name,
// e.g. iris/middleware/logger.(*requestLoggerMiddleware).ServeHTTP-fm to
// github.com/kataras/iris/v12/middleware/logger.(*requestLoggerMiddleware).ServeHTTP-fm
// for convenient between Iris versions.
func SetHandlerName(original string, replacement string) {
	if strings.HasPrefix(original, "iris") {
		original = PackageName + strings.TrimPrefix(original, "iris")
	}

	handlerNamesMu.Lock()
	// If regexp syntax is wrong
	// then its `MatchString` will compare through literal. Fixes an issue
	// when a handler name is declared as it's and cause regex parsing expression error,
	// e.g. `iris/cache/client.(*Handler).ServeHTTP-fm`
	regex, _ := regexp.Compile(original)
	handlerNames[&NameExpr{
		literal:           original,
		normalizedLiteral: normalizeExpression(original),
		regex:             regex,
	}] = replacement

	handlerNamesMu.Unlock()
}

// NameExpr regex or literal comparison through `MatchString`.
type NameExpr struct {
	regex             *regexp.Regexp
	literal           string
	normalizedLiteral string
}

// MatchString reports whether "s" is literal of "literal"
// or it matches the regex expression at "regex".
func (expr *NameExpr) MatchString(s string) bool {
	if expr.literal == s { // if matches as string, as it's.
		return true
	}

	if expr.regex != nil {
		return expr.regex.MatchString(s)
	}

	return false
}

// MatchFilename reports whether "filename" contains the "literal".
func (expr *NameExpr) MatchFilename(filename string) bool {
	if filename == "" {
		return false
	}

	return strings.Contains(filename, expr.normalizedLiteral)
}

// The regular expression to match the versioning and the domain part
var trimFileModuleNameRegex = regexp.MustCompile(`^[\w.]+/(kataras|iris-contrib)/|/v\d+|\.\*`)

func normalizeExpression(str string) string {
	// Replace all occurrences of the regular expression with the replacement string.
	return strings.ToLower(trimFileModuleNameRegex.ReplaceAllString(str, ""))
}

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

func valueOf(v interface{}) reflect.Value {
	if val, ok := v.(reflect.Value); ok {
		return val
	}

	return reflect.ValueOf(v)
}

// HandlerName returns the handler's function name.
// See `Context.HandlerName` method to get function name of the current running handler in the chain.
// See `SetHandlerName` too.
func HandlerName(h interface{}) string {
	pc := valueOf(h).Pointer()
	fn := runtime.FuncForPC(pc)
	name := fn.Name()
	filename, _ := fn.FileLine(fn.Entry())
	filenameLower := strings.ToLower(filename)

	handlerNamesMu.RLock()
	for expr, newName := range handlerNames {
		if expr.MatchString(name) || expr.MatchFilename(filenameLower) {
			name = newName
			break
		}
	}
	handlerNamesMu.RUnlock()

	return trimHandlerName(name)
}

// HandlersNames returns a slice of "handlers" names
// separated by commas. Can be used for debugging
// or to determinate if end-developer
// called the same exactly Use/UseRouter/Done... API methods
// so framework can give a warning.
func HandlersNames(handlers ...interface{}) string {
	if len(handlers) == 1 {
		if hs, ok := handlers[0].(Handlers); ok {
			asInterfaces := make([]interface{}, 0, len(hs))
			for _, h := range hs {
				asInterfaces = append(asInterfaces, h)
			}

			return HandlersNames(asInterfaces...)
		}
	}

	names := make([]string, 0, len(handlers))
	for _, h := range handlers {
		names = append(names, HandlerName(h))
	}

	return strings.Join(names, ",")
}

// HandlerFileLine returns the handler's file and line information.
// See `context.HandlerFileLine` to get the file, line of the current running handler in the chain.
func HandlerFileLine(h interface{}) (file string, line int) {
	pc := valueOf(h).Pointer()
	return runtime.FuncForPC(pc).FileLine(pc)
}

// HandlerFileLineRel same as `HandlerFileLine` but it returns the path
// corresponding to its relative based on the package-level "WorkingDir" variable.
func HandlerFileLineRel(h interface{}) (file string, line int) {
	file, line = HandlerFileLine(h)
	if relFile, err := filepath.Rel(WorkingDir, file); err == nil {
		if !strings.HasPrefix(relFile, "..") {
			// Only if it's relative to this path, not parent.
			file = "./" + relFile
		}
	}

	return
}

// MainHandlerName tries to find the main handler that end-developer
// registered on the provided chain of handlers and returns its function name.
func MainHandlerName(handlers ...interface{}) (name string, index int) {
	if len(handlers) == 0 {
		return
	}

	if hs, ok := handlers[0].(Handlers); ok {
		tmp := make([]interface{}, 0, len(hs))
		for _, h := range hs {
			tmp = append(tmp, h)
		}

		return MainHandlerName(tmp...)
	}

	for i := 0; i < len(handlers); i++ {
		name = HandlerName(handlers[i])
		if name == "" {
			continue
		}

		index = i
		if !ingoreMainHandlerName(name) {
			break
		}
	}

	return
}

func trimHandlerName(name string) string {
	// trim the path for Iris' internal middlewares, e.g.
	// irs/mvc.GRPC.Apply.func1
	if internalName := PackageName; strings.HasPrefix(name, internalName) {
		name = strings.Replace(name, internalName, "iris", 1)
	}

	if internalName := "github.com/iris-contrib"; strings.HasPrefix(name, internalName) {
		name = strings.Replace(name, internalName, "iris-contrib", 1)
	}

	name = strings.TrimSuffix(name, "GRPC.Apply.func1")
	return name
}

var ignoreHandlerNames = [...]string{
	"iris/macro/handler.MakeHandler",
	"iris/hero.makeHandler.func2",
	"iris/core/router.ExecutionOptions.buildHandler",
	"iris/core/router.(*APIBuilder).Favicon",
	"iris/core/router.StripPrefix",
	"iris/core/router.PrefixDir",
	"iris/core/router.PrefixFS",
	"iris/context.glob..func2.1",
}

// IgnoreHandlerName compares a static slice of Iris builtin
// internal methods that should be ignored from trace.
// Some internal methods are kept out of this list for actual debugging.
func IgnoreHandlerName(name string) bool {
	for _, ignore := range ignoreHandlerNames {
		if name == ignore {
			return true
		}
	}

	return false
}

// ingoreMainHandlerName reports whether a main handler of "name" should
// be ignored and continue to match the next.
// The ignored main handler names are literals and respects the `ignoreNameHandlers` too.
func ingoreMainHandlerName(name string) bool {
	if IgnoreHandlerName(name) {
		// If ignored at all, it can't be the main.
		return true
	}

	for _, ignore := range ignoreMainHandlerNames {
		if name == ignore {
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
// Example can be found at: _examples/routing/conditional-chain.
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
			newHandlers := append(currHandlers[:currIdx+1], append(handlers, currHandlers[currIdx+1:]...)...)
			ctx.SetHandlers(newHandlers)
			ctx.Next()
			return
		}
		// if not pass, then just execute the next.
		ctx.Next()
	}
}

// JoinHandlers returns a copy of "h1" and "h2" Handlers slice joined as one slice of Handlers.
func JoinHandlers(h1 Handlers, h2 Handlers) Handlers {
	if len(h1) == 0 {
		return h2
	}

	if len(h2) == 0 {
		return h1
	}

	nowLen := len(h1)
	totalLen := nowLen + len(h2)
	// create a new slice of Handlers in order to merge the "h1" and "h2"
	newHandlers := make(Handlers, totalLen)
	// copy the already Handlers to the just created
	copy(newHandlers, h1)
	// start from there we finish, and store the new Handlers too
	copy(newHandlers[nowLen:], h2)
	return newHandlers
}

// UpsertHandlers like `JoinHandlers` but it does
// NOT copies the handlers entries and it does remove duplicates.
func UpsertHandlers(h1 Handlers, h2 Handlers) Handlers {
reg:
	for _, handler := range h2 {
		name := HandlerName(handler)
		for i, registeredHandler := range h1 {
			registeredName := HandlerName(registeredHandler)
			if name == registeredName {
				h1[i] = handler // replace this handler with the new one.
				continue reg    // break and continue to the next handler.
			}
		}

		h1 = append(h1, handler) // or just insert it.
	}

	return h1
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

// HandlerExists reports whether a handler exists in the "handlers" slice.
func HandlerExists(handlers Handlers, handlerNameOrFunc any) bool {
	if handlerNameOrFunc == nil {
		return false
	}

	var matchHandler func(any) bool

	switch v := handlerNameOrFunc.(type) {
	case string:
		matchHandler = func(handler any) bool {
			return HandlerName(handler) == v
		}
	case Handler:
		handlerName := HandlerName(v)
		matchHandler = func(handler any) bool {
			return HandlerName(handler) == handlerName
		}
	default:
		matchHandler = func(handler any) bool {
			return reflect.TypeOf(handler) == reflect.TypeOf(v)
		}
	}

	for _, handler := range handlers {
		if matchHandler(handler) {
			return true
		}
	}

	return false
}
