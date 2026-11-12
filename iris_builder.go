package iris

import (
	stdContext "context"
	stdErrors "errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/kataras/iris/v14/rest/errors"

	"github.com/kataras/iris/v14/middleware/accesslog"
	"github.com/kataras/iris/v14/middleware/compress"
	"github.com/kataras/iris/v14/middleware/cors"
	"github.com/kataras/iris/v14/middleware/modrevision"
	"github.com/kataras/iris/v14/middleware/rate"
	"github.com/kataras/iris/v14/middleware/recovery"
)

// NewBuilder returns a step-by-step builder for a production-ready Iris
// application: the common middleware, the error handling and the dependency
// container are set up in the right order for you.
//
// The order is enforced by the compiler, not by documentation. Each step
// returns a narrower interface that exposes only the steps that may follow:
//
//  1. Prefix(path) or NoPrefix(): where the API routes live
//  2. AllowOrigin(line) or Cors(*cors.CORS): CORS
//  3. Compression(bool): request and response compression
//  4. LogRequests(bool) or LogRequestsWith(*accesslog.AccessLog): the access log
//  5. Health(bool, env, developer) or NoHealth(): the /health probe
//  6. Errors(*errors.Options), Deferrables(...), Services(...)
//  7. RouterMiddlewares(...), then Middlewares(...)
//  8. API(path, controllers...), repeatable
//  9. Build() *Application
//
// Steps 1 to 6 only record what they are given. The middleware is registered
// on the first RouterMiddlewares or Middlewares call, guarded by a sync.Once,
// which is why the order it runs in is fixed no matter how the chain is
// written; see RouterMiddlewares for that order.
//
// A failure status that reaches the client with nothing written answers with
// the rest/errors JSON envelope, next to the errors the application reports
// through rest/errors itself. That covers:
//
//   - a status of 400 or above set by a route handler or by a middleware
//     given to RouterMiddlewares, Middlewares or API, whether or not it
//     stopped the chain: ctx.StopWithStatus(403), ctx.NotFound(), the 401 of
//     the jwt and basicauth middleware, a file the file server does not have;
//   - an unknown path (404);
//   - a method a known path does not serve (405, when
//     Options.Router.FireMethodNotAllowed is on; it is off by default, and
//     the answer is then the 404 one).
//
// Two failures are answered ahead of that handler and keep their bare status:
//
//   - a rejection from a handler a Configurator registered with UseRouter.
//     It runs before the builder's own middleware, this one included;
//   - the 415 of the compression middleware, for a request body that is not
//     what its Content-Encoding says.
//
// What the client reads for an error stored on the Context
// (ctx.StopWithError, ctx.StopWithPlainError, ctx.SetErr): nothing of a
// private error (iris.PrivateError), and no mapping is asked about one. It
// renders the status text below 500 ("Unauthorized") and, at 500 and above,
// the generic "server error" while the error is logged. An error a mapping
// recognises, and that is not private, renders through its mapping. Any other
// error: below 500 its text is the message; at 500 and above the text is
// logged and the client reads the generic "server error", except for a stored
// *errors.Error that is not private, which is sent as it is.
//
// A response that has a body, that declares a Content-Length, or whose status
// line is already on the wire, is left as it is, and so is a hijacked
// connection. To answer 404 or 405 yourself, register HandleNoRoute or
// HandleNoMethod from a Configurator given here: the builder fills those two
// slots only when they are empty at Build.
//
// Parameters:
//   - configurators: Optional Iris application configurators for advanced setup
//
// For a worked example see the package examples for this function.
func NewBuilder(configurators ...Configurator) Builder {
	app := New(configurators...)
	app.RegisterDependency(app) // register itself as a dependency, so controllers/custom routers can use it directly.

	// Install the rest/errors handlers here and not in the once-block below:
	// a child Router copies the context options and the macros of its parent
	// when Prefix creates it, which is the very first builder step. Installed
	// any later, a prefixed API would keep the framework's default context
	// error handler while a NoPrefix one renders the rest/errors body.
	app.Macros().SetErrorHandler(errors.DefaultPathParameterTypeErrorHandler)
	app.EditOptions(func(o *Options) {
		o.Context.ErrorHandler = errors.DefaultContextErrorHandler
	})

	return &builder{
		app:  app,
		once: new(sync.Once),
	}
}

type (
	// Builder is the simplified API builder that provides a step-by-step approach
	// to building production-ready Iris applications with common middleware,
	// security features, and best practices built-in.
	//
	// The Builder enforces a logical order of configuration steps to ensure
	// middleware and features are applied in the correct sequence.
	Builder interface {
		// Prefix sets the API Router prefix path for all registered routes.
		// This is useful for versioning (e.g., "/api/v1") or organizing APIs.
		//
		// Parameters:
		//   - prefixPath: The path prefix (must start with "/")
		//
		// Example:
		//   Prefix("/api/v1")  // All routes will be under /api/v1/*
		//
		// Returns CorsBuilder for the next configuration step.
		Prefix(prefixPath string) CorsBuilder

		// NoPrefix disables the API Router prefix path, making routes available
		// directly under the root path. This is equivalent to Prefix("").
		//
		// Example:
		//   NoPrefix()  // Routes available directly under /*
		//
		// Returns CorsBuilder for the next configuration step.
		NoPrefix() CorsBuilder
	}

	// CorsBuilder configures Cross-Origin Resource Sharing (CORS) middleware.
	// CORS is essential for web applications that need to handle requests
	// from different domains, such as frontend applications.
	CorsBuilder interface {
		// AllowOrigin configures CORS allowed origins for cross-domain requests.
		// This is a security-critical setting that controls which domains
		// can access your API from browsers.
		//
		// Parameters:
		//   - originLine: Allowed origins, comma-separated or "*" for all origins
		//
		// Security Notes:
		//   - Use "*" only for public APIs without sensitive data
		//   - For production, specify exact domains: "https://yourdomain.com"
		//   - Multiple origins: "https://app.com,https://admin.app.com"
		//   - Use "none" to disable CORS middleware entirely
		//
		// Examples:
		//   AllowOrigin("*")                                    // Allow all (use cautiously)
		//   AllowOrigin("https://myapp.com")                    // Single domain
		//   AllowOrigin("https://app.com,https://admin.com")    // Multiple domains
		//   AllowOrigin("none")                                 // Disable CORS
		//
		// Returns CompressionBuilder for the next configuration step.
		AllowOrigin(originLine string) CompressionBuilder

		// Cors registers a fully configured CORS value, for the settings
		// AllowOrigin does not reach: credentials, exposed headers, max age,
		// the origin extractor and the error handler.
		//
		// AllowOrigin(line) is the shorthand for:
		//
		//	Cors(cors.New().HandleErrorFunc(errors.FailedPrecondition.Err).AllowOrigin(line))
		//
		// The value is used as given, so call HandleErrorFunc yourself to keep
		// the error mapping AllowOrigin installs.
		//
		// A wildcard origin does not advertise credentials unless they are
		// enabled explicitly, because browsers reject the "*" plus credentials
		// combination. Call AllowCredentials when a non-browser client needs it:
		//
		//	Cors(cors.New().
		//		HandleErrorFunc(errors.FailedPrecondition.Err).
		//		AllowOrigin("*").
		//		AllowCredentials())
		//
		// Returns CompressionBuilder for the next configuration step.
		Cors(c *cors.CORS) CompressionBuilder
	}

	// CompressionBuilder configures HTTP response compression to reduce bandwidth
	// usage and improve response times, especially for JSON APIs and large payloads.
	CompressionBuilder interface {
		// Compression enables or disables automatic response compression using
		// gzip or other client-preferred algorithms (brotli, deflate).
		//
		// The compression middleware automatically:
		//   - Detects client compression support via Accept-Encoding header
		//   - Compresses responses above a minimum size threshold
		//   - Skips compression for already-compressed content types
		//   - Handles streaming responses efficiently
		//
		// Performance Notes:
		//   - Enable for production APIs to reduce bandwidth by 60-80%
		//   - Minimal CPU overhead for most applications
		//   - Particularly effective for JSON responses
		//
		// Parameters:
		//   - enableReadWriteCompression: true to enable, false to disable
		//
		// Example:
		//   Compression(true)   // Enable compression (recommended for production)
		//   Compression(false)  // Disable compression (useful for debugging)
		//
		// Returns RequestLoggingBuilder for the next configuration step.
		Compression(enableReadWriteCompression bool) RequestLoggingBuilder
	}

	// RequestLoggingBuilder configures structured request/response logging for
	// observability, debugging, and compliance requirements.
	RequestLoggingBuilder interface {
		// LogRequests logs every request and response as structured JSON, which is
		// what makes traffic readable to both a log pipeline and a person.
		//
		// The access log captures:
		//   - Request details: method, path, headers, body
		//   - Response details: status code, headers, body, latency
		//   - Client information: IP address, user agent
		//   - Timing information: request duration, timestamps
		//   - Error details: panic logs, error traces
		//
		// Output destinations:
		//   - File: access_log.json (automatically created)
		//   - Console: stdout (can be disabled by modifying the middleware)
		//
		// Parameters:
		//   - enableRequestAccessLog: true to enable logging, false to disable
		//
		// Example:
		//   LogRequests(true)   // Enable for development and production monitoring
		//   LogRequests(false)  // Disable for performance-critical scenarios
		//
		// Returns HealthBuilder for the next configuration step.
		LogRequests(enableRequestAccessLog bool) HealthBuilder

		// LogRequestsWith registers the given pre-configured access log
		// middleware and its closer. Use it when the tuned defaults of
		// LogRequests(true) do not fit: custom output, formatter, fields.
		//
		//   ac := accesslog.File("access_log.json")
		//   ac.AddFields(func(ctx iris.Context, f *accesslog.Fields) { ... })
		//   builder.LogRequestsWith(ac)
		//
		// LogRequests(true) is a shortcut for LogRequestsWith with the
		// framework's tuned default configuration. A nil value is ignored.
		//
		// Returns HealthBuilder for the next configuration step.
		LogRequestsWith(ac *accesslog.AccessLog) HealthBuilder
	}

	// HealthBuilder configures a health check endpoint for monitoring, load balancers,
	// and container orchestration platforms like Kubernetes.
	HealthBuilder interface {
		// Health registers a health check endpoint at /health that reports the
		// application status, the build it is running and the environment.
		//
		// The health endpoint returns:
		//   - Application version and build information
		//   - Environment details (development, staging, production)
		//   - Developer contact information
		//   - Cloud platform detection (AWS, GCP, Azure, etc.)
		//   - Module revision information
		//   - HTTP 200 OK status for healthy applications
		//
		// Use cases:
		//   - Load balancer health checks
		//   - Kubernetes liveness/readiness probes
		//   - Monitoring system integration
		//   - Deployment verification
		//
		// The endpoint is registered on the application's root Router, not under
		// Prefix: a probe lives at /health whatever the API path is, which is
		// what load balancers and orchestrators expect.
		//
		// Parameters:
		//   - enableHealthEndpoint: true to enable the /health endpoint
		//   - env: Environment name (e.g., "development", "staging", "production")
		//   - developer: Developer or team contact information. It is also the
		//     value of the X-Developer response header, set on every response by
		//     the built-in router middleware, even when the endpoint is disabled.
		//
		// Examples:
		//   Health(true, "production", "api-team@company.com")
		//   Health(true, "development", "john.doe@company.com")
		//   Health(false, "", "")  // Disable health endpoint
		//
		// Returns ServiceBuilder for the next configuration step.
		Health(enableHealthEndpoint bool, env, developer string) ServiceBuilder

		// NoHealth disables the health check endpoint. This is equivalent to
		// calling Health(false, "", "").
		//
		// Use this when:
		//   - Health checks are handled by external systems
		//   - Security policies prohibit exposing application information
		//   - Custom health check implementation is preferred
		//
		// Returns ServiceBuilder for the next configuration step.
		NoHealth() ServiceBuilder
	}

	// ServiceBuilder configures error handling, dependency injection and graceful
	// shutdown: the three parts of an application's lifecycle.
	ServiceBuilder interface {
		// Errors configures centralized error handling with custom error mappings,
		// status codes, and response formats for consistent API error responses.
		//
		// The errors system provides:
		//   - Automatic error code mapping (database errors → HTTP status codes)
		//   - Structured error responses with details and correlation IDs
		//   - Custom error handlers for specific error types
		//   - Integration with logging and monitoring systems
		//
		// Common error mappings:
		//   - sql.ErrNoRows → 404 Not Found
		//   - validation errors → 400 Bad Request
		//   - authentication errors → 401 Unauthorized
		//   - authorization errors → 403 Forbidden
		//
		// Parameters:
		//   - options: Error handling configuration from errors.NewOptions()
		//
		// Example:
		//   Errors(errors.NewOptions().
		//       MapErrors(errors.NotFound, sql.ErrNoRows).
		//       MapErrors(errors.Unavailable, driver.ErrBadConn).
		//       MapErrorFunc(errors.DeadlineExceeded, func(err error) error {
		//           // Return an error to answer with this code, nil to pass.
		//           if stdErrors.Is(err, stdContext.DeadlineExceeded) {
		//               return fmt.Errorf("the database took too long: %w", err)
		//           }
		//           return nil
		//       }))
		//
		// Returns ServiceBuilder for chaining.
		Errors(*errors.Options) ServiceBuilder

		// Deferrables registers cleanup functions that will be executed when
		// the server shuts down gracefully. This ensures proper resource cleanup.
		//
		// Supported values (stdContext is the standard library's "context" package):
		//   - func()                         // Simple cleanup function
		//   - func() error                   // Cleanup with error handling
		//   - func(stdContext.Context)       // Cleanup which respects the shutdown context
		//   - func(stdContext.Context) error // Both of the above
		//   - interface{ Close() error }     // Objects with Close method returning error (io.Closer)
		//   - interface{ CloseWithContext(stdContext.Context) error } // Context-aware closers
		//
		// Any other value panics at registration time. An object with a plain
		// Close() method (no error return) is not accepted, pass its method
		// instead: Deferrables(worker.Close).
		//
		// Common use cases:
		//   - Closing database connections
		//   - Stopping background workers
		//   - Flushing caches or logs
		//   - Releasing external resources
		//
		// Parameters:
		//   - closers: One or more cleanup functions or closable objects
		//
		// Examples:
		//   Deferrables(func() { db.Close() })
		//   Deferrables(redisClient, elasticClient)  // Objects with a Close() error method
		//   Deferrables(
		//       func() { worker.Stop() },
		//       func() error { return cache.Flush() },
		//   )
		//
		// Returns ServiceBuilder for chaining.
		Deferrables(closers ...any) ServiceBuilder

		// Services registers dependencies for dependency injection throughout the
		// application. These dependencies can be used in controllers, middleware,
		// and other components.
		//
		// Dependency types supported:
		//   - Service instances (databases, repositories, business services)
		//   - Configuration objects
		//   - Third-party clients (Redis, Elasticsearch, etc.)
		//   - Iris Configurators for advanced application setup
		//   - Functions that return services (factories)
		//
		// Dependency injection features:
		//   - Automatic constructor injection based on parameter types
		//   - Singleton lifecycle management
		//   - Interface-based dependency resolution
		//
		// Three values are treated specially:
		//   - a value with a Close() or Close() error method is registered as a
		//     dependency AND its Close is added to the shutdown closers, so it
		//     does not need a second Deferrables call;
		//   - an iris.Worker is registered as a dependency AND started after the
		//     initializers, then stopped at shutdown; when it is a closer too it
		//     is stopped before it is closed. See iris.Worker;
		//   - an iris.Configurator is applied to the application immediately and
		//     is not registered as a dependency.
		//
		// A func() or a func() error returns nothing injectable and panics
		// here: pass it to Deferrables for cleanup, or implement
		// iris.Initializer for startup work.
		//
		// Parameters:
		//   - dependencies: Services, configurations, or factories to register
		//
		// Examples:
		//   Services(
		//       database.New(config.DatabaseURL),      // Database service
		//       repository.NewUserRepo,                // Repository factory
		//       service.NewAuthService,                // Business service factory
		//       &config.AppConfig{},                   // Configuration object
		//       redis.NewClient(&redis.Options{}),     // Third-party client
		//   )
		//
		// Returns MiddlewareBuilder for the next configuration step.
		Services(dependencies ...any) MiddlewareBuilder
	}

	// MiddlewareBuilder configures the middleware pipeline with clear separation
	// between global middleware (RouterMiddlewares) and API-specific middleware (Middlewares).
	MiddlewareBuilder interface {
		// RouterMiddlewares registers global middleware that executes for ALL requests,
		// including 404s, static files, and health checks. This middleware runs at
		// the top-level router before any route matching occurs.
		//
		// Use RouterMiddlewares for:
		//   - Request logging and metrics collection
		//   - Security headers (CORS, CSP, etc.)
		//   - Rate limiting across the entire application
		//   - Request tracing and correlation IDs
		//   - Path rewriting or request preprocessing
		//   - Global authentication checks
		//
		// Middleware execution order:
		//   1. Built-in router middleware, registered once on the first
		//      RouterMiddlewares or Middlewares call, in this order:
		//      the rest errors options handler (Errors), the access log
		//      (LogRequests/LogRequestsWith), recovery, CORS (AllowOrigin/Cors),
		//      the Server and X-Developer headers, compression (Compression),
		//      the rest errors handler (rest/errors.Handler)
		//   2. RouterMiddlewares (this method), in call order
		//   3. API middleware (Middlewares method)
		//   4. Route-specific middleware
		//   5. Route handlers
		//
		// The order of the first group is the reason the earlier steps only
		// record their configuration: the CORS rejection has to be written
		// through the Errors options and be recorded by the access log, and the
		// access log has to sit above recovery so that a panicking handler is
		// recorded too. The rest errors handler is last so that it writes
		// through the compression writer, and so that the middleware given here
		// runs inside it: a handler that rejects with a bare status, such as
		// ctx.StopWithStatus(429), answers the rest/errors envelope.
		//
		// Because the recovery middleware is registered before the middleware
		// given here, a panic raised by such a handler is recovered as well.
		//
		// Two things sit outside that net. A panic inside the access log or
		// inside the Errors options handler is raised above recovery, so
		// net/http's own per-connection recovery catches it instead: the
		// connection is dropped and the client sees no response at all. And
		// under Prefix the access log and CORS are registered on the prefixed
		// router, while the Errors handler, recovery, the Server and
		// X-Developer headers, compression and the rest errors handler are
		// application-wide - so a request to /health, outside the prefix, is
		// neither logged nor CORS'd.
		//
		// Supported middleware types:
		//   - iris.Handler functions
		//   - Dependency injection functions that return iris.Handler
		//   - Built-in Iris middleware
		//
		// Parameters:
		//   - middlewares: One or more middleware functions or handlers
		//
		// Examples:
		//   RouterMiddlewares(func(ctx iris.Context) {
		//       // Global request logging
		//       ctx.Application().Logger().Infof("Request: %s %s", ctx.Method(), ctx.Path())
		//       ctx.Next()
		//   })
		//
		//   RouterMiddlewares(func(logger *log.Logger) iris.Handler {
		//       // Middleware with dependency injection
		//       return func(ctx iris.Context) {
		//           logger.Printf("Request from IP: %s", ctx.RemoteAddr())
		//           ctx.Next()
		//       }
		//   })
		//
		//   RouterMiddlewares(
		//       securityHeaders,           // Security middleware
		//       requestTracing,            // Tracing middleware
		//       rateLimiter,              // Rate limiting
		//   )
		//
		// Returns MiddlewareBuilder for chaining additional router middleware.
		RouterMiddlewares(middlewares ...any) MiddlewareBuilder

		// Middlewares registers API-specific middleware that executes only for
		// routes registered through the API() method. This middleware runs after
		// router middleware but before individual route handlers.
		//
		// Use Middlewares for:
		//   - API authentication and authorization
		//   - Request validation specific to API routes
		//   - API versioning logic
		//   - Business logic preprocessing
		//   - API-specific rate limiting
		//   - Request/response transformation
		//
		// Middleware scope:
		//   - Applies only to routes registered via API() method
		//   - Does NOT apply to health checks, static files, or other routes
		//   - Executes after RouterMiddlewares
		//   - Can access services registered via dependency injection
		//
		// Parameters:
		//   - middlewares: One or more API-specific middleware functions
		//
		// Examples:
		//   Middlewares(func(ctx iris.Context) {
		//       // API-only authentication
		//       if !isAuthenticated(ctx) {
		//           ctx.StatusCode(401)
		//           return
		//       }
		//       ctx.Next()
		//   })
		//
		//   Middlewares(func(authService *AuthService) iris.Handler {
		//       // Authentication with injected service
		//       return func(ctx iris.Context) {
		//           user, err := authService.ValidateToken(ctx)
		//           if err != nil {
		//               ctx.StatusCode(401)
		//               return
		//           }
		//           ctx.Values().Set("user", user)
		//           ctx.Next()
		//       }
		//   })
		//
		// Returns APIBuilder for the final configuration step.
		Middlewares(middlewares ...any) APIBuilder
	}

	// APIBuilder is the final step for registering API controllers and building
	// the complete application with all configured middleware and services.
	APIBuilder interface {
		// API registers controllers for a specific route group, enabling modular
		// API organization with optional middleware per group.
		//
		// Controller types supported:
		//   - Structs implementing Routing(iris.Router) method
		//   - Factory functions that return controllers (with dependency injection)
		//   - Direct iris.Handler functions
		//   - Middleware functions (using iris.Middleware wrapper)
		//
		// Route group features:
		//   - Each API call creates a new route group under the specified path
		//   - Groups can have their own middleware stack
		//   - Controllers can access injected dependencies
		//   - Automatic method routing based on HTTP verbs (GET, POST, etc.)
		//
		// Parameters:
		//   - prefixPath: Route group prefix (e.g., "/users", "/admin", "/v1/products")
		//   - controllers: Controllers and/or middleware for this group
		//
		// Examples:
		//   // Simple controller registration
		//   API("/users", NewUserController)
		//
		//   // Controller with middleware
		//   API("/admin",
		//       basicauth.Default(map[string]string{"admin": "secret"}),
		//       NewAdminController)
		//
		//   // Multiple controllers in one group
		//   API("/api",
		//       NewUserController,
		//       NewProductController,
		//       NewOrderController)
		//
		//   // Factory function with dependency injection
		//   API("/posts", func(db *sql.DB, cache *redis.Client) *PostController {
		//       return &PostController{db: db, cache: cache}
		//   })
		//
		// Controller example:
		//   type UserController struct {
		//       userService *UserService
		//   }
		//
		//   func NewUserController(service *UserService) *UserController {
		//       return &UserController{userService: service}
		//   }
		//
		//   func (c *UserController) Routing(r iris.Router) {
		//       r.Get("/", c.listUsers)      // GET /users/
		//       r.Post("/", c.createUser)    // POST /users/
		//       r.Get("/{id}", c.getUser)    // GET /users/{id}
		//       r.Put("/{id}", c.updateUser) // PUT /users/{id}
		//       r.Delete("/{id}", c.deleteUser) // DELETE /users/{id}
		//   }
		//
		// Returns APIBuilder for registering additional API groups.
		API(prefixPath string, controllers ...any) APIBuilder

		// Build compiles and finalizes the application configuration, creating
		// the complete Iris application instance ready for deployment.
		//
		// Build process:
		//   1. Validates all registered dependencies
		//   2. Initializes the dependency injection container
		//   3. Applies all registered middleware in correct order
		//   4. Registers all API routes and controllers
		//   5. Sets up error handlers and recovery middleware
		//   6. Validates route patterns and conflicts
		//
		// When to call Build():
		//   - Always: it is the terminal builder step, Listen and Run are methods
		//     of the *Application it returns
		//   - Useful for testing (build without starting the server)
		//   - Required for custom server configurations
		//
		// Returns:
		//   - *Application: The fully configured Iris application instance
		//
		// The returned Application is already built. Its mux was compiled from
		// the routes registered up to this point, and a route registered on it
		// afterwards is never served (the router logs a warning naming it and
		// the file it came from). Register those through API instead, before
		// Build.
		//
		// The returned Application can be used for:
		//   - Custom middleware registration on the pre-routing chain
		//   - Testing with httptest
		//   - Custom server startup with specific options
		//
		// Examples:
		//   // Simple usage
		//   app := builder.Build()
		//   app.Listen(":8080")
		//
		//   // Custom server configuration
		//   app := builder.Build()
		//   app.Run(iris.ServerOptions{Addr: ":443", CertFile: "cert.pem", KeyFile: "key.pem"})
		//
		//   // Testing
		//   app := builder.Build()
		//   e := httptest.New(t, app)
		//   e.GET("/api/users").Expect().Status(httptest.StatusOK)
		//
		// Note: Build() panics if there are configuration errors. This is
		// intentional to catch issues early during application startup.
		Build() *Application
	}
)

// Initializer is implemented by services registered through Builder.Services
// (or RegisterDependency) that need a startup step after dependency injection
// completes and before the server starts: cache warm-up, connection priming,
// background pool setup. Init runs exactly once, inside Builder.Build, in
// service registration order; a returned error stops the application from
// building. Implementations derive their own timeouts from ctx.
// It is the startup counterpart of Builder.Deferrables.
type Initializer interface {
	Init(ctx stdContext.Context) error
}

// Worker is implemented by a long-running service registered through
// Builder.Services: a scheduler, a queue consumer, a cache refresher. The
// builder starts it after every Initializer has returned and stops it before
// the closers run, so it lives inside the startup and shutdown chain the
// builder already enforces instead of in a goroutine next to it.
//
// A Worker is registered as a dependency too, exactly like a value with a
// Close method is, so the same value can be injected into controllers.
//
// It is the long-running counterpart of Initializer.
type Worker interface {
	// Run blocks until the given context is done or the work fails. The
	// context is cancelled at shutdown, just before Stop is called.
	//
	// An error returned before shutdown is logged with the worker's type name
	// and nothing else: a failed worker does not take the HTTP server with it.
	Run(ctx stdContext.Context) error
	// Stop releases what Run acquired. It is called once, at shutdown, with a
	// bounded context: the Application.Shutdown context when that one carries
	// a deadline, otherwise a ten second one, so a Stop which hangs cannot
	// hold the process open forever.
	//
	// Shutdown then waits for Run to return, within the same budget, so a
	// worker is really finished when Shutdown reports success. A Run which
	// does not return in time is logged and reported as a shutdown error.
	//
	// That ten second budget is per worker, not per shutdown: the closers run
	// one after another, so N stuck workers cost N times ten seconds. Pass a
	// deadline to Application.Shutdown to bound the whole chain instead; a
	// caller-supplied deadline is used as it is and is shared by every closer.
	Stop(ctx stdContext.Context) error
}

// workerStopTimeout bounds Worker.Stop, and the wait for Run after it, when
// the shutdown context does not carry a deadline of its own
// (app.Shutdown(context.Background())). It is spent per worker.
const workerStopTimeout = 10 * time.Second

// workerRegistration is the builder's bookkeeping for one registered Worker.
// The context and its cancel are created at registration, before any goroutine
// exists, so the field is never written after the value is published and a
// shutdown which arrives before the start still cancels the worker.
type workerRegistration struct {
	worker Worker
	ctx    stdContext.Context
	cancel stdContext.CancelFunc
	// done is closed when the Run goroutine returns. Shutdown waits on it, so a
	// worker is really finished when Shutdown reports success.
	done chan struct{}
	// startMu makes the start decision and the stop decision mutually exclusive.
	// Two independent atomics are not enough: startWorker loads stopping then
	// stores started, stopWorker stores stopping then loads started, and that
	// shape lets both sides miss each other, so Close could run while a Run was
	// about to be launched. The lock is never held across Stop or the wait.
	startMu sync.Mutex
	// started reports whether the Run goroutine was ever launched: nothing waits
	// on done for a worker which never started. Written under startMu.
	started atomic.Bool
	// stopping tells the Run goroutine that the error it is about to return
	// comes from the shutdown it was asked to perform, not from a failure. It
	// also keeps a worker from starting after its shutdown step already ran.
	stopping atomic.Bool
	stopOnce sync.Once
}

type builder struct {
	app    *Application
	router Router
	once   *sync.Once

	initOnce sync.Once
	workers  []*workerRegistration

	developer    string
	errorOptions *errors.Options
	// cors and accessLog are kept here and registered inside the once-block,
	// after the errors options handler, so both observe the Errors
	// configuration: the access log above recovery so a panicking handler is
	// still recorded, CORS below it so a rejection is both recovered and
	// logged. See RouterMiddlewares for the full order.
	cors              *cors.CORS
	accessLog         *accesslog.AccessLog
	enableCompression bool
}

func (g *builder) Prefix(prefixPath string) CorsBuilder {
	g.router = g.app.Router(prefixPath)
	return g
}

func (g *builder) NoPrefix() CorsBuilder {
	g.router = g.app
	return g
}

func (g *builder) AllowOrigin(originLine string) CompressionBuilder {
	// Register the CORS middleware.
	originLine = strings.ToLower(strings.TrimSpace(originLine))
	if originLine == "none" {
		return g
	}

	return g.Cors(cors.New().HandleErrorFunc(errors.FailedPrecondition.Err).AllowOrigin(originLine))
}

func (g *builder) Cors(c *cors.CORS) CompressionBuilder {
	if c == nil {
		return g
	}

	g.cors = c
	return g
}

func (g *builder) Compression(enableReadWriteCompression bool) RequestLoggingBuilder {
	g.enableCompression = enableReadWriteCompression
	return g
}

func (g *builder) LogRequests(enableRequestAccessLog bool) HealthBuilder {
	if enableRequestAccessLog {
		return g.LogRequestsWith(defaultAccessLog())
	}
	return g
}

func (g *builder) LogRequestsWith(ac *accesslog.AccessLog) HealthBuilder {
	if ac != nil {
		g.accessLog = ac
		g.app.AddCloser(ac.Close)
	}
	return g
}

// defaultAccessLog returns the framework's tuned access log configuration,
// used by LogRequests(true): JSON lines to access_log.json and stdout.
func defaultAccessLog() *accesslog.AccessLog {
	ac := accesslog.File("access_log.json")
	// Remove this line to disable logging to console:
	ac.AddOutput(os.Stdout)
	// The default configuration:
	ac.Delim = '|'
	ac.TimeFormat = "2006-01-02 15:04:05"
	ac.Async = false
	ac.IP = true
	ac.BytesReceivedBody = true
	ac.BytesSentBody = true
	ac.BytesReceived = false
	ac.BytesSent = false
	ac.RequestBody = true
	ac.ResponseBody = true
	ac.KeepMultiLineError = true
	ac.PanicLog = accesslog.LogHandler
	// Default line format if formatter is missing:
	// Time|Latency|Code|Method|Path|IP|Path Params Query Fields|Bytes Received|Bytes Sent|Request|Response|
	//
	// Set Custom Formatter:
	ac.SetFormatter(&accesslog.JSON{
		Indent:    "  ",
		HumanTime: true,
	})

	return ac
}

func (g *builder) Health(enableHealthEndpoint bool, env, developer string) ServiceBuilder {
	g.developer = developer

	if enableHealthEndpoint {
		g.app.Get("/health", modrevision.New(modrevision.Options{
			ServerName:     "Iris Server",
			Env:            env,
			Developer:      developer,
			CloudDetection: true,
		}))
	}
	return g
}

func (g *builder) NoHealth() ServiceBuilder {
	return g
}

func (g *builder) Errors(opts *errors.Options) ServiceBuilder {
	g.errorOptions = opts
	return g
}

func (g *builder) Deferrables(closers ...any) ServiceBuilder {
	for _, closer := range closers {
		g.app.AddCloser(closer)
	}

	return g
}

func (g *builder) Services(deps ...any) MiddlewareBuilder {
	dependencies := make([]any, 0, len(deps))
	for _, d := range deps {
		if d == nil {
			continue // don't add it as a dependnecy.
		}

		switch cb := d.(type) {
		case func(), func() error:
			// A function with nothing to inject is not a service. It used to be
			// registered as a shutdown closer, which silently turned a typo'd
			// constructor into cleanup that never provided anything.
			panic(fmt.Errorf("iris: builder: Services: %T provides no service (nothing injectable is returned); use Deferrables for cleanup or implement iris.Initializer for startup work", d))
		case Worker:
			// Before the Close cases on purpose: a value which is both a Worker
			// and a Closer must be stopped before it is closed, and one closer
			// does both, in that order.
			dependencies = append(dependencies, cb)
			g.registerWorker(cb)
		case interface{ Close() }:
			dependencies = append(dependencies, cb)
			g.app.AddCloser(cb.Close)
		case interface{ Close() error }:
			dependencies = append(dependencies, cb)
			g.app.AddCloser(cb.Close)
		case Configurator:
			g.app.Configure(cb) // don't add it as a dependency (bad value: function has zero outputs).
		default:
			dependencies = append(dependencies, d)
		}
	}

	if err := g.router.RegisterDependency(dependencies...); err != nil {
		panic(fmt.Errorf("iris: builder: Services: %w", err))
	}
	return g
}

func (g *builder) RouterMiddlewares(middlewares ...any) MiddlewareBuilder {
	g.once.Do(func() { // Register the top-level router middlewares only once, before the API's middlewares.
		// Register the rest errors options handler first: everything after it,
		// the CORS rejection included, is written through the collector, the
		// renderer and the id generator configured by Errors.
		if g.errorOptions != nil {
			g.app.UseRouter(g.errorOptions.Handler())
		}

		// The access log goes before recovery, not after it. accesslog.Handler
		// does its logging when ctx.Next returns, not in a defer, so it has to
		// be the outer one: recovery turns the panic into a 500 and returns
		// normally, and the access log resumes and records the request. The
		// other way round a panicking handler is logged nowhere and the
		// PanicLog option is unreachable.
		routerHandlers := g.app.RouterHandlers()
		if g.accessLog != nil {
			g.router.UseRouter(g.accessLog.Handler)
		}

		// A user-registered rest/errors.RecoveryHandler ("iris.errors.recover")
		// counts as recovery too — don't stack a second recovery middleware on it.
		if !routerHandlers.Exists("iris.recovery") && !routerHandlers.Exists("iris.errors.recover") {
			// Register the recovery middleware.
			g.app.UseRouter(recovery.WithErrorHandler(errors.Internal.LogErr), recovery.Handler)
		}

		// CORS is below recovery so a panic in its error handler is recovered,
		// and below the access log so a rejection is recorded.
		if g.cors != nil {
			g.router.UseRouter(g.cors.Handler())
		}

		g.app.UseRouter(func(ctx Context) {
			ctx.SetHeader("Server", "Iris")
			if dev := g.developer; dev != "" {
				ctx.SetHeader("X-Developer", dev)
			}

			ctx.Next()
		})

		if g.enableCompression && !routerHandlers.Exists("iris.compression") {
			// Safe, not Handler: an unbounded decompressor turns a ~1MB gzip
			// body into ~1GB of memory.
			g.app.UseRouter(compress.Safe)
		}

		// The rest errors handler goes last. It renders a failure status that
		// nobody gave a body to, so it has to write while the compression writer
		// is still on the Context: after compression, or the envelope goes out
		// uncompressed under a Content-Encoding header. Everything registered
		// after this block (RouterMiddlewares, the API middleware, the routes)
		// runs inside it, so a rejection from any of them is rendered too.
		g.app.UseRouter(errors.Handler)

		// Bind the rate limit exceed handler to the JSON version of errors package if the rate middleware registered.
		rate.RegisterExceedHandler(g.router, errors.DefaultRateLimitExceededHandler)
	})

	for _, handler := range middlewares {
		g.router.UseRouter(g.router.HandlerFunc(handler))
	}

	return g
}

func (g *builder) Middlewares(middlewares ...any) APIBuilder {
	// To ensure that the top-level middlewares are registered before the API's middlewares.
	g.RouterMiddlewares()

	// UseRelevant, not Use: on the root Router (NoPrefix) Use also prepends to
	// the 404 and 405 chains, so an authentication middleware given here would
	// answer 401 for every unknown path. These handlers belong to the routes
	// registered through API.
	g.router.UseRelevant(g.router.HandlerFuncs(middlewares...)...)
	return g
}

func (g *builder) API(prefixPath string, controllers ...any) APIBuilder {
	g.router.Router(prefixPath, controllers...)
	return g
}

// Build builds the application with the prior configuration and returns the
// Iris Application instance for further customizations.
func (g *builder) Build() *Application {
	app := g.app

	g.initOnce.Do(g.initializeServices)
	g.handleRouterErrors()

	if err := app.Build(); err != nil {
		panic(err)
	}
	return app
}

// handleRouterErrors makes the router's own 404 and 405 answer the rest/errors
// envelope, like every other failure of a builder application.
//
// The handler registered through RouterMiddlewares cannot do it: it runs
// before the router, the router's default handler writes the status text, and
// by the time control is back a body is already there. So errors.Handler is
// registered as the no-route and the no-method handler itself.
//
// It runs when the application is built and only fills an empty slot: a
// HandleNoRoute or HandleNoMethod the application registered on its own, from
// a Configurator or on the value Default returns, is found there and kept.
func (g *builder) handleRouterErrors() {
	if g.app.NoRouteHandlers().Count() == 0 {
		g.app.HandleNoRoute(errors.Handler)
	}

	if g.app.NoMethodHandlers().Count() == 0 {
		g.app.HandleNoMethod(errors.Handler)
	}
}

// unbuilt returns the configured Application without building it, so the caller
// can keep registering routes on it. The one-time router setup already ran
// (through the Middlewares step); the service initializers are deferred to the
// application's own Build, which Listen, Run and httptest call.
//
// Used by Default, which hands the application to the end-developer rather than
// a builder step.
func (g *builder) unbuilt() *Application {
	g.app.registerOnBuild(func() error {
		g.initOnce.Do(g.initializeServices)
		g.handleRouterErrors()
		return nil
	})

	return g.app
}

// initializeServices runs the Init method of every registered singleton
// service that implements the Initializer interface, in registration order.
func (g *builder) initializeServices() {
	router := g.router
	if router == nil {
		router = g.app
	}

	ctx := stdContext.Background()
	router.DI().VisitSingletons(func(initializer Initializer) {
		g.app.Logger().Debugf("Initializing service %T", initializer)
		if err := initializer.Init(ctx); err != nil {
			panic(fmt.Errorf("iris: builder: init: %T: %w", initializer, err))
		}
	})

	g.startWorkers()
}

// registerWorker records the worker and registers its shutdown step. The step
// is a closer like any other, so it runs in registration order together with
// the values given to Deferrables.
//
// Run's context is created here and not in startWorker: the shutdown step is
// registered now and may run before the application is ever built, and a
// cancel func written on one goroutine and read on the signal handler's would
// be a race with a leaked worker at the end of it.
func (g *builder) registerWorker(w Worker) {
	ctx, cancel := stdContext.WithCancel(stdContext.Background())

	reg := &workerRegistration{
		worker: w,
		ctx:    ctx,
		cancel: cancel,
		done:   make(chan struct{}),
	}

	g.workers = append(g.workers, reg)
	g.app.AddCloser(g.stopWorker(reg))
}

// startWorkers runs every registered Worker in its own goroutine. It is called
// from initializeServices, after the Initializer pass, so a worker can rely on
// everything the initializers prepared.
func (g *builder) startWorkers() {
	for _, reg := range g.workers {
		g.startWorker(reg)
	}
}

func (g *builder) startWorker(reg *workerRegistration) {
	// The check and the launch are one step: a shutdown which arrives between
	// them would see started as false, skip the wait and Close the worker under
	// the Run about to start.
	reg.startMu.Lock()
	defer reg.startMu.Unlock()

	if reg.stopping.Load() {
		// Shut down before it ever started: the context is already cancelled and
		// Stop has already run, so there is nothing left to start.
		return
	}

	g.app.Logger().Debugf("Starting worker %T", reg.worker)

	reg.started.Store(true)

	go func() {
		defer close(reg.done)

		if err := reg.worker.Run(reg.ctx); err != nil && !reg.stopping.Load() {
			// Log and carry on: the server may still answer every request which
			// does not depend on this worker, and exiting here would take those
			// down with it.
			g.app.Logger().Errorf("iris: builder: worker: %T: %v", reg.worker, err)
		}
	}()
}

// stopWorker returns the shutdown step of a worker: cancel Run's context, call
// Stop, wait for Run to return and, when the worker is a closer too, Close it
// last.
//
// The wait is what makes the shutdown mean something: without it Shutdown
// reports success while the worker is still draining and, under Listen, the
// process exits from under it. It is bounded by the same context Stop got, so
// a Run which ignores its cancellation cannot hold the process open; that case
// is logged at Error level and returned, because a shutdown which did not
// finish is not a successful one.
//
// Close comes after the wait and not before it: closing the connection a
// running Run is still reading from is the crash this ordering avoids.
func (g *builder) stopWorker(reg *workerRegistration) func(stdContext.Context) error {
	return func(ctx stdContext.Context) error {
		var err error

		reg.stopOnce.Do(func() {
			// Under startMu, so this either wins the race with startWorker (which
			// then starts nothing) or loses it (and the started flag below is
			// already true, so the wait happens). Released before Stop: nothing is
			// held across it or across the wait.
			reg.startMu.Lock()
			reg.stopping.Store(true)
			reg.startMu.Unlock()

			reg.cancel()

			stopCtx, cancel := workerStopContext(ctx)
			defer cancel()

			err = reg.worker.Stop(stopCtx)

			if reg.started.Load() {
				select {
				case <-reg.done:
				case <-stopCtx.Done():
					timeoutErr := fmt.Errorf("iris: builder: worker: %T: Run did not return: %w", reg.worker, stopCtx.Err())
					g.app.Logger().Error(timeoutErr)
					err = stdErrors.Join(err, timeoutErr)
				}
			}

			switch closer := reg.worker.(type) {
			case interface{ Close() error }:
				err = stdErrors.Join(err, closer.Close())
			case interface{ Close() }:
				closer.Close()
			}
		})

		return err
	}
}

// workerStopContext bounds Worker.Stop. The shutdown context is used as it is
// when it already carries a deadline: the caller of Application.Shutdown chose
// how long shutdown may take, and that one deadline then bounds the whole
// closer chain. A context without one, context.Background being the common
// case, gets a fresh workerStopTimeout per worker, so the cost of several
// stuck workers adds up; a caller who cares passes a deadline.
func workerStopContext(ctx stdContext.Context) (stdContext.Context, stdContext.CancelFunc) {
	if ctx == nil {
		return stdContext.WithTimeout(stdContext.Background(), workerStopTimeout)
	}

	if _, hasDeadline := ctx.Deadline(); hasDeadline {
		return ctx, func() {}
	}

	return stdContext.WithTimeout(ctx, workerStopTimeout)
}
