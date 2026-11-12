package context

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// The hijack probe used to be a zero-byte Write, which on a live net/http
// writer triggers the implicit WriteHeader(200) and commits the response. Any
// status a later handler sets was then silently dropped.
func TestIsHijackedDoesNotCommitTheResponse(t *testing.T) {
	rec := httptest.NewRecorder()
	w := AcquireResponseWriter(rec)

	if w.IsHijacked() {
		t.Fatal("IsHijacked() reported a hijack on a plain recorder")
	}

	w.WriteHeader(http.StatusNotFound)
	w.Flush()

	if rec.Code != http.StatusNotFound {
		t.Errorf("status code = %d, want %d: asking whether the connection was "+
			"hijacked committed the response", rec.Code, http.StatusNotFound)
	}
}

func TestWriteHeaderIgnoresAnInvalidStatusCode(t *testing.T) {
	rec := httptest.NewRecorder()
	w := AcquireResponseWriter(rec)

	w.WriteHeader(http.StatusTeapot)

	// net/http panics on these, and Context.StatusCode passes its argument
	// through unchecked.
	for _, code := range []int{0, -1, 99, 1000, 1 << 20} {
		w.WriteHeader(code)

		if got := w.StatusCode(); got != http.StatusTeapot {
			t.Errorf("WriteHeader(%d) set the status to %d, want it ignored", code, got)
		}
	}

	w.Flush()

	if rec.Code != http.StatusTeapot {
		t.Errorf("status code = %d, want %d", rec.Code, http.StatusTeapot)
	}
}

func TestWriteHeaderAcceptsTheValidRange(t *testing.T) {
	for _, code := range []int{100, 200, 404, 599, 999} {
		w := AcquireResponseWriter(httptest.NewRecorder())

		w.WriteHeader(code)
		if got := w.StatusCode(); got != code {
			t.Errorf("WriteHeader(%d) produced %d", code, got)
		}
	}
}

func TestAcquireResponseWriterResetsTheHijackedFlag(t *testing.T) {
	w := AcquireResponseWriter(httptest.NewRecorder())
	w.hijacked = true
	ReleaseResponseWriter(w)

	next := AcquireResponseWriter(httptest.NewRecorder())
	if next.IsHijacked() {
		t.Error("a pooled response writer was acquired still marked as hijacked")
	}
}
