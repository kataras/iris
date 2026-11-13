package bodylimit_test

import (
	"compress/gzip"
	"encoding/json"
	stdErrors "errors"
	"fmt"
	"io"
	"net/http"
	nethttptest "net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/context"
	"github.com/kataras/iris/v14/httptest"
	"github.com/kataras/iris/v14/middleware/bodylimit"
	"github.com/kataras/iris/v14/middleware/compress"
	"github.com/kataras/iris/v14/rest/errors"
)

func TestBodyLimit(t *testing.T) {
	limit := int64(10) // Set the limit to 10 bytes

	handler := func(ctx iris.Context) {
		body, err := ctx.Body()
		if err != nil {
			ctx.StopWithPlainError(iris.StatusInternalServerError, err)
			return
		}
		ctx.Write(body)
	}

	app := iris.New()
	app.Use(bodylimit.New(limit))
	app.Post("/", handler)

	e := httptest.New(t, app)

	// Test with a body that is smaller than the limit.
	e.POST("/").WithText("123456789").Expect().Status(iris.StatusOK).Body().IsEqual("123456789")

	// Test with a body that is equal to the limit.
	e.POST("/").WithText("1234567890").Expect().Status(iris.StatusOK).Body().IsEqual("1234567890")

	// Test with a body that is bigger than the limit.
	e.POST("/").WithText("12345678910").Expect().Status(iris.StatusRequestEntityTooLarge)
}

// A body past the limit is one condition, found at one of two moments: before
// the handler runs when the request declares its length, or while the handler
// reads when it does not (a chunked upload). A client must not be able to
// tell which: the status, the code and the message are the same.
func TestBodyLimitAnswersTheSameOnBothPaths(t *testing.T) {
	const (
		limit   = 10
		tooLong = "12345678910"
	)

	var (
		limited []bool
		stored  []error
	)

	app := iris.New()
	app.UseRouter(func(ctx iris.Context) {
		ctx.Next()

		if err := ctx.GetErr(); err != nil {
			stored = append(stored, err)
		}
	})
	app.UseRouter(errors.Handler)
	app.Use(bodylimit.New(limit))
	app.Post("/", func(ctx iris.Context) {
		limited = append(limited, bodylimit.IsBodyLimited(ctx))

		body, err := ctx.Body()
		if err != nil {
			errors.HandleRequestError(ctx, err)
			return
		}

		ctx.Write(body)
	})

	if err := app.Build(); err != nil {
		t.Fatal(err)
	}

	srv := nethttptest.NewServer(app)
	defer srv.Close()

	type answer struct {
		Status        int
		CanonicalName string
		Message       string
		Detail        string
	}

	post := func(body io.Reader) answer {
		t.Helper()

		res, err := srv.Client().Post(srv.URL, "text/plain", body)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()

		var doc struct {
			ErrorCode struct {
				CanonicalName string `json:"canonical_name"`
			} `json:"http_error_code"`
			Message string `json:"message"`
			Detail  string `json:"detail"`
		}

		raw, err := io.ReadAll(res.Body)
		if err != nil {
			t.Fatal(err)
		}

		if err := json.Unmarshal(raw, &doc); err != nil {
			t.Fatalf("expected the rest/errors envelope, got %q: %v", raw, err)
		}

		return answer{res.StatusCode, doc.ErrorCode.CanonicalName, doc.Message, doc.Detail}
	}

	expected := answer{
		Status:        iris.StatusRequestEntityTooLarge,
		CanonicalName: "TOO_LARGE",
		Message:       "request body too large",
	}

	// A *strings.Reader has a length net/http knows, so the request declares
	// it and the middleware refuses it before the handler runs.
	if got := post(strings.NewReader(tooLong)); got != expected {
		t.Errorf("declared length: expected %+v, got %+v", expected, got)
	}

	if len(limited) != 0 {
		t.Fatalf("declared length: the handler ran for a request the middleware refused")
	}

	if len(stored) != 1 || !stdErrors.Is(stored[0], context.ErrRequestEntityTooLarge) {
		t.Errorf("declared length: expected the stored error to be context.ErrRequestEntityTooLarge, got %v", stored)
	} else if !stdErrors.Is(stored[0], context.ErrReadRequestBody) {
		// What it was before, for code that already checks for it.
		t.Errorf("declared length: expected the stored error to still be context.ErrReadRequestBody, got %v", stored[0])
	}

	// The same bytes behind a reader of unknown length go out chunked, and the
	// limit is found while the handler reads.
	if got := post(io.NopCloser(strings.NewReader(tooLong))); got != expected {
		t.Errorf("no declared length: expected %+v, got %+v", expected, got)
	}

	if len(limited) != 1 || !limited[0] {
		t.Fatalf("no declared length: expected the handler to run once with a limited body, got %v", limited)
	}
}

type limitedDoc struct {
	Text string `json:"text" form:"text"`
}

// limitedBodies has one entry for each kind of reader the tree puts on a
// request body (the JSON decoder, io.ReadAll, the form parsers): the route of
// newLimitedServer that reads it that way, and a body that carries a text to
// it.
var limitedBodies = []struct {
	path        string
	contentType string
	body        func(text string) string
	// A form has no error to give back, see TestBodyLimitChunkedBodyPastTheLimit.
	form bool
}{
	{path: "/json", contentType: "application/json", body: jsonBody},
	{path: "/body", contentType: "application/json", body: jsonBody},
	{path: "/string", contentType: "text/plain", body: plainBody},
	{path: "/raw", contentType: "text/plain", body: plainBody},
	{path: "/readall", contentType: "text/plain", body: plainBody},
	{path: "/form", contentType: "application/x-www-form-urlencoded", body: urlencodedBody, form: true},
	{path: "/form", contentType: "multipart/form-data; boundary=limit", body: multipartBody, form: true},
}

func jsonBody(text string) string { return `{"text":"` + text + `"}` }

func plainBody(text string) string { return text }

func urlencodedBody(text string) string { return "text=" + text }

func multipartBody(text string) string {
	return "--limit\r\nContent-Disposition: form-data; name=\"text\"\r\n\r\n" + text + "\r\n--limit--\r\n"
}

func multipartFileBody(text string) string {
	return "--limit\r\nContent-Disposition: form-data; name=\"text\"; filename=\"text.txt\"\r\n\r\n" + text +
		"\r\n--limit--\r\n"
}

// newLimitedServer serves the routes of limitedBodies, /file and /read, behind
// errors.Handler and bodylimit.New(limit). A handler that read the body
// answers "OK " and the text the body carried. One that could not gives the
// error to errors.HandleRequestError. Every answer says in X-Declared-Length
// which length the request declared, -1 for none. The handlers of before run
// ahead of bodylimit.New.
func newLimitedServer(t *testing.T, limit int64, before ...iris.Handler) *nethttptest.Server {
	t.Helper()

	answer := func(ctx iris.Context, text string, err error) {
		if err != nil {
			errors.HandleRequestError(ctx, err)
			return
		}

		ctx.WriteString("OK " + text)
	}

	app := iris.New()
	app.UseRouter(errors.Handler)
	app.Use(func(ctx iris.Context) {
		ctx.SetHeader("X-Declared-Length", strconv.FormatInt(ctx.Request().ContentLength, 10))
		ctx.Next()
	})
	app.Use(before...)
	app.Use(bodylimit.New(limit))

	app.Post("/json", func(ctx iris.Context) {
		var doc limitedDoc
		err := ctx.ReadJSON(&doc)
		answer(ctx, doc.Text, err)
	})
	app.Post("/body", func(ctx iris.Context) {
		var doc limitedDoc
		err := ctx.ReadBody(&doc)
		answer(ctx, doc.Text, err)
	})
	app.Post("/string", func(ctx iris.Context) {
		var text string
		err := ctx.ReadBody(&text)
		answer(ctx, text, err)
	})
	app.Post("/raw", func(ctx iris.Context) {
		body, err := ctx.Body()
		answer(ctx, string(body), err)
	})
	app.Post("/readall", func(ctx iris.Context) {
		body, err := io.ReadAll(ctx.Request().Body)
		answer(ctx, string(body), err)
	})
	app.Post("/form", func(ctx iris.Context) {
		var doc limitedDoc
		err := ctx.ReadForm(&doc)
		answer(ctx, doc.Text, err)
	})
	app.Post("/file", func(ctx iris.Context) {
		file, _, err := ctx.FormFile("text")
		if err != nil {
			answer(ctx, "", err)
			return
		}
		defer file.Close()

		body, err := io.ReadAll(file)
		answer(ctx, string(body), err)
	})

	// One Read after the other, the way a decoder reads: what it was given,
	// how much of that came together with the limit error, the error that
	// ended the reading and what one more Read answers.
	app.Post("/read", func(ctx iris.Context) {
		var (
			body      = ctx.Request().Body
			buf       = make([]byte, 64)
			data      []byte
			withError int
			err       error
		)

		for i := 0; i < 100 && err == nil; i++ {
			var n int
			n, err = body.Read(buf)
			data = append(data, buf[:n]...)

			if stdErrors.Is(err, context.ErrRequestEntityTooLarge) {
				withError += n
			}
		}

		n, again := body.Read(buf)
		ctx.Writef("read %q, %d with the limit error, end: %v, then: %d %v", data, withError, err, n, again)
	})

	if err := app.Build(); err != nil {
		t.Fatal(err)
	}

	srv := nethttptest.NewServer(app)
	t.Cleanup(srv.Close)

	return srv
}

// postLimited sends body to a route of newLimitedServer, with its length
// declared or chunked with none, and returns the status and the body of the
// answer. A Content-Encoding is sent when one is given.
func postLimited(t *testing.T, srv *nethttptest.Server, path, contentType, body string, chunked bool, encoding ...string) (int, string) {
	t.Helper()

	var (
		reader   io.Reader = strings.NewReader(body)
		declared           = strconv.Itoa(len(body))
	)

	if chunked {
		// net/http declares the length of a *strings.Reader. Behind another
		// reader it does not know it, and the body goes out chunked.
		reader, declared = io.NopCloser(reader), "-1"
	}

	req, err := http.NewRequest(http.MethodPost, srv.URL+path, reader)
	if err != nil {
		t.Fatal(err)
	}

	req.Header.Set("Content-Type", contentType)
	if len(encoding) > 0 {
		req.Header.Set("Content-Encoding", encoding[0])
	}

	res, err := srv.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	if got := res.Header.Get("X-Declared-Length"); got != declared {
		t.Fatalf("%s: expected the request to declare the length %s, got %q", path, declared, got)
	}

	raw, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}

	return res.StatusCode, string(raw)
}

// isTooLarge reports whether an answer is the one a body past the limit gets
// under errors.Handler: 413 TOO_LARGE, "request body too large".
func isTooLarge(status int, answer string) bool {
	var doc struct {
		ErrorCode struct {
			CanonicalName string `json:"canonical_name"`
		} `json:"http_error_code"`
		Message string `json:"message"`
	}

	if json.Unmarshal([]byte(answer), &doc) != nil {
		return false
	}

	return status == iris.StatusRequestEntityTooLarge && doc.ErrorCode.CanonicalName == "TOO_LARGE" &&
		doc.Message == "request body too large"
}

// short keeps a failure message readable when the answer is the body itself.
func short(answer string) string {
	if len(answer) > 80 {
		return answer[:80] + "..."
	}

	return answer
}

// A body with no declared length passes the limit while the handler reads
// it, so the limit is only as good as what the reader does there. It used to
// return the bytes past the limit together with the error. The JSON decoder
// keeps the bytes of such a Read and drops its error, so ReadJSON decoded a
// chunked body of any size. Whichever way the handler reads, a chunked body
// past the limit is answered like the same body with a declared length.
func TestBodyLimitChunkedBodyPastTheLimit(t *testing.T) {
	srv := newLimitedServer(t, 10)

	for _, size := range []int{300, 200 << 10} {
		text := strings.Repeat("a", size)

		for _, tt := range limitedBodies {
			body := tt.body(text)

			for _, chunked := range []bool{false, true} {
				status, answer := postLimited(t, srv, tt.path, tt.contentType, body, chunked)

				if tt.form && chunked {
					// ReadForm has no error for it: the one the form parser
					// got from the reader stops at context.GetForm, which
					// reports no form. What the reader owes a form is that
					// nothing of a body past the limit is bound.
					if answer != "OK " && !isTooLarge(status, answer) {
						t.Errorf("%s (%s), %d bytes, chunked: expected no form value to be bound, got %d %s",
							tt.path, tt.contentType, len(body), status, short(answer))
					}

					continue
				}

				if !isTooLarge(status, answer) {
					t.Errorf("%s (%s), %d bytes, chunked=%v: expected 413 TOO_LARGE, got %d %s",
						tt.path, tt.contentType, len(body), chunked, status, short(answer))
				}
			}
		}
	}
}

// What the handler's own Read calls see of a chunked body: the bytes up to
// the limit, then the error and nothing with it, on every Read from there on.
// Not one byte past the limit, with or without the error.
func TestBodyLimitReaderStopsAtTheLimit(t *testing.T) {
	const atLimit = "0123456789"

	srv := newLimitedServer(t, int64(len(atLimit)))

	tests := []struct {
		body     string
		expected string
	}{
		{
			body: atLimit,
			expected: fmt.Sprintf("read %q, 0 with the limit error, end: %v, then: 0 %v",
				atLimit, io.EOF, io.EOF),
		},
		{
			body: atLimit + "a",
			expected: fmt.Sprintf("read %q, 0 with the limit error, end: %v, then: 0 %v",
				atLimit, context.ErrRequestEntityTooLarge, context.ErrRequestEntityTooLarge),
		},
		{
			body: atLimit + strings.Repeat("a", 300),
			expected: fmt.Sprintf("read %q, 0 with the limit error, end: %v, then: 0 %v",
				atLimit, context.ErrRequestEntityTooLarge, context.ErrRequestEntityTooLarge),
		},
	}

	for _, tt := range tests {
		status, answer := postLimited(t, srv, "/read", "text/plain", tt.body, true)
		if status != iris.StatusOK || answer != tt.expected {
			t.Errorf("%d bytes, chunked:\nexpected: %s\n     got: %d %s", len(tt.body), tt.expected, status, short(answer))
		}
	}
}

// The limit is the last byte a body may have. A body of exactly the limit is
// read in full, declared or not, and one byte more is refused.
func TestBodyLimitAcceptsABodyOfExactlyTheLimit(t *testing.T) {
	const text = "0123456789"

	for _, tt := range limitedBodies {
		var (
			body     = tt.body(text)
			exact    = newLimitedServer(t, int64(len(body)))
			oneShort = newLimitedServer(t, int64(len(body)-1))
		)

		for _, chunked := range []bool{false, true} {
			status, answer := postLimited(t, exact, tt.path, tt.contentType, body, chunked)
			if status != iris.StatusOK || answer != "OK "+text {
				t.Errorf("%s (%s), a body of the limit, chunked=%v: expected 200 OK %s, got %d %s",
					tt.path, tt.contentType, chunked, text, status, short(answer))
			}

			if tt.form && chunked {
				continue // see TestBodyLimitChunkedBodyPastTheLimit.
			}

			status, answer = postLimited(t, oneShort, tt.path, tt.contentType, body, chunked)
			if !isTooLarge(status, answer) {
				t.Errorf("%s (%s), a body one byte past the limit, chunked=%v: expected 413 TOO_LARGE, got %d %s",
					tt.path, tt.contentType, chunked, status, short(answer))
			}
		}
	}
}

// An upload goes through the multipart parser, and ctx.FormFile returns the
// error of that parser. The limit falls inside the content of the file here.
// Where it falls inside the header lines of a part, the parser reports a
// malformed header for a chunked body and the limit error is lost to it.
func TestBodyLimitChunkedUploadPastTheLimit(t *testing.T) {
	const contentType = "multipart/form-data; boundary=limit"

	var (
		text    = strings.Repeat("a", 300)
		body    = multipartFileBody(text)
		exact   = newLimitedServer(t, int64(len(body)))
		tooLong = newLimitedServer(t, int64(len(body)-200))
	)

	for _, chunked := range []bool{false, true} {
		status, answer := postLimited(t, exact, "/file", contentType, body, chunked)
		if status != iris.StatusOK || answer != "OK "+text {
			t.Errorf("an upload of the limit, chunked=%v: expected 200 with the file, got %d %s", chunked, status, short(answer))
		}

		status, answer = postLimited(t, tooLong, "/file", contentType, body, chunked)
		if !isTooLarge(status, answer) {
			t.Errorf("an upload past the limit, chunked=%v: expected 413 TOO_LARGE, got %d %s", chunked, status, short(answer))
		}
	}
}

// A compressed body declares its length on the wire. With compress.Handler
// registered before bodylimit.New the limit counts the bytes the handler
// reads, the decompressed ones, so the declared length says nothing of them
// and the reader is the only check, chunked or not. ReadJSON used to decode
// such a body in full.
func TestBodyLimitDecompressedBodyPastTheLimit(t *testing.T) {
	const limit = 1024

	srv := newLimitedServer(t, limit, compress.Handler)

	gzipped := func(body string) string {
		var wire strings.Builder

		w := gzip.NewWriter(&wire)
		if _, err := w.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}

		if err := w.Close(); err != nil {
			t.Fatal(err)
		}

		return wire.String()
	}

	var (
		atLimit = strings.Repeat("a", limit-len(jsonBody("")))
		exact   = gzipped(jsonBody(atLimit))
		tooLong = gzipped(jsonBody(strings.Repeat("a", 200<<10)))
	)

	if len(tooLong) > limit {
		t.Fatalf("expected the long body to be within the limit on the wire, got %d bytes", len(tooLong))
	}

	for _, path := range []string{"/json", "/body"} {
		for _, chunked := range []bool{false, true} {
			status, answer := postLimited(t, srv, path, "application/json", exact, chunked, "gzip")
			if status != iris.StatusOK || answer != "OK "+atLimit {
				t.Errorf("%s, %d bytes decompressed, chunked=%v: expected 200 with the text, got %d %s",
					path, limit, chunked, status, short(answer))
			}

			status, answer = postLimited(t, srv, path, "application/json", tooLong, chunked, "gzip")
			if !isTooLarge(status, answer) {
				t.Errorf("%s, %d bytes on the wire, chunked=%v: expected 413 TOO_LARGE, got %d %s",
					path, len(tooLong), chunked, status, short(answer))
			}
		}
	}
}
