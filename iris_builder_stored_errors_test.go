package iris_test

import (
	"encoding/json"
	stdErrors "errors"
	nethttptest "net/http/httptest"
	"strings"
	"testing"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/auth/oauth2"
	"github.com/kataras/iris/v14/httptest"
	"github.com/kataras/iris/v14/middleware/jwt"
	"github.com/kataras/iris/v14/middleware/rewrite"
	"github.com/kataras/iris/v14/middleware/sessions"
	"github.com/kataras/iris/v14/middleware/view"
	"github.com/kataras/iris/v14/rest/errors"
)

type storeUserSession struct{ Username string }

// errSessionStoreDown is what a session database returns when its store is
// unreachable: the driver's own words, with a host and a port in them.
var errSessionStoreDown = stdErrors.New("dial tcp 10.0.0.7:6379: connect: connection refused")

// failingSessionDatabase is a server-side session database whose store is
// down, on the read or on the write.
type failingSessionDatabase struct {
	opts       sessions.Options
	failLoad   bool
	failCommit bool
}

func (db *failingSessionDatabase) Get(ctx iris.Context, name string) (*sessions.Session[storeUserSession], error) {
	return sessions.Load(ctx, db, name)
}

func (db *failingSessionDatabase) Load(iris.Context, string, string) (*sessions.Session[storeUserSession], error) {
	if db.failLoad {
		return nil, errSessionStoreDown
	}

	session := sessions.NewSession[storeUserSession](db.opts)
	session.ID = "7c9e6679-7425-40de-944b-e07fc1f90ae7"
	session.IsNew = true
	return session, nil
}

func (db *failingSessionDatabase) Commit(iris.Context, *sessions.Session[storeUserSession]) error {
	if db.failCommit {
		return errSessionStoreDown
	}

	return nil
}

// sessions.Handler answers a store failure with a 500 and stores the error on
// the Context. Under the builder that 500 is rendered, and the driver's text
// (a host, a port, sometimes a user and a database name) must not be what the
// client reads.
func TestBuilderSessionsStoreFailureTextStaysOnTheServer(t *testing.T) {
	opts := sessions.NewOptions(
		[]byte("D*G-KaPdSgUkXp2s5v8y/B?E(H+MbQeThWmYq3t6w9z$C&F)J@NcRfUjXn2r4u7x"),
		[]byte("VkYp3s6v9y$B&E)H@McQfTjWmZq4t7w!")).Apply()

	for name, db := range map[string]*failingSessionDatabase{
		"Get":    {opts: opts, failLoad: true},
		"Commit": {opts: opts, failCommit: true},
	} {
		t.Run(name, func(t *testing.T) {
			var (
				stored   error
				isPublic bool
			)

			app := iris.NewBuilder().
				NoPrefix().
				AllowOrigin("none").
				Compression(false).
				LogRequests(false).
				NoHealth().
				Services().
				RouterMiddlewares(func(ctx iris.Context) {
					ctx.Next()
					isPublic, stored = ctx.ErrPublic()
				}).
				Middlewares().
				API("/", func(r iris.Router) {
					r.Get("/me", sessions.Handler(db, "session"), func(ctx iris.Context) {
						ctx.WriteString("signed in")
					})
				}).
				Build()

			body := httptest.New(t, app).GET("/me").Expect().
				Status(httptest.StatusInternalServerError).Body().Raw()

			// The error is still on the Context for the server's own handlers,
			// and it is marked private where it is stored: the text stays off
			// the wire under any error handler, not only this one.
			if !stdErrors.Is(stored, errSessionStoreDown) {
				t.Fatalf("expected the store failure on the Context, got: %v", stored)
			}

			if isPublic {
				t.Fatal("the session store failure is stored as an error a client may read")
			}

			if strings.Contains(body, "10.0.0.7") || strings.Contains(body, "connection refused") {
				t.Fatalf("the store failure text reached the client: %s", body)
			}

			if strings.Contains(body, "signed in") {
				t.Fatalf("the route's body went out with the failure: %s", body)
			}

			bareEnvelope(t, []byte(body), httptest.StatusInternalServerError, "INTERNAL")
		})
	}
}

// The jwt middleware stops a bad token with a private error and a 401. That is
// the client's fault, not the server's, and the body has to say so: the status
// text, where a private error used to read "server error".
func TestBuilderJWTBadTokenReadsUnauthorized(t *testing.T) {
	verifier := jwt.NewVerifier(jwt.HS256, []byte("sercrethatmaycontainch@r$"))

	app := iris.NewBuilder().
		NoPrefix().
		AllowOrigin("none").
		Compression(false).
		LogRequests(false).
		NoHealth().
		Services().
		RouterMiddlewares().
		Middlewares().
		API("/", func(r iris.Router) {
			r.Get("/protected", verifier.Verify(func() any { return new(storeUserSession) }), func(ctx iris.Context) {
				ctx.WriteString("in")
			})
		}).
		Build()

	e := httptest.New(t, app)

	for name, authorization := range map[string]string{
		"no token":      "",
		"garbage token": "Bearer not.a.token",
	} {
		req := e.GET("/protected")
		if authorization != "" {
			req = req.WithHeader("Authorization", authorization)
		}

		body := req.Expect().Status(httptest.StatusUnauthorized).Body().Raw()

		doc := bareEnvelope(t, []byte(body), httptest.StatusUnauthorized, "UNAUTHENTICATED")
		if doc.Message != "Unauthorized" || doc.Detail != "" {
			t.Fatalf("%s: expected the message %q and no detail, got: %s", name, "Unauthorized", body)
		}
	}
}

// The same under Errors(errors.NewOptions()), whose default mappings know the
// errors of kataras/jwt and answer them as Unauthenticated. The middleware
// stores its error as private, and a private error is not handed to a
// mapping: the client reads "Unauthorized", not why the token was refused.
func TestBuilderJWTBadTokenReadsUnauthorizedUnderErrorsOptions(t *testing.T) {
	verifier := jwt.NewVerifier(jwt.HS256, []byte("sercrethatmaycontainch@r$"))

	var collected []string
	opts := errors.NewOptions().SetCollector(func(ctx iris.Context, err *errors.Error) {
		collected = append(collected, err.Message)
	})

	app := iris.NewBuilder().
		NoPrefix().
		AllowOrigin("none").
		Compression(false).
		LogRequests(false).
		NoHealth().
		Errors(opts).
		Services().
		RouterMiddlewares().
		Middlewares().
		API("/", func(r iris.Router) {
			r.Get("/protected", verifier.Verify(func() any { return new(storeUserSession) }), func(ctx iris.Context) {
				ctx.WriteString("in")
			})
		}).
		Build()

	e := httptest.New(t, app)

	body := e.GET("/protected").Expect().Status(httptest.StatusUnauthorized).Body().Raw()

	doc := bareEnvelope(t, []byte(body), httptest.StatusUnauthorized, "UNAUTHENTICATED")
	if doc.Message != "Unauthorized" || doc.Detail != "" {
		t.Fatalf("expected the message %q and no detail, got: %s", "Unauthorized", body)
	}

	// The reason is for the server: the collector still receives it.
	if len(collected) != 1 || collected[0] == "" || collected[0] == "Unauthorized" {
		t.Fatalf("expected the collector to receive why the token was refused, got %q", collected)
	}
}

// The framework's own middleware store an internal failure as a private error:
// why a token was refused, what a template's data source said, which rewrite
// target did not parse. Whatever error handler the application runs, that text
// is not a message for the client. Checked at the source, on an application
// built with iris.New(), with errors.Handler installed to read the answer.
func TestStoredInternalErrorsArePrivate(t *testing.T) {
	rewriter, err := rewrite.New(rewrite.Options{
		// Code 0: rewrite the request in place, no redirect.
		RedirectMatch: []string{"0 /seo/(.*) /$1"},
	})
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name     string
		register func(app *iris.Application)
		target   string
		status   int
		message  string // what the client reads.
		internal string // a piece of the stored error's text.
	}{
		{
			name: "oauth2.AllowScopes without a token",
			register: func(app *iris.Application) {
				app.Get("/admin", oauth2.AllowScopes("users:read"), func(ctx iris.Context) {
					ctx.WriteString("in")
				})
			},
			target:   "/admin",
			status:   httptest.StatusUnauthorized,
			message:  "Unauthorized",
			internal: "token is empty",
		},
		{
			name: "view.WithTemplate whose data function fails",
			register: func(app *iris.Application) {
				app.Get("/report", view.WithTemplate(&view.Template{
					Name: "report.html",
					Code: httptest.StatusBadGateway,
					DataFunc: func(iris.Context) (any, error) {
						return nil, stdErrors.New("reports upstream 10.0.0.9:8443 timed out")
					},
				}))
			},
			target:   "/report",
			status:   httptest.StatusBadGateway,
			message:  "server error",
			internal: "10.0.0.9",
		},
		{
			name: "rewrite to a target that does not parse",
			register: func(app *iris.Application) {
				app.UseRouter(rewriter.Handler)
			},
			// The decoded path is /seo/%zz, so the target is /%zz: a bad escape.
			target:   "/seo/%25zz",
			status:   iris.StatusMisdirectedRequest,
			message:  "Misdirected Request",
			internal: "invalid URL escape",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var (
				stored   error
				isPublic bool
			)

			app := iris.New()
			app.UseRouter(func(ctx iris.Context) {
				ctx.Next()
				isPublic, stored = ctx.ErrPublic()
			})
			app.UseRouter(errors.Handler)
			tt.register(app)

			app.Logger().SetLevel("disable")
			if err := app.Build(); err != nil {
				t.Fatal(err)
			}

			rec := nethttptest.NewRecorder()
			app.ServeHTTP(rec, nethttptest.NewRequest("GET", "http://example.com"+tt.target, nil))

			body := rec.Body.String()

			if rec.Code != tt.status {
				t.Fatalf("expected status %d, got %d: %s", tt.status, rec.Code, body)
			}

			if stored == nil || !strings.Contains(stored.Error(), tt.internal) {
				t.Fatalf("expected the failure on the Context for the server's own handlers, got: %v", stored)
			}

			if isPublic {
				t.Fatalf("the failure is stored as an error a client may read: %v", stored)
			}

			if strings.Contains(body, tt.internal) {
				t.Fatalf("the failure text reached the client: %s", body)
			}

			var doc envelope
			if err := json.Unmarshal([]byte(body), &doc); err != nil {
				t.Fatalf("expected the rest/errors envelope, got %q: %v", body, err)
			}

			if doc.Message != tt.message {
				t.Fatalf("expected the message %q, got: %s", tt.message, body)
			}
		})
	}
}
