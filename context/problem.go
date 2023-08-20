package context

import (
	"encoding/xml"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Problem Details for HTTP APIs.
// Pass a Problem value to `context.Problem` to
// write an "application/problem+json" response.
//
// Read more at: https://github.com/kataras/iris/blob/main/_examples/routing/http-errors.
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

func (p Problem) getURI(key string) string {
	f, found := p[key]
	if found {
		if typ, ok := f.(string); ok {
			if !isEmptyTypeURI(typ) {
				return typ
			}
		}
	}

	return ""
}

// Updates "type" field to absolute URI, recursively.
func (p Problem) updateURIsToAbs(ctx *Context) {
	if p == nil {
		return
	}

	if uriRef := p.getURI("type"); uriRef != "" && !strings.HasPrefix(uriRef, "http") {
		p.Type(ctx.AbsoluteURI(uriRef))
	}

	if uriRef := p.getURI("instance"); uriRef != "" {
		p.Instance(ctx.AbsoluteURI(uriRef))
	}

	if cause, ok := p["cause"]; ok {
		if causeP, ok := cause.(Problem); ok {
			causeP.updateURIsToAbs(ctx)
		}
	}
}

const (
	problemTempKeyPrefix = "@temp_"
)

// TempKey sets a temporary key-value pair, which is being removed
// on the its first get.
func (p Problem) TempKey(key string, value interface{}) Problem {
	return p.Key(problemTempKeyPrefix+key, value)
}

// GetTempKey returns the temp value based on "key" and removes it.
func (p Problem) GetTempKey(key string) interface{} {
	key = problemTempKeyPrefix + key
	v, ok := p[key]
	if ok {
		delete(p, key)
		return v
	}

	return nil
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
// When "about:blank" is used and "title" was not set-ed,
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

	// if !shouldOverrideTitle {
	// 	typ, found := p["type"]
	// 	shouldOverrideTitle = !found || isEmptyTypeURI(typ.(string))
	// }

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

// DetailErr calls `Detail(err.Error())`.
func (p Problem) DetailErr(err error) Problem {
	if err == nil {
		return p
	}

	return p.Key("detail", err.Error())
}

// Instance sets the problem's instance field.
// A URI reference that identifies the specific
// occurrence of the problem.  It may or may not yield further
// information if dereferenced.
func (p Problem) Instance(instanceURI string) Problem {
	return p.Key("instance", instanceURI)
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

// MarshalXML makes this Problem XML-compatible content to render.
func (p Problem) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	if len(p) == 0 {
		return nil
	}

	err := e.EncodeToken(start)
	if err != nil {
		return err
	}

	// toTitle := cases.Title(language.English)
	// toTitle.String(k)

	for k, v := range p {
		// convert keys like "type" to "Type", "productName" to "ProductName" and e.t.c. when xml.
		err = e.Encode(xmlMapEntry{XMLName: xml.Name{Local: strings.Title(k)}, Value: v})
		if err != nil {
			return err
		}
	}

	return e.EncodeToken(start.End())
}

// DefaultProblemOptions the default options for `Context.Problem` method.
var DefaultProblemOptions = ProblemOptions{
	JSON: JSON{Indent: "  "},
	XML:  XML{Indent: "  "},
}

// ProblemOptions the optional settings when server replies with a Problem.
// See `Context.Problem` method and `Problem` type for more details.
type ProblemOptions struct {
	// JSON are the optional JSON renderer options.
	JSON JSON

	// RenderXML set to true if want to render as XML doc.
	// See `XML` option field too.
	RenderXML bool
	// XML are the optional XML renderer options.
	// Affect only when `RenderXML` field is set to true.
	XML XML

	// RetryAfter sets the Retry-After response header.
	// https://tools.ietf.org/html/rfc7231#section-7.1.3
	// The value can be one of those:
	// time.Time
	// time.Duration for seconds
	// int64, int, float64 for seconds
	// string for duration string or for datetime string.
	//
	// Examples:
	// time.Now().Add(5 * time.Minute),
	// 300 * time.Second,
	// "5m",
	// 300
	RetryAfter interface{}
	// A function that, if specified, can dynamically set
	// retry-after based on the request. Useful for ProblemOptions reusability.
	// Should return time.Time, time.Duration, int64, int, float64 or string.
	//
	// Overrides the RetryAfter field.
	RetryAfterFunc func(*Context) interface{}
}

func parseDurationToSeconds(dur time.Duration) int64 {
	return int64(math.Round(dur.Seconds()))
}

func (o *ProblemOptions) parseRetryAfter(value interface{}, timeLayout string) string {
	// https://tools.ietf.org/html/rfc7231#section-7.1.3
	// Retry-After = HTTP-date / delay-seconds
	switch v := value.(type) {
	case int64:
		return strconv.FormatInt(v, 10)
	case int:
		return o.parseRetryAfter(int64(v), timeLayout)
	case float64:
		return o.parseRetryAfter(int64(math.Round(v)), timeLayout)
	case time.Time:
		return v.Format(timeLayout)
	case time.Duration:
		return o.parseRetryAfter(parseDurationToSeconds(v), timeLayout)
	case string:
		dur, err := time.ParseDuration(v)
		if err != nil {
			t, err := time.Parse(timeLayout, v)
			if err != nil {
				return ""
			}

			return o.parseRetryAfter(t, timeLayout)
		}

		return o.parseRetryAfter(parseDurationToSeconds(dur), timeLayout)
	}

	return ""
}

// Apply accepts a Context and applies specific response-time options.
func (o *ProblemOptions) Apply(ctx *Context) {
	retryAfterHeaderValue := ""
	timeLayout := ctx.Application().ConfigurationReadOnly().GetTimeFormat()

	if o.RetryAfterFunc != nil {
		retryAfterHeaderValue = o.parseRetryAfter(o.RetryAfterFunc(ctx), timeLayout)
	} else if o.RetryAfter != nil {
		retryAfterHeaderValue = o.parseRetryAfter(o.RetryAfter, timeLayout)
	}

	if retryAfterHeaderValue != "" {
		ctx.Header("Retry-After", retryAfterHeaderValue)
	}
}
