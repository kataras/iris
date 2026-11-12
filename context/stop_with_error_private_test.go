package context_test

import (
	"encoding/json"
	stdErrors "errors"
	"fmt"
	"strings"
	"testing"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/httptest"
	"github.com/kataras/iris/v14/rest/errors"
)

// StopWithError writes the error's text as the body unless the error is
// private. The private mark does not have to be the outermost value:
// errors.Private adds it to the error's own chain, and a private error can be
// wrapped again on its way up to the handler. Both used to have their whole
// text written, because only the outermost value was looked at.
//
// Whatever the shape, a private error answers what an outermost private error
// answers: the status and nothing of the error. ctx.GetErr still returns the
// error the handler passed.
func TestStopWithErrorPrivateMarkInTheChain(t *testing.T) {
	const (
		secretText = "SECRET tenant db-eu-7 refused the connection"
		publicText = "name is required"
	)

	var (
		errSecret = stdErrors.New(secretText)
		errPublic = stdErrors.New(publicText)

		outermost = iris.PrivateError(errSecret)
		marked    = errors.Private(errSecret)
		rewrapped = fmt.Errorf("load: %w", iris.PrivateError(errSecret))
	)

	for _, test := range []struct {
		name               string
		errorsHandler      bool
		statusCodeResponse bool
	}{
		{name: "WithoutErrorsHandler"},
		{name: "WithErrorsHandler", errorsHandler: true},
		// The router's own handler for a failure status nobody gave a body
		// to. It asks ctx.ErrPublic and writes the text of a public error, so
		// the mark has to be on the stored value too.
		{name: "WithStatusCodeResponse", statusCodeResponse: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			stored := make(map[string]error)

			app := iris.New()
			app.UseRouter(func(ctx iris.Context) {
				ctx.Next()
				stored[ctx.Path()] = ctx.GetErr()
			})

			if test.errorsHandler {
				app.UseRouter(errors.Handler)
			}

			if test.statusCodeResponse {
				app.EnableStatusCodeResponse()
			}

			passed := map[string]error{
				"/outermost": outermost,
				"/marked":    marked,
				"/rewrapped": rewrapped,
				"/public":    errPublic,
			}

			for path, err := range passed {
				app.Get(path, func(ctx iris.Context) {
					ctx.StopWithError(iris.StatusBadRequest, err)
				})
			}

			e := httptest.New(t, app)

			get := func(path string) string {
				t.Helper()
				return e.GET(path).Expect().Status(iris.StatusBadRequest).Body().Raw()
			}

			// What an outermost private error answers today. The other two
			// shapes must answer the same.
			expected := get("/outermost")
			if strings.Contains(expected, "SECRET") {
				t.Fatalf("/outermost: the private error text reached the client: %q", expected)
			}

			for _, path := range []string{"/marked", "/rewrapped"} {
				body := get(path)
				if strings.Contains(body, "SECRET") || strings.Contains(body, "db-eu-7") {
					t.Errorf("%s: the private error text reached the client: %q", path, body)
					continue
				}

				if !test.errorsHandler {
					if body != expected {
						t.Errorf("%s: expected what an outermost private error answers, %q, got %q", path, expected, body)
					}

					continue
				}

				// Under errors.Handler the body is the envelope, with an id
				// of its own each time.
				var doc struct {
					ErrorCode struct {
						CanonicalName string `json:"canonical_name"`
					} `json:"http_error_code"`
					Message string `json:"message"`
					Detail  string `json:"detail"`
				}

				if err := json.Unmarshal([]byte(body), &doc); err != nil {
					t.Errorf("%s: expected the rest/errors envelope, got %q: %v", path, body, err)
					continue
				}

				if doc.ErrorCode.CanonicalName != "INVALID_ARGUMENT" || doc.Message != "Bad Request" || doc.Detail != "" {
					t.Errorf("%s: expected INVALID_ARGUMENT with the status text and no detail, got %q", path, body)
				}
			}

			// An error that is not private still has its text written.
			if body := get("/public"); body != publicText {
				t.Errorf("/public: expected the error's text %q as the body, got %q", publicText, body)
			}

			// The server's own handlers read the error they were given. An
			// outermost private wrapper is taken off by ctx.GetErr, as it
			// always was; an error that carries the mark inside comes back
			// whole.
			for path, err := range map[string]error{
				"/outermost": errSecret,
				"/marked":    marked,
				"/rewrapped": rewrapped,
				"/public":    errPublic,
			} {
				if got := stored[path]; got != err {
					t.Errorf("%s: expected ctx.GetErr to return %q, got %v", path, err, got)
				}
			}
		})
	}
}
