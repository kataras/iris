// Package errors is the HTTP error handling system for Iris web applications. It
// implements RFC 7807 (Problem Details for HTTP APIs) and lets you control how errors
// are processed, logged and rendered to clients.
//
// # Core Concepts
//
// The package is built around canonical error names (ErrorCodeName) that map to HTTP status codes,
// providing a consistent way to handle errors across your application. Each error name corresponds
// to a specific HTTP status code and can be used to send structured error responses to clients.
//
// # Basic Usage
//
// Register custom error codes during application initialization:
//
//	var UserNotFound = errors.Register("USER_NOT_FOUND", http.StatusNotFound)
//
// Use error codes in your handlers:
//
//	func getUserHandler(ctx iris.Context) {
//		userID := ctx.Params().Get("id")
//		user, err := userService.GetUser(userID)
//		if err != nil {
//			if errors.Is(err, sql.ErrNoRows) {
//				errors.NotFound.Message(ctx, "user not found")
//				return
//			}
//			errors.Internal.LogErr(ctx, err) // logs error and sends generic message
//			return
//		}
//		ctx.JSON(user)
//	}
//
// # Error Handling Strategies
//
// The package offers multiple approaches for error handling:
//
//  1. Direct error sending with custom messages:
//     errors.InvalidArgument.Detail(ctx, "validation failed", "email format is invalid")
//
//  2. Automatic error processing with HandleError:
//     if err := someOperation(); err != nil {
//     errors.HandleError(ctx, err) // automatically determines appropriate response
//     }
//
//  3. Error mapping and transformation:
//     opts := errors.NewOptions()
//     opts.MapErrors(errors.NotFound, sql.ErrNoRows, myapp.ErrResourceNotFound)
//
// # RFC 7807 Problem Details
//
// The package supports RFC 7807 problem details format for structured error responses:
//
//	errors.InvalidArgument.Problem(ctx,
//		"Validation Failed",
//		"The email field contains an invalid email address",
//		"https://example.com/problems/validation-failed",
//		"/users/create")
//
// # Error Logging and Collection
//
// Configure custom error logging and collection:
//
//	opts := errors.NewOptions().
//		SetLogger(func(ctx iris.Context, err error) {
//			// Custom logging logic
//			logger.Error("HTTP Error", "error", err, "path", ctx.Path())
//		}).
//		SetCollector(func(ctx iris.Context, err *errors.Error) {
//			// Send to monitoring service
//			monitoring.ReportError(err)
//		})
//
// # Content Negotiation
//
// The package automatically handles content negotiation, sending:
// - JSON responses by default
// - HTML error pages when Accept: text/html
// - Problem+JSON when using Problem method with appropriate renderer
//
// # Security Considerations
//
// The package distinguishes between public and private errors:
//   - Public errors show detailed messages to clients
//   - Private errors (marked with Private() or context.ErrPrivate) show generic messages
//     while logging full details server-side
//
// # Integration with Validation
//
// The package works with the validation sub-package for field-level validation
// errors, formatting them for the client without extra wiring.
//
// See the validation sub-package documentation for detailed validation capabilities.
package errors

import (
	"encoding/json"
	"encoding/json/jsontext"
	jsonv2 "encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"reflect"
	"slices"
	"strings"

	"github.com/kataras/iris/v14/client"
	"github.com/kataras/iris/v14/context"
	"github.com/kataras/iris/v14/core/pkg/jsonutil"
	"github.com/kataras/iris/v14/core/pkg/reflex"

	// "github.com/goccy/go-json"
	"github.com/iris-contrib/schema"
)

// func init() {
// 	context.SetHandlerName("iris/rest/errors.*", "iris.errors")
// }

// SkipCanceled is a package-level setting which by default
// skips the logging of a canceled response or operation.
// See the "Context.IsCanceled()" method and "iris.IsCanceled()" function
// that decide if the error is caused by a canceled operation.
//
// Change of this setting MUST be done on initialization of the program.
var SkipCanceled = true

type (
	// ErrorCodeName represents a canonical error identifier that maps to an HTTP status code.
	// It serves as a type-safe way to reference specific error types throughout your application
	// while providing methods for generating structured error responses.
	//
	// ErrorCodeName instances are typically created during application initialization using
	// the Register function and then used throughout the application to send consistent
	// error responses to clients.
	//
	// Key characteristics:
	// - Type-safe error identification
	// - Automatic HTTP status code mapping
	// - Multiple response formatting options (Message, Detail, Problem, etc.)
	// - Integration with logging and error collection systems
	// - Support for validation errors and custom data
	//
	// Example usage:
	//	var UserNotFound = errors.Register("USER_NOT_FOUND", http.StatusNotFound)
	//
	//	// In handler:
	//	UserNotFound.Detail(ctx, "user not found", "user with ID %s does not exist", userID)
	//
	// Available methods include Message, Detail, Problem, Data, Validation, Err, Log, and LogErr.
	// Each method provides different levels of detail and formatting options for error responses.
	ErrorCodeName string

	// ErrorCode represents the structured metadata for an HTTP error, containing both
	// the canonical name and associated HTTP status code. This type is used internally
	// for error processing and is included in JSON error responses sent to clients.
	//
	// The ErrorCode provides the bridge between logical error names (like "USER_NOT_FOUND")
	// and HTTP protocol specifics (like status code 404), enabling consistent error
	// handling across different layers of the application.
	//
	// This type is automatically populated when registering error codes and should not
	// be created directly by application code.
	ErrorCode struct {
		CanonicalName ErrorCodeName `json:"canonical_name" yaml:"CanonicalName"`
		Status        int           `json:"status" yaml:"Status"`
	}
)

// A read-only map of valid http error codes.
var errorCodeNameMap = make(map[ErrorCodeName]ErrorCode)
var codeNamesByStatusCode = make(map[int][]ErrorCodeName) // in order to find what errors based on status code.
var statusCodesOrder = make([]int, 0, 598)

// Register creates and registers a new canonical error name with its associated HTTP status code,
// returning an ErrorCodeName that can be used throughout the application for consistent error handling.
//
// This function should be called during application initialization (typically in package init() functions
// or early in main()) to register all custom error types before the HTTP server starts. The internal
// error registry is not protected by mutex for performance reasons.
//
// Parameters:
//   - httpErrorCanonicalName: A descriptive, uppercase string identifier for the error type.
//     Convention suggests using UPPER_SNAKE_CASE format (e.g., "USER_NOT_FOUND", "INVALID_CREDENTIALS").
//   - httpStatusCode: The HTTP status code that should be returned when this error occurs.
//
// Returns:
//   - ErrorCodeName: A typed identifier that can be used to generate error responses.
//
// Example usage:
//
//	var (
//		UserNotFound     = errors.Register("USER_NOT_FOUND", http.StatusNotFound)
//		InvalidEmail     = errors.Register("INVALID_EMAIL", http.StatusBadRequest)
//		PaymentRequired  = errors.Register("PAYMENT_REQUIRED", http.StatusPaymentRequired)
//	)
//
//	// Later in handlers:
//	func getUser(ctx iris.Context) {
//		if user == nil {
//			UserNotFound.Detail(ctx, "user not found", "user with ID %s does not exist", userID)
//			return
//		}
//	}
//
// See also RegisterErrorCode and RegisterErrorCodeMap for alternative registration methods.
//
// Thread Safety: This function must be called before the HTTP server starts and is not thread-safe.
// Calling this after server startup may result in undefined behavior.
func Register(httpErrorCanonicalName string, httpStatusCode int) ErrorCodeName {
	canonicalName := ErrorCodeName(httpErrorCanonicalName)
	RegisterErrorCode(canonicalName, httpStatusCode)
	return canonicalName
}

// RegisterErrorCode registers a canonical error name with its associated HTTP status code
// in the global error registry. This function is used internally by Register() and can be
// called directly when you need more control over the registration process.
//
// The function updates multiple internal data structures:
// - Maps the canonical name to its ErrorCode metadata
// - Maintains ordered lists of status codes for consistent error processing
// - Groups error names by status code for lookup operations
//
// Parameters:
//   - canonicalName: The ErrorCodeName identifier for this error type
//   - httpStatusCode: The HTTP status code that should be returned for this error
//
// Example usage:
//
//	const UserBlocked ErrorCodeName = "USER_BLOCKED"
//	errors.RegisterErrorCode(UserBlocked, http.StatusForbidden)
//
// Thread Safety: This function must be called during application initialization before
// the HTTP server starts. The internal registry is not protected by mutex for performance
// reasons, making concurrent registration unsafe.
//
// See also Register() for a more convenient registration method that returns the ErrorCodeName.
func RegisterErrorCode(canonicalName ErrorCodeName, httpStatusCode int) {
	errorCodeNameMap[canonicalName] = ErrorCode{
		CanonicalName: canonicalName,
		Status:        httpStatusCode,
	}

	if !slices.Contains(statusCodesOrder, httpStatusCode) { // add the status code only once, so we can loop through them faster than i:=0; i<=599.
		statusCodesOrder = append(statusCodesOrder, httpStatusCode)
		// sort.Sort(sort.IntSlice(statusCodesOrder))
		slices.Sort(statusCodesOrder) // sort by asceding order.
	}

	statusesNames := codeNamesByStatusCode[httpStatusCode]
	if statusCanonicalNameExists := slices.Contains(statusesNames, canonicalName); !statusCanonicalNameExists {
		// More than one canonical name per status code exists.
		codeNamesByStatusCode[httpStatusCode] = append(statusesNames, canonicalName)
	}
}

// RegisterErrorCodeMap registers multiple error codes at once from a map of canonical names
// to HTTP status codes. This function is useful when you have many error codes to register
// or when loading error configurations from external sources.
//
// The function iterates through the provided map and calls RegisterErrorCode for each entry,
// maintaining all the same guarantees and requirements as individual registration.
//
// Parameters:
//   - errorMap: A map where keys are ErrorCodeName identifiers and values are HTTP status codes
//
// Example usage:
//
//	errorMap := map[errors.ErrorCodeName]int{
//		"USER_NOT_FOUND":     http.StatusNotFound,
//		"INVALID_TOKEN":      http.StatusUnauthorized,
//		"QUOTA_EXCEEDED":     http.StatusTooManyRequests,
//		"MAINTENANCE_MODE":   http.StatusServiceUnavailable,
//	}
//	errors.RegisterErrorCodeMap(errorMap)
//
// The function safely handles empty maps and will return immediately without side effects
// if the provided map is nil or empty.
//
// Thread Safety: Like other registration functions, this must be called during application
// initialization before the HTTP server starts. Concurrent registration is not safe.
//
// See also Register() and RegisterErrorCode() for individual error registration.
func RegisterErrorCodeMap(errorMap map[ErrorCodeName]int) {
	if len(errorMap) == 0 {
		return
	}

	for canonicalName, httpStatusCode := range errorMap {
		RegisterErrorCode(canonicalName, httpStatusCode)
	}
}

// List of default error codes a server should follow and send back to the client.
//
// Known error code names:
// - InvalidArgument
// - FailedPrecondition
// - OutOfRange
// - UnsupportedMediaType
// - DeadlineExceeded
// - NotFound
// - MethodNotAllowed
// - AlreadyExists
// - Aborted
// - PermissionDenied
// - Unauthenticated
// - ResourceExhausted
// - TooLarge
// - Unimplemented
// - Unavailable
// - BadGateway
// - Internal
// - DataLoss
// - Unknown
// - PaymentRequired
// - NotAcceptable
// - ProxyAuthRequired
// - RequestTimeout
// - Cancelled
// - Gone
// - LengthRequired
// - PreconditionFailed
// - RequestURITooLong
// - RequestedRangeNotSatisfiable
// - ExpectationFailed
// - ImATeapot
// - MisdirectedRequest
// - UnprocessableEntity
// - Locked
// - FailedDependency
// - TooEarly
// - UpgradeRequired
// - PreconditionRequired
// - RequestHeaderFieldsTooLarge
// - UnavailableForLegalReasons
// - HTTPVersionNotSupported
// - VariantAlsoNegotiates
// - InsufficientStorage
// - LoopDetected
// - NotExtended
// - NetworkAuthenticationRequired
// - PageExpired
// - BlockedByWindowsParentalControls
// - InvalidToken
// - TokenRequired
// - BandwidthLimitExceeded
// - InvalidSSLCertificate
// - SiteOverloaded
// - SiteFrozen
// - NetworkReadTimeout
var (
	// InvalidArgument indicates that the client specified an invalid argument.
	InvalidArgument ErrorCodeName = Register("INVALID_ARGUMENT", http.StatusBadRequest)
	// FailedPrecondition indicates that the operation was rejected because the system is not in a state required for the operation's execution.
	FailedPrecondition ErrorCodeName = Register("FAILED_PRECONDITION", http.StatusBadRequest)
	// OutOfRange indicates that an operation was attempted past the valid range.
	OutOfRange ErrorCodeName = Register("OUT_OF_RANGE", http.StatusBadRequest)
	// UnsupportedMediaType indicates that the request entity has a media type which the server or resource does not support.
	UnsupportedMediaType ErrorCodeName = Register("UNSUPPORTED_MEDIA_TYPE", http.StatusUnsupportedMediaType)
	// DeadlineExceeded indicates that the deadline expired before the operation could complete.
	DeadlineExceeded ErrorCodeName = Register("DEADLINE_EXCEEDED", http.StatusGatewayTimeout)
	// NotFound indicates that the requested resource could not be found.
	NotFound ErrorCodeName = Register("NOT_FOUND", http.StatusNotFound)
	// MethodNotAllowed indicates that the HTTP method used is not supported by the resource.
	MethodNotAllowed ErrorCodeName = Register("METHOD_NOT_ALLOWED", http.StatusMethodNotAllowed)
	// AlreadyExists indicates that the resource that a client tried to create already exists.
	AlreadyExists ErrorCodeName = Register("ALREADY_EXISTS", http.StatusConflict)
	// Aborted indicates that the operation was aborted, typically due to a concurrency issue.
	Aborted ErrorCodeName = Register("ABORTED", http.StatusConflict)
	// PermissionDenied indicates that the caller does not have permission to execute the specified operation.
	PermissionDenied ErrorCodeName = Register("PERMISSION_DENIED", http.StatusForbidden)
	// Unauthenticated indicates that the request does not have valid authentication credentials for the operation.
	Unauthenticated ErrorCodeName = Register("UNAUTHENTICATED", http.StatusUnauthorized)
	// ResourceExhausted indicates that some resource has been exhausted, perhaps a per-user quota, or perhaps the entire file system is out of space.
	ResourceExhausted ErrorCodeName = Register("RESOURCE_EXHAUSTED", http.StatusTooManyRequests)
	// TooLarge indicates that the request entity is larger than limits defined by server.
	TooLarge ErrorCodeName = Register("TOO_LARGE", http.StatusRequestEntityTooLarge)
	// Unimplemented indicates that the operation is not implemented or not supported/enabled in this service.
	Unimplemented ErrorCodeName = Register("UNIMPLEMENTED", http.StatusNotImplemented)
	// Unavailable indicates that the service is currently unavailable.
	Unavailable ErrorCodeName = Register("UNAVAILABLE", http.StatusServiceUnavailable)
	// BadGateway indicates that an upstream this service depends on failed.
	//
	// It is distinct from Unavailable, which says this service cannot serve the
	// request. Any proxy or gateway needs the difference, and the set had 503 and
	// 504 without it.
	BadGateway ErrorCodeName = Register("BAD_GATEWAY", http.StatusBadGateway)
	// Internal indicates an internal server error.
	Internal ErrorCodeName = Register("INTERNAL", http.StatusInternalServerError)
	// DataLoss indicates unrecoverable data loss or corruption.
	DataLoss ErrorCodeName = Register("DATA_LOSS", http.StatusInternalServerError)
	// Unknown indicates an unknown error.
	Unknown ErrorCodeName = Register("UNKNOWN", http.StatusInternalServerError)

	// PaymentRequired indicates that payment is required to access the resource.
	PaymentRequired ErrorCodeName = Register("PAYMENT_REQUIRED", http.StatusPaymentRequired)
	// NotAcceptable indicates that the resource is not capable of generating content acceptable according to the Accept headers sent in the request.
	NotAcceptable ErrorCodeName = Register("NOT_ACCEPTABLE", http.StatusNotAcceptable)
	// ProxyAuthRequired indicates that the client must first authenticate itself with the proxy.
	ProxyAuthRequired ErrorCodeName = Register("PROXY_AUTH_REQUIRED", http.StatusProxyAuthRequired)
	// RequestTimeout indicates that the server timed out waiting for the request.
	RequestTimeout ErrorCodeName = Register("REQUEST_TIMEOUT", http.StatusRequestTimeout)
	// Cancelled indicates that the operation was cancelled, typically by the
	// caller (gRPC CANCELLED).
	//
	// The status is 408 Request Timeout, the closest standard status for "the
	// client stopped waiting". The non-standard 499 that nginx logs for a
	// client-closed request is already taken by TokenRequired
	// (context.StatusTokenRequired) and stays there.
	Cancelled ErrorCodeName = Register("CANCELLED", http.StatusRequestTimeout)
	// Gone indicates that the resource requested is no longer available and will not be available again.
	Gone ErrorCodeName = Register("GONE", http.StatusGone)
	// LengthRequired indicates that the request did not specify the length of its content, which is required by the requested resource.
	LengthRequired ErrorCodeName = Register("LENGTH_REQUIRED", http.StatusLengthRequired)
	// PreconditionFailed indicates that the server does not meet one of the preconditions that the requester put on the request.
	PreconditionFailed ErrorCodeName = Register("PRECONDITION_FAILED", http.StatusPreconditionFailed)
	// RequestURITooLong indicates that the URI provided was too long for the server to process.
	RequestURITooLong ErrorCodeName = Register("REQUEST_URI_TOO_LONG", http.StatusRequestURITooLong)
	// RequestedRangeNotSatisfiable indicates that the server cannot supply the portion of the file requested.
	RequestedRangeNotSatisfiable ErrorCodeName = Register("REQUESTED_RANGE_NOT_SATISFIABLE", http.StatusRequestedRangeNotSatisfiable)
	// ExpectationFailed indicates that the server cannot meet the requirements of the Expect request-header field.
	ExpectationFailed ErrorCodeName = Register("EXPECTATION_FAILED", http.StatusExpectationFailed)
	// ImATeapot indicates that the server refuses to brew coffee because it is, permanently, a teapot.
	ImATeapot ErrorCodeName = Register("IM_A_TEAPOT", http.StatusTeapot)
	// MisdirectedRequest indicates that the request was directed at a server that is not able to produce a response.
	MisdirectedRequest ErrorCodeName = Register("MISDIRECTED_REQUEST", http.StatusMisdirectedRequest)
	// UnprocessableEntity indicates that the server understands the content type of the request entity, but was unable to process the contained instructions.
	UnprocessableEntity ErrorCodeName = Register("UNPROCESSABLE_ENTITY", http.StatusUnprocessableEntity)
	// Locked indicates that the resource that is being accessed is locked.
	Locked ErrorCodeName = Register("LOCKED", http.StatusLocked)
	// FailedDependency indicates that the request failed due to failure of a previous request.
	FailedDependency ErrorCodeName = Register("FAILED_DEPENDENCY", http.StatusFailedDependency)
	// TooEarly indicates that the server is unwilling to risk processing a request that might be replayed.
	TooEarly ErrorCodeName = Register("TOO_EARLY", http.StatusTooEarly)
	// UpgradeRequired indicates that the client should switch to a different protocol.
	UpgradeRequired ErrorCodeName = Register("UPGRADE_REQUIRED", http.StatusUpgradeRequired)
	// PreconditionRequired indicates that the origin server requires the request to be conditional.
	PreconditionRequired ErrorCodeName = Register("PRECONDITION_REQUIRED", http.StatusPreconditionRequired)
	// RequestHeaderFieldsTooLarge indicates that the server is unwilling to process the request because its header fields are too large.
	RequestHeaderFieldsTooLarge ErrorCodeName = Register("REQUEST_HEADER_FIELDS_TOO_LARGE", http.StatusRequestHeaderFieldsTooLarge)
	// UnavailableForLegalReasons indicates that the resource is unavailable for legal reasons.
	UnavailableForLegalReasons ErrorCodeName = Register("UNAVAILABLE_FOR_LEGAL_REASONS", http.StatusUnavailableForLegalReasons)
	// HTTPVersionNotSupported indicates that the server does not support the HTTP protocol version used in the request.
	HTTPVersionNotSupported ErrorCodeName = Register("HTTP_VERSION_NOT_SUPPORTED", http.StatusHTTPVersionNotSupported)
	// VariantAlsoNegotiates indicates that the server has an internal configuration error.
	VariantAlsoNegotiates ErrorCodeName = Register("VARIANT_ALSO_NEGOTIATES", http.StatusVariantAlsoNegotiates)
	// InsufficientStorage indicates that the server is unable to store the representation needed to complete the request.
	InsufficientStorage ErrorCodeName = Register("INSUFFICIENT_STORAGE", http.StatusInsufficientStorage)
	// LoopDetected indicates that the server detected an infinite loop while processing a request.
	LoopDetected ErrorCodeName = Register("LOOP_DETECTED", http.StatusLoopDetected)
	// NotExtended indicates that further extensions to the request are required for the server to fulfill it.
	NotExtended ErrorCodeName = Register("NOT_EXTENDED", http.StatusNotExtended)
	// NetworkAuthenticationRequired indicates that the client needs to authenticate to gain network access.
	NetworkAuthenticationRequired ErrorCodeName = Register("NETWORK_AUTHENTICATION_REQUIRED", http.StatusNetworkAuthenticationRequired)

	//
	// Iris-specific ones.
	//

	// PageExpired indicates that the page has expired.
	PageExpired ErrorCodeName = Register("PAGE_EXPIRED", context.StatusPageExpired)
	// BlockedByWindowsParentalControls indicates that the request was blocked by Windows Parental Controls.
	BlockedByWindowsParentalControls ErrorCodeName = Register("BLOCKED_BY_WINDOWS_PARENTAL_CONTROLS", context.StatusBlockedByWindowsParentalControls)
	// InvalidToken indicates that the token provided is invalid.
	InvalidToken ErrorCodeName = Register("INVALID_TOKEN", context.StatusInvalidToken)
	// TokenRequired indicates that a token is required to access the resource.
	TokenRequired ErrorCodeName = Register("TOKEN_REQUIRED", context.StatusTokenRequired)
	// BandwidthLimitExceeded indicates that the bandwidth limit has been exceeded.
	BandwidthLimitExceeded ErrorCodeName = Register("BANDWIDTH_LIMIT_EXCEEDED", context.StatusBandwidthLimitExceeded)
	// InvalidSSLCertificate indicates that the SSL certificate is invalid.
	InvalidSSLCertificate ErrorCodeName = Register("INVALID_SSL_CERTIFICATE", context.StatusInvalidSSLCertificate)
	// SiteOverloaded indicates that the site is overloaded.
	SiteOverloaded ErrorCodeName = Register("SITE_OVERLOADED", context.StatusSiteOverloaded)
	// SiteFrozen indicates that the site is frozen.
	SiteFrozen ErrorCodeName = Register("SITE_FROZEN", context.StatusSiteFrozen)
	// NetworkReadTimeout indicates that there was a network read timeout error.
	NetworkReadTimeout ErrorCodeName = Register("NETWORK_READ_TIMEOUT", context.StatusNetworkReadTimeout)
)

var (
	// ErrRequest is a sentinel error which can be used to indicate that error came from the request body.
	ErrRequest = fmt.Errorf("request error")
	// ErrResponse is a sentinel error which can be used to indicate that error came from the server.
	ErrResponse = fmt.Errorf("response error")
)

// HandleError provides automatic error processing and client response generation based on
// error type analysis and registered error mappings. This function serves as the primary
// entry point for error handling in Iris applications using this package.
//
// The function performs intelligent error analysis and routing:
//
// 1. Determines if the error originated from client request vs server processing
// 2. Checks for validation errors and formats them appropriately
// 3. Applies registered error mappings (see Options.MapErrors, MapErrorFunc)
// 4. Handles special error types (database errors, JSON errors, API errors)
// 5. Wraps unrecognized errors with appropriate error codes
// 6. Sends formatted responses to clients
//
// Error Processing Priority:
//   - ValidationError/ValidationErrors: Formatted as field-level validation failures
//   - ErrorCodeName wrapped errors: Uses the specific error code for response
//   - Mapped errors: Applied through ErrorCodeMap transformations
//   - Registered ErrorResolvers: e.g. the pgerrors subpackage maps
//     pg.ErrNoRows → NotFound and duplicate key → AlreadyExists
//   - JSON parsing errors: Detailed syntax error information
//   - API errors: Proxied from remote services
//   - Unknown errors: Wrapped as Internal server errors
//
// Parameters:
//   - ctx: The Iris request context for the current HTTP request
//   - err: The error to be processed and sent to the client
//
// Returns:
//   - bool: true if the error was successfully handled and a response sent, false otherwise
//
// Example usage:
//
//	func createUser(ctx iris.Context) {
//		var user User
//		if err := ctx.ReadJSON(&user); err != nil {
//			if errors.HandleError(ctx, err) {
//				return // Error was handled and response sent
//			}
//		}
//
//		if err := userService.CreateUser(user); err != nil {
//			errors.HandleError(ctx, err) // Automatic error type detection
//			return
//		}
//
//		ctx.JSON(user)
//	}
//
// The function respects the ErrRequest sentinel error for distinguishing between
// client-side and server-side errors, affecting whether errors are logged as
// InvalidArgument (400) or Internal (500) responses.
//
// See also HandleRequestError for forcing client-side error interpretation,
// and the Options type for configuring error mapping and processing behavior.
func HandleError(ctx *context.Context, err error) bool {
	if err == nil {
		return false
	}

	if ctx.IsStopped() {
		return false
	}

	if loggedAfterBody(ctx, err) {
		return true
	}

	// Only the ErrRequest sentinel marks a client error here. A private
	// error (context.ErrPrivate) is a server error like any other
	// unrecognized one: it renders as Internal with a generic message and
	// its text stays in the log.
	isRequestErr := errors.Is(err, ErrRequest)
	return handleError(ctx, !isRequestErr, err)
}

// loggedAfterBody reports whether a body already went out on this response.
// Then no error document can follow it (the status and the partial body are
// committed), so the error is only logged. Callers report it as handled so
// the handler stops instead of writing more.
func loggedAfterBody(ctx *context.Context, err error) bool {
	if ctx.ResponseWriter().Written() <= 0 {
		return false
	}

	getReadOnlyOptions(ctx).logError(ctx, err)
	ctx.StopExecution()
	return true
}

// HandleRequestError processes errors with the assumption that they originated from client
// request issues rather than server-side problems. This function behaves identically to
// HandleError except that unrecognized errors are treated as client errors (InvalidArgument/400)
// rather than server errors (Internal/500).
//
// This function is useful in scenarios where you're processing user input, validating
// request data, or handling operations where errors are likely to be caused by invalid
// client requests rather than internal server issues.
//
// Parameters:
//   - ctx: The Iris request context for the current HTTP request
//   - err: The error to be processed as a client request error
//
// Returns:
//   - bool: true if the error was successfully handled and a response sent, false otherwise
//
// Example usage:
//
//	func validateUserInput(ctx iris.Context) {
//		var input UserInput
//		if err := ctx.ReadJSON(&input); err != nil {
//			errors.HandleRequestError(ctx, err) // Treat as client error (400)
//			return
//		}
//
//		if err := validateBusinessRules(input); err != nil {
//			errors.HandleRequestError(ctx, err) // Client sent invalid data
//			return
//		}
//	}
//
// The key difference from HandleError is error classification:
// - HandleError: Unknown errors → Internal (500)
// - HandleRequestError: Unknown errors → InvalidArgument (400)
//
// All other error processing logic (validation errors, mapped errors, etc.) remains identical.
//
// See HandleError for complete error processing documentation.
func HandleRequestError(ctx *context.Context, err error) bool {
	if err == nil {
		return false
	}

	if ctx.IsStopped() {
		return false
	}

	if loggedAfterBody(ctx, err) {
		return true
	}

	return handleError(ctx, false, err)
}

// HandleErrorf wraps "err" with request-specific detail and handles the result
// exactly like HandleError. The wrap uses %w, so mappings registered through
// Options.MapErrors, MapErrorTypes and errors.Is/As classification still apply:
//
//	// renders the mapped 404 with message "plan with id: 42: plan not found".
//	errors.HandleErrorf(ctx, err, "plan with id: %s", id)
//
// Mapping functions that inspect the error text observe the prefixed message.
// It returns false without writing a response when "err" is nil.
func HandleErrorf(ctx *context.Context, err error, format string, args ...any) bool {
	if err == nil {
		return false
	}

	return HandleError(ctx, fmt.Errorf(format+": %w", appendErr(args, err)...))
}

// appendErr returns args followed by err in a fresh slice: appending in place
// could write err into the caller's backing array when it has spare capacity.
func appendErr(args []any, err error) []any {
	out := make([]any, 0, len(args)+1)
	return append(append(out, args...), err)
}

// HandleRequestErrorf is the HandleRequestError counterpart of HandleErrorf:
// same wrapping and classification, but unrecognized errors render as client
// errors (InvalidArgument) instead of internal ones.
func HandleRequestErrorf(ctx *context.Context, err error, format string, args ...any) bool {
	if err == nil {
		return false
	}

	return HandleRequestError(ctx, fmt.Errorf(format+": %w", appendErr(args, err)...))
}

// jsonKindName maps a Go type to a JSON type name, so a decode failure can be
// reported without disclosing the server's own type names.
func jsonKindName(typ reflect.Type) string {
	if typ == nil {
		return "a different type"
	}

	switch typ.Kind() {
	case reflect.String:
		return "string"
	case reflect.Bool:
		return "boolean"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return "number"
	case reflect.Slice, reflect.Array:
		return "array"
	case reflect.Map, reflect.Struct:
		return "object"
	case reflect.Pointer, reflect.Interface:
		return jsonKindName(typ.Elem())
	default:
		return "a different type"
	}
}

// jsonKindOfToken is the wire-side sibling of jsonKindName: it names the JSON
// kind a json/v2 semantic error reports for the client's own input.
func jsonKindOfToken(k jsontext.Kind) string {
	switch k {
	case 'n':
		return "null"
	case 't', 'f':
		return "boolean"
	case '"':
		return "string"
	case '0':
		return "number"
	case '{':
		return "object"
	case '[':
		return "array"
	default:
		return "a different kind"
	}
}

// returns true if error was non-nil and it's handled.
func handleError(ctx *context.Context, serverError bool, err error) bool {
	if err == nil {
		return false
	}

	if handleErrorValue(ctx, serverError, err) {
		return true
	}

	// Note(@kataras): honour "serverError" on the fallback path too.
	//
	// It was threaded all the way down here and then discarded, so every
	// unrecognized error rendered as Internal (500) plus a server-side error log
	// - contradicting HandleRequestError's documented contract right above.
	// The decoders wrap XML, YAML, MsgPack and protobuf failures as plain
	// fmt.Errorf values that no mapping matches, so one malformed byte in an XML
	// body answered 500 and wrote an error line, at request rate, from an
	// unauthenticated client.
	if serverError {
		Internal.LogErr(ctx, err)
	} else {
		InvalidArgument.Err(ctx, err)
	}

	return true
}

func handleErrorValue(ctx *context.Context, serverError bool, err error) bool {
	opts := getReadOnlyOptions(ctx)
	// DBUG.
	// fmt.Printf("Incoming error: %#+v\n", err)
	// if opts != nil && len(opts.codeMap) > 0 {
	// 	for c, es := range opts.codeMap {
	// 		fmt.Printf("error code: %s, funcs len: %d\n", c.String(), len(es))
	// 		for _, e := range es {
	// 			if e == nil {
	// 				fmt.Printf("error func is nil!\n")
	// 			}
	// 			fmt.Printf("result error: %#+v\n", e(err))
	// 		}
	// 	}
	// }
	//
	if opts.handleError(ctx, err) {
		return true
	}

	if resolveError(ctx, err) {
		// Handled by a registered ErrorResolver (e.g. the pgerrors
		// subpackage's PostgreSQL mapping).
		return true
	}

	if errors.Is(err, context.ErrRequestEntityTooLarge) {
		TooLarge.Message(ctx, "request body too large")
		return true
	}

	if recoverErr, ok := err.(*context.ErrPanicRecovery); ok {
		Internal.LogErr(ctx, recoverErr)
		ctx.StopExecution()
		return true
	}

	if m, ok := err.(schema.MultiError); ok {
		msg := "invalid request value"
		if len(m) > 1 {
			msg += "s"
		}

		// convert schema: error converting value for \"size\" (and 1 other error)"
		// To: "invalid request value" and "error converting value for  \"size\" and error converting value for \"size\"
		errorMessages := make([]string, 0, len(m))
		for _, e := range m {
			errMsg := e.Error()
			errMsg = strings.TrimPrefix(errMsg, "schema: ")

			errorMessages = append(errorMessages, errMsg)
		}

		InvalidArgument.Detail(ctx, msg, strings.Join(errorMessages, " and "))
		return true
	}

	if handleJSONError(ctx, serverError, err) {
		return true
	}

	// Unwrap and collect the errors slice so the error result doesn't contain the ErrorCodeName type
	// and fire the error status code and title based on this error code name itself.
	var asErrCode ErrorCodeName
	if As(err, &asErrCode) {
		if unwrapJoined, ok := err.(joinedErrors); ok {
			errs := unwrapJoined.Unwrap()
			errsToKeep := make([]error, 0, len(errs)-1)
			for _, src := range errs {
				if _, isErrorCodeName := src.(ErrorCodeName); !isErrorCodeName {
					errsToKeep = append(errsToKeep, src)
				}
			}

			if len(errsToKeep) > 0 {
				err = errors.Join(errsToKeep...)
			}
		}

		asErrCode.Err(ctx, err)
		return true
	}

	if validationError, ok := errors.AsType[ValidationError](err); ok {
		if validationError == nil {
			return false // consider as not error for any case, this should never happen.
		}

		if validationError.GetReason() == "" {
			return false // consider as not error if the validation error has no reason.
		}

		InvalidArgument.Validation(ctx, validationError)
		return true
	}

	if validationErrors, ok := errors.AsType[ValidationErrors](err); ok {
		if len(validationErrors) == 0 {
			return false // consider as not error for any case, this should never happen.
		}

		InvalidArgument.Validation(ctx, validationErrors...)
		return true
	}

	if apiErr, ok := client.GetError(err); ok {
		handleAPIError(ctx, apiErr)
		return true
	}

	return false // not handled by this function.
}

// String returns the string representation of the ErrorCodeName, implementing the fmt.Stringer interface.
// This is primarily used for logging, debugging, and error message formatting.
//
// Example output: "USER_NOT_FOUND", "INVALID_CREDENTIALS", "INTERNAL"
func (e ErrorCodeName) String() string {
	return string(e)
}

// Error implements the error interface for ErrorCodeName, returning an empty string.
// This method exists solely to make ErrorCodeName compatible with Go's error interface,
// enabling it to be used in error wrapping and joining operations.
//
// The empty string return is intentional - ErrorCodeName serves as a tag/marker
// in error chains rather than contributing descriptive text. The actual error
// messages are provided through the various response methods (Message, Detail, etc.).
//
// Usage in error wrapping:
//
//	err = errors.InvalidArgument.Wrap(originalErr)
//	// or
//	err = fmt.Errorf("validation failed: %w", errors.InvalidArgument)
//
// This allows error handling code to detect specific error types:
//
//	var argErr errors.ErrorCodeName
//	if errors.As(err, &argErr) {
//		// Handle specific error code
//	}
func (e ErrorCodeName) Error() string {
	return ""
}

type joinedErrors interface{ Unwrap() []error }

// Wrap creates a new error by joining the ErrorCodeName with the provided error,
// enabling the HandleError function to automatically detect and process the error
// with the appropriate error code and HTTP status code.
//
// This method is particularly useful when you want to tag an error with a specific
// error code without immediately sending a response to the client. The wrapped error
// can be returned from functions and later processed by HandleError or other error
// handling mechanisms.
//
// Parameters:
//   - err: The original error to be wrapped with this error code
//
// Returns:
//   - error: A joined error containing both the ErrorCodeName and the original error
//
// Example usage:
//
//	func getUserFromDB(userID string) (*User, error) {
//		user, err := db.GetUser(userID)
//		if err != nil {
//			if errors.Is(err, sql.ErrNoRows) {
//				return nil, errors.NotFound.Wrap(err)
//			}
//			return nil, errors.Internal.Wrap(err)
//		}
//		return user, nil
//	}
//
//	// In handler:
//	func getUserHandler(ctx iris.Context) {
//		user, err := getUserFromDB(ctx.Params().Get("id"))
//		if err != nil {
//			errors.HandleError(ctx, err) // Automatically detects error code
//			return
//		}
//		ctx.JSON(user)
//	}
//
// The wrapped error maintains the original error's information while adding
// error code metadata that can be detected during error processing.
//
// See HandleError function for automatic error code detection and processing.
func (e ErrorCodeName) Wrap(err error) error {
	return errors.Join(e, err)
}

// Message sends an immediate HTTP error response to the client with the specified message
// and the error code's associated HTTP status code. This is the simplest method for
// sending error responses when you only need to provide a basic error message.
//
// The response will be formatted according to the configured renderer (JSON by default)
// and will include the error code's canonical name and status code in the response structure.
//
// Parameters:
//   - ctx: The Iris request context for the current HTTP request
//   - message: The human-readable error message to send to the client
//
// Example usage:
//
//	errors.NotFound.Message(ctx, "user not found")
//	errors.InvalidArgument.Message(ctx, "email address is required")
//	errors.Internal.Message(ctx, "service temporarily unavailable")
//
// Response example (JSON):
//
//	{
//		"id": "req-123456",
//		"http_error_code": {
//			"canonical_name": "NOT_FOUND",
//			"status": 404
//		},
//		"message": "user not found"
//	}
//
// This method immediately stops request processing and sends the response.
// Use Wrap() instead if you need to return errors from functions for later processing.
//
// See also Detail() for messages with additional details, and Problem() for RFC 7807 format.
func (e ErrorCodeName) Message(ctx *context.Context, message string) {
	fail(ctx, e, message, "", "", "", nil, nil, false)
}

// Messagef sends an HTTP error response with a formatted message, providing a convenient
// way to include dynamic values in error messages without manual string formatting.
//
// This method is equivalent to calling Message(ctx, fmt.Sprintf(format, args...)) but
// with optimized formatting that only calls fmt.Sprintf when arguments are provided.
//
// Parameters:
//   - ctx: The Iris request context for the current HTTP request
//   - format: The format string for the error message (printf-style)
//   - args: Arguments to be formatted into the message
//
// Example usage:
//
//	errors.NotFound.Messagef(ctx, "user with ID %s not found", userID)
//	errors.InvalidArgument.Messagef(ctx, "field %q must be between %d and %d characters", fieldName, min, max)
//	errors.TooLarge.Messagef(ctx, "file size %d bytes exceeds maximum of %d bytes", actualSize, maxSize)
//
// Response example (JSON):
//
//	{
//		"id": "req-123456",
//		"http_error_code": {
//			"canonical_name": "NOT_FOUND",
//			"status": 404
//		},
//		"message": "user with ID abc123 not found"
//	}
//
// This method immediately stops request processing and sends the response.
func (e ErrorCodeName) Messagef(ctx *context.Context, format string, args ...any) {
	fail(ctx, e, sprintf(format, args...), "", "", "", nil, nil, false)
}

// Detail sends an HTTP error response with both a main message and additional detail information.
// This method is ideal when you need to provide both a concise error summary and more specific
// information about what went wrong, helping clients understand both what happened and why.
//
// The detail field provides additional context that can be useful for debugging, user guidance,
// or programmatic error handling by client applications.
//
// Parameters:
//   - ctx: The Iris request context for the current HTTP request
//   - msg: The primary error message - a concise summary of what went wrong
//   - detail: Additional detailed information about the error, its cause, or resolution steps
//
// Example usage:
//
//	errors.InvalidArgument.Detail(ctx, "validation failed", "email field contains an invalid email address")
//	errors.NotFound.Detail(ctx, "resource not found", "user with ID 'user123' does not exist in the system")
//	errors.PermissionDenied.Detail(ctx, "access denied", "user lacks the 'admin' role required for this operation")
//
// Response example (JSON):
//
//	{
//		"id": "req-123456",
//		"http_error_code": {
//			"canonical_name": "INVALID_ARGUMENT",
//			"status": 400
//		},
//		"message": "validation failed",
//		"detail": "email field contains an invalid email address"
//	}
//
// This method immediately stops request processing and sends the response.
//
// See also Message() for simple messages, and Problem() for RFC 7807 problem details format.
func (e ErrorCodeName) Detail(ctx *context.Context, msg, detail string) {
	fail(ctx, e, msg, detail, "", "", nil, nil, false)
}

// Detailf sends an HTTP error response with a main message and a formatted detail string,
// providing a convenient way to include dynamic values in the detailed error information.
//
// This method combines the structured approach of Detail() with the formatting convenience
// of printf-style string formatting for the detail field.
//
// Parameters:
//   - ctx: The Iris request context for the current HTTP request
//   - msg: The primary error message - a concise summary of what went wrong
//   - detail: The format string for the detailed error information (printf-style)
//   - detailArgs: Arguments to be formatted into the detail string
//
// Example usage:
//
//	errors.InvalidArgument.Detailf(ctx, "validation failed",
//		"field %q has length %d but must be between %d and %d characters",
//		fieldName, actualLength, minLength, maxLength)
//
//	errors.NotFound.Detailf(ctx, "user not found",
//		"user with ID %s was not found in organization %s", userID, orgID)
//
//	errors.TooLarge.Detailf(ctx, "file too large",
//		"uploaded file size %d MB exceeds limit of %d MB", actualMB, limitMB)
//
// Response example (JSON):
//
//	{
//		"id": "req-123456",
//		"http_error_code": {
//			"canonical_name": "INVALID_ARGUMENT",
//			"status": 400
//		},
//		"message": "validation failed",
//		"detail": "field \"email\" has length 5 but must be between 10 and 100 characters"
//	}
//
// This method immediately stops request processing and sends the response.
func (e ErrorCodeName) Detailf(ctx *context.Context, msg, detail string, detailArgs ...any) {
	e.Detail(ctx, msg, sprintf(detail, detailArgs...))
}

// Problem sends an RFC 7807 compliant "Problem Details for HTTP APIs" error response.
// This method creates structured error responses that follow the RFC 7807 standard,
// providing machine-readable error information that can be easily processed by clients.
//
// RFC 7807 defines a standard format for HTTP error responses that includes:
// - type: A URI reference identifying the problem type
// - title: A short, human-readable summary of the problem type
// - status: The HTTP status code (automatically set from the ErrorCodeName)
// - detail: A human-readable explanation specific to this occurrence
// - instance: A URI reference identifying the specific occurrence
//
// The response will be rendered as "application/problem+json" if no custom renderer
// is configured, otherwise it uses the configured renderer with problem details.
//
// Parameters:
//   - ctx: The Iris request context for the current HTTP request
//   - title: A short, human-readable summary of the problem type
//   - detail: A human-readable explanation specific to this occurrence of the problem
//   - typeURI: A URI reference that identifies the problem type (e.g., "https://example.com/probs/out-of-credit")
//   - instanceURI: A URI reference that identifies the specific occurrence (e.g., "/account/kataras/msgs/1")
//   - extra: Optional additional data to include in the problem details
//
// Example usage:
//
//	errors.InvalidArgument.Problem(ctx,
//		"Validation Failed",
//		"The email field contains an invalid email address",
//		"https://example.com/problems/validation-failed",
//		"/users/create")
//
// Response example (application/problem+json):
//
//	{
//		"type": "https://example.com/problems/validation-failed",
//		"title": "Validation Failed",
//		"status": 400,
//		"detail": "The email field contains an invalid email address",
//		"instance": "/users/create"
//	}
//
// Multiple extra values are combined into an array, single values are included directly.
// Nil values in extra are automatically filtered out.
//
// This method immediately stops request processing and sends the response.
// See RFC 7807: https://datatracker.ietf.org/doc/html/rfc7807 for complete specification.
func (e ErrorCodeName) Problem(ctx *context.Context, title, detail, typeURI, instanceURI string, extra ...any) {
	msg := title

	var dataValue any
	switch len(extra) {
	case 0:
		dataValue = nil
	case 1:
		dataValue = extra[0]
	default:
		values := make([]any, 0, len(extra))
		for _, v := range extra {
			if v == nil {
				continue
			}

			values = append(values, v)
		}

		dataValue = values
	}

	fail(ctx, e, msg, detail, typeURI, instanceURI, nil, dataValue, false)
}

// Data sends an HTTP error response with a message and additional structured data.
// This method is useful when you need to include extra information beyond the basic
// error message, such as validation details, suggested actions, or context-specific data.
//
// The data parameter can be any value that can be marshaled to JSON, including structs,
// maps, slices, or primitive values. The data will be included in the "data" field
// of the JSON error response.
//
// Parameters:
//   - ctx: The Iris request context for the current HTTP request
//   - msg: The primary error message to send to the client
//   - data: Additional structured data to include in the error response
//
// Example usage:
//
//	// Simple data
//	errors.InvalidArgument.Data(ctx, "validation failed", map[string]string{
//		"field": "email",
//		"requirement": "valid email format",
//	})
//
//	// Complex structured data
//	errors.NotFound.Data(ctx, "resource not found", struct {
//		ResourceType string   `json:"resource_type"`
//		SearchCriteria []string `json:"search_criteria"`
//		Suggestions    []string `json:"suggestions"`
//	}{
//		ResourceType: "user",
//		SearchCriteria: []string{"id:123", "email:user@example.com"},
//		Suggestions: []string{"check user ID", "verify email spelling"},
//	})
//
// Response example (JSON):
//
//	{
//		"id": "req-123456",
//		"http_error_code": {
//			"canonical_name": "INVALID_ARGUMENT",
//			"status": 400
//		},
//		"message": "validation failed",
//		"data": {
//			"field": "email",
//			"requirement": "valid email format"
//		}
//	}
//
// This method immediately stops request processing and sends the response.
// See also DataWithDetail() for including both detail and data fields.
func (e ErrorCodeName) Data(ctx *context.Context, msg string, data any) {
	fail(ctx, e, msg, "", "", "", nil, data, false)
}

// DataWithDetail sends an HTTP error response with a message, detail, and additional structured data.
// It combines Detail() and Data(), so one call carries both the explanation and the
// structured supplementary information.
//
// This is particularly useful for complex error scenarios where you need to provide both
// human-readable explanations and machine-readable data for client processing.
//
// Parameters:
//   - ctx: The Iris request context for the current HTTP request
//   - msg: The primary error message - a concise summary of what went wrong
//   - detail: Additional detailed information about the error
//   - data: Additional structured data to include in the error response
//
// Example usage:
//
//	errors.InvalidArgument.DataWithDetail(ctx,
//		"validation failed",
//		"multiple fields contain invalid values",
//		map[string]any{
//			"invalid_fields": []string{"email", "phone", "age"},
//			"valid_examples": map[string]string{
//				"email": "user@example.com",
//				"phone": "+1-555-123-4567",
//				"age": "21",
//			},
//		})
//
// Response example (JSON):
//
//	{
//		"id": "req-123456",
//		"http_error_code": {
//			"canonical_name": "INVALID_ARGUMENT",
//			"status": 400
//		},
//		"message": "validation failed",
//		"detail": "multiple fields contain invalid values",
//		"data": {
//			"invalid_fields": ["email", "phone", "age"],
//			"valid_examples": {
//				"email": "user@example.com",
//				"phone": "+1-555-123-4567",
//				"age": "21"
//			}
//		}
//	}
//
// This method immediately stops request processing and sends the response.
func (e ErrorCodeName) DataWithDetail(ctx *context.Context, msg, detail string, data any) {
	fail(ctx, e, msg, detail, "", "", nil, data, false)
}

// Validation sends an HTTP error response specifically designed for handling multiple validation failures.
// This method is optimized for scenarios where you need to report multiple field-level validation errors
// simultaneously, providing structured feedback about all validation issues in a single response.
//
// The method accepts multiple ValidationError implementations, each containing detailed information
// about specific validation failures including field names, invalid values, and validation rules
// that were violated, so a client can tell the user exactly what to correct in every
// form field or input parameter at once.
//
// Parameters:
//   - ctx: The Iris request context for the current HTTP request
//   - validationErrors: One or more ValidationError implementations containing detailed validation failure information
//
// Example usage:
//
//	// Single validation error
//	emailErr := &MyValidationError{
//		Field: "email",
//		Value: "invalid-email",
//		Tag:   "email",
//		Message: "must be a valid email address",
//	}
//	errors.InvalidArgument.Validation(ctx, emailErr)
//
//	// Multiple validation errors
//	emailErr := &MyValidationError{Field: "email", Message: "must be a valid email address"}
//	phoneErr := &MyValidationError{Field: "phone", Message: "must be a valid phone number"}
//	ageErr := &MyValidationError{Field: "age", Message: "must be between 18 and 120"}
//	errors.InvalidArgument.Validation(ctx, emailErr, phoneErr, ageErr)
//
//	// Using validation package utilities
//	var validationErrors []ValidationError
//	if err := validation.ValidateEmail(email); err != nil {
//		validationErrors = append(validationErrors, err.(ValidationError))
//	}
//	if err := validation.ValidatePhone(phone); err != nil {
//		validationErrors = append(validationErrors, err.(ValidationError))
//	}
//	if len(validationErrors) > 0 {
//		errors.InvalidArgument.Validation(ctx, validationErrors...)
//	}
//
// Response example (JSON) for single validation error:
//
//	{
//		"id": "req-123456",
//		"http_error_code": {
//			"canonical_name": "INVALID_ARGUMENT",
//			"status": 400
//		},
//		"message": "validation failure",
//		"detail": "fields were invalid",
//		"validation": {
//			"field": "email",
//			"value": "invalid-email",
//			"tag": "email",
//			"message": "must be a valid email address"
//		}
//	}
//
// Response example (JSON) for multiple validation errors:
//
//	{
//		"id": "req-123456",
//		"http_error_code": {
//			"canonical_name": "INVALID_ARGUMENT",
//			"status": 400
//		},
//		"message": "validation failure",
//		"detail": "fields were invalid",
//		"validation": [
//			{
//				"field": "email",
//				"message": "must be a valid email address"
//			},
//			{
//				"field": "phone",
//				"message": "must be a valid phone number"
//			},
//			{
//				"field": "age",
//				"message": "must be between 18 and 120"
//			}
//		]
//	}
//
// This method is particularly useful for:
//   - Form validation with multiple fields
//   - API parameter validation across multiple inputs
//   - Batch input validation scenarios
//   - Complex business rule validation
//
// The validation errors will be included in the response under the "validation" field,
// allowing clients to programmatically handle validation failures and provide specific
// user feedback for each field that failed validation.
//
// This method immediately stops request processing and sends the response.
func (e ErrorCodeName) Validation(ctx *context.Context, validationErrors ...ValidationError) {
	e.validation(ctx, validationErrors)
}

func (e ErrorCodeName) validation(ctx *context.Context, validationErrors any) {
	fail(ctx, e, "validation failure", "fields were invalid", "", "", validationErrors, nil, false)
}

// Err sends an HTTP error response using the error's message text as the client-visible message.
// This method provides automatic error type detection and intelligent handling of different error
// scenarios, making it ideal for generic error processing where you want to expose the actual
// error message to clients.
//
// The method performs several automatic behaviors:
// - Validation errors are automatically routed to the Validation() method for structured handling
// - ValidationErrors (multiple validation failures) are handled collectively
// - Existing Error struct instances are sent directly with the current error code
// - Nested errors with data are automatically extracted and included in the response
// - Private errors (context.ErrPrivate) trigger server-side logging while sending generic messages
//
// Error Processing Features:
// - Automatic detail extraction from errors implementing detailedError interface
// - JSON data extraction from struct-type errors for rich error responses
// - Proper handling of wrapped and joined errors with data preservation
// - Security-conscious private error handling that protects sensitive information
//
// Parameters:
//   - ctx: The Iris request context for the current HTTP request
//   - err: The error to be processed and sent to the client (can be nil)
//
// Example usage:
//
//	// Simple error forwarding
//	if err := userService.GetUser(userID); err != nil {
//		errors.NotFound.Err(ctx, err) // Uses err.Error() as message
//		return
//	}
//
//	// Automatic validation error handling
//	if err := validation.ValidateEmail(email); err != nil {
//		errors.InvalidArgument.Err(ctx, err) // Automatically uses Validation() method
//		return
//	}
//
//	// Complex error with embedded data
//	type UserError struct {
//		UserID   string `json:"user_id"`
//		Reason   string `json:"reason"`
//		Attempts int    `json:"attempts"`
//	}
//	userErr := &UserError{UserID: "123", Reason: "invalid credentials", Attempts: 3}
//	errors.Unauthenticated.Err(ctx, userErr) // Data automatically extracted
//
// Response example (JSON) for simple error:
//
//	{
//		"id": "req-123456",
//		"http_error_code": {
//			"canonical_name": "NOT_FOUND",
//			"status": 404
//		},
//		"message": "user not found"
//	}
//
// Response example (JSON) for error with embedded data:
//
//	{
//		"id": "req-123456",
//		"http_error_code": {
//			"canonical_name": "UNAUTHENTICATED",
//			"status": 401
//		},
//		"message": "invalid credentials",
//		"data": {
//			"user_id": "123",
//			"reason": "invalid credentials",
//			"attempts": 3
//		}
//	}
//
// Security Note: This method exposes the actual error message to clients. Use ErrPrivate()
// when the error might contain sensitive information that should not be exposed to clients.
//
// This method immediately stops request processing and sends the response.
// Use Wrap() instead if you need to return errors from functions for later processing.
//
// See ErrPrivate() for security-conscious error handling that hides sensitive details.
func (e ErrorCodeName) Err(ctx *context.Context, err error) {
	e.err(ctx, err, false)
}

// ErrPrivate sends an HTTP error response with a generic message to the client while logging
// the full error details server-side. This method is essential for security-conscious error
// handling where you need to prevent sensitive information from being exposed to clients.
//
// This method is particularly important when dealing with:
// - Database errors that might contain schema information
// - Internal service errors with implementation details
// - Authentication errors with sensitive credential information
// - File system errors with server path information
// - Third-party service errors with internal details
//
// The method performs the same intelligent error processing as Err() but ensures that
// only generic, safe messages are sent to clients while maintaining complete error
// information in server-side logs for debugging and monitoring.
//
// That holds for every error type. A ValidationError and a *Error, which Err()
// renders as they are, get the generic message here as well.
//
// Security Features:
// - Automatically logs full error details server-side for debugging
// - Sends only generic "server error" messages to clients
// - Preserves error context for internal monitoring and alerting
// - Prevents information leakage through error messages
// - Maintains error traceability for operational support
//
// Parameters:
//   - ctx: The Iris request context for the current HTTP request
//   - err: The error to be logged and handled (can be nil)
//
// Example usage:
//
//	// Database error with sensitive information
//	user, err := db.Query("SELECT * FROM users WHERE secret_token = ?", token)
//	if err != nil {
//		errors.Internal.ErrPrivate(ctx, err) // Logs full SQL error, sends generic message
//		return
//	}
//
//	// External service error with internal details
//	if err := paymentService.ProcessPayment(details); err != nil {
//		errors.Internal.ErrPrivate(ctx, err) // Hides payment provider internals
//		return
//	}
//
//	// File system error with server paths
//	if err := ioutil.ReadFile("/etc/app/config.json"); err != nil {
//		errors.Internal.ErrPrivate(ctx, err) // Hides server file structure
//		return
//	}
//
// Response example (JSON) - client sees only generic message:
//
//	{
//		"id": "req-123456",
//		"http_error_code": {
//			"canonical_name": "INTERNAL",
//			"status": 500
//		},
//		"message": "server error",
//		"detail": "an unexpected error occurred while processing the request"
//	}
//
// Server-side log will contain the full error details for debugging and monitoring.
//
// Best Practices:
// - Use ErrPrivate() for any error that might contain sensitive information
// - Use Err() only when you're certain the error message is safe for client consumption
// - Consider using error mapping (Options.MapErrors) for automatic private error detection
// - Always log the full error server-side when using private error handling
//
// This method immediately stops request processing and sends the response.
func (e ErrorCodeName) ErrPrivate(ctx *context.Context, err error) {
	if err == nil {
		return
	}

	// Not through err: its first branches render a validation error with its
	// fields and send a *Error as it is, whatever isPrivate says. Nothing of a
	// private error is for the client, whatever its type.
	e.failErr(ctx, err, true)
}

// Handle renders "err" as this error code and reports whether it did, so a
// fallible step is one guarded line:
//
//	if errors.Unavailable.Handle(ctx, api.users.UpdateEmail(ctx, old, new)) {
//		return
//	}
//
// A nil error writes nothing and reports false, so the handler continues.
//
// It is the code-forcing counterpart of the package-level HandleError, which
// maps the error through the registry and answers 500 for one it does not
// recognise; Handle states the code at the call site instead, for a step whose
// failure means one thing there and nothing in general (a third-party call
// that is Unavailable, say).
//
// Handle is Err plus the answer to "did it fire". Err stays as it is: its
// signature matches LogErrorFunc and cors.HandleErrorFunc, so
// errors.FailedPrecondition.Err is passed to them as a bare method value, and
// a return value would take that away.
func (e ErrorCodeName) Handle(ctx *context.Context, err error) bool {
	if err == nil {
		return false
	}

	e.err(ctx, err, false)
	return true
}

func (e ErrorCodeName) err(ctx *context.Context, err error, isPrivate bool) {
	if err == nil {
		return
	}

	if vErr, ok := err.(ValidationError); ok {
		if vErr == nil {
			return // consider as not error for any case, this should never happen.
		}

		e.Validation(ctx, vErr)
		return
	}

	if vErrs, ok := err.(ValidationErrors); ok {
		if len(vErrs) == 0 {
			return // consider as not error for any case, this should never happen.
		}

		e.Validation(ctx, vErrs...)
		return
	}

	// If it's already an Error type then send it directly.
	if httpErr, ok := err.(*Error); ok {
		if errorCode, ok := errorCodeNameMap[e]; ok {
			httpErr.ErrorCode = errorCode
			ctx.StopWithJSON(errorCode.Status, httpErr) // here we override the fail function and send the error as it is.
			// renderError(ctx, nil, httpErr,false)
			return
		}
	}

	e.failErr(ctx, err, isPrivate)
}

// failErr renders err as e from its text, its detail and the data of the
// errors it wraps. It is the part of err after the branches that render an
// error by its type (a validation error, a *Error sent as it is). A caller
// whose error is private starts here with isPrivate set (see ErrPrivate and
// serveServerError), so none of those branches can show it: the collector
// receives the error in full and the client reads the generic message.
func (e ErrorCodeName) failErr(ctx *context.Context, err error, isPrivate bool) {
	// extract any detail and/or data from the error value.
	detail := ""
	if errWithDetail, ok := err.(detailedError); ok {
		detail = errWithDetail.Detail()
	}

	// error generated by fmt.Errorf or a Wrap function or a Join function.
	// ordered children error descriptions with message:data key-values if more than one error with data.
	var childrenErrorsWithData []childErrorWithData
	var dataValue any = nil                   // declare it here so defer func below has access to its final version.
	for _, childErr := range UnwrapAll(err) { // returns all errors, including the "err".

		if childData, ok := readErrorStructValueAsJSON(childErr); ok {
			childDetail := ""
			if errWithDetail, ok := childErr.(detailedError); ok {
				childDetail = errWithDetail.Detail()
			}
			childrenErrorsWithData = append(childrenErrorsWithData, childErrorWithData{
				Message: childErr.Error(),
				Detail:  childDetail,
				Data:    childData,
			})
		}

		if !isPrivate { // Important piece of code for security.
			// if it's not a private one already, means that probably this isn't coming from the LogErr method,
			// then try to log the whole error as well. This will run only once, as we check for isPrivate for this reason too:
			// if at least one of the errors is private then the whole error should be private.
			if _, ok := childErr.(context.ErrPrivate); ok { // See privateErrorWrapper type too.
				isPrivate = true

				defer func() {
					errText := err.Error()
					if dataValue != nil {
						if b, _ := json.MarshalIndent(dataValue, "", "  "); len(b) > 2 {
							errText += " (with data)"
							errText += ":\n" + string(b)
						}
					}

					opts := getReadOnlyOptions(ctx)
					opts.logError(ctx, New(errText))
				}()
			}
		}
	}

	// if contained only one joined error with data, then no need to give it a key in the data, let it be in the root "data" key.
	msg := err.Error()
	switch len(childrenErrorsWithData) {
	case 0:
		// As a special case for simple wrapped and then joined error:
		// fmt.Errorf("user json schema validation failed: %w", errors.Join(nested1, nested2))
		// We should display the root error message and then
		// the nested errors as a sub-errors:
		/*
		   {
		       ...
		       "message": "user json schema validation failed",
		       "data": [
		           {
		               "message": "missing required field firstname"
		           },
		           {
		               "message": "missing required field lastname"
		           }
		       ]
		   }
		*/
		var childrenErrorsWithData []*childErrorWithData
		msg, detail, childrenErrorsWithData = parseRootError(err)
		if len(childrenErrorsWithData) > 0 { // fixes the empty "null" on data.
			dataValue = childrenErrorsWithData
		}
	case 1:
		dataValue = childrenErrorsWithData[0].Data
	default:
		dataValue = childrenErrorsWithData
	}

	fail(ctx, e, msg, detail, "", "", nil, dataValue, isPrivate)
}

// Private wraps the given error with the private error wrapper.
// It's useful when you want to log the full error but send a generic message to the client,
// from the `MapErrorFunc`, `MapErrorFuncCtx`, `MapErrors` and `MapErrorTypes`
// methods of the `Options` type.
//
// Usage:
//
//	MapErrorFunc(errors.Internal, func(err error) error {
//		if errText := err.Error(); strings.Contains(errText, `syntax error at or near \"`) ||
//			strings.Contains(errText, "transaction: ") { // contains query information.
//			return errors.Private(err)
//		}
//
//		return nil
//	}).
//
// // And then you may continue with error type mapping as well.
// MapErrorTypes(errors.InvalidArgument, app.MyDomainLevelError1{}, app.MyDomainLevelError2{},
// ).
func Private(err error) error {
	return context.PrivateErrorf(err)
}

type childErrorWithData struct {
	Message string          `json:"message"`
	Detail  string          `json:"detail,omitempty"`
	Data    json.RawMessage `json:"data,omitempty"`
}

type detailedError interface {
	Detail() string
}

func readErrorStructValueAsJSON(err error) (json.RawMessage, bool) {
	var jsonErrorData json.RawMessage
	if errValue := reflex.IndirectValue(err); errValue.IsValid() && errValue.Kind() == reflect.Struct && errValue.NumField() > 0 && errValue.CanInterface() {
		jsonErrorData, _ = jsonv2.Marshal(errValue.Interface(), jsonutil.DefaultMarshalOptionsSlice...)
	}

	return jsonErrorData, len(jsonErrorData) > 2
}

func parseRootError(err error) (message string, detail string, children []*childErrorWithData) {
	if err == nil {
		return
	}

	message = err.Error()

	if unwrapped := errors.Unwrap(err); unwrapped != nil {
		unwrappedMsg := unwrapped.Error()
		// Find the last ": " in message.
		if idx := strings.LastIndex(message, ": "); idx > 0 && idx+2 <= len(message) {
			// Check if the substring after ": " matches the start of the unwrapped error message.
			if strings.HasPrefix(unwrappedMsg, message[idx+2:]) {
				message = message[:idx]
			}
		}
		children = parseNestedErrors(unwrapped)
	}
	// An error that does not unwrap reports no children. Parsing the root error itself was
	// tried and dropped:
	//
	// children = parseNestedErrors(err)
	//
	// if len(children) == 1 && children[0].Message == message {
	// 	// If there's only one child error and its message matches the root error message,
	// 	// we can consider it as a part of the root error. This is a common case
	// 	// when the error is a simple wrapped error,
	// 	children = nil
	// }

	if errWithDetail, ok := err.(detailedError); ok {
		detail = errWithDetail.Detail()
	}

	return
}

func parseNestedErrors(err error) (children []*childErrorWithData) {
	if unwrapped := errors.Unwrap(err); unwrapped != nil {
		// If the unwrapped error is a joined error, use its nested as children.
		if joined, ok := unwrapped.(interface{ Unwrap() []error }); ok {
			for _, nested := range joined.Unwrap() {
				children = append(children, parseNestedErrors(nested)...)
			}
		} else {
			children = append(children, parseNestedErrors(unwrapped)...)
		}
	} else if joined, ok := err.(interface{ Unwrap() []error }); ok {
		for _, nested := range joined.Unwrap() {
			children = append(children, parseNestedErrors(nested)...)
		}
	} else {
		detail := ""
		if errWithDetail, ok := err.(detailedError); ok {
			detail = errWithDetail.Detail()
		}
		children = append(children, &childErrorWithData{
			Message: err.Error(),
			Detail:  detail,
		})
	}

	return
}

// Log creates a formatted error message and handles it through the configured logging and error
// processing pipeline. This method is designed for scenarios where you need to log specific
// formatted messages while potentially sending appropriate error responses to clients.
//
// The method automatically handles cancellation detection, error logging, and client response
// generation. For Internal error codes, it sends generic messages to clients while logging
// the full details server-side for security purposes.
//
// Cancellation Handling:
// The method respects the global SkipCanceled setting and returns early only when
// the client's own request context is canceled (ctx.IsCanceled()), so nothing is
// logged or written for a caller that is no longer listening.
//
// The shape of the error does not get to decide that, and neither does the shape
// of an argument. An upstream dial or read timeout is a net.Error whose Timeout()
// is true and it satisfies errors.Is(err, context.DeadlineExceeded) too, yet it is
// the failure the client is waiting to hear about and the one an operator most
// needs to see. See Report, which follows the same rule.
//
// Parameters:
//   - ctx: The Iris request context for the current HTTP request
//   - format: Printf-style format string for the error message
//   - args: Arguments to be formatted into the error message
//
// Example usage:
//
//	// Simple formatted logging
//	errors.Internal.Log(ctx, "failed to process user %s: operation timeout", userID)
//
//	// Complex error with multiple values
//	errors.InvalidArgument.Log(ctx,
//		"validation failed for field %s: value %v is not in range [%d, %d]",
//		fieldName, fieldValue, minValue, maxValue)
//
//	// Database operation error
//	errors.Internal.Log(ctx, "database query failed for table %s: %v", tableName, dbErr)
//
//	// Performance monitoring
//	if processingTime > threshold {
//		errors.Internal.Log(ctx, "slow operation detected: %s took %v (threshold: %v)",
//			operation, processingTime, threshold)
//	}
//
// Behavior by Error Code:
// - Internal errors: Logs full details server-side, sends generic message to client
// - Other error codes: Logs and sends the formatted message to the client
//
// Response example (JSON) for Internal error code:
//
//	{
//		"id": "req-123456",
//		"http_error_code": {
//			"canonical_name": "INTERNAL",
//			"status": 500
//		},
//		"message": "server error",
//		"detail": "an unexpected error occurred while processing the request"
//	}
//
// Server-side log will contain: "failed to process user user123: operation timeout"
//
// Response example (JSON) for other error codes:
//
//	{
//		"id": "req-123456",
//		"http_error_code": {
//			"canonical_name": "INVALID_ARGUMENT",
//			"status": 400
//		},
//		"message": "validation failed for field email: value invalid@domain is not in range [10, 100]"
//	}
//
// This method immediately stops request processing and sends the response unless the
// operation is canceled or the context is already stopped.
//
// See LogErr() for logging existing error instances, and ErrPrivate() for always private logging.
func (e ErrorCodeName) Log(ctx *context.Context, format string, args ...any) {
	if SkipCanceled && ctx.IsCanceled() {
		return
	}

	err := fmt.Errorf(format, args...)
	e.logErr(ctx, err)
}

// LogErr logs an existing error through the configured logging and error processing pipeline
// while handling appropriate client responses. This method is the primary way to log errors
// that have already been created while ensuring proper error handling and client communication.
//
// The method provides automatic cancellation detection, server-side logging, and intelligent
// client response generation. For Internal error codes, it maintains security by sending
// generic messages to clients while preserving full error details in server-side logs.
//
// Key Features:
// - Automatic cancellation detection and handling
// - Intelligent error processing through the configured error handling pipeline
// - Security-conscious client response generation
// - Integration with custom logging and error collection systems
// - Proper error context preservation for debugging and monitoring
//
// Cancellation Handling:
// The method respects the global SkipCanceled setting and automatically skips processing
// when encountering canceled contexts or canceled errors, preventing unnecessary overhead
// and log noise during client disconnections or timeouts.
//
// Parameters:
//   - ctx: The Iris request context for the current HTTP request
//   - err: The error to be logged and processed (can be nil)
//
// Example usage:
//
//	// Database operation error
//	user, err := userRepository.GetUser(userID)
//	if err != nil {
//		errors.Internal.LogErr(ctx, err) // Logs full error, sends generic response
//		return
//	}
//
//	// External service error
//	if err := paymentService.ProcessPayment(amount); err != nil {
//		errors.Internal.LogErr(ctx, err) // Protects sensitive payment details
//		return
//	}
//
//	// Validation error from business logic
//	if err := validateBusinessRules(request); err != nil {
//		errors.InvalidArgument.LogErr(ctx, err) // Shows validation details to client
//		return
//	}
//
//	// File system operation error
//	if err := os.WriteFile(filepath, data, 0644); err != nil {
//		errors.Internal.LogErr(ctx, err) // Hides server file system details
//		return
//	}
//
// Behavior by Error Code:
// - Internal errors: Full error logged server-side, generic message sent to client
// - Other error codes: Error logged and original message sent to client (unless private)
//
// Response example (JSON) for Internal error:
//
//	{
//		"id": "req-123456",
//		"http_error_code": {
//			"canonical_name": "INTERNAL",
//			"status": 500
//		},
//		"message": "server error",
//		"detail": "an unexpected error occurred while processing the request"
//	}
//
// Server-side logs will contain the complete error details with stack trace and context
// information for debugging and operational monitoring.
//
// Error Processing Pipeline:
// 1. Cancellation detection and early return if applicable
// 2. Error logging through configured logging infrastructure
// 3. Error collection for monitoring and alerting systems
// 4. Client response generation with appropriate security filtering
// 5. Request processing termination
//
// This method immediately stops request processing and sends the response unless the
// operation is canceled or the context is already stopped.
//
// See Log() for formatted error logging, and ErrPrivate() for explicit private error handling.
func (e ErrorCodeName) LogErr(ctx *context.Context, err error) {
	if SkipCanceled && ctx.IsCanceled() {
		return
	}

	e.logErr(ctx, err)
}

// logErr is a helper method that logs the given error and sends a generic error to the client if the error code is Internal.
//
// Parameters:
// - ctx: The context of the request.
// - err: The error to be logged.
func (e ErrorCodeName) logErr(ctx *context.Context, err error) {
	opts := getReadOnlyOptions(ctx)
	opts.logError(ctx, err)

	isPrivate := e == Internal // if it's an internal error then send a generic message to the client.
	e.err(ctx, err, isPrivate)
}

// HandleAPIError handles remote server errors.
// Optionally, use it when you write your server's HTTP clients using the the /client package.
// When the HTTP Client sends data to a remote server but that remote server
// failed to accept the request as expected, then the error will be proxied
// to this server's end-client.
//
// When the given "err" is not a type of client.APIError then
// the error will be sent using the "Internal.LogErr" method which sends
// HTTP internal server error to the end-client and
// prints the "err" using the "LogError" package-level function.
func HandleAPIError(ctx *context.Context, err error) {
	// Error expected and came from the external server,
	// save its body so we can forward it to the end-client.
	if apiErr, ok := client.GetError(err); ok {
		handleAPIError(ctx, apiErr)
		return
	}

	Internal.LogErr(ctx, err)
}

func handleAPIError(ctx *context.Context, apiErr client.APIError) {
	// Error expected and came from the external server,
	// save its body so we can forward it to the end-client.
	// GetErrorCode tolerates an APIError built without a response (0).
	statusCode := client.GetErrorCode(apiErr)
	if statusCode >= 400 && statusCode < 500 {
		InvalidArgument.DataWithDetail(ctx, "remote server error", "invalid client request", apiErr.Body)
	} else {
		Internal.Data(ctx, "remote server error", apiErr.Body)
	}

	// Unavailable.DataWithDetail(ctx, "remote server error", "unavailable", apiErr.Body)
}

// HandleRequestJSONError handles a failure to decode a request body as
// JSON. It answers 400 with a detail that names only wire-level facts (an
// empty body, malformed JSON, the offending field and the expected kind),
// never the server's Go types. It reports whether a response was sent;
// a nil error and an error that is not a JSON one report false.
func HandleRequestJSONError(ctx *context.Context, err error) bool {
	return handleJSONError(ctx, false, err)
}

// HandleResponseJSONError handles a failure to encode a response as JSON.
// It answers 500 with a generic detail and reports whether a response was
// sent; a nil error and an error that is not a JSON one report false.
func HandleResponseJSONError(ctx *context.Context, err error) bool {
	return handleJSONError(ctx, true, err)
}

// handleJSONError is the shared body of HandleRequestJSONError (serverError
// false: 400) and HandleResponseJSONError (serverError true: 500).
//
// The function performs the following steps:
// 1. If the error is nil, it returns false indicating no error was handled.
// 2. Sets the default error message and error code based on whether the error is a server error or client error.
// 3. Checks if the error is an EOF or unexpected EOF error and sends a detailed error response with an "empty body" message.
// 4. Checks if the error is a JSON syntax error and sends a detailed error response with a "malformed JSON" detail (the byte offset is not disclosed to the client).
// 5. Checks if the error is a JSON unmarshal type error and sends a detailed error response with the specific type error message.
// 6. Checks if the error is a JSON unsupported value error and sends a detailed error response with the error message.
// 7. Checks if the error is a JSON unsupported type error and sends a detailed error response with the error message.
// 8. Checks if the error is a JSON invalid unmarshal error and sends a detailed error response with the error message.
// 9. If none of the specific JSON errors are matched, it returns false indicating no error was handled.
func handleJSONError(ctx *context.Context, serverError bool, err error) bool {
	if err == nil {
		return false
	}

	messageText := "unable to parse request body"
	wireError := InvalidArgument
	if serverError {
		messageText = "unable to parse response body"
		wireError = Internal
	}

	// encoding/json/v2 errors first (the codebase decodes through v2); the
	// v1 branches below remain as a fallback for user code still on v1.
	//
	// The syntactic check comes before the EOF one on purpose: a truncated
	// document such as `{"a":` is a SyntacticError that wraps
	// io.ErrUnexpectedEOF, and it is malformed JSON, not an empty body.
	if _, ok := errors.AsType[*jsontext.SyntacticError](err); ok {
		wireError.Detail(ctx, messageText, "malformed JSON") // the byte offset is not disclosed to the client.
		return true
	}

	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		wireError.Detail(ctx, messageText, "empty body")
		return true
	}

	if sem, ok := errors.AsType[*jsonv2.SemanticError](err); ok {
		// Never forward sem.Error() to the client: it renders GoType, the
		// server's Go type graph. JSONPointer and JSONKind are wire-level
		// facts the client sent and are safe (and useful) to disclose.
		ptr := string(sem.JSONPointer)
		switch {
		case errors.Is(err, jsonv2.ErrUnknownName):
			wireError.Detail(ctx, messageText, fmt.Sprintf("unknown field %q", ptr))
		case sem.JSONKind != 0 && sem.GoType != nil:
			want, got := jsonKindName(sem.GoType), jsonKindOfToken(sem.JSONKind)
			if ptr != "" {
				wireError.Detail(ctx, messageText, fmt.Sprintf("field %q: expected %s, got %s", ptr, want, got))
			} else {
				wireError.Detail(ctx, messageText, fmt.Sprintf("expected %s, got %s", want, got))
			}
		default: // marshal-side unsupported value/type, invalid unmarshal target, etc.
			if ptr != "" {
				wireError.Detail(ctx, messageText, fmt.Sprintf("invalid value at %q", ptr))
			} else {
				wireError.Detail(ctx, messageText, "invalid value")
			}
		}

		return true
	}

	if _, ok := errors.AsType[*json.SyntaxError](err); ok {
		wireError.Detail(ctx, messageText, "malformed JSON")
		return true
	}

	if unmarshalErr, ok := errors.AsType[*json.UnmarshalTypeError](err); ok {
		// Note(@kataras): never forward unmarshalErr.Error() to the client.
		//
		// It renders the server's Go type graph verbatim, e.g.
		//	json: cannot unmarshal string into Go struct field
		//	CreateUserRequest.internal_role of type models.Role
		// which walks an attacker through internal type and field names,
		// including fields that are not otherwise observable. Report only the
		// wire-level facts: which JSON field, and what kind was expected.
		// Kind fields, not error-text sniffing: Go 1.27 no longer
		// guarantees the v1 error text.
		if unmarshalErr.Field != "" {
			wireError.Detail(ctx, messageText, fmt.Sprintf("field %q: expected %s, got %s",
				unmarshalErr.Field, jsonKindName(unmarshalErr.Type), unmarshalErr.Value))
		} else {
			wireError.Detail(ctx, messageText, fmt.Sprintf("expected %s, got %s",
				jsonKindName(unmarshalErr.Type), unmarshalErr.Value))
		}

		return true
	}

	// The remaining v1 errors describe the server's own values and types
	// (their text names the Go type), so the client gets a fixed detail.
	if _, ok := errors.AsType[*json.UnsupportedValueError](err); ok {
		wireError.Detail(ctx, messageText, "unsupported value")
		return true
	}

	if _, ok := errors.AsType[*json.UnsupportedTypeError](err); ok {
		wireError.Detail(ctx, messageText, "unsupported type")
		return true
	}

	if _, ok := errors.AsType[*json.InvalidUnmarshalError](err); ok {
		wireError.Detail(ctx, messageText, "invalid decode target")
		return true
	}

	return false
}

var (
	// ErrUnexpected is the HTTP error which sent to the client
	// when server fails to send an error, it's a fallback error.
	// The server fails to send an error on two cases:
	// 1. when the provided error code name is not registered (the error value is the ErrUnexpectedErrorCode)
	// 2. when the error contains data but cannot be encoded to json (the value of the error is the result error of json.Marshal).
	ErrUnexpected = Register("UNEXPECTED_ERROR", http.StatusInternalServerError)
	// ErrUnexpectedErrorCode is the error which logged
	// when the given error code name is not registered.
	ErrUnexpectedErrorCode = New("unexpected error code name")
)

// Error represents the JSON form of "http wire errors".
//
// Examples can be found at:
//
//	https://github.com/kataras/iris/tree/main/_examples/08-errors/central-error-map.
type Error struct {
	// ID helps to track the error, it's a unique identifier.
	ID string `json:"id,omitempty" yaml:"ID,omitempty"`
	// ErrorCode is the canonical name and the status code of the error.
	ErrorCode ErrorCode `json:"http_error_code" yaml:"HTTPErrorCode"`
	// Message is the human-readable message of the error.
	Message string `json:"message,omitempty" yaml:"Message"`
	// Detail is the human-readable explanation specific to this occurrence of the error.
	Detail string `json:"detail,omitempty" yaml:"Detail"`
	// Problem-Specific.

	// Type is the URI reference that identifies the problem type.
	// It's a field, part of the RFC 7807. Example: https://example.com/probs/out-of-credit.
	Type string `json:"type,omitempty" yaml:"Type,omitempty"`
	// Instance is the URI reference that identifies the specific occurrence of the problem.
	// It's a field, part of the RFC 7807. Example: /account/kataras/messages/1.
	Instance string `json:"instance,omitempty" yaml:"Instance,omitempty"`
	//

	// Validation is the validation error, if any.
	Validation any `json:"validation,omitempty" yaml:"Validation,omitempty"`
	// Data is the custom (JSON) extra data of the error, if any.
	Data json.RawMessage `json:"data,omitempty" yaml:"Data,omitempty"`
}

// Attrs returns the slog's attributes (and golog supports them) for this error. It's just a helper function to make your life easier.
func (err *Error) Attrs() []slog.Attr {
	attrs := []slog.Attr{
		{Key: "id", Value: slog.StringValue(err.ID)},
		{Key: "http_error_code", Value: slog.StringValue(string(err.ErrorCode.CanonicalName))},
		{Key: "http_status_code", Value: slog.IntValue(err.ErrorCode.Status)},
		{Key: "message", Value: slog.StringValue(err.Message)},
	}

	if err.Detail != "" {
		attr := slog.Attr{Key: "detail", Value: slog.StringValue(err.Detail)}
		attrs = append(attrs, attr)
	}

	if err.Type != "" {
		attr := slog.Attr{Key: "type", Value: slog.StringValue(err.Type)}
		attrs = append(attrs, attr)
	}

	if err.Instance != "" {
		attr := slog.Attr{Key: "instance", Value: slog.StringValue(err.Instance)}
		attrs = append(attrs, attr)
	}

	if err.Validation != nil {
		attr := slog.Attr{Key: "validation", Value: slog.AnyValue(err.Validation)}
		attrs = append(attrs, attr)
	}
	if err.Data != nil {
		attr := slog.Attr{Key: "data", Value: slog.AnyValue(err.Data)}
		attrs = append(attrs, attr)
	}

	return attrs
}

// Error method completes the error interface. It just returns the canonical name, status code, message and detail.
// Empty fields print as placeholders; the receiver is never modified, since
// the same value is logged first and rendered on the wire right after.
func (err *Error) Error() string {
	message, detail := err.Message, err.Detail
	if message == "" {
		message = "<empty>"
	}

	if detail == "" {
		detail = "<empty>"
	}

	canonicalName, status := err.ErrorCode.CanonicalName, err.ErrorCode.Status
	if canonicalName == "" {
		canonicalName = ErrUnexpected
	}

	if status <= 0 {
		status = http.StatusInternalServerError
	}

	return sprintf("iris http wire error: canonical name: %s, http status code: %d, message: %s, detail: %s, type: %s, instance: %s",
		canonicalName, status, message, detail, err.Type, err.Instance)
}

func fail(ctx *context.Context, codeName ErrorCodeName, msg, detail, typeURI, instanceURI string, validationErrors any /* it always comes as slice here. */, dataValue any, isPrivate bool) {
	// for _, frame := range reflex.GetInternalStackTrace() { // DBUG.
	// 	fmt.Println(frame.String())
	// }

	errorCode, ok := errorCodeNameMap[codeName]
	if !ok {
		opts := getReadOnlyOptions(ctx)
		// This SHOULD NEVER happen, all ErrorCodeNames MUST be registered.
		opts.logError(ctx, ErrUnexpectedErrorCode)
		fail(ctx, ErrUnexpected, msg, detail, "", "", validationErrors, dataValue, isPrivate)
		return
	}

	// A body already went out: the status and those bytes are committed, so
	// an error document appended to them would corrupt what the client is
	// reading. Log the error and stop, like HandleError does. The old guard
	// also required a stopped chain, which let `ctx.WriteString("partial")`
	// followed by `NotFound.Message(ctx, ...)` append the document to a 200.
	//
	// Under middleware/recorder's ResponseRecorder, Written() counts the
	// buffered bytes, so a buffered partial body suppresses the render as
	// well; a caller that wants the error instead clears the recorded body
	// first (ResponseRecorder.SetBody(nil)).
	//
	// A hijacked connection (a websocket upgrade) has no response left to
	// write, and Hijack leaves Written at StatusCodeWritten, so it is asked
	// about on its own. This is where a handler that panics after the hijack
	// ends up, through the recovery handler.
	if w := ctx.ResponseWriter(); w.Written() > 0 || w.IsHijacked() {
		getReadOnlyOptions(ctx).logError(ctx, &Error{ErrorCode: errorCode, Message: msg, Detail: detail})
		ctx.StopExecution()
		return
	}

	errorCode.ServeFailure(ctx, msg, detail, typeURI, instanceURI, validationErrors, dataValue, isPrivate)
}

// ServeFailure handles the error response for a given Iris request context.
// It constructs an error object with the provided details and sends it to the client.
// The method also logs the error and optionally sends a generic message to the client if the error is marked as private.
//
// Parameters:
// - ctx: The context of the request.
// - msg: The error message to be sent to the client.
// - detail: Additional details about the error.
// - typeURI: A URI reference that identifies the problem type.
// - instanceURI: A URI reference that identifies the specific occurrence of the problem.
// - validationErrors: Any validation errors related to the request.
// - dataValue: Additional data to be included in the error response.
// - isPrivate: A boolean indicating whether the error details should be hidden from the client.
//
// The method performs the following steps:
// 1. Converts the dataValue to a JSON raw message if it is not nil.
// 2. Constructs an Error object with the provided details.
// 3. Logs the error and collects it using the configured options.
// 4. If the error is marked as private, replaces the error details with generic messages.
// 5. Sets the HTTP status code and stops further execution.
// 6. Renders the error response as HTML if the requested content type is HTML,
// otherwise uses a custom renderer or defaults to JSON.
func (errorCode ErrorCode) ServeFailure(ctx *context.Context, msg, detail, typeURI, instanceURI string, validationErrors any, dataValue any, isPrivate bool) {
	opts := getReadOnlyOptions(ctx)

	var data json.RawMessage
	if dataValue != nil {
		switch v := dataValue.(type) {
		case json.RawMessage: // == jsontext.Value, its alias since Go 1.27.
			data = v
		case []byte:
			data = v
		case error:
			if msg == "" {
				msg = v.Error()
			} else if detail == "" {
				detail = v.Error()
			} else {
				// JSON-string-encode the text: raw error text placed where
				// valid JSON is required made the error response itself fail
				// to render.
				data, _ = json.Marshal(v.Error())
			}
		default:
			b, err := jsonv2.Marshal(v, jsonutil.DefaultMarshalOptionsSlice...)
			if err != nil {
				opts.logError(ctx, err)
				fail(ctx, ErrUnexpected, err.Error(), "", "", "", nil, nil, isPrivate)
				return
			}
			data = b
		}

		// Caller-supplied raw bytes must be valid JSON or the whole error
		// response fails to marshal; fall back to carrying them as a string.
		if len(data) > 0 && !jsontext.Value(data).IsValid() {
			data, _ = json.Marshal(string(data))
		}
	}

	err := &Error{
		ID:         opts.getErrorID(ctx),
		ErrorCode:  errorCode,
		Message:    msg,
		Detail:     detail,
		Type:       typeURI,
		Instance:   instanceURI,
		Data:       data,
		Validation: validationErrors,
	}

	renderError(ctx, opts, err, isPrivate)
	// fmt.Printf("HERE: %#+v\n", err.Validation.([]ValidationError)[0])
}

func renderError(ctx *context.Context, opts *Options, err *Error, isPrivate bool) {
	opts.collect(ctx, err)
	// Do not handle the error at all, it may run in a separate goroutine.
	// if err := collect(ctx, err); err != nil {
	// 	LogError(ctx, err) // if collect failed, log it.
	// }

	if isPrivate { // if it's private send a generic message to the client, but after the error was sent to the collector (see ErrorCodeName.LogErr method).
		err.Message = "server error"
		err.Detail = "an unexpected error occurred while processing the request"
		err.Type = ""
		err.Instance = ""
		err.Validation = nil
		err.Data = nil
	}

	writeError(ctx, opts, err)
}

// writeError sends err to the client as it is, through the renderer the
// request asked for or the options configure. It is the part of renderError
// after the collector and the redaction; a caller that has done both itself
// (see servePrivateClientError) starts here.
func writeError(ctx *context.Context, opts *Options, err *Error) {
	// Stop the execution before any custom renderer fired.
	ctx.StopExecution()

	// As an exception, if there no a custom renderer registered
	// and the error has a type, instance, and title(as message),
	// then we render the error as a problem+json.
	if err.Type != "" && err.Instance != "" && err.Message != "" && !opts.HasRenderer() {
		Problem(ctx, err) // status code is written based on the given ErrorCodeName inside the function itself.
		// Note that, we don't allow custom status codes for problem+json as
		// all status codes are covered already through the available ErrorCodeNames.
		return
	}

	// Set the status code of the error's code.
	ctx.StatusCode(err.ErrorCode.Status)

	// If the requested content type is HTML, then render the error as an HTML page.
	if ctx.IsMediaType(context.ContentHTMLHeaderValue) {
		HTML(ctx, err)
		return
	}
	// Try to find a custom registered renderer.
	// Or JSON.
	opts.render(ctx, err)
	// ctx.SetErr(&err)
	// ctx.StopWithJSON(errorCode.Status, err)
	// Defaults to JSON rendering.
	// ctx.JSON(err)
}
