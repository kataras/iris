package iris_test

import (
	stdContext "context"
	"fmt"
	"strings"
	"testing"

	"github.com/kataras/iris/v14"
)

type warmupService struct{ inits int }

func (s *warmupService) Init(ctx stdContext.Context) error {
	if ctx == nil {
		return fmt.Errorf("nil context")
	}
	s.inits++
	return nil
}

type failingInitService struct{}

func (failingInitService) Init(stdContext.Context) error {
	return fmt.Errorf("warmup failed")
}

// Services implementing Initializer are initialized exactly once by Build,
// after dependency injection, even when registered twice or built twice.
func TestBuilderRunsInitializersOnce(t *testing.T) {
	svc := &warmupService{}

	b := iris.NewBuilder().
		NoPrefix().
		AllowOrigin("none").
		Compression(false).
		LogRequests(false).
		NoHealth().
		Services(svc, svc).
		RouterMiddlewares().
		Middlewares()

	b.Build()
	if svc.inits != 1 {
		t.Fatalf("expected exactly one Init call after Build, got %d", svc.inits)
	}

	b.Build() // building again must not re-initialize.
	if svc.inits != 1 {
		t.Fatalf("expected Init to run once across builds, got %d", svc.inits)
	}
}

// Constructor-registered services get their outputs initialized too.
func TestBuilderRunsInitializerOnConstructorOutput(t *testing.T) {
	var captured *warmupService
	ctor := func() *warmupService {
		captured = &warmupService{}
		return captured
	}

	iris.NewBuilder().
		NoPrefix().
		AllowOrigin("none").
		Compression(false).
		LogRequests(false).
		NoHealth().
		Services(ctor).
		RouterMiddlewares().
		Middlewares().
		Build()

	if captured == nil {
		t.Fatal("expected the constructor to run during registration")
	}
	if captured.inits != 1 {
		t.Fatalf("expected the constructor output to be initialized once, got %d", captured.inits)
	}
}

// A failing Init stops the application from building, like every other
// builder misconfiguration.
func TestBuilderInitializerFailurePanics(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected Build to panic on Init failure")
		}
		if !strings.Contains(fmt.Sprint(r), "warmup failed") {
			t.Fatalf("unexpected panic value: %v", r)
		}
	}()

	iris.NewBuilder().
		NoPrefix().
		AllowOrigin("none").
		Compression(false).
		LogRequests(false).
		NoHealth().
		Services(&failingInitService{}).
		RouterMiddlewares().
		Middlewares().
		Build()
}

// A function with no service to provide is a mistake, not a shutdown closer:
// it used to be swallowed and registered as one.
func TestBuilderServicesRejectsZeroOutputFunctions(t *testing.T) {
	for _, dep := range []any{func() {}, func() error { return nil }} {
		func() {
			defer func() {
				r := recover()
				if r == nil {
					t.Fatalf("expected Services(%T) to panic", dep)
				}
				msg := fmt.Sprint(r)
				if !strings.Contains(msg, "iris: builder: Services:") ||
					!strings.Contains(msg, "provides no service") ||
					!strings.Contains(msg, "Deferrables") {
					t.Fatalf("unexpected panic value for %T: %v", dep, r)
				}
			}()

			iris.NewBuilder().
				NoPrefix().
				AllowOrigin("none").
				Compression(false).
				LogRequests(false).
				NoHealth().
				Services(dep)
		}()
	}
}
