package errors

import (
	"reflect"

	"github.com/kataras/iris/v14/context"
	"github.com/kataras/iris/v14/core/pkg/reflex"
)

// ErrorCodeMap provides a sophisticated error mapping and transformation system that enables
// automatic classification of errors into appropriate HTTP status codes and ErrorCodeName categories.
// This type serves as the core engine for intelligent error handling in Iris applications.
//
// The ErrorCodeMap associates ErrorCodeName identifiers with lists of transformation functions
// that can analyze, modify, or replace incoming errors. When an error occurs, the map is
// consulted to determine the most appropriate HTTP response based on error characteristics.
//
// Key Features:
// - Automatic error classification using multiple strategies
// - Priority-based processing ordered by HTTP status codes
// - Support for error wrapping and unwrapping chains
// - Type-based error matching with interface support
// - Custom transformation functions for complex error scenarios
// - Fallback mechanisms for unhandled error types
//
// Error Processing Pipeline:
// 1. Iterate through registered ErrorCodeNames ordered by HTTP status code (lower to higher)
// 2. For each ErrorCodeName, execute associated transformation functions in registration order
// 3. Return the first non-nil error result along with its associated ErrorCodeName
// 4. If no functions handle the error, return the original error without classification
//
// Mapping Strategies:
//
//	// Direct error instance mapping
//	codeMap.MapErrors(errors.NotFound, sql.ErrNoRows, gorm.ErrRecordNotFound)
//
//	// Type-based mapping
//	codeMap.MapErrorTypes(errors.InvalidArgument, MyValidationError{})
//
//	// Custom transformation functions
//	codeMap.MapErrorFunc(errors.Internal, func(err error) error {
//		if strings.Contains(err.Error(), "connection refused") {
//			return errors.Private(err) // Hide sensitive details
//		}
//		return nil // Let other mappings handle this error
//	})
//
// Usage Examples:
//
//	// Create and configure error mapping
//	codeMap := make(ErrorCodeMap)
//	codeMap.MapErrors(errors.NotFound, sql.ErrNoRows)
//	codeMap.MapErrorTypes(errors.InvalidArgument, ValidationError{})
//
//	// Use with Options for automatic error handling
//	opts := errors.NewOptions()
//	opts.codeMap = codeMap
//
//	// Manual error classification (ctx may be nil for direct calls)
//	errorCode, transformedErr := codeMap.Filter(ctx, originalErr)
//	if errorCode != "" {
//		errorCode.Err(ctx, transformedErr)
//	}
//
// Thread Safety: ErrorCodeMap instances are not thread-safe and should be configured
// during application initialization before the HTTP server starts. Concurrent access
// during runtime may result in undefined behavior.
//
// See Options.MapErrors, Options.MapErrorTypes, Options.MapErrorFunc and
// Options.MapErrorFuncCtx for convenient configuration methods that wrap this
// functionality.
type ErrorCodeMap map[ErrorCodeName][]MapErrorFn

// MapErrorFn is the stored form of every error mapping function: it receives
// the request context (nil when Filter is called directly, outside a request)
// and the error to classify, returning the transformed error or nil when it
// does not handle it. Context-unaware functions registered through
// MapErrorFunc, MapErrors and MapErrorTypes are adapted to this form.
type MapErrorFn func(ctx *context.Context, err error) error

// handle processes an error through the ErrorCodeMap and automatically sends an appropriate
// HTTP response to the client if a mapping is found. This method serves as the primary
// entry point for automatic error processing within the error handling pipeline.
//
// The method attempts to classify the incoming error using the Filter method and, if
// successful, automatically generates and sends an HTTP response using the determined
// ErrorCodeName. This enables transparent error handling where application code can
// return any error type and have it automatically converted to appropriate HTTP responses.
//
// Processing Flow:
// 1. Apply error mapping and transformation using Filter()
// 2. If a mapping is found (ErrorCodeName is not empty), send HTTP response
// 3. Return true to indicate the error was handled
// 4. If no mapping is found, return false to allow fallback handling
//
// Parameters:
//   - ctx: The Iris request context for sending the HTTP response
//   - err: The error to be processed and potentially handled
//
// Returns:
//   - bool: true if the error was mapped and a response was sent, false otherwise
//
// Usage Context:
// This method is primarily used internally by the error handling system, particularly
// in Options.handleError() and the broader HandleError() function. Application code
// typically doesn't call this method directly.
//
// Example internal usage:
//
//	// Within error handling pipeline
//	if errorCodeMap.handle(ctx, err) {
//		return // Error was handled, response sent
//	}
//	// Continue with fallback error handling
//
// The method ensures that only successfully mapped errors result in HTTP responses,
// allowing the error handling system to try multiple mapping strategies before
// falling back to default error processing.
func (m ErrorCodeMap) handle(ctx *context.Context, err error) bool {
	if errorCodeName, errToSend := m.Filter(ctx, err); errorCodeName.String() != "" {
		errorCodeName.Err(ctx, errToSend)
		return true
	}

	return false
}

// Filter performs error classification and transformation by applying all registered
// mapping functions to determine the most appropriate ErrorCodeName and transformed error
// for the given input error. This method implements the core logic of the error mapping system.
//
// The method processes mappings in a specific order to ensure consistent and predictable
// error classification:
// 1. Iterate through HTTP status codes in ascending order (400, 401, 404, 500, etc.)
// 2. For each status code, process all associated ErrorCodeNames
// 3. For each ErrorCodeName, execute mapping functions in registration order
// 4. Return the first successful mapping (non-nil error result)
//
// This ordering ensures that more specific error types (typically with lower status codes)
// are checked before more general ones, allowing for precise error classification while
// maintaining predictable fallback behavior.
//
// Parameters:
//   - ctx: The request context, forwarded to context-aware mapping functions
//     (registered through MapErrorFuncCtx). It may be nil when Filter is called
//     directly, outside a request; context-aware functions must tolerate that.
//   - err: The original error to be classified and potentially transformed
//
// Returns:
//   - ErrorCodeName: The determined error code name, empty string if no mapping found
//   - error: The transformed error to be sent to the client, original error if no transformation
//
// Transformation Process:
// Each mapping function receives the original error and can:
// - Return nil to indicate it doesn't handle this error type
// - Return the same error unchanged (classification only)
// - Return a modified error with additional context or formatting
// - Return a completely different error (replacement)
//
// Example mappings and their results:
//
//	// Direct error instance mapping
//	codeMap.MapErrors(errors.NotFound, sql.ErrNoRows)
//	// sql.ErrNoRows -> (errors.NotFound, sql.ErrNoRows)
//
//	// Type-based mapping with transformation
//	codeMap.MapErrorTypes(errors.InvalidArgument, ValidationError{})
//	// MyValidationError{Field: "email"} -> (errors.InvalidArgument, MyValidationError{Field: "email"})
//
//	// Custom transformation
//	codeMap.MapErrorFunc(errors.Internal, func(err error) error {
//		if strings.Contains(err.Error(), "database") {
//			return fmt.Errorf("storage service temporarily unavailable")
//		}
//		return nil
//	})
//	// database error -> (errors.Internal, "storage service temporarily unavailable")
//
// Performance Characteristics:
// - Time complexity: O(n*m*k) where n=status codes, m=error codes per status, k=functions per error code
// - Space complexity: O(1) for processing (mapping storage is separate)
// - Early termination on first successful mapping for efficiency
//
// Thread Safety: This method is read-only and safe for concurrent use after the
// ErrorCodeMap has been fully configured during application initialization.
func (m ErrorCodeMap) Filter(ctx *context.Context, err error) (ErrorCodeName, error) {
	// ordered by status code.
	for _, statusCode := range statusCodesOrder {
		errorCodeNames, ok := codeNamesByStatusCode[statusCode]
		if !ok {
			continue // not required but for the logic presentation.
		}

		for _, errorCodeName := range errorCodeNames {
			if errorFuncs, ok := m[errorCodeName]; ok {
				for _, errorFunc := range errorFuncs {
					if errToSend := errorFunc(ctx, err); errToSend != nil {
						// fmt.Printf("error code name: %s, error: %s\n", errorCodeName.String(), errToSend)
						return errorCodeName, errToSend
					}
				}
			}
		}
	}

	// not ordered by status code:
	// for errorCodeName, errorFuncs := range m {
	// 	for _, errorFunc := range errorFuncs {
	// 		if errToSend := errorFunc(err); errToSend != nil {
	// 			// fmt.Printf("error code name: %s, error: %s\n", errorCodeName.String(), errToSend)
	// 			return errorCodeName, errToSend
	// 		}
	// 	}
	// }

	return "", err
}

// MapErrorFunc registers a custom error transformation function that provides the most flexible
// error mapping capability. This method allows complex error analysis, conditional processing,
// and sophisticated error transformation logic that goes beyond simple type or instance matching.
//
// The transformation function receives an error and can:
// - Return nil to indicate it doesn't handle this error (allows other mappings to be tried)
// - Return the same error unchanged (classification without transformation)
// - Return a modified version of the error (enhancement or formatting)
// - Return a completely different error (replacement or abstraction)
//
// Multiple functions can be registered for the same ErrorCodeName and will be executed
// in registration order until one returns a non-nil error. This enables layered processing
// and fallback mechanisms.
//
// Parameters:
//   - e: The ErrorCodeName to associate with this transformation function
//   - fn: The transformation function that processes incoming errors
//
// Function Signature:
//   - Input: error - The original error to be analyzed and potentially transformed
//   - Output: error - The processed error, or nil if this function doesn't handle it
//
// Example implementations:
//
//	// Database error abstraction
//	codeMap.MapErrorFunc(errors.Internal, func(err error) error {
//		if strings.Contains(err.Error(), "connection refused") {
//			return errors.Private(fmt.Errorf("database service unavailable"))
//		}
//		if strings.Contains(err.Error(), "timeout") {
//			return fmt.Errorf("operation timed out, please try again")
//		}
//		return nil // Let other mappings handle this error
//	})
//
//	// Third-party service error mapping
//	codeMap.MapErrorFunc(errors.Unavailable, func(err error) error {
//		if stripeErr, ok := err.(*stripe.Error); ok {
//			return &errors.Error{
//				Message: "payment processing temporarily unavailable",
//				Detail:  stripeErr.Msg,
//				Data:    json.RawMessage(fmt.Sprintf(`{"stripe_code":"%s"}`, stripeErr.Code)),
//			}
//		}
//		return nil
//	})
//
//	// Validation error enhancement
//	codeMap.MapErrorFunc(errors.InvalidArgument, func(err error) error {
//		if validationErr, ok := err.(ValidationError); ok {
//			return fmt.Errorf("validation failed for field '%s': %s",
//				validationErr.Field(), validationErr.Reason())
//		}
//		return nil
//	})
//
// Thread Safety: This method must be called during application initialization before
// the HTTP server starts, as the internal mapping structures are not protected by mutex.
//
// Registration Order: Functions are stored in a slice and executed in registration order,
// so registering the more specific handlers before the general ones gives you
// priority-based processing.
func (m ErrorCodeMap) MapErrorFunc(e ErrorCodeName, fn func(error) error) ErrorCodeMap {
	return m.MapErrorFuncCtx(e, func(_ *context.Context, err error) error {
		return fn(err)
	})
}

// MapErrorFuncCtx registers a context-aware error transformation function: it
// works exactly like MapErrorFunc but the function also receives the request
// context, so the transformed error can carry request detail (path, user,
// values) or trigger request-scoped side effects.
//
//	codeMap.MapErrorFuncCtx(errors.NotFound, func(ctx *context.Context, err error) error {
//		if errors.Is(err, ErrPlanNotFound) {
//			return fmt.Errorf("%s: %w", ctx.Path(), err)
//		}
//		return nil
//	})
//
// The context is nil when Filter is called directly, outside a request;
// context-aware functions must tolerate a nil context.
func (m ErrorCodeMap) MapErrorFuncCtx(e ErrorCodeName, fn MapErrorFn) ErrorCodeMap {
	m[e] = append(m[e], fn)
	return m
}

// MapErrors registers specific error instances that should be automatically mapped to
// a particular ErrorCodeName when encountered. This method provides exact error matching
// using errors.Is() semantics, making it ideal for mapping well-known errors from
// standard library packages or third-party dependencies.
//
// Error matching uses errors.Is(), which means:
// - Exact error instances are matched
// - Wrapped errors containing the target error are matched
// - Error chains are traversed to find matches
// - Custom As() and Is() implementations are respected
//
// When any of the target errors is encountered (directly or through wrapping),
// the original error is passed through unchanged with the specified ErrorCodeName.
// This preserves error context while providing appropriate HTTP classification.
//
// Parameters:
//   - e: The ErrorCodeName to use when any of the target errors is encountered
//   - targets: One or more specific error instances to map to this ErrorCodeName
//
// Example usage:
//
//	// Standard library errors
//	codeMap.MapErrors(errors.NotFound,
//		sql.ErrNoRows,           // Database: no results
//		os.ErrNotExist)          // File system: file not found
//
//	// Third-party library errors
//	codeMap.MapErrors(errors.AlreadyExists,
//		gorm.ErrDuplicateKey,    // GORM: duplicate key violation
//		mongo.ErrDuplicateKey)   // MongoDB: duplicate key
//
//	// Custom application errors
//	codeMap.MapErrors(errors.PermissionDenied,
//		auth.ErrInsufficientPermissions,
//		auth.ErrTokenExpired,
//		auth.ErrAccountSuspended)
//
//	// Network and connectivity errors
//	codeMap.MapErrors(errors.Unavailable,
//		redis.ErrPoolExhausted,
//		kafka.ErrBrokerNotAvailable)
//
// Implementation Details:
// This method internally creates a transformation function that iterates through
// the target errors and uses errors.Is() to check for matches. The first match
// returns the original error, while no matches return nil to allow other mappings.
//
// Thread Safety: This method must be called during application initialization
// before the HTTP server starts, as it modifies internal mapping structures.
func (m ErrorCodeMap) MapErrors(e ErrorCodeName, targets ...error) ErrorCodeMap {
	targetErrors := make([]error, 0, len(targets))
	for _, target := range targets {
		if target != nil {
			targetErrors = append(targetErrors, target)
		}
	}

	if len(targetErrors) == 0 {
		return m
	}

	return m.MapErrorFunc(e, func(err error) error {
		for _, target := range targetErrors {
			if Is(err, target) {
				return err
			}
		}

		return nil
	})
}

// MapErrorTypes registers error types that should be automatically mapped to a particular
// ErrorCodeName when encountered. This method provides type-based error matching using
// Go's type system and reflection, making it ideal for mapping custom error types,
// interface implementations, and hierarchical error structures.
//
// Type matching works with:
// - Concrete types (structs, built-in types)
// - Interface types (any error implementing the interface)
// - Pointer types (both *Type and Type will match appropriately)
// - Embedded error types within larger structures
//
// When a matching error type is encountered, the original error is returned unchanged,
// preserving all error context and details while providing appropriate HTTP classification.
//
// Parameters:
//   - e: The ErrorCodeName to use when errors of the target types are encountered
//   - targets: One or more type instances representing the types to map
//
// Type Instance Format:
// Pass instances of the types you want to match, not reflect.Type objects:
//
//	// Concrete types
//	codeMap.MapErrorTypes(errors.InvalidArgument,
//		MyValidationError{},     // Value type
//		&MyBusinessError{})      // Pointer type
//
//	// Interface types (use typed nil)
//	codeMap.MapErrorTypes(errors.Unauthenticated,
//		(*AuthError)(nil))       // Interface type
//
// Example usage:
//
//	// Custom application error types
//	codeMap.MapErrorTypes(errors.InvalidArgument,
//		ValidationError{},       // Custom validation errors
//		BusinessRuleError{})     // Business logic violations
//
//	// Third-party library error types
//	codeMap.MapErrorTypes(errors.Internal,
//		&pq.Error{},            // PostgreSQL driver errors
//		&json.SyntaxError{},    // JSON parsing errors
//		&url.Error{})           // URL parsing errors
//
//	// Interface-based error handling
//	codeMap.MapErrorTypes(errors.RequestTimeout,
//		(*TimeoutError)(nil),   // Any error implementing TimeoutError
//		context.DeadlineExceeded) // Context timeouts
//
// Advanced Type Matching:
// The type matching system supports sophisticated error hierarchies:
//
//	// Base error interface
//	type DomainError interface {
//		error
//		Domain() string
//	}
//
//	// Specific implementations
//	type UserError struct { /* ... */ }
//	func (e UserError) Domain() string { return "user" }
//
//	type PaymentError struct { /* ... */ }
//	func (e PaymentError) Domain() string { return "payment" }
//
//	// Map the interface to catch all domain errors
//	codeMap.MapErrorTypes(errors.InvalidArgument, (*DomainError)(nil))
//
// Thread Safety: This method must be called during application initialization
// before the HTTP server starts, as it modifies internal mapping structures.
//
// See MapErrorTypesTransformed() for cases where you want to return the typed
// error instance rather than the original error.
func (m ErrorCodeMap) MapErrorTypes(e ErrorCodeName, targets ...any) ErrorCodeMap {
	return m.mapErrorTypes(e, false, targets...)
}

// MapErrorTypesTransformed registers error types that should be mapped to a particular
// ErrorCodeName, but returns the typed error instance rather than the original error.
// This method is useful when you want to normalize error types or extract specific
// error instances from error chains for consistent handling.
//
// Unlike MapErrorTypes() which returns the original error unchanged, this method
// returns the specific typed error instance that matched, potentially extracted
// from an error chain. This enables error normalization and type-specific processing.
//
// Parameters:
//   - e: The ErrorCodeName to use when errors of the target types are encountered
//   - targets: One or more type instances representing the types to map and extract
//
// Transformation Behavior:
// When an error chain contains multiple wrapped errors and one matches the target type,
// this method extracts and returns the matching typed error instance rather than
// the full wrapped error. This can be useful for standardizing error formats.
//
// Example usage:
//
//	// Extract specific validation errors from chains
//	codeMap.MapErrorTypesTransformed(errors.InvalidArgument, ValidationError{})
//
//	// Input:  fmt.Errorf("user creation failed: %w", ValidationError{Field: "email"})
//	// Output: ValidationError{Field: "email"} (extracted from chain)
//
//	// Normalize database errors
//	codeMap.MapErrorTypesTransformed(errors.Internal, &sql.Error{})
//
//	// Input:  fmt.Errorf("database operation failed: %w", &sql.Error{Code: "23505"})
//	// Output: &sql.Error{Code: "23505"} (extracted and normalized)
//
//	// Extract timeout errors for consistent handling
//	codeMap.MapErrorTypesTransformed(errors.RequestTimeout, (*TimeoutError)(nil))
//
// Use Cases:
// - Error normalization across different error sources
// - Extracting typed errors from complex error chains
// - Standardizing error formats for consistent client responses
// - Type-specific error processing and formatting
// - Removing wrapper context while preserving core error information
//
// Comparison with MapErrorTypes():
// - MapErrorTypes(): Returns original error, preserves full context
// - MapErrorTypesTransformed(): Returns typed instance, normalizes format
//
// Thread Safety: This method must be called during application initialization
// before the HTTP server starts, as it modifies internal mapping structures.
//
// Implementation Note: This method uses the same underlying type matching logic
// as MapErrorTypes() but with different return behavior for the transformation function.
func (m ErrorCodeMap) MapErrorTypesTransformed(e ErrorCodeName, targets ...any) ErrorCodeMap {
	return m.mapErrorTypes(e, true, targets...)
}

func (m ErrorCodeMap) mapErrorTypes(e ErrorCodeName, transformToSpecificTarget bool, targets ...any) ErrorCodeMap {
	for _, target := range targets {
		fn := func(target any) MapErrorFn {
			targetType := reflex.IndirectType(reflect.TypeOf(target))
			if !reflex.IsError(targetType) {
				panic("expected a target type to be an error but got: " + targetType.String())
			}
			var targetErr = reflect.New(targetType).Elem()

			return func(_ *context.Context, err error) error {
				if as(err, target, targetErr, targetType) { // let's use this to support wrapped errors.
					if transformToSpecificTarget {
						return targetErr.Interface().(error)
					}
					return err
				}

				return nil
			}
		}
		m[e] = append(m[e], fn(target))
	}

	return m
}

// clopy of internal errors "as".
func as(err error, target any, targetElemVal reflect.Value, targetType reflect.Type) bool {
	for {
		if reflect.TypeOf(err).AssignableTo(targetType) {
			targetElemVal.Set(reflect.ValueOf(err))
			return true
		}
		if x, ok := err.(interface{ As(any) bool }); ok && x.As(target) {
			return true
		}
		switch x := err.(type) {
		case interface{ Unwrap() error }:
			err = x.Unwrap()
			if err == nil {
				return false
			}
		case interface{ Unwrap() []error }:
			for _, err := range x.Unwrap() {
				if err == nil {
					continue
				}
				if as(err, target, targetElemVal, targetType) {
					return true
				}
			}
			return false
		default:
			return false
		}
	}
}
