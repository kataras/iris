package errors

import (
	"fmt"
	"strings"

	"github.com/kataras/iris/v14/context"
)

var (
	// spaceToUnderlineReplacer is a strings.Replacer used to convert HTTP status text
	// (which contains spaces) into valid ErrorCodeName format (which uses underscores).
	// This enables automatic generation of canonical error names from standard HTTP
	// status codes when no specific ErrorCodeName is registered.
	//
	// Example transformations:
	// - "Not Found" -> "NOT_FOUND"
	// - "Internal Server Error" -> "INTERNAL_SERVER_ERROR"
	// - "Bad Request" -> "BAD_REQUEST"
	//
	// This replacement is applied in conjunction with strings.ToUpper() to create
	// consistent, predictable error code names that match the package's naming conventions.
	spaceToUnderlineReplacer = strings.NewReplacer(" ", "_")

	// availableRouterErrorCodeMap maps specific HTTP status codes to their corresponding
	// ErrorCode instances for router-level errors. These are errors that occur at the
	// routing layer before any application handlers are executed.
	//
	// Currently supported router errors:
	// - 404 (Not Found): When no registered route matches the request path
	// - 405 (Method Not Allowed): When a route exists but doesn't support the HTTP method
	//
	// This map enables automatic error handling for common routing failures, ensuring
	// they are processed through the same error handling pipeline as application errors
	// and formatted consistently for client consumption.
	//
	// The map is populated during package initialization using the global errorCodeNameMap
	// to ensure consistency with registered error codes.
	availableRouterErrorCodeMap = map[int]ErrorCode{
		404: errorCodeNameMap[NotFound],
		405: errorCodeNameMap[MethodNotAllowed],
	}
)

// Handler answers both router-level errors (404, 405) and application-level errors with
// one consistent JSON shape and the matching HTTP status code.
//
// This handler serves as a universal error processing middleware that can be applied globally
// or to specific route groups to ensure all errors are handled uniformly throughout the application.
//
// Error Processing Flow:
// 1. Determine if the current request matched a registered route
// 2. For router errors (404/405): Generate appropriate error responses with helpful details
// 3. For application errors: Process any errors set via ctx.StopWithError() or similar methods
// 4. Apply error transformation and formatting through the configured error handling pipeline
// 5. Send structured JSON responses to clients with appropriate HTTP status codes
//
// Router Error Handling:
// - 404 Not Found: sends the HTTP status text ("Not Found") as the message and the fixed
// "The requested page could not be found." text as the detail; no request path is included
// - 405 Method Not Allowed: Includes allowed methods from the Allow header when available
//
// Application Error Handling:
//
// After the chain returns, a status of 400 or above with nothing written is
// rendered. The chain does not have to be stopped: ctx.StopWithStatus(403)
// and ctx.NotFound() are treated alike. What the client reads depends on the
// error stored on the Context, if any (ctx.StopWithError,
// ctx.StopWithPlainError, ctx.SetErr):
//
//   - no stored error: the status text is the message ("Forbidden");
//   - a private error (context.PrivateError, ctx.SetErrPrivate, or an error
//     with a private one anywhere in its chain, which is what errors.Private
//     builds): nothing of it is shown, whatever its type, and no mapping is
//     asked about it. The status stays the one the handler set. Below 500 the
//     status text is the message and there is no detail. At 500 and above the
//     error goes to the configured logger and the client reads the generic
//     "server error" message. The private text reaches the collector, not
//     the client;
//   - an error a mapping recognises, and that is not private (Options.MapErrors
//     and its siblings, a registered ErrorResolver, a validation error, a
//     wrapped ErrorCodeName): it renders through that mapping, with the
//     mapping's status;
//   - any other error, on a status of 500 or above: the error goes to the
//     configured logger and the client reads the generic "server error"
//     message. A store that is down must not have its address read by a
//     client because whoever stored the error forgot to mark it. The status
//     stays the one the handler set: a 503 is a 503. There is one exception:
//     a stored *errors.Error that is not private is sent as it is, because it
//     was built for the client;
//   - any other error, below 500: its text is the message. It was written
//     for the client ("name is required").
//
// Nothing is rendered when a body was written, when the status line is
// already on the wire (a flush with no body yet) or when the connection was
// hijacked: there is no response left to write, and a document appended to
// one the client has started to read corrupts it. Nothing is rendered either
// when the response declares a Content-Length: whoever declared it chose the
// body, an empty one included, as a standard handler does when it relays an
// upstream's empty 401. A status below 400 is never touched.
//
// Where it is registered decides what it covers:
//
//   - app.Use(errors.Handler) on the root application covers the routes and the
//     router's own 404 and 405. A HandleNoRoute or HandleNoMethod handler
//     registered after it never runs: in those two chains this handler answers
//     and does not call the next one.
//   - app.UseRouter(errors.Handler) covers the routes and everything registered
//     after it with UseRouter. It does not cover 404 and 405: it runs before the
//     router, and by the time the chain returns the router's default handler
//     has written the status text.
//   - app.HandleNoRoute(errors.Handler) and app.HandleNoMethod(errors.Handler)
//     cover 404 and 405 only.
//
// iris.NewBuilder registers it for you: with UseRouter as the last of its
// built-in middleware, after compression, and as the 404 and the 405 handler
// when the application registered none of its own. httptest.NewController
// registers it with UseRouter and as the 404 handler.
//
// Usage Examples:
//
//	// Global error handling for entire application
//	app.Use(errors.Handler)
//
//	// Router-level error handling: the routes and the pre-routing middleware
//	// registered after it, and the router's own 404 and 405
//	app.UseRouter(errors.Handler)
//	app.HandleNoRoute(errors.Handler)
//	app.HandleNoMethod(errors.Handler)
//
//	// Route group specific error handling
//	apiGroup := app.Party("/api")
//	apiGroup.Use(errors.Handler)
//
//	// Combined with custom error configuration
//	opts := errors.NewOptions().
//		MapErrors(errors.NotFound, sql.ErrNoRows).
//		SetRenderer(errors.HTML)
//	app.Use(opts.Handler(), errors.Handler)
//
// Error Response Format:
// All errors are formatted as JSON responses following the standard Error struct format:
//
//	{
//		"id": "req-12345",
//		"http_error_code": {
//			"canonical_name": "NOT_FOUND",
//			"status": 404
//		},
//		"message": "Not Found",
//		"detail": "The requested page could not be found."
//	}
//
// Security Considerations:
// - A private error (context.PrivateError, ctx.SetErrPrivate, errors.Private) never has its text sent to a client, through a mapping or otherwise
// - An unmapped error on a status of 500 or above is logged and never has its text sent either, unless it is a *errors.Error that is not private
// - Error IDs enable correlation between client-facing errors and server-side logs
//
// Performance Notes:
// - Minimal overhead for successful requests (early returns)
// - Efficient status code mapping using pre-computed maps
// - Lazy error code generation only when needed
//
// Integration: this handler honours the rest of the error handling system, including
// Options configuration, error mapping, custom renderers and logging.
func Handler(ctx *context.Context) {
	if ctx.CurrentRoute().IsRoute() {
		// if it's a registered route, act like enable response status code feature, run the route and then get the result of status code.
		ctx.Next()
	} else {
		// if it's not a route, it means no_route (404) or no_method_allowed (405) by the router itself, not a registered route.
		// in that case, the context is not stopped yet. Registered through app.Use(errors.Handler) or app.UseRouter(errors.Handler).
		routerStatusCode := ctx.GetStatusCode()

		errorCode, ok := availableRouterErrorCodeMap[routerStatusCode]
		if !ok {
			// should never happen, as the only status code we get from router is 404 and 405.
			ErrUnexpected.LogErr(ctx, fmt.Errorf("unexpected status code by a no route (REPORT IT AS BUG): %d", routerStatusCode))
			return
		}

		details := ""
		if routerStatusCode == 405 {
			if allowHeader := ctx.ResponseHeader("Allow"); allowHeader != "" {
				details = fmt.Sprintf("Allowed methods: %s", allowHeader)
			}
		} else if routerStatusCode == 404 { // when no registered route was found.
			details = "The requested page could not be found."
		}
		errorCode.ServeFailure(ctx, context.StatusText(routerStatusCode), details, "", "", nil, nil, false)
		return
	}

	status := ctx.GetStatusCode()
	if status < 400 {
		return
	}

	// Render only a response nobody has touched. Written reports NoWritten for
	// that state alone: StatusCodeWritten means the status line and the headers
	// are already on the wire (a flush, a hijack), and anything above it that a
	// body is there or on its way. A document written in either of those states
	// would be appended to a response the client has already started to read.
	// The hijack check is for a connection taken over underneath the writer.
	//
	// A declared Content-Length is a body somebody chose, an empty one
	// included. The writer defers WriteHeader, so a standard handler that sets
	// "Content-Length: 0" and calls w.WriteHeader(401), which is what
	// httputil.ReverseProxy relays for an upstream's empty 401, still reports
	// NoWritten here. net/http refuses every byte past a declared length, so the
	// document would never arrive: the client would read an empty body
	// labelled application/json.
	w := ctx.ResponseWriter()
	if w.Written() != context.NoWritten || w.IsHijacked() || w.Header().Get(context.ContentLengthHeaderKey) != "" {
		return
	}

	errorCode := errorCodeOfStatus(status)

	// The stored error, if any, and its visibility. The chain does not have to be
	// stopped: ctx.NotFound() and the file server set a failure status and return.
	isPublic, err := ctx.ErrPublic()
	if err == nil {
		// A failure status and nothing else, e.g. ctx.StopWithStatus(407). The
		// status text is the message, like app.EnableStatusCodeResponse but as JSON.
		errorCode.ServeFailure(ctx, context.StatusText(status), "", "", "", nil, nil, false)
		return
	}

	// A private error is answered before anything looks at what it is.
	// ErrPublic hands back the error without its private wrapper, so past this
	// point only isPublic remembers the mark: a mapping would render the text
	// with the mapping's status, a wrapped ErrorCodeName or a validation error
	// would render it too, and a *Error would be sent as it is.
	if !isPublic || markedPrivate(err) {
		if status >= 500 {
			serveServerError(ctx, errorCode, err, true)
		} else {
			// The client's fault: the status text, not "server error".
			servePrivateClientError(ctx, errorCode, err)
		}

		return
	}

	// A known error value renders through its own mapping first. The second
	// argument is "is this the server's fault", which only a JSON error asks:
	// the status stored next to the error answers it, so a body the client
	// sent malformed, stored with a 400, is not answered as a 500.
	if handleErrorValue(ctx, status >= 500, err) {
		return
	}

	if status >= 500 {
		// The server's own trouble. Its text is for the log, whether or not
		// whoever stored it remembered to mark it private.
		serveServerError(ctx, errorCode, err, false)
		return
	}

	// A message written for the client.
	errorCode.ServeFailure(ctx, err.Error(), "", "", "", nil, nil, false)
}

// markedPrivate reports whether err, or an error it wraps, is a private one
// (context.ErrPrivate). Context.ErrPublic looks at the stored value alone.
// errors.Private adds the mark to an error's chain and keeps the chain's own
// type, and a private error can be wrapped again on its way up to the handler
// that stores it, so the outermost value does not always tell.
func markedPrivate(err error) bool {
	for _, childErr := range UnwrapAll(err) {
		if _, private := childErr.(context.ErrPrivate); private {
			return true
		}
	}

	return false
}

// errorCodeOfStatus returns the canonical ErrorCode of a status code, the first
// one registered for it. A status no code is registered for gets one made from
// its status text, so a name is the same whichever path rendered it
// (HandleError or Handler): 400 is always INVALID_ARGUMENT, never BAD_REQUEST.
func errorCodeOfStatus(status int) ErrorCode {
	var name ErrorCodeName
	if names := codeNamesByStatusCode[status]; len(names) > 0 {
		name = names[0] // the 1st one wins.
	} else {
		name = ErrorCodeName(strings.ToUpper(spaceToUnderlineReplacer.Replace(context.StatusText(status))))
	}

	if errorCode, ok := errorCodeNameMap[name]; ok && errorCode.Status == status {
		return errorCode
	}

	// Not registered: create it at serve-time.
	return ErrorCode{CanonicalName: name, Status: status}
}

// serveServerError answers a stored error on a status of 500 or above, either
// a private one or one no mapping recognised, the way HandleError answers an
// error no mapping recognises: the error goes to the configured logger, the
// collector receives it in full and the client reads the generic message. The
// one difference is the status, which stays the one the handler set: a 503 is
// still a 503.
//
// An error that is not private keeps one exception, the one Internal.LogErr
// has: a *Error is sent as it is, because somebody built that document for
// the client. It is logged like the rest, and the collector is not called for
// it. A private error has no exception.
func serveServerError(ctx *context.Context, errorCode ErrorCode, err error, private bool) {
	if SkipCanceled && ctx.IsCanceled() {
		return
	}

	getReadOnlyOptions(ctx).logError(ctx, err)

	codeName := errorCode.CanonicalName
	if _, registered := errorCodeNameMap[codeName]; !registered {
		errorCode.ServeFailure(ctx, err.Error(), "", "", "", nil, nil, true)
		return
	}

	if private {
		codeName.failErr(ctx, err, true)
		return
	}

	// The same call Internal.LogErr ends in, with this status's code.
	codeName.err(ctx, err, true)
}

// servePrivateClientError answers a private stored error on a status below
// 500. The client reads the status text and nothing else: a rejected token is
// "Unauthorized", not a server fault. The private text stays off the wire and
// reaches the collector, as it did when this case rendered "server error".
func servePrivateClientError(ctx *context.Context, errorCode ErrorCode, err error) {
	opts := getReadOnlyOptions(ctx)
	id := opts.getErrorID(ctx)

	opts.collect(ctx, &Error{ID: id, ErrorCode: errorCode, Message: err.Error()})

	writeError(ctx, opts, &Error{
		ID:        id,
		ErrorCode: errorCode,
		Message:   context.StatusText(errorCode.Status),
	})
}

// Func defines a function signature for error-aware HTTP handlers that can return errors
// instead of manually handling them. This type enables a more functional programming style
// for HTTP handlers where error handling is automatically managed by the framework.
//
// Functions of this type return errors that go through the error handling pipeline:
// mapping, transformation, logging and the client response. Individual handlers do not
// repeat that work.
//
// Function Signature:
//   - Input: iris.Context - The Iris request context
//   - Output: error - Any error that occurred during request processing, or nil for success
//
// Usage Examples:
//
//	// Simple handler function
//	func getUserByID(ctx iris.Context) error {
//		userID := ctx.Params().Get("id")
//		user, err := userService.GetUser(userID)
//		if err != nil {
//			return err // Automatically handled by framework
//		}
//
//		ctx.JSON(user)
//		return nil
//	}
//
//	// Register with automatic error handling
//	app.Get("/users/{id}", errors.FuncHandler(getUserByID))
//
//	// Database operation with automatic error mapping
//	func createUser(ctx iris.Context) error {
//		var user User
//		if err := ctx.ReadJSON(&user); err != nil {
//			return err // JSON parsing errors automatically handled
//		}
//
//		if err := userService.Create(user); err != nil {
//			return err // Database errors automatically mapped
//		}
//
//		ctx.StatusCode(201)
//		ctx.JSON(user)
//		return nil
//	}
//
// Error Handling Benefits:
// - Automatic error mapping through configured Options
// - Consistent error response formatting
// - Built-in logging and monitoring integration
// - Security-conscious private error handling
// - Validation error processing
//
// This type is particularly useful with FuncHandler, FuncHandlers, and FuncMiddleware
// functions that provide the automatic error handling infrastructure.
type Func = func(*context.Context) error

// FuncHandler converts an error-returning function into a standard Iris handler with
// automatic error handling. This enables a cleaner, more functional programming style
// for HTTP handlers by eliminating repetitive error handling boilerplate code.
//
// The returned handler sends any error through the error handling pipeline: mapping,
// transformation, logging and the client response.
//
// Parameters:
//   - fn: An error-returning function that implements the Func signature
//
// Returns:
//   - context.Handler: A standard Iris handler that automatically handles errors
//   - nil: If the input function is nil
//
// Error Processing:
// When the wrapped function returns an error, it is automatically processed through:
// 1. Error mapping and transformation (if configured via Options)
// 2. Validation error handling for structured field-level errors
// 3. Security-conscious private error filtering
// 4. Appropriate HTTP status code determination
// 5. JSON response generation with error details
// 6. Logging and error collection (if configured)
//
// Example usage:
//
//	// Define error-returning handler function
//	func getUserProfile(ctx iris.Context) error {
//		userID := ctx.Values().GetString("user_id")
//		if userID == "" {
//			return errors.Unauthenticated.Wrap(fmt.Errorf("user not authenticated"))
//		}
//
//		profile, err := profileService.GetProfile(userID)
//		if err != nil {
//			return err // Automatic error handling
//		}
//
//		ctx.JSON(profile)
//		return nil
//	}
//
//	// Register with automatic error handling
//	app.Get("/profile", errors.FuncHandler(getUserProfile))
//
//	// Using with middleware chain
//	app.Get("/admin/users",
//		authMiddleware,
//		errors.FuncHandler(listUsers),
//		auditMiddleware)
//
// Advantages over traditional handlers:
// - Eliminates repetitive error handling code
// - Ensures consistent error response formatting
// - Automatic integration with error mapping and logging
// - Cleaner, more readable handler code
// - Better error context preservation
//
// Performance: The wrapper introduces minimal overhead, primarily the error check
// and potential error processing. Successful requests have negligible impact.
//
// Integration: Fully compatible with Iris middleware chains, route groups, and
// the broader error handling ecosystem including Options configuration.
func FuncHandler(fn Func) context.Handler {
	if fn == nil {
		return nil
	}

	return func(ctx *context.Context) {
		if err := fn(ctx); err != nil {
			HandleError(ctx, err)
		}
	}
}

// FuncHandlers converts multiple error-returning functions into a slice of standard Iris
// handlers with automatic error handling. It converts several Func values at once and
// drops any that are nil.
//
// Each function in the input slice is individually wrapped with automatic error handling
// through FuncHandler, creating a slice of context.Handler instances that can be used
// directly with Iris routing methods that accept multiple handlers.
//
// Parameters:
//   - funcs: A variadic list of error-returning functions implementing the Func signature
//
// Returns:
//   - context.Handlers: A slice of standard Iris handlers with automatic error handling
//   - nil: If no valid functions are provided or all functions are nil
//
// Filtering Behavior:
// - Nil functions are automatically filtered out and excluded from the result
// - The resulting slice contains only valid, wrapped handlers
// - If all input functions are nil, the function returns nil
//
// Example usage:
//
//	// Define multiple error-returning handlers
//	func validateRequest(ctx iris.Context) error {
//		// Validation logic
//		return nil
//	}
//
//	func processData(ctx iris.Context) error {
//		// Data processing logic
//		return nil
//	}
//
//	func generateResponse(ctx iris.Context) error {
//		// Response generation logic
//		return nil
//	}
//
//	// Convert all functions to handlers with error handling
//	handlers := errors.FuncHandlers(validateRequest, processData, generateResponse)
//
//	// Use with route registration
//	app.Post("/api/process", handlers...)
//
//	// Combine with traditional handlers
//	allHandlers := append([]context.Handler{authMiddleware}, handlers...)
//	app.Put("/api/update", allHandlers...)
//
// Advanced usage with conditional handlers:
//
//	// Some handlers might be conditionally nil
//	var optionalHandler errors.Func
//	if features.EnableAdvancedProcessing {
//		optionalHandler = advancedProcessingHandler
//	}
//
//	handlers := errors.FuncHandlers(
//		validateRequest,
//		optionalHandler, // Automatically filtered if nil
//		generateResponse,
//	)
//
// Performance: Each function is individually wrapped, so the performance characteristics
// are identical to calling FuncHandler multiple times. The overhead is minimal and
// proportional to the number of valid functions provided.
//
// Use Cases:
// - Pipeline-style request processing with multiple stages
// - Batch conversion of error-returning functions
// - Building reusable handler chains with automatic error handling
// - Simplifying route registration with multiple handlers
func FuncHandlers(funcs ...Func) context.Handlers {
	fnsLen := len(funcs)
	if fnsLen == 0 {
		return nil
	}

	handlers := make(context.Handlers, 0, fnsLen)
	for _, fn := range funcs {
		handler := FuncHandler(fn)
		if handler == nil {
			continue
		}

		handlers = append(handlers, handler)
	}

	return handlers
}

// FuncMiddleware converts an error-returning function into a middleware handler with
// automatic error handling and request pipeline continuation. This function enables
// middleware components to use the same error-handling patterns as regular handlers
// while maintaining proper middleware semantics.
//
// Unlike FuncHandler which is designed for terminal handlers, FuncMiddleware is
// specifically designed for middleware components that need to:
// 1. Perform preprocessing or validation
// 2. Potentially stop the request pipeline on errors
// 3. Continue to the next middleware/handler on success
//
// Parameters:
//   - handler: An error-returning function that implements the Func signature
//
// Returns:
//   - context.Handler: A standard Iris middleware that automatically handles errors
//
// Middleware Behavior:
// - On Success (nil error): Calls ctx.Next() to continue the request pipeline
// - On Error (non-nil error): Processes error through HandleError() and stops execution
// - Error handling includes all standard features: mapping, logging, response generation
//
// Example usage:
//
//	// Authentication middleware
//	func authenticateUser(ctx iris.Context) error {
//		token := ctx.Header("Authorization")
//		if token == "" {
//			return errors.Unauthenticated.Wrap(fmt.Errorf("missing authorization header"))
//		}
//
//		user, err := authService.ValidateToken(token)
//		if err != nil {
//			return errors.Unauthenticated.Wrap(err)
//		}
//
//		ctx.Values().Set("user", user)
//		return nil // Success - continue pipeline
//	}
//
//	// Register as middleware
//	app.Use(errors.FuncMiddleware(authenticateUser))
//
//	// Input validation middleware
//	func validateJSONInput(ctx iris.Context) error {
//		var input map[string]any
//		if err := ctx.ReadJSON(&input); err != nil {
//			return errors.InvalidArgument.Wrap(err)
//		}
//
//		if err := inputValidator.Validate(input); err != nil {
//			return errors.InvalidArgument.Wrap(err)
//		}
//
//		ctx.Values().Set("validated_input", input)
//		return nil
//	}
//
//	// Rate limiting middleware
//	func rateLimitMiddleware(ctx iris.Context) error {
//		clientIP := ctx.RemoteAddr()
//		if !rateLimiter.Allow(clientIP) {
//			return errors.ResourceExhausted.Wrap(fmt.Errorf("rate limit exceeded"))
//		}
//		return nil
//	}
//
//	// Chain multiple error-aware middleware
//	app.Use(
//		errors.FuncMiddleware(rateLimitMiddleware),
//		errors.FuncMiddleware(authenticateUser),
//		errors.FuncMiddleware(validateJSONInput),
//	)
//
// Advanced usage with conditional processing:
//
//	func conditionalProcessing(ctx iris.Context) error {
//		if ctx.Method() == "POST" {
//			// Only validate POST requests
//			if err := validatePostData(ctx); err != nil {
//				return err
//			}
//		}
//
//		// Always log request
//		logRequest(ctx)
//		return nil
//	}
//
// Error Handling Flow:
// 1. Middleware function executes
// 2. If error returned: HandleError() processes it and stops request
// 3. If nil returned: ctx.Next() continues to next middleware/handler
// 4. All error processing features available: mapping, logging, collection, etc.
//
// Performance: Minimal overhead for successful requests (single error check + ctx.Next()).
// Error cases have the same performance characteristics as other error handling paths.
//
// Thread Safety: Fully thread-safe and compatible with concurrent request processing.
//
// Integration: works alongside other Iris middleware and honours the error handling
// Options.
func FuncMiddleware(handler Func) context.Handler {
	return func(ctx *context.Context) {
		if err := handler(ctx); err != nil {
			HandleError(ctx, err) // stops the execution as well, just in case.
			return
		}

		ctx.Next()
	}
}
