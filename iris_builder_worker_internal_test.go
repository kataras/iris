package iris

import (
	stdContext "context"
	"sync"
	"testing"
	"time"
)

// interleavingWorker records whether Close ever ran while Run was live, which
// is the failure the start/stop mutual exclusion exists to prevent.
type interleavingWorker struct {
	mu      sync.Mutex
	running bool
	badness int
	closes  int
}

func (w *interleavingWorker) Run(ctx stdContext.Context) error {
	w.mu.Lock()
	w.running = true
	w.mu.Unlock()

	<-ctx.Done()

	w.mu.Lock()
	w.running = false
	w.mu.Unlock()

	return nil
}

func (w *interleavingWorker) Stop(stdContext.Context) error { return nil }

func (w *interleavingWorker) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.closes++
	if w.running {
		w.badness++
	}

	return nil
}

func newWorkerTestBuilder() *builder {
	return &builder{app: New(), once: new(sync.Once)}
}

// The shutdown step and startWorker cannot interleave: while the start
// decision is being taken, the stop decision waits. Holding startMu from the
// test is what makes this deterministic; the window it closes is otherwise
// only reachable when a signal arrives inside Build.
func TestWorkerStartAndStopAreMutuallyExclusive(t *testing.T) {
	g := newWorkerTestBuilder()
	w := &interleavingWorker{}

	g.registerWorker(w)
	reg := g.workers[0]

	reg.startMu.Lock() // stand in for startWorker having taken the decision.

	stopped := make(chan struct{})
	go func() {
		_ = g.stopWorker(reg)(stdContext.Background())
		close(stopped)
	}()

	select {
	case <-stopped:
		t.Fatal("expected the shutdown step to wait for the start decision")
	case <-time.After(100 * time.Millisecond):
	}

	reg.startMu.Unlock()

	select {
	case <-stopped:
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for the shutdown step")
	}

	// The stop decision won, so a later start must launch nothing: a goroutine
	// started here would run against a worker which is already closed.
	g.startWorker(reg)

	if reg.started.Load() {
		t.Fatal("expected a stopped worker not to be started")
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	if w.closes != 1 {
		t.Fatalf("expected exactly one Close, got: %d", w.closes)
	}

	if w.badness != 0 {
		t.Fatalf("Close ran while Run was live %d time(s)", w.badness)
	}
}

// The same invariant under contention, which is the shape a signal handler
// racing Build actually produces.
func TestWorkerStartAndStopUnderContention(t *testing.T) {
	for range 200 {
		g := newWorkerTestBuilder()
		w := &interleavingWorker{}

		g.registerWorker(w)
		reg := g.workers[0]

		var wg sync.WaitGroup
		wg.Add(2)

		go func() {
			defer wg.Done()
			g.startWorker(reg)
		}()

		go func() {
			defer wg.Done()
			_ = g.stopWorker(reg)(stdContext.Background())
		}()

		wg.Wait()

		if reg.started.Load() {
			<-reg.done // the goroutine was launched: it must return on its own.
		}

		w.mu.Lock()
		badness := w.badness
		w.mu.Unlock()

		if badness != 0 {
			t.Fatalf("Close ran while Run was live %d time(s)", badness)
		}
	}
}
