package errors

import (
	"database/sql"
	"encoding/json/jsontext"
	jsonv2 "encoding/json/v2"
	"fmt"
	"log/slog"
	"maps"
	"strings"

	"github.com/kataras/iris/v14/context"
	"github.com/kataras/iris/v14/core/pkg/jsonutil"
	"github.com/kataras/iris/v14/core/pkg/uid"
	"github.com/kataras/iris/v14/middleware/view"

	"github.com/kataras/jwt"
)

func init() {
	context.SetHandlerName("iris/rest/errors.Options.*", "iris.rest.errors.options")
}

type (
	// LogErrorFunc defines the signature for error logging functions used throughout the error handling system.
	// These functions are responsible for recording errors in logs, external monitoring systems, or other
	// persistence mechanisms. The function receives both the request context and the error to be logged.
	//
	// The context provides access to request-specific information such as request ID, user information,
	// request path, and other metadata that can be useful for error correlation and debugging.
	//
	// Implementation Examples:
	//
	//	// Simple logging to application logger
	//	func simpleLogger(ctx iris.Context, err error) {
	//		ctx.Application().Logger().Error(err)
	//	}
	//
	//	// Structured logging with request context
	//	func structuredLogger(ctx iris.Context, err error) {
	//		logger.Error("HTTP request error",
	//			"error", err.Error(),
	//			"request_id", ctx.ID(),
	//			"path", ctx.Path(),
	//			"method", ctx.Method(),
	//			"user_id", ctx.Values().GetString("user_id"))
	//	}
	//
	//	// External service logging (Sentry, LogRocket, etc.)
	//	func sentryLogger(ctx iris.Context, err error) {
	//		sentry.WithScope(func(scope *sentry.Scope) {
	//			scope.SetTag("request_id", ctx.ID())
	//			scope.SetContext("request", map[string]any{
	//				"path": ctx.Path(),
	//				"method": ctx.Method(),
	//			})
	//			sentry.CaptureException(err)
	//		})
	//	}
	//
	// The default implementation logs to the application's configured logger. Custom implementations
	// can be set globally via the LogError package variable or per-route via Options.SetLogger().
	//
	// Thread Safety: Implementations should be thread-safe as they may be called concurrently
	// from multiple goroutines handling different requests.
	LogErrorFunc = func(ctx *context.Context, err error)

	// ContextErrorFunc defines the signature for functions that process Error instances within
	// the context of an HTTP request. These functions are used for both error collection
	// (monitoring, analytics) and error rendering (response formatting).
	//
	// The function receives the current request context and a fully-constructed Error instance
	// containing all error details including ID, error code, message, validation information,
	// and any additional data.
	//
	// Collection Use Cases:
	// - Sending errors to monitoring services (Datadog, New Relic, etc.)
	// - Storing errors in databases for analysis
	// - Triggering alerts based on error patterns
	// - Building error dashboards and metrics
	//
	// Rendering Use Cases:
	// - Custom JSON formatting
	// - HTML error page generation
	// - XML or other format responses
	// - Problem+JSON (RFC 7807) formatting
	//
	// Implementation Examples:
	//
	//	// Error collection for monitoring
	//	func monitoringCollector(ctx iris.Context, err *Error) {
	//		go func() { // Non-blocking collection
	//			monitoring.ReportError(monitoring.ErrorEvent{
	//				ID:         err.ID,
	//				Code:       string(err.ErrorCode.CanonicalName),
	//				Message:    err.Message,
	//				Timestamp:  time.Now(),
	//				RequestID:  ctx.ID(),
	//				UserID:     ctx.Values().GetString("user_id"),
	//			})
	//		}()
	//	}
	//
	//	// Custom JSON rendering
	//	func customJSONRenderer(ctx iris.Context, err *Error) {
	//		response := map[string]any{
	//			"success": false,
	//			"error_code": err.ErrorCode.CanonicalName,
	//			"message": err.Message,
	//			"timestamp": time.Now().Unix(),
	//		}
	//		ctx.JSON(response)
	//	}
	//
	// Performance Note: Collection functions should avoid blocking operations or perform
	// them in separate goroutines to prevent request processing delays.
	ContextErrorFunc = func(ctx *context.Context, err *Error)

	// IDGeneratorFunc defines the signature for functions that generate unique identifiers
	// for error instances. These IDs are used for error tracking, correlation across systems,
	// and debugging purposes.
	//
	// The function receives the current request context, which can be used to derive IDs
	// from existing request metadata or to access application-specific ID generation services.
	//
	// ID Requirements:
	// - Must be unique across all errors in the system
	// - Should be easily searchable in logs and monitoring systems
	// - Should maintain some correlation with the request context when possible
	//
	// Implementation Examples:
	//
	//	// UUID-based generation (default behavior)
	//	func uuidGenerator(ctx iris.Context) string {
	//		return uuid.New().String()
	//	}
	//
	//	// Request-ID based generation
	//	func requestBasedGenerator(ctx iris.Context) string {
	//		requestID := ctx.ID()
	//		errorSeq := atomic.AddInt64(&errorCounter, 1)
	//		return fmt.Sprintf("%s-err-%d", requestID, errorSeq)
	//	}
	//
	//	// Timestamp-based generation
	//	func timestampGenerator(ctx iris.Context) string {
	//		return fmt.Sprintf("err-%d-%s", time.Now().UnixNano(),
	//			randomString(8))
	//	}
	//
	//	// External service integration
	//	func serviceGenerator(ctx iris.Context) string {
	//		return errorTrackingService.GenerateErrorID(ctx.ID())
	//	}
	//
	// The default implementation generates UUIDs. Custom generators can be configured
	// via Options.SetIDGenerator() or Options.SetID() for static IDs.
	IDGeneratorFunc = func(ctx *context.Context) string
	// Options configures HTTP error handling behavior in Iris applications.
	// It is a middleware builder that must be configured before request handlers, and it
	// decides how errors are processed, logged, collected and rendered.
	//
	// The Options struct enables:
	// - Error mapping and transformation based on error types or patterns
	// - Custom logging integration with external services
	// - Error collection for monitoring and analytics
	// - Custom response rendering (JSON, HTML, Problem+JSON, etc.)
	// - Unique error ID generation for tracking and correlation
	//
	// Configuration Workflow:
	// 1. Create new Options instance with NewOptions()
	// 2. Configure error mapping, logging, collection, and rendering
	// 3. Apply the configuration using Handler() or HandlerWithError()
	// 4. Register the middleware before your application handlers
	//
	// Example Usage:
	//
	//	// Basic configuration with error mapping
	//	opts := errors.NewOptions().
	//		MapErrors(errors.NotFound, sql.ErrNoRows, myapp.ErrUserNotFound).
	//		MapErrors(errors.InvalidArgument, myapp.ErrInvalidEmail).
	//		SetLogger(customLogger).
	//		SetCollector(monitoringCollector).
	//		SetRenderer(errors.HTML)
	//
	//	// Apply to specific routes
	//	userRoutes := app.Party("/users")
	//	userRoutes.Use(opts.Handler())
	//
	//	// Apply globally
	//	app.Use(opts.Handler())
	//
	// Error Processing Pipeline:
	// When an error occurs, the Options configuration determines:
	// 1. Whether the error matches any registered mappings
	// 2. How the error should be logged (custom logger vs default)
	// 3. Whether the error should be collected for monitoring
	// 4. How the error response should be rendered to the client
	// 5. What unique ID should be assigned to the error
	//
	// Thread Safety: Options instances should be configured during application
	// initialization before the HTTP server starts, as the internal error mapping
	// is not protected by mutex for performance reasons.
	Options struct {
		// codeMap contains error mapping configurations that determine how different
		// error types should be processed and which ErrorCodeName they should be
		// associated with. This enables automatic error classification and response
		// generation based on error characteristics.
		codeMap ErrorCodeMap

		// logger defines the function used for error logging. If nil, the package-level
		// LogError function is used. Custom loggers enable integration with external
		// logging services, structured logging, or specialized error tracking systems.
		logger LogErrorFunc

		// idGenerator provides custom error ID generation logic. If nil, UUID generation
		// is used by default. Custom generators can create IDs that correlate with
		// request tracking, user sessions, or external monitoring systems.
		idGenerator IDGeneratorFunc

		// collector defines the function used for error collection and monitoring.
		// This is called synchronously before the error response is sent, so
		// time-consuming operations should be performed in separate goroutines.
		// Used for sending errors to monitoring services, databases, or analytics systems.
		collector ContextErrorFunc

		// renderer defines the function used for generating error responses to clients.
		// If nil, JSON rendering is used by default. Custom renderers enable
		// HTML error pages, Problem+JSON (RFC 7807), XML, or other response formats.
		renderer ContextErrorFunc
	}
)

// NewOptions creates and returns a new Options instance for configuring HTTP error handling
// in Iris applications. This function initializes a configuration that can be
// customized using the fluent API methods before being applied as middleware.
//
// The returned Options instance starts with the following defaults:
// - A default error code mapping, seeded by makeDefaultErrorCodeMap
// - Default UUID-based error ID generation
// - Default application logger for error logging
// - No custom error collection or rendering
//
// The default mapping already answers sql.ErrNoRows with NotFound, the kataras/jwt
// errors with Unauthenticated, view.ErrNoEngine with Internal and an error text of
// "encoding/hex: invalid byte:" with InvalidArgument, without any MapErrors call of
// your own.
//
// Configuration should be done during application initialization before the HTTP
// server starts, as the internal error mapping structures are not thread-safe.
//
// Returns:
//   - *Options: A new Options instance ready for configuration
//
// Example usage:
//
//	// Basic configuration
//	opts := errors.NewOptions()
//
//	// Fluent configuration with method chaining
//	opts := errors.NewOptions().
//		MapErrors(errors.NotFound, sql.ErrNoRows).
//		MapErrors(errors.InvalidArgument, myapp.ValidationError{}).
//		SetLogger(customLogger).
//		SetCollector(monitoringCollector).
//		SetRenderer(errors.Problem)
//
//	// Apply to application routes
//	app.Use(opts.Handler())
//
// The Options instance must be applied as middleware using Handler() or HandlerWithError()
// before any route handlers that should benefit from the custom error handling configuration.
//
// See the Options struct documentation for detailed configuration options and the
// Handler() method for middleware application.
func NewOptions() *Options {
	return &Options{codeMap: makeDefaultErrorCodeMap()}
}

func makeDefaultErrorCodeMap() ErrorCodeMap {
	// The PostgreSQL-specific mappings (pg.ErrNoRows, duplicate keys,
	// SQLSTATE text trimming) live in the optional pgerrors subpackage.
	return ErrorCodeMap{}.
		MapErrors(NotFound, sql.ErrNoRows).
		MapErrors(Unauthenticated,
			jwt.ErrBlocked,
			jwt.ErrDecrypt,
			jwt.ErrEmptyKid,
			jwt.ErrExpected,
			jwt.ErrExpired,
			jwt.ErrInvalidKey,
			jwt.ErrIssuedInTheFuture,
			jwt.ErrMissing,
			jwt.ErrMissingKey,
			jwt.ErrNotValidYet,
			jwt.ErrTokenAlg,
			jwt.ErrTokenForm,
			jwt.ErrTokenSignature,
			jwt.ErrUnknownKid,
		).
		MapErrors(Internal, view.ErrNoEngine).
		MapErrorFunc(InvalidArgument, func(err error) error {
			if strings.Contains(err.Error(), "encoding/hex: invalid byte:") {
				return fmt.Errorf("invalid secret format: %w", err)
			}

			return nil
		})
}

// ErrorCodeMap returns the internal error code mapping configuration for direct access.
// This method provides read-only access to the configured error mappings, which can be
// useful for debugging, testing, or introspection of the current error handling setup.
//
// The returned ErrorCodeMap contains all configured error transformations including:
// - Error type mappings (MapErrorTypes)
// - Specific error instance mappings (MapErrors)
// - Custom error transformation functions (MapErrorFunc)
//
// Returns:
//   - ErrorCodeMap: The current error code mapping configuration
//
// Example usage:
//
//	opts := errors.NewOptions().
//		MapErrors(errors.NotFound, sql.ErrNoRows).
//		MapErrors(errors.InvalidArgument, myapp.ValidationError{})
//
//	// Inspect current mappings
//	codeMap := opts.ErrorCodeMap()
//	fmt.Printf("Configured mappings: %d\n", len(codeMap))
//
//	// Test error mapping behavior
//	testErr := sql.ErrNoRows
//	if codeMap.handle(ctx, testErr) {
//		fmt.Println("Error would be handled by configured mappings")
//	}
//
// Thread Safety: The returned map should be treated as read-only. Modifications
// should be made through the Options configuration methods rather than direct
// map manipulation.
//
// See ErrorCodeMap type documentation for details on mapping functionality.
func (opts *Options) ErrorCodeMap() ErrorCodeMap {
	if opts == nil {
		// getReadOnlyOptions returns nil for a request that never went through
		// the options middleware, which is every request in an application that
		// did not register any. Ranging a nil map is fine; dereferencing a nil
		// receiver is not, and every sibling accessor here already guards.
		return nil
	}

	return opts.codeMap
}

func (opts *Options) handleError(ctx *context.Context, err error) bool {
	if opts == nil || len(opts.codeMap) == 0 {
		return false
	}

	return opts.codeMap.handle(ctx, err)
}

// MapErrorFuncCtx registers a context-aware error transformation function:
// like MapErrorFunc, but the function also receives the request context, so
// the transformed error can carry request detail (path, user, values).
// The context is nil when the code map's Filter is called directly, outside
// a request.
//
//	opts.MapErrorFuncCtx(errors.NotFound, func(ctx *context.Context, err error) error {
//		if errors.Is(err, ErrPlanNotFound) {
//			return fmt.Errorf("%s: %w", ctx.Path(), err)
//		}
//		return nil
//	})
func (opts *Options) MapErrorFuncCtx(e ErrorCodeName, fn MapErrorFn) *Options {
	if fn == nil {
		return opts
	}

	if opts.codeMap == nil {
		opts.codeMap = make(ErrorCodeMap)
	}

	opts.codeMap.MapErrorFuncCtx(e, fn)
	return opts
}

// MapErrorFunc registers a custom error transformation function that processes incoming errors
// and optionally transforms them before they are sent to clients. This method provides the
// most flexible error mapping capability, allowing complex error analysis and transformation logic.
//
// The provided function receives an error and can:
// - Return the same error unchanged (pass-through)
// - Return a modified version of the error (transformation)
// - Return nil to indicate this function doesn't handle the error (fallback to other mappings)
// - Return a completely different error (replacement)
//
// When multiple MapErrorFunc functions are registered for the same ErrorCodeName, they are
// executed in registration order until one returns a non-nil error. This enables layered
// error processing and fallback mechanisms.
//
// Parameters:
//   - e: The ErrorCodeName that should be used when this function handles an error
//   - fn: The transformation function that processes incoming errors
//
// Function Signature:
//   - Input: error - The original error to be processed
//   - Output: error - The processed error, or nil if this function doesn't handle it
//
// Example usage:
//
//	// Database error transformation
//	opts.MapErrorFunc(errors.Internal, func(err error) error {
//		if strings.Contains(err.Error(), "connection refused") {
//			return errors.Private(err) // Hide connection details from clients
//		}
//		return nil // Let other mappings handle this error
//	})
//
//	// Validation error enhancement
//	opts.MapErrorFunc(errors.InvalidArgument, func(err error) error {
//		if validationErr, ok := err.(MyValidationError); ok {
//			// Add additional context
//			return fmt.Errorf("validation failed for %s: %w", validationErr.Field, err)
//		}
//		return nil
//	})
//
//	// External service error mapping
//	opts.MapErrorFunc(errors.Unavailable, func(err error) error {
//		if isExternalServiceError(err) {
//			return fmt.Errorf("external service temporarily unavailable")
//		}
//		return nil
//	})
//
// Thread Safety: This method must be called during application initialization before
// the HTTP server starts, as the internal mapping structures are not thread-safe.
//
// Returns: *Options for method chaining
func (opts *Options) MapErrorFunc(e ErrorCodeName, fn func(error) error) *Options {
	if opts.codeMap == nil {
		opts.codeMap = make(ErrorCodeMap)
	}
	opts.codeMap.MapErrorFunc(e, fn)

	return opts
}

// MapErrors registers specific error instances or error values that should be automatically
// mapped to a particular ErrorCodeName when encountered. This method provides exact error
// matching using errors.Is() semantics, making it ideal for mapping well-known errors
// from standard library or third-party packages.
//
// When any of the registered target errors is encountered (either directly or through
// error wrapping), the associated ErrorCodeName will be used to generate the HTTP response.
// This enables automatic translation of internal errors to appropriate HTTP status codes.
//
// Error Matching:
// The method uses errors.Is() for matching, which means:
// - Exact error instances are matched
// - Wrapped errors containing the target are matched
// - Error chains are traversed to find matches
//
// Parameters:
//   - e: The ErrorCodeName to use when any of the target errors is encountered
//   - targets: One or more specific error instances to map to this ErrorCodeName
//
// Example usage:
//
//	// Database errors
//	opts.MapErrors(errors.NotFound, sql.ErrNoRows, gorm.ErrRecordNotFound)
//	opts.MapErrors(errors.AlreadyExists, sql.ErrDuplicateKey)
//
//	// File system errors
//	opts.MapErrors(errors.NotFound, os.ErrNotExist)
//	opts.MapErrors(errors.PermissionDenied, os.ErrPermission)
//
//	// Custom application errors
//	opts.MapErrors(errors.InvalidArgument,
//		myapp.ErrInvalidEmail,
//		myapp.ErrInvalidPassword,
//		myapp.ErrInvalidUserData)
//
//	// External service errors
//	opts.MapErrors(errors.Unavailable, redis.ErrPoolExhausted, kafka.ErrBrokerNotAvailable)
//
// Multiple Mappings: The same error can be mapped to different ErrorCodeNames in different
// contexts by using separate Options instances for different route groups.
//
// Thread Safety: This method must be called during application initialization before
// the HTTP server starts, as the internal mapping structures are not thread-safe.
//
// Returns: *Options for method chaining
func (opts *Options) MapErrors(e ErrorCodeName, targets ...error) *Options {
	if opts.codeMap == nil {
		opts.codeMap = make(ErrorCodeMap)
	}
	opts.codeMap.MapErrors(e, targets...)

	return opts
}

// MapErrorTypes registers error types that should be automatically mapped to a particular
// ErrorCodeName when encountered. This method provides type-based error matching using
// Go's type assertion and interface satisfaction, making it ideal for mapping custom
// error types and interface implementations.
//
// The method accepts type instances (not Type objects) and uses reflection to determine
// if incoming errors are assignable to the registered types. This enables flexible
// error handling for custom error hierarchies and interface-based error designs.
//
// Type Matching:
// - Uses type assertion for interface types
// - Uses type comparison for concrete types
// - Supports pointer and value type matching
// - Works with embedded error types
//
// Parameters:
//   - e: The ErrorCodeName to use when errors of the target types are encountered
//   - targets: One or more type instances (value or pointer) representing the types to map
//
// Example usage:
//
//	// Custom error types
//	opts.MapErrorTypes(errors.InvalidArgument,
//		myapp.ValidationError{},    // Value type
//		&myapp.BusinessRuleError{}) // Pointer type
//
//	// Interface-based error types
//	opts.MapErrorTypes(errors.Unauthenticated,
//		(*myapp.AuthError)(nil),     // Interface type
//		myapp.TokenExpiredError{})   // Concrete implementation
//
//	// Third-party library error types
//	opts.MapErrorTypes(errors.Internal,
//		&pq.Error{},           // PostgreSQL driver errors
//		&json.SyntaxError{},   // JSON parsing errors
//		&url.Error{})          // URL errors
//
//	// Built-in error types
//	opts.MapErrorTypes(errors.RequestTimeout,
//		&net.OpError{},        // Network operation errors
//		context.DeadlineExceeded) // Context timeout errors
//
// Interface Handling: When mapping interface types, pass a typed nil pointer:
//
//	// Correct way to map interface types
//	opts.MapErrorTypes(errors.InvalidArgument, (*ValidationError)(nil))
//
//	// This will match any error that implements ValidationError interface
//
// Polymorphism: Type mapping supports polymorphic error handling where different
// concrete types implementing the same interface can be mapped to different ErrorCodeNames
// based on their specific type.
//
// Thread Safety: This method must be called during application initialization before
// the HTTP server starts, as the internal mapping structures are not thread-safe.
//
// Returns: *Options for method chaining
func (opts *Options) MapErrorTypes(e ErrorCodeName, targets ...any) *Options {
	if opts.codeMap == nil {
		opts.codeMap = make(ErrorCodeMap)
	}
	opts.codeMap.MapErrorTypes(e, targets...)

	return opts
}

// LogError is the default global error logging function used throughout the error handling system.
// This function serves as the fallback logger when no custom logger is configured via Options.SetLogger().
// Modifying this variable affects all error logging globally across the application.
//
// The default implementation:
// - Uses the structured logger (slog) when no context is available
// - Uses the application's configured logger when context is provided
// - Automatically extracts relevant information from the context for enhanced logging
//
// Default Behavior:
// - Context available: Uses ctx.Application().Logger().Error(err)
// - No context: Uses slog.Error(err.Error())
//
// Global Customization Example:
//
//	// Replace global logger with custom implementation
//	errors.LogError = func(ctx iris.Context, err error) {
//		if ctx == nil {
//			myLogger.Error("Context-less error", "error", err)
//			return
//		}
//
//		myLogger.Error("HTTP request error",
//			"error", err.Error(),
//			"request_id", ctx.ID(),
//			"path", ctx.Path(),
//			"method", ctx.Method(),
//			"user_agent", ctx.Header("User-Agent"))
//	}
//
// For route-specific or group-specific logging configuration, use Options.SetLogger()
// instead of modifying this global variable.
//
// Thread Safety: This variable should be modified during application initialization
// before the HTTP server starts. Concurrent modification during runtime is not safe.
var LogError LogErrorFunc = func(ctx *context.Context, err error) {
	if ctx == nil {
		slog.Error(err.Error())
		return
	}

	ctx.Application().Logger().Error(err)
}

// SetLogger configures a custom error logging function for this Options instance,
// overriding the default global LogError function. This enables route-specific or
// group-specific error logging behavior without affecting the global logging configuration.
//
// The custom logger receives both the request context and the error, providing access
// to request-specific information for enhanced logging, monitoring, and debugging.
// This is particularly useful for integrating with external logging services,
// implementing structured logging, or creating custom error handling workflows.
//
// Parameters:
//   - fn: The custom logging function to use for errors in routes configured with this Options instance
//
// Custom Logger Capabilities:
// - Access to full request context (headers, user info, request ID, etc.)
// - Integration with external monitoring services (Sentry, DataDog, etc.)
// - Structured logging with custom fields and formatting
// - Conditional logging based on error types or request characteristics
// - Custom alerting and notification logic
//
// Example usage:
//
//	// Structured logging with request context
//	opts.SetLogger(func(ctx iris.Context, err error) {
//		logger.Error("API error occurred",
//			"error", err.Error(),
//			"request_id", ctx.ID(),
//			"endpoint", ctx.Path(),
//			"method", ctx.Method(),
//			"user_id", ctx.Values().GetString("user_id"),
//			"ip_address", ctx.RemoteAddr())
//	})
//
//	// Integration with external monitoring
//	opts.SetLogger(func(ctx iris.Context, err error) {
//		// Log locally
//		ctx.Application().Logger().Error(err)
//
//		// Send to monitoring service
//		monitoring.RecordError(monitoring.ErrorEvent{
//			Error:     err,
//			RequestID: ctx.ID(),
//			UserID:    ctx.Values().GetString("user_id"),
//			Path:      ctx.Path(),
//			Method:    ctx.Method(),
//		})
//	})
//
//	// Conditional logging based on error severity
//	opts.SetLogger(func(ctx iris.Context, err error) {
//		if isHighSeverityError(err) {
//			alertingService.SendAlert(err, ctx)
//		}
//		defaultLogger.Log(ctx, err)
//	})
//
// Performance Note: Custom loggers should avoid blocking operations or perform them
// in separate goroutines to prevent request processing delays.
//
// Returns: *Options for method chaining
func (opts *Options) SetLogger(fn LogErrorFunc) *Options {
	opts.logger = fn
	return opts
}

func (opts *Options) logError(ctx *context.Context, err error) {
	if opts == nil || opts.logger == nil {
		LogError(ctx, err)
		return
	}

	opts.logger(ctx, err)
}

func (opts *Options) getErrorID(ctx *context.Context) string {
	if opts == nil || opts.idGenerator == nil {
		return uid.NewString()
	}

	return opts.idGenerator(ctx)
}

// SetIDGenerator configures a custom error ID generation function for this Options instance,
// replacing the default UUID-based ID generation. This enables context-aware ID generation
// that can correlate with request tracking, user sessions, or external monitoring systems.
//
// Error IDs serve multiple purposes:
// - Unique identification for error tracking across systems
// - Correlation between client-side and server-side logs
// - Reference points for customer support and debugging
// - Integration with external monitoring and alerting systems
//
// The custom generator receives the current request context, providing access to:
// - Request ID from the context
// - User identification information
// - Session data and metadata
// - Custom application-specific data
//
// Parameters:
//   - fn: The custom ID generation function that receives context and returns a unique string
//
// ID Generation Strategies:
//
//	// Request-based ID generation
//	opts.SetIDGenerator(func(ctx iris.Context) string {
//		requestID := ctx.ID()
//		errorSequence := atomic.AddInt64(&errorCounter, 1)
//		return fmt.Sprintf("%s-err-%d", requestID, errorSequence)
//	})
//
//	// User-session based ID generation
//	opts.SetIDGenerator(func(ctx iris.Context) string {
//		userID := ctx.Values().GetString("user_id")
//		sessionID := ctx.Values().GetString("session_id")
//		timestamp := time.Now().UnixNano()
//		return fmt.Sprintf("usr-%s-sess-%s-%d", userID, sessionID, timestamp)
//	})
//
//	// External service integration
//	opts.SetIDGenerator(func(ctx iris.Context) string {
//		return monitoringService.GenerateTraceID(ctx.ID())
//	})
//
//	// Hierarchical ID generation
//	opts.SetIDGenerator(func(ctx iris.Context) string {
//		serviceID := "api"
//		instanceID := os.Getenv("INSTANCE_ID")
//		requestID := ctx.ID()
//		return fmt.Sprintf("%s.%s.%s.%s", serviceID, instanceID, requestID, uuid.New().String()[:8])
//	})
//
// Best Practices:
// - Ensure IDs are unique across all application instances
// - Include sufficient information for correlation and debugging
// - Keep IDs reasonably short for efficient storage and transmission
// - Consider using structured formats that can be easily parsed
// - Avoid including sensitive information in error IDs
//
// Default Behavior: If no custom generator is set, the system generates UUIDs using
// the standard library's uuid package, providing guaranteed uniqueness across all instances.
//
// Returns: *Options for method chaining
func (opts *Options) SetIDGenerator(fn IDGeneratorFunc) *Options {
	opts.idGenerator = fn
	return opts
}

// SetID configures a static error ID that will be used for all errors handled by this
// Options instance. This method is useful for scenarios where you want consistent,
// predictable error IDs for testing, development, or specific application requirements.
//
// Using static IDs can be helpful for:
// - Unit testing with predictable error responses
// - Development environments with consistent error identification
// - Specific API endpoints that need fixed error reference codes
// - Integration testing scenarios requiring reproducible error IDs
//
// Parameters:
//   - id: The static string that will be used as the error ID for all errors
//
// Example usage:
//
//	// Development environment with predictable IDs
//	if app.IsDebugMode() {
//		opts.SetID("debug-error-001")
//	}
//
//	// Testing configuration
//	testOpts := errors.NewOptions().SetID("test-error-id")
//
//	// API-specific error IDs
//	authOpts := errors.NewOptions().SetID("auth-service-error")
//	paymentOpts := errors.NewOptions().SetID("payment-service-error")
//
// Warning: Using static IDs in production environments is generally not recommended
// as it prevents proper error correlation and tracking. Static IDs should primarily
// be used for development, testing, or very specific use cases where consistent
// error identification is required.
//
// Implementation Note: This method internally uses SetIDGenerator with a function
// that always returns the provided static ID, ignoring the context parameter.
//
// Returns: *Options for method chaining
func (opts *Options) SetID(id string) *Options {
	return opts.SetIDGenerator(func(*context.Context) string {
		return id
	})
}

func (opts *Options) collect(ctx *context.Context, err *Error) {
	if opts == nil || opts.collector == nil {
		return
	}

	opts.collector(ctx, err)
}

// SetCollector configures a custom error collection function for monitoring, analytics,
// and external error tracking systems. The collector receives fully-constructed Error
// instances and can process them for various purposes without affecting the client response.
//
// Error collection is called synchronously before the error response is sent to the client,
// so time-consuming operations should be performed in separate goroutines to avoid
// blocking request processing and degrading user experience.
//
// Common Use Cases:
// - Sending errors to monitoring services (Datadog, New Relic, Prometheus)
// - Storing errors in databases for analysis and reporting
// - Triggering alerts based on error patterns or frequency
// - Building error dashboards and metrics
// - Feeding data to machine learning systems for anomaly detection
// - Creating audit trails for compliance and debugging
//
// Parameters:
//   - fn: The collection function that processes Error instances
//
// Collection Function Signature:
//   - Input: ctx (iris.Context) - The current request context
//   - Input: err (*Error) - The fully-constructed Error instance with all details
//   - Output: None (void function)
//
// Example implementations:
//
//	// Basic monitoring integration
//	opts.SetCollector(func(ctx iris.Context, err *Error) {
//		go func() { // Non-blocking collection
//			monitoring.ReportError(monitoring.ErrorEvent{
//				ID:           err.ID,
//				Code:         string(err.ErrorCode.CanonicalName),
//				Status:       err.ErrorCode.Status,
//				Message:      err.Message,
//				Detail:       err.Detail,
//				RequestID:    ctx.ID(),
//				UserID:       ctx.Values().GetString("user_id"),
//				Endpoint:     ctx.Path(),
//				Method:       ctx.Method(),
//				Timestamp:    time.Now(),
//				UserAgent:    ctx.Header("User-Agent"),
//				IPAddress:    ctx.RemoteAddr(),
//			})
//		}()
//	})
//
//	// Database storage for analysis
//	opts.SetCollector(func(ctx iris.Context, err *Error) {
//		go func() { // Non-blocking storage
//			errorRecord := &ErrorRecord{
//				ID:          err.ID,
//				ErrorCode:   string(err.ErrorCode.CanonicalName),
//				HTTPStatus:  err.ErrorCode.Status,
//				Message:     err.Message,
//				Detail:      err.Detail,
//				ValidationData: err.Validation,
//				RequestPath: ctx.Path(),
//				RequestMethod: ctx.Method(),
//				UserID:      ctx.Values().GetString("user_id"),
//				CreatedAt:   time.Now(),
//			}
//			database.InsertErrorRecord(errorRecord)
//		}()
//	})
//
//	// Alert triggering based on error patterns
//	opts.SetCollector(func(ctx iris.Context, err *Error) {
//		// Check for high-severity errors
//		if err.ErrorCode.Status >= 500 {
//			go alerting.TriggerAlert(alerting.Alert{
//				Severity:    "high",
//				Message:     fmt.Sprintf("Server error: %s", err.Message),
//				ErrorID:     err.ID,
//				RequestPath: ctx.Path(),
//				UserID:      ctx.Values().GetString("user_id"),
//			})
//		}
//
//		// Track error frequency
//		metrics.IncrementCounter("http_errors_total",
//			"code", string(err.ErrorCode.CanonicalName),
//			"endpoint", ctx.Path())
//	})
//
// Performance Considerations:
// - Use goroutines for time-consuming operations (database writes, external API calls)
// - Implement circuit breakers for external service integration
// - Consider batching for high-volume error scenarios
// - Avoid depending on the Iris context in goroutines due to connection pooling
// - Extract all needed data from context before starting goroutines
//
// Thread Safety: Collection functions should be thread-safe as they may be called
// concurrently from multiple request handlers.
//
// Returns: *Options for method chaining
func (opts *Options) SetCollector(fn ContextErrorFunc) *Options {
	opts.collector = fn
	return opts
}

func (opts *Options) render(ctx *context.Context, err *Error) {
	if opts == nil || opts.renderer == nil {
		ctx.JSON(err)
		return
	}

	opts.renderer(ctx, err)
}

// SetRenderer configures a custom error response renderer that overrides the default JSON
// rendering behavior. This enables applications to customize how error responses are formatted
// and sent to clients, supporting various content types and presentation formats.
//
// The renderer function receives the current request context and a fully-constructed Error
// instance, providing complete control over the response generation process. This includes
// setting custom headers, status codes, and response body formatting.
//
// Built-in Renderers:
// The package provides several pre-built renderers:
// - Default: JSON rendering (used when no custom renderer is set)
// - errors.HTML: HTML error page rendering with template support
// - errors.Problem: RFC 7807 Problem Details for HTTP APIs format
//
// Parameters:
//   - fn: The custom rendering function that generates error responses
//
// Renderer Function Signature:
//   - Input: ctx (iris.Context) - The current request context
//   - Input: err (*Error) - The Error instance to be rendered
//   - Output: None (the function should write directly to the response)
//
// Example implementations:
//
//	// Custom JSON rendering with additional metadata
//	opts.SetRenderer(func(ctx iris.Context, err *Error) {
//		response := map[string]any{
//			"success":    false,
//			"error_code": err.ErrorCode.CanonicalName,
//			"message":    err.Message,
//			"detail":     err.Detail,
//			"request_id": err.ID,
//			"timestamp":  time.Now().Unix(),
//			"path":       ctx.Path(),
//		}
//		if err.Data != nil {
//			response["data"] = err.Data
//		}
//		ctx.JSON(response)
//	})
//
//	// XML error rendering
//	opts.SetRenderer(func(ctx iris.Context, err *Error) {
//		xmlError := struct {
//			XMLName xml.Name `xml:"error"`
//			Code    string   `xml:"code"`
//			Message string   `xml:"message"`
//			Detail  string   `xml:"detail,omitempty"`
//		}{
//			Code:    string(err.ErrorCode.CanonicalName),
//			Message: err.Message,
//			Detail:  err.Detail,
//		}
//		ctx.XML(xmlError)
//	})
//
//	// Content negotiation rendering
//	opts.SetRenderer(func(ctx iris.Context, err *Error) {
//		accept := ctx.Header("Accept")
//		switch {
//		case strings.Contains(accept, "application/xml"):
//			renderXMLError(ctx, err)
//		case strings.Contains(accept, "text/html"):
//			errors.HTML(ctx, err)
//		case strings.Contains(accept, "application/problem+json"):
//			errors.Problem(ctx, err)
//		default:
//			ctx.JSON(err) // Default JSON
//		}
//	})
//
// Using Built-in Renderers:
//
//	// HTML error pages with template support
//	opts.SetRenderer(errors.HTML)
//
//	// RFC 7807 Problem Details format
//	opts.SetRenderer(errors.Problem)
//
//	// Custom wrapper around built-in renderers
//	opts.SetRenderer(func(ctx iris.Context, err *Error) {
//		// Add custom headers
//		ctx.Header("X-Error-ID", err.ID)
//		ctx.Header("X-Request-ID", ctx.ID())
//
//		// Use built-in renderer
//		errors.Problem(ctx, err)
//	})
//
// Response Responsibility: Custom renderers are responsible for:
// - Setting appropriate HTTP status codes (usually done automatically)
// - Setting content-type headers
// - Writing the response body
// - Handling any rendering errors gracefully
//
// Thread Safety: Renderer functions should be thread-safe as they may be called
// concurrently from multiple request handlers.
//
// Returns: *Options for method chaining
func (opts *Options) SetRenderer(fn ContextErrorFunc) *Options {
	opts.renderer = fn
	return opts
}

// HasRenderer reports whether this Options instance has a custom error renderer configured.
// This method is useful for conditional logic and debugging to determine if custom rendering
// will be applied to error responses.
//
// When HasRenderer returns false, the default JSON rendering will be used for error responses.
// When it returns true, the custom renderer configured via SetRenderer will be used.
//
// Returns:
//   - bool: true if a custom renderer is configured, false if default JSON rendering will be used
//
// Example usage:
//
//	opts := errors.NewOptions()
//	fmt.Println(opts.HasRenderer()) // false
//
//	opts.SetRenderer(errors.HTML)
//	fmt.Println(opts.HasRenderer()) // true
//
//	// Conditional rendering logic
//	if opts.HasRenderer() {
//		fmt.Println("Custom error rendering is configured")
//	} else {
//		fmt.Println("Using default JSON error rendering")
//	}
//
// This method is primarily used internally by the error handling system to determine
// the appropriate rendering strategy, but can also be useful for testing and debugging.
func (opts *Options) HasRenderer() bool {
	return opts != nil && opts.renderer != nil
}

// HTML is a renderer which renders the error as an HTML page.
// The view engine should be registered before using this renderer.
// The error's status code is used as a part of the template name e.g. /errors/500.html.
//
// Usage:
//
//	opts := errors.NewOptions()
//	opts.SetRenderer(errors.HTML)
func HTML(ctx *context.Context, err *Error) {
	fallbackViewMessage := fmt.Sprintf(`<html><body><h1>%d %s</h1><p>%s</p></body></html>`, err.ErrorCode.Status, err.ErrorCode.CanonicalName, err.Message)

	ext := view.Ext(ctx)
	if ext == "" {
		// means no view engine was registered, let's use a default html page.
		ctx.HTML(fallbackViewMessage)
		return
	}

	// try by finding the template, if not found, then throw the fallback html page.
	tmplName := fmt.Sprintf("errors/%d%s", err.ErrorCode.Status, view.Ext(ctx))
	if view.Render(ctx, tmplName, "", err) != nil {
		// fallback again, view failed, maybe errors/500.html is missing.
		ctx.HTML(fallbackViewMessage)
	}
}

// Problem is a renderer which renders an error as a problem details object as defined in RFC 7807.
// The object must contain the following members:
// - type: A URI reference that identifies the problem type.
// - title: A short, human-readable summary of the problem type.
// - status: The HTTP status code generated by the origin server for this occurrence of the problem.
// - detail: A human-readable explanation specific to this occurrence of the problem.
// - instance: A URI reference that identifies the specific occurrence of the problem.
//
// For more details, see RFC 7807: https://datatracker.ietf.org/doc/html/rfc7807.
func Problem(ctx *context.Context, err *Error) {
	problemStatus := err.ErrorCode.Status
	problemTitle := err.Message
	problemType := err.Type
	problemDetail := err.Detail
	problemInstance := err.Instance

	prob := make(map[string]any, 7)
	prob["id"] = err.ID
	prob["status"] = problemStatus

	if problemTitle != "" {
		prob["title"] = problemTitle
	}

	if problemType != "" {
		prob["type"] = problemType
	}

	if problemDetail != "" {
		prob["detail"] = problemDetail
	}

	if problemInstance != "" {
		prob["instance"] = problemInstance
	}

	if err.Validation != nil {
		prob["validation"] = err.Validation
	}

	if err.Data != nil { // extra data as one level map (... merge).
		extras := make(map[string]any)
		// Duplicate names allowed: the Data bytes may come from
		// rest/types.Merge, which deliberately produces them (last wins).
		if extraErr := jsonv2.Unmarshal(err.Data, &extras, jsontext.AllowDuplicateNames(true)); extraErr != nil {
			// possible more than one extras are passed as array,
			// we cannot combine them, let them render as "data".
			prob["data"] = err.Data
		} else { // merge them.
			maps.Copy(prob, extras)
		}
	}

	ctx.ResponseWriter().Header().Set(context.ContentTypeHeaderKey, context.ContentJSONProblemHeaderValue)
	ctx.StatusCode(problemStatus)

	// Framework marshal defaults: sorted map keys (problem+json readers and
	// golden tests depend on the stable member order), duplicate names,
	// lenient UTF-8 and Duration-as-nanos. Buffer-then-write so a marshal
	// failure cannot leave a partial body before HandleResponseJSONError.
	probBody, encodeErr := jsonv2.Marshal(prob, jsonutil.DefaultMarshalOptionsSlice...)
	if encodeErr == nil {
		probBody = append(probBody, '\n')
		_, writeErr := ctx.Write(probBody)
		if writeErr != nil {
			return
		}
	}
	if encodeErr != nil {
		// we should fire the error without any custom options, so
		// renderer will not try to render the error as a problematic problem.
		if prevOpts := getReadOnlyOptions(ctx); prevOpts != nil {
			setOptions(ctx, nil)
			defer setOptions(ctx, prevOpts)
		}

		// fail(ctx, Internal, encodeErr.Error(),"",nil, nil,false)
		HandleResponseJSONError(ctx, encodeErr)
	}
}

// Handler returns the middleware handler which should be registered to the Iris router.
// If nothing was set, then it returns nil.
func (opts *Options) Handler() context.Handler {
	// if nothing was set, then do not register the middleware at all.
	if opts == nil || (len(opts.codeMap) == 0 && opts.idGenerator == nil && opts.collector == nil && opts.renderer == nil) {
		return nil
	}

	return func(ctx *context.Context) {
		setOptions(ctx, opts)
		ctx.Next()
	}
}

// HandlerWithError returns the middleware handler which should be registered to the Iris errors router.
func (opts *Options) HandlerWithError() Func {
	h := opts.Handler()
	if h == nil {
		return nil
	}

	return func(ctx *context.Context) error {
		h(ctx)
		return nil
	}
}

const optionsContextKey = "iris.rest.errors.options"

func setOptions(ctx *context.Context, opts *Options) {
	ctx.Values().Set(optionsContextKey, opts)
}

func getReadOnlyOptions(ctx *context.Context) *Options {
	v := ctx.Values().Get(optionsContextKey)
	if v == nil {
		return nil
	}

	return v.(*Options)
}
