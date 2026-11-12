# RFC: HTTP Wire Errors

## Overview

This RFC proposes a standardized approach for handling and representing HTTP wire errors in a consistent and structured manner. The goal is to provide clear and detailed error responses that can be easily understood and processed by clients.

## Error Response Structure

Each error response will follow a standardized JSON structure with the following fields:

- `id`: A unique identifier for the error instance.
- `http_error_code`: An object containing:
  - `canonical_name`: A string representing the canonical name of the error.
  - `status`: An integer representing the HTTP status code.
- `message`: A short, human-readable summary of the error.
- `detail (optional)`: A human-readable explanation specific to this occurrence of the error.
- `data` (optional): Additional data related to the error.
- `validation` (optional): An array of validation errors, each containing:
  - `field`: The name of the field that caused the validation error.
  - `value`: The invalid value provided.
  - `reason`: A human-readable explanation of why the value is invalid.

### Example

```json
{
    "id": "unique-error-id",
    "http_error_code": {
        "canonical_name": "ERROR_NAME",
        "status": 400
    },
    "message": "short summary of the error",
    "detail": "detailed explanation of the error",
    "data": {
        "additional": "context-specific data"
    },
    "validation": [
        {
            "field": "field_name",
            "value": "invalid_value",
            "reason": "explanation of why the value is invalid"
        }
    ]
}
```

## Canonical Error Names

The following canonical error names are defined:

- `INVALID_ARGUMENT` (400): Indicates that the client provided an invalid argument.
- `FAILED_PRECONDITION` (400): Indicates that the operation was rejected because the system is not in a state required for the operation's execution.
- `OUT_OF_RANGE` (400): Indicates that an operation was attempted past the valid range.
- `UNSUPPORTED_MEDIA_TYPE` (415): Indicates that the request entity has a media type which the server or resource does not support.
- `DEADLINE_EXCEEDED` (504): Indicates that the deadline expired before the operation could complete.
- `NOT_FOUND` (404): Indicates that the requested resource could not be found.
- `METHOD_NOT_ALLOWED` (405): Indicates that the HTTP method used is not supported by the resource.
- `ALREADY_EXISTS` (409): Indicates that the resource that a client tried to create already exists.
- `ABORTED` (409): Indicates that the operation was aborted, typically due to a concurrency issue.
- `PERMISSION_DENIED` (403): Indicates that the caller does not have permission to execute the specified operation.
- `UNAUTHENTICATED` (401): Indicates that the request does not have valid authentication credentials for the operation.
- `RESOURCE_EXHAUSTED` (429): Indicates that some resource has been exhausted, perhaps a per-user quota, or perhaps the entire file system is out of space.
- `TOO_LARGE` (413): Indicates that the request entity is larger than limits defined by server.
- `UNIMPLEMENTED` (501): Indicates that the operation is not implemented or not supported/enabled in this service.
- `UNAVAILABLE` (503): Indicates that the service is currently unavailable.
- `BAD_GATEWAY` (502): Indicates that an upstream this service depends on failed.
- `INTERNAL` (500): Indicates an internal server error.
- `DATA_LOSS` (500): Indicates unrecoverable data loss or corruption.
- `UNKNOWN` (500): Indicates an unknown error.
- `PAYMENT_REQUIRED` (402): Indicates that payment is required to access the resource.
- `NOT_ACCEPTABLE` (406): Indicates that the resource is not capable of generating content acceptable according to the Accept headers sent in the request.
- `PROXY_AUTH_REQUIRED` (407): Indicates that the client must first authenticate itself with the proxy.
- `REQUEST_TIMEOUT` (408): Indicates that the server timed out waiting for the request.
- `CANCELLED` (408): Indicates that the operation was cancelled, typically by the caller. Uses 408 because the non-standard 499 is taken by `TOKEN_REQUIRED`.
- `GONE` (410): Indicates that the resource requested is no longer available and will not be available again.
- `LENGTH_REQUIRED` (411): Indicates that the request did not specify the length of its content, which is required by the requested resource.
- `PRECONDITION_FAILED` (412): Indicates that the server does not meet one of the preconditions that the requester put on the request.
- `REQUEST_URI_TOO_LONG` (414): Indicates that the URI provided was too long for the server to process.
- `REQUESTED_RANGE_NOT_SATISFIABLE` (416): Indicates that the server cannot supply the portion of the file requested.
- `EXPECTATION_FAILED` (417): Indicates that the server cannot meet the requirements of the Expect request-header field.
- `IM_A_TEAPOT` (418): Indicates that the server refuses to brew coffee because it is, permanently, a teapot.
- `MISDIRECTED_REQUEST` (421): Indicates that the request was directed at a server that is not able to produce a response.
- `UNPROCESSABLE_ENTITY` (422): Indicates that the server understands the content type of the request entity, but was unable to process the contained instructions.
- `LOCKED` (423): Indicates that the resource that is being accessed is locked.
- `FAILED_DEPENDENCY` (424): Indicates that the request failed due to failure of a previous request.
- `TOO_EARLY` (425): Indicates that the server is unwilling to risk processing a request that might be replayed.
- `UPGRADE_REQUIRED` (426): Indicates that the client should switch to a different protocol.
- `PRECONDITION_REQUIRED` (428): Indicates that the origin server requires the request to be conditional.
- `REQUEST_HEADER_FIELDS_TOO_LARGE` (431): Indicates that the server is unwilling to process the request because its header fields are too large.
- `UNAVAILABLE_FOR_LEGAL_REASONS` (451): Indicates that the resource is unavailable for legal reasons.
- `HTTP_VERSION_NOT_SUPPORTED` (505): Indicates that the server does not support the HTTP protocol version used in the request.
- `VARIANT_ALSO_NEGOTIATES` (506): Indicates that the server has an internal configuration error.
- `INSUFFICIENT_STORAGE` (507): Indicates that the server is unable to store the representation needed to complete the request.
- `LOOP_DETECTED` (508): Indicates that the server detected an infinite loop while processing a request.
- `NOT_EXTENDED` (510): Indicates that further extensions to the request are required for the server to fulfill it.
- `NETWORK_AUTHENTICATION_REQUIRED` (511): Indicates that the client needs to authenticate to gain network access.
- `PAGE_EXPIRED` (419): Indicates that the page has expired.
- `BLOCKED_BY_WINDOWS_PARENTAL_CONTROLS` (450): Indicates that the request was blocked by Windows Parental Controls.
- `INVALID_TOKEN` (498): Indicates that the token provided is invalid.
- `TOKEN_REQUIRED` (499): Indicates that a token is required to access the resource.
- `BANDWIDTH_LIMIT_EXCEEDED` (509): Indicates that the bandwidth limit has been exceeded.
- `INVALID_SSL_CERTIFICATE` (526): Indicates that the SSL certificate is invalid.
- `SITE_OVERLOADED` (529): Indicates that the site is overloaded.
- `SITE_FROZEN` (530): Indicates that the site is frozen.
- `NETWORK_READ_TIMEOUT` (598): Indicates that there was a network read timeout error.
- `UNEXPECTED_ERROR` (500): The fallback the server sends when it fails to send an error (an unregistered error code name, or a failure while marshaling the error response).

## Error Handling

### Invalid Argument

When the client provides an invalid argument, the server should respond with an `INVALID_ARGUMENT` error.

#### Example

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

### Failed Precondition

When the operation is rejected due to a failed precondition, the server should respond with a `FAILED_PRECONDITION` error.

#### Example

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

### Unauthenticated

When the request lacks valid authentication credentials, the server should respond with an `UNAUTHENTICATED` error.

#### Example

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

### Unauthorized Access

Due to invalid [JWK](https://datatracker.ietf.org/doc/html/rfc7517).

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

### Permission Denied

When the caller does not have permission to execute the specified operation, the server should respond with a `PERMISSION_DENIED` error.

#### Example

```json
{
    "id": "d4e5f6a7-b8c9-0123-def4-567890123456",
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

### Not Found

When the requested resource is not found, the server should respond with a `NOT_FOUND` error.

#### Example

```json
{
    "id": "e5f6a7b8-c9d0-1234-ef56-789012345678",
    "http_error_code": {
        "canonical_name": "NOT_FOUND",
        "status": 404
    },
    "message": "transaction not found",
    "detail": "the transaction with id 'tx1234567890' was not found"
}
```

#### NotFoundID

The "this row is not there" answer is one call. `NotFoundID` renders the `NOT_FOUND` envelope with the detail `"<resource> <id> not found"`.

```go
func NotFoundID(ctx *context.Context, resource string, id any) bool
```

```go
func getPlan(ctx iris.Context) {
    plan, err := svc.GetPlan(ctx, id)
    if errors.HandleError(ctx, err) {
        return
    }

    if plan == nil {
        errors.NotFoundID(ctx, "plan", id)
        return
    }

    ctx.OK(plan, nil)
}
```

The result follows the `Context.OK` convention, `true` meaning the handler succeeded, so it is always `false` here. `HandleError` in the same package reports the opposite, `true` when it wrote the response, which is why the example tests one and ignores the other.

### Validation Error

When the request contains invalid fields, the server should respond with an `INVALID_ARGUMENT` error and include validation details.

#### Example

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
            "value": "jo",
            "reason": "name must be at least 3 characters long"
        },
        {
            "field": "email",
            "value": "invalid@comcom",
            "reason": "email must be a valid email address"
        }
    ]
}
```

#### Internal Server Error

When an unexpected error occurs while processing the request, the server should respond with an `INTERNAL` error.

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

### Complex Error with Nested Data

When an error involves multiple nested errors, the server should provide detailed context for each part of the error message.

#### Example

```json
{
    "id": "9f8b7c6d-4e5f-4a2b-8c3d-1e2f3a4b5c6d",
    "http_error_code": {
        "canonical_name": "FAILED_PRECONDITION",
        "status": 400
    },
    "message": "transaction failed: insufficient funds: daily withdrawal limit exceeded",
    "data": [
        {
            "message": "transaction failed",
            "data": {
                "transaction_id": "tx1234567890",
                "amount": 150.00,
                "currency": "USD"
            }
        },
        {
            "message": "insufficient funds",
            "data": {
                "account_id": "acc987654321",
                "current_balance": 50.00,
                "required_balance": 150.00
            }
        },
        {
            "message": "daily withdrawal limit exceeded",
            "data": {
                "account_id": "acc987654321",
                "daily_limit": 1000.00,
                "amount_attempted": 1500.00
            }
        }
    ]
}
```

In this example, the error message is composed of three parts:
1. `transaction failed`
2. `insufficient funds`
3. `daily withdrawal limit exceeded`

Each part of the error message has its own section of data, providing detailed context for each part of the error. This structure allows clients to understand the specific context and data related to each part of the error message, making it easier to debug and resolve issues.

## Mapping Helpers

Beside `MapErrors`, `MapErrorTypes`, `MapErrorFunc` and `MapErrorFuncCtx`, two helpers cover the mappings which need the error's own fields or only its text.

### MapErrorAs

`MapErrorAs` looks for a typed error anywhere in the error tree, as the standard `errors.AsType` does, and hands it to a function which builds the `*Error` to send. A `nil` result means "not mine", so the next mapping gets a turn. It is a generic method of `*Options` and of `ErrorCodeMap`, so it chains like the other mapping methods, and the error type is inferred from the function.

```go
func (opts *Options) MapErrorAs[E error](code ErrorCodeName, render func(E) *Error) *Options
func (m ErrorCodeMap) MapErrorAs[E error](code ErrorCodeName, render func(E) *Error) ErrorCodeMap
```

```go
opts := errors.NewOptions().
    MapErrors(errors.NotFound, sql.ErrNoRows).
    MapErrorAs(errors.ResourceExhausted, func(e *QuotaError) *errors.Error {
        return &errors.Error{
            Message: "quota exceeded",
            Detail:  fmt.Sprintf("the limit is %d", e.Limit),
        }
    })
```

A `nil` function classifies without transforming, the same as `MapErrorTypes`. The type then has to be named, because there is nothing left to infer it from:

```go
opts.MapErrorAs[*QuotaError](errors.ResourceExhausted, nil)
```

### MapErrorContains

`MapErrorContains` answers an error whose text contains a substring, with the given message replacing the original text. The match is case-sensitive.

```go
func (opts *Options) MapErrorContains(e ErrorCodeName, substr, message string) *Options
func (m ErrorCodeMap) MapErrorContains(e ErrorCodeName, substr, message string) ErrorCodeMap
```

```go
opts.MapErrorContains(errors.InvalidArgument, "violates check constraint", "invalid value")
```

This exists for the SQL and vendor errors which arrive as text only. Prefer a typed mapping wherever one is possible, because a driver is free to reword its text in the next release.

## Problem Details

The [Problem Details RFC](https://tools.ietf.org/html/rfc7807) is a standardized format for representing errors in HTTP APIs. It defines a structured way to provide detailed information about errors in a machine-readable format, making it easier for clients to understand and handle errors effectively.

### Example

```json
{
    "status": 400,
    "title": "Insufficient Funds",
    "detail": "The account '987654321' has insufficient funds for the transaction.",
    "type": "https://example.com/probs/insufficient-funds",
    "instance": "/accounts/12345/transactions/67890",
    "id": "b2c3d4e5-f6a7-8901-bcde-f23456789012",
    "current_balance": 50.00,
    "required_balance": 100.00
}
```

### Differences Between HTTP Wire Errors and Problem Details

While both HTTP Wire Errors and Problem Details provide a structured way to represent errors in HTTP APIs, there are some key differences:

1. **Flexibility**: HTTP Wire Errors offer a more flexible structure that can include additional fields such as `id`, `http_error_code`, `message`, `data`, and `validation`. This allows for more detailed and context-specific error information.

2. **Canonical Error Names**: HTTP Wire Errors define a set of canonical error names that standardize common error scenarios, making it easier to categorize and handle errors consistently across different parts of an application.

3. **Nested Data**: HTTP Wire Errors support nested data structures, so each part of a complex error message carries its own context. This is particularly useful for errors that involve multiple related issues.

4. **Validation Errors**: HTTP Wire Errors include a dedicated `validation` field to provide detailed information about validation errors, including the specific fields and reasons for the validation failure.

### What Problem Details Were Missing

Problem Details (RFC 7807) provide a standardized format for representing errors, but they lack some features that are addressed by HTTP Wire Errors:

- **Canonical Error Names**: Problem Details do not define a set of canonical error names, which can lead to inconsistencies in error categorization.
- **Nested Data**: Problem Details do not natively support nested data structures, making it harder to provide detailed context for complex errors.
- **Validation Errors**: Problem Details do not have a dedicated field for validation errors, which can make it harder to provide detailed information about validation issues.

### How HTTP Wire Errors Provide a Solution

HTTP Wire Errors address these limitations by:

- Defining a set of canonical error names for consistent error categorization.
- Supporting nested data structures to provide detailed context for complex errors.
- Including a dedicated `validation` field for detailed validation error information.

## Author

[Gerasimos Maropoulos](https://github.com/kataras) is the author of the [Iris web framework](https://www.iris-go.com) for Go, which he has developed in the open since 2016, along with articles and tutorials on building web services in Go.

He wrote this RFC because error responses in HTTP APIs are usually invented per project, which leaves clients guessing at the shape of a failure. A fixed structure means a client can read an error the same way every time, and a developer reading a log can see what actually went wrong.

## Summary

This RFC defines one structure for HTTP wire errors: a stable set of fields, nested data for complex failures, and a dedicated `validation` field. A client that knows the structure can handle any error from any endpoint without special cases.
