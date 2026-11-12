# Iris REST Sub-Package Documentation

The `rest` tree holds the packages that write API errors and typed payloads: `rest/errors` (this document), `rest/types` (scalar JSON types such as `SimpleDate`) and `rest/math`. The response helpers themselves are methods of the Iris Context, so a handler decodes its input, calls a service, and hands the result over in one line: `ctx.BindJSON`, `ctx.OK`, `ctx.Created`, `ctx.NoContent`, `ctx.Count` and `ctx.Paginated`. This document covers the `rest/errors` package and its options, together with those Context methods and `errors.HandleError`.

## rest/errors Package

The `rest/errors` package provides utilities for error handling, including custom error options and predefined error functions.

```go
import "github.com/kataras/iris/v14/rest/errors"
```

### Aliases

The package includes aliases for standard error functions:

- `Is`: Alias for [errors.Is](https://pkg.go.dev/errors#Is).
- `As`: Alias for [errors.As](https://pkg.go.dev/errors#As).
- `New`: Alias for [errors.New](https://pkg.go.dev/errors#New).
- `Unwrap`: Alias for [errors.Unwrap](https://pkg.go.dev/errors#Unwrap).
- `Join`: Alias for [errors.Join](https://pkg.go.dev/errors#Join).

### Options

The `Options` struct configures error handling. Its methods set the error collector, the renderer and the ID generator.

#### Creating Options

To create a new `Options` instance:

```go
opts := errors.NewOptions()
```

#### Setting Custom Error Collector

To set a custom error collector:

```go
opts.SetCollector(func(ctx iris.Context, err *errors.Error) {
    // Custom error collection logic
})
```

#### Setting Custom Error Renderer

To set a custom error renderer:

```go
opts.SetRenderer(func(ctx iris.Context, err *errors.Error) {
    // Custom error rendering logic
})
```

#### Setting Custom ID Generator

To set a custom ID generator:

```go
opts.SetIDGenerator(func(ctx iris.Context) string {
    return "custom-id"
})
```

#### Registering Error Functions

To register custom error functions with predefined HTTP wire errors:

```go
// Define some custom domain-level errors.
var (
    ErrNoRows = errors.New("resource not found")
    ErrDuplicated = errors.New("resource already exists")
    ErrForeignKey = errors.New("reference entity does not exist")
)

```

In another part of your application, when initializing options, map the domain-level errors to HTTP wire errors. This ensures that any time a service call returns an error, it can be automatically handled by the Iris HTTP layer.

```go
// Mapping HTTP Wire Errors with Domain-Level Errors
opts.
    MapErrors(errors.NotFound, ErrNoRows).
    MapErrors(errors.AlreadyExists, ErrDuplicated).
    MapErrors(errors.InvalidArgument, ErrForeignKey)
```

### HTML Renderer

The `HTML` function renders errors as HTML pages. With a view engine registered it renders the `errors/$status_code.$ext` template (for example `errors/500.html`); without one, or when that template is missing, it writes a built-in page with the status, the canonical name and the message.

```go
opts.SetRenderer(errors.HTML)
```

### Applying Options Middleware

To apply the options middleware to the Iris router:

```go
app.Use(opts.Handler())
```

### Error Handling Methods

The `errors` package provides several methods to handle and send errors to the client.

#### Err

The `Err` method sends the error's text as a message to the client. If the given error is a type of validation error, then the `Validation` method is called instead.

```go
func (e ErrorCodeName) Err(ctx iris.Context, err error)
```

##### Example

```go
func handleRequest(ctx iris.Context) {
    err := someFunction()
    if err != nil {
        errors.InvalidArgument.Err(ctx, err)
        return
    }
}
```

#### Message

The `Message` and `Messagef` methods send an error with a simple message to the client.

```go
func (e ErrorCodeName) Message(ctx iris.Context, message string)
func (e ErrorCodeName) Messagef(ctx iris.Context, format string, args ...any)
```

##### Example

```go
func handleRequest(ctx iris.Context) {
    if someCondition {
        errors.InvalidArgument.Message(ctx, "Invalid argument provided")
        return
    }
}
```

JSON output:

```json
{
    "id": "c30adf71-25c0-4a2b-a1f2-c3ee17efcd17",
    "http_error_code": {
        "canonical_name": "INVALID_ARGUMENT",
        "status": 400
    },
    "message": "Invalid argument provided"
}
```

#### Detail

The `Detail` and `Detailf` methods send an error with a message and details to the client.

```go
func (e ErrorCodeName) Detail(ctx iris.Context, msg, detail string)
func (e ErrorCodeName) Detailf(ctx iris.Context, msg, detail string, detailsArgs ...any)
```

##### Example

```go
func handleRequest(ctx iris.Context) {
    if someCondition {
        errors.InvalidArgument.Detailf(ctx, "Invalid argument", "The provided argument %s is invalid", "argValue")
        return
    }
}
```

JSON output:

```json
{
    "id": "c30adf71-25c0-4a2b-a1f2-c3ee17efcd17",
    "http_error_code": {
        "canonical_name": "INVALID_ARGUMENT",
        "status": 400
    },
    "message": "Invalid argument",
    "detail": "The provided argument argValue is invalid"
}
```

#### Data

The `Data` method sends an error with a message and JSON data to the client.

```go
func (e ErrorCodeName) Data(ctx iris.Context, msg string, data any)
```

##### Example

```go
func handleRequest(ctx iris.Context) {
    if someCondition {
        errors.InvalidArgument.Data(ctx, "Invalid argument", iris.Map{"field": "value"})
        return
    }
}
```

JSON output:

```json
{
    "id": "c30adf71-25c0-4a2b-a1f2-c3ee17efcd17",
    "http_error_code": {
        "canonical_name": "INVALID_ARGUMENT",
        "status": 400
    },
    "message": "Invalid argument",
    "data": {
        "field": "value"
    }
}
```

#### Data With Detail

The `DataWithDetail` method sends an error with a message, details, and JSON data to the client.

```go
func (e ErrorCodeName) DataWithDetail(ctx iris.Context, msg, detail string, data any)
```

##### Example

```go
func handleRequest(ctx iris.Context) {
    if someCondition {
        errors.InvalidArgument.DataWithDetail(ctx, "Invalid argument", "The provided argument is invalid", iris.Map{"field": "value"})
        return
    }
}
```

JSON output:

```json
{
    "id": "c30adf71-25c0-4a2b-a1f2-c3ee17efcd17",
    "http_error_code": {
        "canonical_name": "INVALID_ARGUMENT",
        "status": 400
    },
    "message": "Invalid argument",
    "detail": "The provided argument is invalid",
    "data": {
        "field": "value"
    }
}
```

#### Log

The `Log` method logs the given message and sends a generic error to the client if the `ErrorCodeName` is `Internal`. The output target can be customized by setting the `LogError` package-level variable, or per options instance with `Options.SetLogger`.

```go
func (e ErrorCodeName) Log(ctx iris.Context, format string, args ...any)
```

##### Example

```go
func handleRequest(ctx iris.Context) {
    if someCondition {
        errors.InvalidArgument.Log(ctx, "An error occurred: %s", "details")
        return
    }
}
```

#### LogErr

The `LogErr` method logs the given error and sends a generic error to the client if the `ErrorCodeName` is `Internal`.

```go
func (e ErrorCodeName) LogErr(ctx iris.Context, err error)
```

##### Example

```go
func handleRequest(ctx iris.Context) {
    err := someFunction()
    if err != nil {
        errors.InvalidArgument.LogErr(ctx, err)
        return
    }
}
```

#### Validation Errors

The `Validation` method sends an error which renders the invalid fields to the client.

```go
func (e ErrorCodeName) Validation(ctx iris.Context, validationErrors ...ValidationError)
```

##### Example

```go
func handleRequest(ctx iris.Context) {
    if someCondition {
        errors.InvalidArgument.Validation(ctx,
            &validation.FieldError[string]{Field: "name", Value: "Jo", Reason: "name must be at least 3 characters long"},
            &validation.FieldError[string]{Field: "email", Value: " ", Reason: "email must be a valid email address"},
        )
        return
    } // there are some pre-defined helpers for the FieldError, see the rest/errors/validation package for more.
}
```

> Note that the `Validation` call is optionally because every `err` error implements the `ValidationError` interface can be handled as validation error by the `Err` method as well. This example showcases how you can check and maybe skip these type of errors from your request logging logic.


JSON output:

```json
{
    "id": "c30adf71-25c0-4a2b-a1f2-c3ee17efcd17",
    "http_error_code": {
        "canonical_name": "INVALID_ARGUMENT",
        "status": 400
    },
    "message": "validation failure",
    "detail": "fields were invalid",
    "validation": [
        {
            "field": "name",
            "value": "Jo",
            "reason": "name must be at least 3 characters long"
        },
        {
            "field": "email",
            "value": " ",
            "reason": "email must be a valid email address"
        } 
    ]
}
```

### Problem Details

The `Problem` method sends an error as a [RFC 7807 Problem Details](https://tools.ietf.org/html/rfc7807) response. This RFC defines a standardized format for representing errors in HTTP APIs, making it easier for clients to understand and handle errors effectively.

#### Problem Details Structure

Each problem details response will follow a standardized JSON structure with the following fields:

- `type`: A URI reference that identifies the problem type. This URI should provide human-readable documentation for the problem type when dereferenced.
- `title`: A short, human-readable summary of the problem type. It should not change from occurrence to occurrence of the problem, except for localization purposes.
- `status`: The HTTP status code generated by the origin server for this occurrence of the problem.
- `detail`: A human-readable explanation specific to this occurrence of the problem.
- `instance`: A URI reference that identifies the specific occurrence of the problem. It may or may not yield further information if dereferenced.
- `id`: A unique identifier for the error instance.
- `extra_custom_key`: Any additional custom key-value pairs that provide more context about the error.

#### Example

```json
{
    "id": "bd7026f3-28c9-4fb8-9a09-3e1ce0671226",
    "status": 400,
    "title": "problem_title",
    "detail": "problem_detail",
    "type": "problem_type",
    "instance": "problem_instance",
    "extra_custom_key": "extra_custom_value",
    "extra_custom_key2": "extra_custom_value2"
}
```

#### Usage

The `Problem` method can be used to send a problem details response. Here is an example of how to use it in a handler:

```go
func handleRequest(ctx iris.Context) {
    if someCondition {
        errors.InvalidArgument.Problem(ctx,
            "problem_title",
            "problem_detail",
            "problem_type",
            "problem_instance",
            iris.Map{
                "extra_custom_key":  "extra_custom_value",
                "extra_custom_key2": "extra_custom_value2",
            },
        )
        return
    }
}
```

### JSON Output

```json
{
    "id": "bd7026f3-28c9-4fb8-9a09-3e1ce0671226",
    "status": 400,
    "title": "problem_title",
    "detail": "problem_detail",
    "type": "problem_type",
    "instance": "problem_instance",
    "extra_custom_key": "extra_custom_value",
    "extra_custom_key2": "extra_custom_value2"
}
```

### More Information

RFC 7807, titled "Problem Details for HTTP APIs," provides a standardized format for expressing errors in HTTP responses. This format is designed to be both human-readable and machine-readable, making it easier for clients to understand and handle errors. The RFC specifies that the problem details object should be used as the response body when the server encounters an error that it wants to convey to the client.

For more details, see [RFC 7807](https://datatracker.ietf.org/doc/html/rfc7807).

### Predefined HTTP Wire Errors

The `errors` package provides several predefined HTTP wire errors, each associated with its own status code, to handle common error scenarios.

#### `errors.FailedPrecondition`

Indicates that the operation was rejected because the system is not in a state required for the operation's execution.

##### Example

```go
func handleRequest(ctx iris.Context) {
    if someCondition {
        errors.FailedPrecondition.Message(ctx, "Failed precondition")
        return
    }
}
```

#### `errors.OutOfRange`

Indicates that an operation was attempted past the valid range.

##### Example

```go
func handleRequest(ctx iris.Context) {
    if someCondition {
        errors.OutOfRange.Message(ctx, "Out of range")
        return
    }
}
```

#### `errors.UnsupportedMediaType`

Indicates that the request entity has a media type which the server or resource does not support.

##### Example

```go
func handleRequest(ctx iris.Context) {
    if someCondition {
        errors.UnsupportedMediaType.Message(ctx, "Unsupported media type")
        return
    }
}
```

#### `errors.DeadlineExceeded`

Indicates that the deadline expired before the operation could complete.

##### Example

```go
func handleRequest(ctx iris.Context) {
    if someCondition {
        errors.DeadlineExceeded.Message(ctx, "Deadline exceeded")
        return
    }
}
```

#### `errors.NotFound`

Indicates that the requested resource was not found.

##### Example

```go
func handleRequest(ctx iris.Context) {
    if someCondition {
        errors.NotFound.Message(ctx, "Resource not found")
        return
    }
}
```

#### `errors.MethodNotAllowed`

Indicates that the HTTP method is not allowed for the requested resource.

##### Example

```go
func handleRequest(ctx iris.Context) {
    if someCondition {
        errors.MethodNotAllowed.Message(ctx, "Method not allowed")
        return
    }
}
```

#### `errors.AlreadyExists`

Indicates that the resource already exists.

##### Example

```go
func handleRequest(ctx iris.Context) {
    if someCondition {
        errors.AlreadyExists.Message(ctx, "Resource already exists")
        return
    }
}
```

#### `errors.Aborted`

Indicates that the operation was aborted, typically due to a concurrency issue.

##### Example

```go
func handleRequest(ctx iris.Context) {
    if someCondition {
        errors.Aborted.Message(ctx, "Operation aborted")
        return
    }
}
```

#### `errors.PermissionDenied`

Indicates that the caller does not have permission to execute the specified operation.

##### Example

```go
func handleRequest(ctx iris.Context) {
    if someCondition {
        errors.PermissionDenied.Message(ctx, "Permission denied")
        return
    }
}
```

#### `errors.Unauthenticated`

Indicates that the request does not have valid authentication credentials for the operation.

##### Example

```go
func handleRequest(ctx iris.Context) {
    if someCondition {
        errors.Unauthenticated.Message(ctx, "Unauthenticated")
        return
    }
}
```

#### `errors.ResourceExhausted`

Indicates that some resource has been exhausted, perhaps a per-user quota, or perhaps the entire file system is out of space.

##### Example

```go
func handleRequest(ctx iris.Context) {
    if someCondition {
        errors.ResourceExhausted.Message(ctx, "Resource exhausted")
        return
    }
}
```

#### `errors.Unimplemented`

Indicates that the operation is not implemented or not supported/enabled in this service.

##### Example

```go
func handleRequest(ctx iris.Context) {
    if someCondition {
        errors.Unimplemented.Message(ctx, "Operation not implemented")
        return
    }
}
```

#### `errors.Unavailable`

Indicates that the service is currently unavailable. This is a most likely a transient condition and may be corrected by retrying with a backoff.

##### Example

```go
func handleRequest(ctx iris.Context) {
    if someCondition {
        errors.Unavailable.Message(ctx, "Service unavailable")
        return
    }
}
```

#### `errors.Internal`

Indicates an internal server error.

##### Example

```go
func handleRequest(ctx iris.Context) {
    if someCondition {
        errors.Internal.Message(ctx, "Internal server error")
        return
    }
}
```

#### `errors.DataLoss`

Indicates that unrecoverable data loss or corruption has occurred.

##### Example

```go
func handleRequest(ctx iris.Context) {
    if someCondition {
        errors.DataLoss.Message(ctx, "Data loss")
        return
    }
}
```

<details><summary>and more...</summary>

- TooLarge
- BadGateway
- Unknown
- PaymentRequired
- NotAcceptable
- ProxyAuthRequired
- RequestTimeout
- Cancelled
- Gone
- LengthRequired
- PreconditionFailed
- RequestURITooLong
- RequestedRangeNotSatisfiable
- ExpectationFailed
- ImATeapot
- MisdirectedRequest
- UnprocessableEntity
- Locked
- FailedDependency
- TooEarly
- UpgradeRequired
- PreconditionRequired
- RequestHeaderFieldsTooLarge
- UnavailableForLegalReasons
- HTTPVersionNotSupported
- VariantAlsoNegotiates
- InsufficientStorage
- LoopDetected
- NotExtended
- NetworkAuthenticationRequired
- PageExpired
- BlockedByWindowsParentalControls
- InvalidToken
- TokenRequired
- BandwidthLimitExceeded
- InvalidSSLCertificate
- SiteOverloaded
- SiteFrozen
- NetworkReadTimeout

</details>

### Handling Detailed Errors with Nested Data

An error can carry nested data, so each part of a complex failure keeps its own context instead of being flattened into one message.

#### Example

Consider the following example where an error message is composed of multiple parts, each with its own section of data:

```json
{
    "id": "1843a33a-1ad9-4b63-853a-9e11eab06521",
    "http_error_code": {
        "canonical_name": "INVALID_ARGUMENT",
        "status": 400
    },
    "message": "error generating new weekly plan: error generating new daily plan: no matched planning entry found",
    "data": [
        {
            "message": "error generating new weekly plan",
            "data": {
                "generate_options": {
                    "include_diet_type": true,
                    "include_food_preferences": true,
                    "waking_time": "7:04 AM",
                    "bed_time": "11:55 PM",
                    "lunch_time": "3:04 PM",
                    "intermittent_fasting_start_time": "1:04 PM",
                    "intermittent_fasting_end_time": "2:04 PM",
                    "daily_meal_types": [
                        1,
                        3,
                        5
                    ]
                }
            }
        },
        {
            "message": "error generating new daily plan",
            "data": {
                "date": "2024-11-25",
                "weekday": "monday"
            }
        },
        {
            "message": "no matched planning entry found",
            "data": {
                "initial_intake_target": 2578,
                "diet_type": 0,
                "fitness_goal": 5,
                "daily_meal_types": [
                    1,
                    3,
                    5
                ],
                "biggest_meal_type": 4,
                "has_prescribed_activity": true
            }
        }
    ]
}
```

In this example, the error message is composed of three parts:
1. `error generating new weekly plan`
2. `error generating new daily plan`
3. `no matched planning entry found`

Each part of the error message has its own section of data, providing detailed context for each part of the error. This structure allows clients to understand the specific context and data related to each part of the error message, making it easier to debug and resolve issues.

### JSON Output Examples

Here are additional JSON output examples for HTTP wire errors, considering various business scenarios such as a banking system:

#### Example 1: Invalid Account Number

```json
{
    "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "http_error_code": {
        "canonical_name": "INVALID_ARGUMENT",
        "status": 400
    },
    "message": "invalid account number provided",
    "detail": "the account number '123456789' does not exist in our records"
}
```

#### Example 2: Insufficient Funds

```json
{
    "id": "b2c3d4e5-f6a7-8901-bcde-f23456789012",
    "http_error_code": {
        "canonical_name": "FAILED_PRECONDITION",
        "status": 400
    },
    "message": "insufficient funds",
    "detail": "the account '987654321' has insufficient funds for the transaction",
    "data": {
        "current_balance": 50.00,
        "required_balance": 100.00
    }
}
```

#### Example 3: Unauthorized Access

```json
{
    "id": "c3d4e5f6-a7b8-9012-cdef-345678901234",
    "http_error_code": {
        "canonical_name": "UNAUTHENTICATED",
        "status": 401
    },
    "message": "unauthorized access",
    "detail": "the provided credentials are invalid"
}
```

OR...

```json
{
    "id": "b445b036-fd2e-4583-b4cc-61c2b8fb0b73",
    "http_error_code": {
        "canonical_name": "UNAUTHENTICATED",
        "status": 401
    },
    "message": "auth: verify: jwt: unknown kid"
}
```

#### Example 4: Account Locked

```json
{"id": "d4e5f6a7-b8c9-0123-def4-567890123456",
    "http_error_code": {
        "canonical_name": "PERMISSION_DENIED",
        "status": 403
    },
    "message": "account locked",
    "detail": "the account '123456789' is locked due to multiple failed login attempts",
    "data": {
        "lock_duration": "24 hours"
    }
}
```

#### Example 5: Transaction Not Found

```json
{
    "id": "e5f6a7b8-c9d0-1234-ef56-789012345678",
    "http_error_code": {
        "canonical_name": "NOT_FOUND",
        "status": 404
    },
    "message": "transaction not found",
    "detail": "the transaction with ID 'tx1234567890' was not found"
}
```

#### Example 6: Internal Server Error

```json
{
    "id": "f6a7b8c9-d0e1-2345-f678-901234567890",
    "http_error_code": {
        "canonical_name": "INTERNAL",
        "status": 500
    },
    "message": "server error",
    "detail": "an unexpected error occurred while processing the request"
}
```

## ctx.BindJSON

The `BindJSON` generic method reads a JSON payload and returns it as a generic type `Request`. It also returns a boolean value indicating whether the read was successful or not. If the read fails, it sends an appropriate error response to the client.

When `Request` implements the `Validatable` interface (`Validate() error`), the method runs automatically after decoding, and a non-nil result rejects the request with a client error response. Handlers do not call `Validate` themselves. Use `BindJSONOptional` when the request body is optional: an empty body returns the zero `Request` with `true` and skips validation.

```go
func (ctx *Context) BindJSON[Request any]() (Request, bool)
func (ctx *Context) BindJSONOptional[Request any]() (Request, bool)
```

#### Example

```go
func createFoodBookmark(ctx iris.Context) {
    entry, ok := ctx.BindJSON[customer.CustomerFoodBookmark]()
    if !ok {
        return
    }
    // [...]
}
```

## Response Handling Methods

The Iris Context provides several methods to handle API responses. Each one takes the service call's result and error, renders a non-nil error through the `rest/errors` pipeline and returns a boolean indicating whether the handle was successful. The types they use (`PageOptions`, `Page[T]`, `CountResponse[N]`, `IDPayload[T]`, `Validatable`) are exported by the `context` package and aliased by the root `iris` package.

### ctx.OK

The `OK` method handles a generic response and error from a service call and sends a JSON response to the client. It returns a boolean indicating whether the handle was successful.

```go
func (ctx *Context) OK(resp any, err error) bool
```

#### Description

- **Error Handling**: If the error is not `nil`, `ctx.OK` calls `HandleError` to send an appropriate error response to the client.
- **Response Handling**: Sends a JSON response with the provided data. A `nil` response writes no body.
- **Return Value**: Returns a boolean value indicating whether the handle was successful or not.

#### Example

```go
func getFoodBookmark(ctx iris.Context) {
    id := getFoodBookmarkID(ctx)

    entry, err := api.presenters.GetCustomerFoodBookmark(ctx, id)
    ctx.OK(entry, err)
}
```

### ctx.Created

The `Created` method handles a generic response and error from a service call, sending a 201 JSON response to the client. This method simplifies error handling and response formatting for create operations in your APIs.

```go
func (ctx *Context) Created(resp any, err error) bool
```

#### Description

- **Error Handling**:  If the error is not `nil`, `ctx.Created` calls `HandleError` to send an appropriate error response to the client.
- **Response Handling**: If the response value is a `string`, `int` (including `int64` and `uint64`), or the standard library's `uuid.UUID` (the `context.IDType` set), it writes a JSON object with a single `id` field containing the given response value (`context.IDPayload`). Any other value, including a `[]string`, is written as it is.
- **Return Value**: Returns a boolean value indicating whether the handle was successful or not.

#### Example

```go
func createFoodBookmark(ctx iris.Context) {
    entry, ok := ctx.BindJSON[customer.CustomerFoodBookmark]()
    if !ok {
        return
    }
    entry.CustomerID = getCustomerID(ctx)

    id, err := api.customers.CreateCustomerFoodBookmark(ctx, entry)
    ctx.Created(id, err)
}
```

### ctx.NoContent

The `NoContent` method handles an error from a service call and sends a 204 No Content or a 304 Not Modified empty response to the client. It returns a boolean indicating whether the handle was successful.

```go
func (ctx *Context) NoContent(modified bool, err error) bool
```

#### Description

- **Error Handling**:  If the error is not `nil`, `ctx.NoContent` calls `HandleError` to send an appropriate error response to the client.
- **Response Handling**: A `true` "modified" sends `204` No Content, a `false` one `304` Not Modified. Both bodies are empty.
- **Return Value**: Returns a boolean value indicating whether the handle was successful or not.

#### Example

```go
func deleteFoodBookmark(ctx iris.Context) {
    id := getFoodBookmarkID(ctx)
    customerID := getCustomerID(ctx)

    deleted, err := api.customers.DeleteCustomerFoodBookmarkByID(ctx, customerID, id)
    ctx.NoContent(deleted, err)
}
```

### ctx.JSONOrError

The `JSONOrError` method sends a JSON response and, when encoding or writing it fails, renders that failure through the same error pipeline as a server error. It is the response half of `OK`, `Created`, `Count` and `Paginated`; use it when the status code or the value needs preparing first.

```go
func (ctx *Context) JSONOrError(resp any) bool
```

### ctx.Paginated

The `Paginated` generic method reads pagination options from the request context, calls the given collect function to collect items, and returns them in a paginated response. It returns a boolean indicating whether the handle was successful.

```go
func (ctx *Context) Paginated[Filter any, Item any, TotalCount CountType](collect ItemsCollectorFunc[Filter, Item, TotalCount]) bool
```

#### Description

- **Error Handling**:  If the error is not `nil`, `ctx.Paginated` calls `HandleError` to send an appropriate error response to the client.
- **Response Handling**: Collects items using the provided `collect` function and sends a paginated JSON response.
- **Return Value**: Returns a boolean value indicating whether the handle was successful or not.

#### Example

```go
func listCustomerFoodBookmarks(ctx iris.Context) {
    customerID := getCustomerID(ctx)

    ctx.Paginated(func(_ context.Context, pageOpts iris.PageOptions, filter customer.Filter) ([]customer.CustomerFoodBookmark, int64, error) {
        items, totalCount, err := api.presenters.ListCustomerFoodBookmarks(ctx, pageOpts, filter, customerID)
        return items, totalCount, err
    })
}
```

#### JSON Output Example

```sh
curl -i -X POST "http://localhost:8080/api/bookmarks/list?page=1&size=20"
```

Here is an example of the JSON output returned by the `Paginated` function:

```json
{
    "current_page": 1,
    "page_size": 2,
    "total_pages": 1,
    "total_items": 2,
    "has_next_page": false,
    "filter": null,
    "items": [
        {
            "id": 1,
            "name": "Bookmark 1"
        },
        {
            "id": 2,
            "name": "Bookmark 2"
        }
    ]
}
```

This JSON output represents a paginated response with two items, indicating that there is only one page and no next page available.

Note: The `filter` field represents the filter used to retrieve the items. The filter is optionally a JSON request body and can be included if needed.

### errors.HandleError

The `HandleError` function handles the error and sends an appropriate response to the client. It is what the Context methods above call for a non-nil error.

```go
func HandleError(ctx iris.Context, err error) bool
```

### Description

- **Error Handling**: Handles the error and sends an appropriate response to the client.
- **Return Value**: Returns a boolean value indicating whether an error response was written; when it is `true` the handler should `return`.

#### Example

```go
if errors.HandleError(ctx, err) {
    return
}
```

### Migration

This section demonstrates how you can reduce boilerplate code in your handlers by using the Context's binding and response methods with the `rest/errors` pipeline. Below is a comparison between the old and new code for creating a food bookmark.

#### Old Code

In the old code, you manually handle JSON parsing, error checking, and response formatting. This results in repetitive and verbose code.

```go
func (api *FoodBookmarksAPI) createFoodBookmark(ctx iris.Context) {
    var entry customer.CustomerFoodBookmark
    err := ctx.ReadJSON(&entry)
    if err != nil {
        ctx.StopWithJSON(iris.StatusBadRequest, iris.Map{
            "message": "unable to parse body",
            "detail": err.Error()},
        )
        return
    }
    entry.CustomerID = getCustomerID(ctx)
    id, err := api.customers.CreateCustomerFoodBookmark(ctx, entry)
    if err != nil {
        if entity.IsErrDuplicate(err) {
            ctx.StopWithJSON(iris.StatusConflict, iris.Map{"message": "already exists"})
        } else if entity.IsErrForeignKey(err) {
            ctx.StopWithJSON(iris.StatusBadRequest, iris.Map{"message": "reference is invalid"})
        } else {
            ctx.StopWithJSON(iris.StatusInternalServerError, iris.Map{"message": "server error"})
            ctx.Application().Logger().Error(err)
        }
        return
    }
    ctx.StatusCode(iris.StatusCreated)
    ctx.JSON(iris.Map{"id": id})
}
```

#### New Code

In the new code, the Context methods simplify the process by handling JSON parsing, error handling, and response formatting. This reduces boilerplate code and makes the handler more concise and readable.

```go
func (api *FoodBookmarksAPI) createFoodBookmark(ctx iris.Context) {
    entry, ok := ctx.BindJSON[customer.CustomerFoodBookmark]()
    if !ok {
        return
    }
    entry.CustomerID = getCustomerID(ctx)
    id, err := api.customers.CreateCustomerFoodBookmark(ctx, entry)
    ctx.Created(id, err)
}
```

#### Explanation

- **Old Code**:
  - Manually reads JSON from the request body using `ctx.ReadJSON`.
  - Checks for errors and sends appropriate HTTP responses using `ctx.StopWithJSON`.
  - Handles different error scenarios (e.g., duplicate entry, foreign key violation) with specific status codes.
  - Sets the status code and sends the JSON response manually.

- **New Code**:
  - Uses `ctx.BindJSON` to read JSON from the request body and handle errors automatically.
  - Simplifies error handling by using `ctx.Created` to send a 201 Created response or handle errors.
  - Reduces boilerplate code, making the handler more concise and easier to maintain.

## Benefits

- **Less code**: Error handling and response formatting stop being repeated in every handler.
- **Consistency**: One response shape and one error shape across the whole API.
- **Readability**: What a handler does is visible in the handler.