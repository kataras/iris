package router

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMakeWrapperFunc(t *testing.T) {
	var (
		firstBody    = []byte("1")
		secondBody   = []byte("2")
		mainBody     = []byte("3")
		expectedBody = append(firstBody, append(secondBody, mainBody...)...)
	)

	pre := func(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
		w.Header().Set("X-Custom", "data")
		next(w, r)
	}

	first := func(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
		w.Write(firstBody)
		next(w, r)
	}

	second := func(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
		w.Write(secondBody)
		next(w, r)
	}

	mainHandler := func(w http.ResponseWriter, r *http.Request) {
		w.Write(mainBody)
	}

	wrapper := makeWrapperFunc(second, first)
	wrapper = makeWrapperFunc(wrapper, pre)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "https://iris-go.com", nil)
	wrapper(w, r, mainHandler)

	if got := w.Body.Bytes(); !bytes.Equal(expectedBody, got) {
		t.Fatalf("expected boy: %s but got: %s", string(expectedBody), string(got))
	}

	if expected, got := "data", w.Header().Get("X-Custom"); expected != got {
		t.Fatalf("expected x-custom header: %s but got: %s", expected, got)
	}
}
