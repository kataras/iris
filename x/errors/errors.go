package errors

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/kataras/iris/v12/context"
	"github.com/kataras/iris/v12/x/client"
)

// LogErrorFunc is an alias of a function type which accepts the Iris request context and an error
// and it's fired whenever an error should be logged.
//
// See "OnErrorLog" variable to change the way an error is logged,
// by default the error is logged using the Application's Logger's Error method.
type LogErrorFunc = func(ctx *context.Context, err error)

// LogError can be modified to customize the way an error is logged to the server (most common: internal server errors, database errors et.c.).
// Can be used to customize the error logging, e.g. using Sentry (cloud-based error console).
var LogError LogErrorFunc = func(ctx *context.Context, err error) {
	if ctx == nil {
		slog.Error(err.Error())
		return
	}

	ctx.Application().Logger().Error(err)
}

// SkipCanceled is a package-level setting which by default
// skips the logging of a canceled response or operation.
// See the "Context.IsCanceled()" method and "iris.IsCanceled()" function
// that decide if the error is caused by a canceled operation.
//
// Change of this setting MUST be done on initialization of the program.
var SkipCanceled = true

type (
	// ErrorCodeName is a custom string type represents canonical error names.
	//
	// It contains functionality for safe and easy error populating.
	// See its "Message", "Details", "Data" and "Log" methods.
	ErrorCodeName string

	// ErrorCode represents the JSON form ErrorCode of the Error.
	ErrorCode struct {
		CanonicalName ErrorCodeName `json:"canonical_name" yaml:"CanonicalName"`
		Status        int           `json:"status" yaml:"Status"`
	}
)

// A read-only map of valid http error codes.
var errorCodeMap = make(map[ErrorCodeName]ErrorCode)

// Deprecated: Use Register instead.
var E = Register

// Register registers a custom HTTP Error and returns its canonical name for future use.
// The method "New" is reserved and was kept as it is for compatibility
// with the standard errors package, therefore the "Register" name was chosen instead.
// The key stroke "e" is near and accessible while typing the "errors" word
// so developers may find it easy to use.
//
// See "RegisterErrorCode" and "RegisterErrorCodeMap" for alternatives.
//
// Example:
//
//		var (
//	   		NotFound = errors.Register("NOT_FOUND", http.StatusNotFound)
//		)
//		...
//		NotFound.Details(ctx, "resource not found", "user with id: %q was not found", userID)
//
// This method MUST be called on initialization, before HTTP server starts as
// the internal map is not protected by mutex.
func Register(httpErrorCanonicalName string, httpStatusCode int) ErrorCodeName {
	canonicalName := ErrorCodeName(httpErrorCanonicalName)
	RegisterErrorCode(canonicalName, httpStatusCode)
	return canonicalName
}

// RegisterErrorCode registers a custom HTTP Error.
//
// This method MUST be called on initialization, before HTTP server starts as
// the internal map is not protected by mutex.
func RegisterErrorCode(canonicalName ErrorCodeName, httpStatusCode int) {
	errorCodeMap[canonicalName] = ErrorCode{
		CanonicalName: canonicalName,
		Status:        httpStatusCode,
	}
}

// RegisterErrorCodeMap registers one or more custom HTTP Errors.
//
// This method MUST be called on initialization, before HTTP server starts as
// the internal map is not protected by mutex.
func RegisterErrorCodeMap(errorMap map[ErrorCodeName]int) {
	if len(errorMap) == 0 {
		return
	}

	for canonicalName, httpStatusCode := range errorMap {
		RegisterErrorCode(canonicalName, httpStatusCode)
	}
}

// List of default error codes a server should follow and send back to the client.
var (
	Cancelled          ErrorCodeName = Register("CANCELLED", context.StatusTokenRequired)
	Unknown            ErrorCodeName = Register("UNKNOWN", http.StatusInternalServerError)
	InvalidArgument    ErrorCodeName = Register("INVALID_ARGUMENT", http.StatusBadRequest)
	DeadlineExceeded   ErrorCodeName = Register("DEADLINE_EXCEEDED", http.StatusGatewayTimeout)
	NotFound           ErrorCodeName = Register("NOT_FOUND", http.StatusNotFound)
	AlreadyExists      ErrorCodeName = Register("ALREADY_EXISTS", http.StatusConflict)
	PermissionDenied   ErrorCodeName = Register("PERMISSION_DENIED", http.StatusForbidden)
	Unauthenticated    ErrorCodeName = Register("UNAUTHENTICATED", http.StatusUnauthorized)
	ResourceExhausted  ErrorCodeName = Register("RESOURCE_EXHAUSTED", http.StatusTooManyRequests)
	FailedPrecondition ErrorCodeName = Register("FAILED_PRECONDITION", http.StatusBadRequest)
	Aborted            ErrorCodeName = Register("ABORTED", http.StatusConflict)
	OutOfRange         ErrorCodeName = Register("OUT_OF_RANGE", http.StatusBadRequest)
	Unimplemented      ErrorCodeName = Register("UNIMPLEMENTED", http.StatusNotImplemented)
	Internal           ErrorCodeName = Register("INTERNAL", http.StatusInternalServerError)
	Unavailable        ErrorCodeName = Register("UNAVAILABLE", http.StatusServiceUnavailable)
	DataLoss           ErrorCodeName = Register("DATA_LOSS", http.StatusInternalServerError)
)

// errorFuncCodeMap is a read-only map of error code names and their error functions.
// See HandleError package-level function.
var errorFuncCodeMap = make(map[ErrorCodeName][]func(error) error)

// HandleError handles an error by sending it to the client
// based on the registered error code names and their error functions.
// Returns true if the error was handled, otherwise false.
// If the given "err" is nil then it returns false.
// If the given "err" is a type of validation error then it sends it to the client
// using the "Validation" method.
// If the given "err" is a type of client.APIError then it sends it to the client
// using the "HandleAPIError" function.
//
// See ErrorCodeName.MapErrorFunc and MapErrors methods too.
func HandleError(ctx *context.Context, err error) bool {
	if err == nil {
		return false
	}

	if ctx.IsStopped() {
		return false
	}

	for errorCodeName, errorFuncs := range errorFuncCodeMap {
		for _, errorFunc := range errorFuncs {
			if errToSend := errorFunc(err); errToSend != nil {
				errorCodeName.Err(ctx, errToSend)
				return true
			}
		}
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

	if handleJSONError(ctx, err) {
		return true
	}

	if vErr, ok := err.(ValidationError); ok {
		if vErr == nil {
			return false // consider as not error for any case, this should never happen.
		}

		InvalidArgument.Validation(ctx, vErr)
		return true
	}

	if vErrs, ok := err.(ValidationErrors); ok {
		if len(vErrs) == 0 {
			return false // consider as not error for any case, this should never happen.
		}

		InvalidArgument.Validation(ctx, vErrs...)
		return true
	}

	if apiErr, ok := client.GetError(err); ok {
		handleAPIError(ctx, apiErr)
		return true
	}

	Internal.LogErr(ctx, err)
	return true
}

// Error returns an empty string, it is only declared as a method of ErrorCodeName type in order
// to be a compatible error to be joined within other errors:
//
//	err = fmt.Errorf("%w%w", errors.InvalidArgument, err) OR
//	err = errors.InvalidArgument.Wrap(err)
func (e ErrorCodeName) Error() string {
	return ""
}

type joinedErrors interface{ Unwrap() []error }

// Wrap wraps the given error with this ErrorCodeName.
// It calls the standard errors.Join package-level function.
// See HandleError function for more.
func (e ErrorCodeName) Wrap(err error) error {
	return errors.Join(e, err)
}

// MapErrorFunc registers a function which will validate the incoming error and
// return the same error or overriden in order to be sent to the client, wrapped by this ErrorCodeName "e".
//
// This method MUST be called on initialization, before HTTP server starts as
// the internal map is not protected by mutex.
//
// Example Code:
//
//	errors.InvalidArgument.MapErrorFunc(func(err error) error {
//		stripeErr, ok := err.(*stripe.Error)
//		if !ok {
//			return nil
//		}
//
//		return &errors.Error{
//			Message: stripeErr.Msg,
//			Details: stripeErr.DocURL,
//		}
//	})
func (e ErrorCodeName) MapErrorFunc(fn func(error) error) {
	errorFuncCodeMap[e] = append(errorFuncCodeMap[e], fn)
}

// MapError registers one or more errors which will be sent to the client wrapped by this "e" ErrorCodeName
// when the incoming error matches to at least one of the given "targets" one.
//
// This method MUST be called on initialization, before HTTP server starts as
// the internal map is not protected by mutex.
func (e ErrorCodeName) MapErrors(targets ...error) {
	e.MapErrorFunc(func(err error) error {
		for _, target := range targets {
			if Is(err, target) {
				return err
			}
		}

		return nil
	})
}

// Message sends an error with a simple message to the client.
func (e ErrorCodeName) Message(ctx *context.Context, format string, args ...interface{}) {
	fail(ctx, e, sprintf(format, args...), "", nil, nil)
}

// Details sends an error with a message and details to the client.
func (e ErrorCodeName) Details(ctx *context.Context, msg, details string, detailsArgs ...interface{}) {
	fail(ctx, e, msg, sprintf(details, detailsArgs...), nil, nil)
}

// Data sends an error with a message and json data to the client.
func (e ErrorCodeName) Data(ctx *context.Context, msg string, data interface{}) {
	fail(ctx, e, msg, "", nil, data)
}

// DataWithDetails sends an error with a message, details and json data to the client.
func (e ErrorCodeName) DataWithDetails(ctx *context.Context, msg, details string, data interface{}) {
	fail(ctx, e, msg, details, nil, data)
}

// Validation sends an error which renders the invalid fields to the client.
func (e ErrorCodeName) Validation(ctx *context.Context, validationErrors ...ValidationError) {
	e.validation(ctx, validationErrors)
}

func (e ErrorCodeName) validation(ctx *context.Context, validationErrors interface{}) {
	fail(ctx, e, "validation failure", "fields were invalid", validationErrors, nil)
}

// Err sends the error's text as a message to the client.
// In exception, if the given "err" is a type of validation error
// then the Validation method is called instead.
func (e ErrorCodeName) Err(ctx *context.Context, err error) {
	if err == nil {
		return
	}

	if vErr, ok := err.(ValidationError); ok {
		if vErr == nil {
			return // consider as not error for any case, this should never happen.
		}

		e.Validation(ctx, vErr)
	}

	if vErrs, ok := err.(ValidationErrors); ok {
		if len(vErrs) == 0 {
			return // consider as not error for any case, this should never happen.
		}

		e.Validation(ctx, vErrs...)
	}

	// If it's already an Error type then send it directly.
	if httpErr, ok := err.(*Error); ok {
		if errorCode, ok := errorCodeMap[e]; ok {
			httpErr.ErrorCode = errorCode
			ctx.StopWithJSON(errorCode.Status, httpErr) // here we override the fail function and send the error as it is.
			return
		}
	}

	e.Message(ctx, err.Error())
}

// Log sends an error of "format" and optional "args" to the client and prints that
// error using the "LogError" package-level function, which can be customized.
//
// See "LogErr" too.
func (e ErrorCodeName) Log(ctx *context.Context, format string, args ...interface{}) {
	if SkipCanceled {
		if ctx.IsCanceled() {
			return
		}

		for _, arg := range args {
			if err, ok := arg.(error); ok {
				if context.IsErrCanceled(err) {
					return
				}
			}
		}
	}

	err := fmt.Errorf(format, args...)
	e.LogErr(ctx, err)
}

// LogErr sends the given "err" as message to the client and prints that
// error to using the "LogError" package-level function, which can be customized.
func (e ErrorCodeName) LogErr(ctx *context.Context, err error) {
	if SkipCanceled && (ctx.IsCanceled() || context.IsErrCanceled(err)) {
		return
	}

	LogError(ctx, err)

	e.Message(ctx, "server error")
}

// HandleAPIError handles remote server errors.
// Optionally, use it when you write your server's HTTP clients using the the /x/client package.
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
	statusCode := apiErr.Response.StatusCode
	if statusCode >= 400 && statusCode < 500 {
		InvalidArgument.DataWithDetails(ctx, "remote server error", "invalid client request", apiErr.Body)
	} else {
		Internal.Data(ctx, "remote server error", apiErr.Body)
	}

	// Unavailable.DataWithDetails(ctx, "remote server error", "unavailable", apiErr.Body)
}

func handleJSONError(ctx *context.Context, err error) bool {
	var syntaxErr *json.SyntaxError
	if errors.As(err, &syntaxErr) {
		InvalidArgument.Details(ctx, "unable to parse body", "syntax error at byte offset %d", syntaxErr.Offset)
		return true
	}

	var unmarshalErr *json.UnmarshalTypeError
	if errors.As(err, &unmarshalErr) {
		InvalidArgument.Details(ctx, "unable to parse body", "unmarshal error for field %q", unmarshalErr.Field)
		return true
	}

	return false
	// else {
	// 	InvalidArgument.Details(ctx, "unable to parse body", err.Error())
	// }
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
//	https://github.com/kataras/iris/tree/main/_examples/routing/http-wire-errors.
type Error struct {
	ErrorCode  ErrorCode       `json:"http_error_code" yaml:"HTTPErrorCode"`
	Message    string          `json:"message,omitempty" yaml:"Message"`
	Details    string          `json:"details,omitempty" yaml:"Details"`
	Validation interface{}     `json:"validation,omitempty" yaml:"Validation,omitempty"`
	Data       json.RawMessage `json:"data,omitempty" yaml:"Data,omitempty"` // any other custom json data.
}

// Error method completes the error interface. It just returns the canonical name, status code, message and details.
func (err *Error) Error() string {
	if err.Message == "" {
		err.Message = "<empty>"
	}

	if err.Details == "" {
		err.Details = "<empty>"
	}

	if err.ErrorCode.CanonicalName == "" {
		err.ErrorCode.CanonicalName = ErrUnexpected
	}

	if err.ErrorCode.Status <= 0 {
		err.ErrorCode.Status = http.StatusInternalServerError
	}

	return sprintf("iris http wire error: canonical name: %s, http status code: %d, message: %s, details: %s", err.ErrorCode.CanonicalName, err.ErrorCode.Status, err.Message, err.Details)
}

func fail(ctx *context.Context, codeName ErrorCodeName, msg, details string, validationErrors interface{}, dataValue interface{}) {
	errorCode, ok := errorCodeMap[codeName]
	if !ok {
		// This SHOULD NEVER happen, all ErrorCodeNames MUST be registered.
		LogError(ctx, ErrUnexpectedErrorCode)
		fail(ctx, ErrUnexpected, msg, details, validationErrors, dataValue)
		return
	}

	var data json.RawMessage
	if dataValue != nil {
		switch v := dataValue.(type) {
		case json.RawMessage:
			data = v
		case []byte:
			data = v
		case error:
			if msg == "" {
				msg = v.Error()
			} else if details == "" {
				details = v.Error()
			} else {
				data = json.RawMessage(v.Error())
			}
		default:
			b, err := json.Marshal(v)
			if err != nil {
				LogError(ctx, err)
				fail(ctx, ErrUnexpected, err.Error(), "", nil, nil)
				return
			}
			data = b

		}
	}

	err := Error{
		ErrorCode:  errorCode,
		Message:    msg,
		Details:    details,
		Data:       data,
		Validation: validationErrors,
	}

	// ctx.SetErr(&err)
	ctx.StopWithJSON(errorCode.Status, err)
}
