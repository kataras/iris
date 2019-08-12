package context

import (
	"fmt"
	"net/http"
)

// Problem Details for HTTP APIs.
// Pass a Problem value to `context.Problem` to
// write an "application/problem+json" response.
//
// Read more at: https://github.com/kataras/iris/wiki/Routing-error-handlers
type Problem map[string]interface{}

// NewProblem retruns a new Problem.
// Head over to the `Problem` type godoc for more.
func NewProblem() Problem {
	p := make(Problem)
	return p
}

func (p Problem) keyExists(key string) bool {
	if p == nil {
		return false
	}

	_, found := p[key]
	return found
}

// DefaultProblemStatusCode is being sent to the client
// when Problem's status is not a valid one.
var DefaultProblemStatusCode = http.StatusBadRequest

func (p Problem) getStatus() (int, bool) {
	statusField, found := p["status"]
	if !found {
		return DefaultProblemStatusCode, false
	}

	status, ok := statusField.(int)
	if !ok {
		return DefaultProblemStatusCode, false
	}

	if !StatusCodeNotSuccessful(status) {
		return DefaultProblemStatusCode, false
	}

	return status, true
}

func isEmptyTypeURI(uri string) bool {
	return uri == "" || uri == "about:blank"
}

func (p Problem) getType() string {
	typeField, found := p["type"]
	if found {
		if typ, ok := typeField.(string); ok {
			if !isEmptyTypeURI(typ) {
				return typ
			}
		}
	}

	return ""
}

// Updates "type" field to absolute URI, recursively.
func (p Problem) updateTypeToAbsolute(ctx Context) {
	if p == nil {
		return
	}

	if uriRef := p.getType(); uriRef != "" {
		p.Type(ctx.AbsoluteURI(uriRef))
	}

	if cause, ok := p["cause"]; ok {
		if causeP, ok := cause.(Problem); ok {
			causeP.updateTypeToAbsolute(ctx)
		}
	}
}

// Key sets a custom key-value pair.
func (p Problem) Key(key string, value interface{}) Problem {
	p[key] = value
	return p
}

// Type URI SHOULD resolve to HTML [W3C.REC-html5-20141028]
// documentation that explains how to resolve the problem.
// Example: "https://example.net/validation-error"
//
// Empty URI or "about:blank", when used as a problem type,
// indicates that the problem has no additional semantics beyond that of the HTTP status code.
// When "about:blank" is used,
// the title is being automatically set the same as the recommended HTTP status phrase for that code
// (e.g., "Not Found" for 404, and so on) on `Status` call.
//
// Relative paths are also valid when writing this Problem to an Iris Context.
func (p Problem) Type(uri string) Problem {
	return p.Key("type", uri)
}

// Title sets the problem's title field.
// Example: "Your request parameters didn't validate."
// It is set to status Code text if missing,
// (e.g., "Not Found" for 404, and so on).
func (p Problem) Title(title string) Problem {
	return p.Key("title", title)
}

// Status sets HTTP error code for problem's status field.
// Example: 404
//
// It is required.
func (p Problem) Status(statusCode int) Problem {
	shouldOverrideTitle := !p.keyExists("title")

	if !shouldOverrideTitle {
		typ, found := p["type"]
		shouldOverrideTitle = !found || isEmptyTypeURI(typ.(string))
	}

	if shouldOverrideTitle {
		// Set title by code.
		p.Title(http.StatusText(statusCode))
	}
	return p.Key("status", statusCode)
}

// Detail sets the problem's detail field.
// Example: "Optional details about the error...".
func (p Problem) Detail(detail string) Problem {
	return p.Key("detail", detail)
}

// Cause sets the problem's cause field.
// Any chain of problems.
func (p Problem) Cause(cause Problem) Problem {
	if !cause.Validate() {
		return p
	}

	return p.Key("cause", cause)
}

// Validate reports whether this Problem value is a valid problem one.
func (p Problem) Validate() bool {
	// A nil problem is not a valid one.
	if p == nil {
		return false
	}

	return p.keyExists("type") &&
		p.keyExists("title") &&
		p.keyExists("status")
}

// Error method completes the go error.
// Returns the "[Status] Title" string form of this Problem.
// If Problem is not a valid one, it returns "invalid problem".
func (p Problem) Error() string {
	if !p.Validate() {
		return "invalid problem"
	}

	return fmt.Sprintf("[%d] %s", p["status"], p["title"])
}
