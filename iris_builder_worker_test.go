package iris_test

import (
	"bytes"
	stdContext "context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/kataras/iris/v14"
)

const workerTestTimeout = 5 * time.Second

// testWorker records what the builder did to it: how many times Stop was
// called, which error Run saw from its context and, for the ordering test,
// whether the initializer had already run.
type testWorker struct {
	runErr error
	// runLinger delays the return of Run after its context is done, so a test
	// can tell whether Shutdown waits for it.
	runLinger time.Duration
	// blockRun keeps Run from returning at all until the channel is closed.
	blockRun chan struct{}

	initialized bool

	mu         sync.Mutex
	runErrIn   error
	stopCtx    stdContext.Context
	stops      int
	closes     int
	returnedAt time.Time

	started chan struct{}
	ran     chan struct{}
}

func newTestWorker(runErr error) *testWorker {
	return &testWorker{
		runErr:  runErr,
		started: make(chan struct{}, 1),
		ran:     make(chan struct{}),
	}
}

func (w *testWorker) Init(stdContext.Context) error {
	w.initialized = true
	return nil
}

func (w *testWorker) Run(ctx stdContext.Context) error {
	w.started <- struct{}{}

	if w.runErr != nil {
		close(w.ran)
		return w.runErr
	}

	<-ctx.Done()

	w.mu.Lock()
	w.runErrIn = ctx.Err()
	w.mu.Unlock()

	if w.blockRun != nil {
		<-w.blockRun
	}

	if w.runLinger > 0 {
		time.Sleep(w.runLinger)
	}

	w.mu.Lock()
	w.returnedAt = time.Now()
	w.mu.Unlock()

	close(w.ran)
	return nil
}

func (w *testWorker) Stop(ctx stdContext.Context) error {
	w.mu.Lock()
	w.stops++
	w.stopCtx = ctx
	w.mu.Unlock()

	return nil
}

func (w *testWorker) Close() error {
	w.mu.Lock()
	w.closes++
	w.mu.Unlock()
	return nil
}

func (w *testWorker) counts() (stops, closes int) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.stops, w.closes
}

func (w *testWorker) contextError() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.runErrIn
}

func (w *testWorker) stopDeadline() (time.Time, bool) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.stopCtx == nil {
		return time.Time{}, false
	}

	return w.stopCtx.Deadline()
}

func (w *testWorker) runReturnedAt() time.Time {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.returnedAt
}

func workerBuilder(configurators ...iris.Configurator) iris.ServiceBuilder {
	return iris.NewBuilder(configurators...).
		NoPrefix().
		AllowOrigin("none").
		Compression(false).
		LogRequests(false).
		NoHealth()
}

func buildWithWorker(t *testing.T, configurators []iris.Configurator, deps ...any) *iris.Application {
	t.Helper()

	return workerBuilder(configurators...).
		Services(deps...).
		RouterMiddlewares().
		Middlewares().
		Build()
}

func waitFor(t *testing.T, ch <-chan struct{}, what string) {
	t.Helper()

	select {
	case <-ch:
	case <-time.After(workerTestTimeout):
		t.Fatalf("timeout waiting for %s", what)
	}
}

// Run gets a context which is cancelled at shutdown, and Stop runs once even
// when Shutdown is called twice.
func TestBuilderWorkerRunAndStop(t *testing.T) {
	w := newTestWorker(nil)

	app := buildWithWorker(t, nil, w)
	waitFor(t, w.started, "Run to start")

	if err := app.Shutdown(stdContext.Background()); err != nil {
		t.Fatal(err)
	}

	waitFor(t, w.ran, "Run to return")

	if err := w.contextError(); err != stdContext.Canceled {
		t.Fatalf("expected Run's context to be cancelled, got: %v", err)
	}

	// A value which is both a Worker and a Closer gets both, Stop then Close.
	if stops, closes := w.counts(); stops != 1 || closes != 1 {
		t.Fatalf("expected one Stop and one Close, got: %d, %d", stops, closes)
	}

	if err := app.Shutdown(stdContext.Background()); err != nil {
		t.Fatal(err)
	}

	if stops, _ := w.counts(); stops != 1 {
		t.Fatalf("expected Stop to run once across shutdowns, got: %d", stops)
	}
}

// Shutdown does not return while Run is still running: a worker which is still
// draining must not have the process exit from under it.
func TestBuilderWorkerShutdownWaitsForRun(t *testing.T) {
	w := newTestWorker(nil)
	w.runLinger = 300 * time.Millisecond

	app := buildWithWorker(t, nil, w)
	waitFor(t, w.started, "Run to start")

	if err := app.Shutdown(stdContext.Background()); err != nil {
		t.Fatal(err)
	}

	returnedAt := w.runReturnedAt()
	if returnedAt.IsZero() {
		t.Fatal("expected Shutdown to wait for Run to return")
	}

	select {
	case <-w.ran:
	default:
		t.Fatal("expected Run to have returned by the time Shutdown returned")
	}
}

// A Run which ignores its cancellation is bounded by the stop context: the
// shutdown reports the failure instead of hanging forever.
func TestBuilderWorkerShutdownReportsRunThatDoesNotReturn(t *testing.T) {
	w := newTestWorker(nil)
	w.blockRun = make(chan struct{})
	t.Cleanup(func() { close(w.blockRun) })

	app := buildWithWorker(t, nil, w)
	waitFor(t, w.started, "Run to start")

	ctx, cancel := stdContext.WithTimeout(stdContext.Background(), 100*time.Millisecond)
	defer cancel()

	err := app.Shutdown(ctx)
	if err == nil {
		t.Fatal("expected a shutdown error for a Run which did not return")
	}

	if !strings.Contains(err.Error(), "iris_test.testWorker") ||
		!strings.Contains(err.Error(), "Run did not return") {
		t.Fatalf("unexpected shutdown error: %v", err)
	}
}

// A shutdown before the application is ever built cancels the worker and keeps
// the later build from starting it: nothing is left running with a context
// which will never be cancelled.
func TestBuilderWorkerStoppedBeforeStart(t *testing.T) {
	w := newTestWorker(nil)

	var app *iris.Application
	b := workerBuilder(iris.ConfiguratorFunc(func(a *iris.Application) { app = a })).
		Services(w).
		RouterMiddlewares().
		Middlewares()

	if err := app.Shutdown(stdContext.Background()); err != nil {
		t.Fatal(err)
	}

	b.Build() // must not start a worker which was already stopped.

	select {
	case <-w.started:
		t.Fatal("expected a stopped worker not to be started by a later Build")
	case <-time.After(200 * time.Millisecond):
	}

	if stops, closes := w.counts(); stops != 1 || closes != 1 {
		t.Fatalf("expected one Stop and one Close, got: %d, %d", stops, closes)
	}
}

// A Worker which is also an Initializer runs after its own Init: the builder
// starts the workers once every initializer has returned.
func TestBuilderWorkerStartsAfterInitializer(t *testing.T) {
	w := newTestWorker(nil)

	app := buildWithWorker(t, nil, w)
	t.Cleanup(func() { _ = app.Shutdown(stdContext.Background()) })

	waitFor(t, w.started, "Run to start")

	if !w.initialized {
		t.Fatal("expected Init to have run before Run")
	}
}

// A Worker is registered as a dependency too, so a controller can take it.
func TestBuilderWorkerIsInjectable(t *testing.T) {
	w := newTestWorker(nil)

	var injected *testWorker
	ctor := func(dep *testWorker) string {
		injected = dep
		return "ok"
	}

	app := buildWithWorker(t, nil, w, ctor)
	t.Cleanup(func() { _ = app.Shutdown(stdContext.Background()) })

	if injected != w {
		t.Fatalf("expected the worker to be injectable as a dependency, got: %v", injected)
	}
}

// Stop gets the Application.Shutdown context when that one carries a deadline,
// and a ten second budget when it does not.
func TestBuilderWorkerStopContextBudget(t *testing.T) {
	t.Run("shutdown deadline is passed through", func(t *testing.T) {
		w := newTestWorker(nil)
		app := buildWithWorker(t, nil, w)
		waitFor(t, w.started, "Run to start")

		deadline := time.Now().Add(2 * time.Second)
		ctx, cancel := stdContext.WithDeadline(stdContext.Background(), deadline)
		defer cancel()

		if err := app.Shutdown(ctx); err != nil {
			t.Fatal(err)
		}

		got, ok := w.stopDeadline()
		if !ok {
			t.Fatal("expected Stop to get a bounded context")
		}

		if !got.Equal(deadline) {
			t.Fatalf("expected the shutdown deadline %v but got: %v", deadline, got)
		}
	})

	t.Run("a context without a deadline gets the ten second budget", func(t *testing.T) {
		w := newTestWorker(nil)
		app := buildWithWorker(t, nil, w)
		waitFor(t, w.started, "Run to start")

		before := time.Now()
		if err := app.Shutdown(stdContext.Background()); err != nil {
			t.Fatal(err)
		}

		got, ok := w.stopDeadline()
		if !ok {
			t.Fatal("expected Stop to get a bounded context")
		}

		if budget := got.Sub(before); budget < 9*time.Second || budget > 11*time.Second {
			t.Fatalf("expected a ten second stop budget, got: %v", budget)
		}
	})
}

// A Run error before shutdown is logged with the worker's type name and does
// not stop the application.
func TestBuilderWorkerRunErrorIsLogged(t *testing.T) {
	w := newTestWorker(fmt.Errorf("consumer lost its connection"))

	logs := newLogCapture("consumer lost its connection")
	configurators := []iris.Configurator{
		iris.ConfiguratorFunc(func(app *iris.Application) {
			app.Logger().SetOutput(logs)
		}),
	}

	app := buildWithWorker(t, configurators, w)
	t.Cleanup(func() { _ = app.Shutdown(stdContext.Background()) })

	waitFor(t, w.ran, "Run to return")
	waitFor(t, logs.matched, "the Run error to be logged")

	output := logs.String()
	if !strings.Contains(output, "iris_test.testWorker") {
		t.Fatalf("expected the worker's type name in the log output, got: %s", output)
	}
}

// logCapture is a logger output which signals when the wanted text is written.
type logCapture struct {
	want string

	mu      sync.Mutex
	buf     bytes.Buffer
	once    sync.Once
	matched chan struct{}
}

func newLogCapture(want string) *logCapture {
	return &logCapture{want: want, matched: make(chan struct{})}
}

func (w *logCapture) Write(p []byte) (int, error) {
	w.mu.Lock()
	n, err := w.buf.Write(p)
	found := strings.Contains(w.buf.String(), w.want)
	w.mu.Unlock()

	if found {
		w.once.Do(func() { close(w.matched) })
	}

	return n, err
}

func (w *logCapture) String() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.buf.String()
}
