package fasthttp

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"sort"
	"testing"
	"time"
)

func TestNewVHostPathRewriter(t *testing.T) {
	var ctx RequestCtx
	var req Request
	req.Header.SetHost("foobar.com")
	req.SetRequestURI("/foo/bar/baz")
	ctx.Init(&req, nil, nil)

	f := NewVHostPathRewriter(0)
	path := f(&ctx)
	expectedPath := "/foobar.com/foo/bar/baz"
	if string(path) != expectedPath {
		t.Fatalf("unexpected path %q. Expecting %q", path, expectedPath)
	}

	ctx.Request.Reset()
	ctx.Request.SetRequestURI("https://aaa.bbb.cc/one/two/three/four?asdf=dsf")
	f = NewVHostPathRewriter(2)
	path = f(&ctx)
	expectedPath = "/aaa.bbb.cc/three/four"
	if string(path) != expectedPath {
		t.Fatalf("unexpected path %q. Expecting %q", path, expectedPath)
	}
}

func TestNewVHostPathRewriterMaliciousHost(t *testing.T) {
	var ctx RequestCtx
	var req Request
	req.Header.SetHost("/../../../etc/passwd")
	req.SetRequestURI("/foo/bar/baz")
	ctx.Init(&req, nil, nil)

	f := NewVHostPathRewriter(0)
	path := f(&ctx)
	expectedPath := "/invalid-host/foo/bar/baz"
	if string(path) != expectedPath {
		t.Fatalf("unexpected path %q. Expecting %q", path, expectedPath)
	}
}

func TestServeFileHead(t *testing.T) {
	var ctx RequestCtx
	var req Request
	req.Header.SetMethod("HEAD")
	req.SetRequestURI("http://foobar.com/baz")
	ctx.Init(&req, nil, nil)

	ServeFile(&ctx, "fs.go")

	var resp Response
	resp.SkipBody = true
	s := ctx.Response.String()
	br := bufio.NewReader(bytes.NewBufferString(s))
	if err := resp.Read(br); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	ce := resp.Header.Peek("Content-Encoding")
	if len(ce) > 0 {
		t.Fatalf("Unexpected 'Content-Encoding' %q", ce)
	}

	body := resp.Body()
	if len(body) > 0 {
		t.Fatalf("unexpected response body %q. Expecting empty body", body)
	}

	expectedBody, err := getFileContents("/fs.go")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	contentLength := resp.Header.ContentLength()
	if contentLength != len(expectedBody) {
		t.Fatalf("unexpected Content-Length: %d. expecting %d", contentLength, len(expectedBody))
	}
}

func TestServeFileCompressed(t *testing.T) {
	var ctx RequestCtx
	var req Request
	req.SetRequestURI("http://foobar.com/baz")
	req.Header.Set("Accept-Encoding", "gzip")
	ctx.Init(&req, nil, nil)

	ServeFile(&ctx, "fs.go")

	var resp Response
	s := ctx.Response.String()
	br := bufio.NewReader(bytes.NewBufferString(s))
	if err := resp.Read(br); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	ce := resp.Header.Peek("Content-Encoding")
	if string(ce) != "gzip" {
		t.Fatalf("Unexpected 'Content-Encoding' %q. Expecting %q", ce, "gzip")
	}

	body, err := resp.BodyGunzip()
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	expectedBody, err := getFileContents("/fs.go")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if !bytes.Equal(body, expectedBody) {
		t.Fatalf("unexpected body %q. expecting %q", body, expectedBody)
	}
}

func TestServeFileUncompressed(t *testing.T) {
	var ctx RequestCtx
	var req Request
	req.SetRequestURI("http://foobar.com/baz")
	req.Header.Set("Accept-Encoding", "gzip")
	ctx.Init(&req, nil, nil)

	ServeFileUncompressed(&ctx, "fs.go")

	var resp Response
	s := ctx.Response.String()
	br := bufio.NewReader(bytes.NewBufferString(s))
	if err := resp.Read(br); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	ce := resp.Header.Peek("Content-Encoding")
	if len(ce) > 0 {
		t.Fatalf("Unexpected 'Content-Encoding' %q", ce)
	}

	body := resp.Body()
	expectedBody, err := getFileContents("/fs.go")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if !bytes.Equal(body, expectedBody) {
		t.Fatalf("unexpected body %q. expecting %q", body, expectedBody)
	}
}

func TestFSByteRangeConcurrent(t *testing.T) {
	fs := &FS{
		Root:            ".",
		AcceptByteRange: true,
	}
	h := fs.NewRequestHandler()

	concurrency := 10
	ch := make(chan struct{}, concurrency)
	for i := 0; i < concurrency; i++ {
		go func() {
			for j := 0; j < 5; j++ {
				testFSByteRange(t, h, "/fs.go")
				testFSByteRange(t, h, "/README.md")
			}
			ch <- struct{}{}
		}()
	}

	for i := 0; i < concurrency; i++ {
		select {
		case <-time.After(time.Second):
			t.Fatalf("timeout")
		case <-ch:
		}
	}
}

func TestFSByteRangeSingleThread(t *testing.T) {
	fs := &FS{
		Root:            ".",
		AcceptByteRange: true,
	}
	h := fs.NewRequestHandler()

	testFSByteRange(t, h, "/fs.go")
	testFSByteRange(t, h, "/README.md")
}

func testFSByteRange(t *testing.T, h RequestHandler, filePath string) {
	var ctx RequestCtx
	ctx.Init(&Request{}, nil, nil)

	expectedBody, err := getFileContents(filePath)
	if err != nil {
		t.Fatalf("cannot read file %q: %s", filePath, err)
	}

	fileSize := len(expectedBody)
	startPos := rand.Intn(fileSize)
	endPos := rand.Intn(fileSize)
	if endPos < startPos {
		startPos, endPos = endPos, startPos
	}

	ctx.Request.SetRequestURI(filePath)
	ctx.Request.Header.SetByteRange(startPos, endPos)
	h(&ctx)

	var resp Response
	s := ctx.Response.String()
	br := bufio.NewReader(bytes.NewBufferString(s))
	if err := resp.Read(br); err != nil {
		t.Fatalf("unexpected error: %s. filePath=%q", err, filePath)
	}
	if resp.StatusCode() != StatusPartialContent {
		t.Fatalf("unexpected status code: %d. Expecting %d. filePath=%q", resp.StatusCode(), StatusPartialContent, filePath)
	}
	cr := resp.Header.Peek("Content-Range")

	expectedCR := fmt.Sprintf("bytes %d-%d/%d", startPos, endPos, fileSize)
	if string(cr) != expectedCR {
		t.Fatalf("unexpected content-range %q. Expecting %q. filePath=%q", cr, expectedCR, filePath)
	}
	body := resp.Body()
	bodySize := endPos - startPos + 1
	if len(body) != bodySize {
		t.Fatalf("unexpected body size %d. Expecting %d. filePath=%q, startPos=%d, endPos=%d",
			len(body), bodySize, filePath, startPos, endPos)
	}

	expectedBody = expectedBody[startPos : endPos+1]
	if !bytes.Equal(body, expectedBody) {
		t.Fatalf("unexpected body %q. Expecting %q. filePath=%q, startPos=%d, endPos=%d",
			body, expectedBody, filePath, startPos, endPos)
	}
}

func getFileContents(path string) ([]byte, error) {
	path = "." + path
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return ioutil.ReadAll(f)
}

func TestParseByteRangeSuccess(t *testing.T) {
	testParseByteRangeSuccess(t, "bytes=0-0", 1, 0, 0)
	testParseByteRangeSuccess(t, "bytes=1234-6789", 6790, 1234, 6789)

	testParseByteRangeSuccess(t, "bytes=123-", 456, 123, 455)
	testParseByteRangeSuccess(t, "bytes=-1", 1, 0, 0)
	testParseByteRangeSuccess(t, "bytes=-123", 456, 333, 455)

	// End position exceeding content-length. It should be updated to content-length-1.
	// See https://www.w3.org/Protocols/rfc2616/rfc2616-sec14.html#sec14.35
	testParseByteRangeSuccess(t, "bytes=1-2345", 234, 1, 233)
	testParseByteRangeSuccess(t, "bytes=0-2345", 2345, 0, 2344)

	// Start position overflow. Whole range must be returned.
	// See https://www.w3.org/Protocols/rfc2616/rfc2616-sec14.html#sec14.35
	testParseByteRangeSuccess(t, "bytes=-567", 56, 0, 55)
}

func testParseByteRangeSuccess(t *testing.T, v string, contentLength, startPos, endPos int) {
	startPos1, endPos1, err := ParseByteRange([]byte(v), contentLength)
	if err != nil {
		t.Fatalf("unexpected error: %s. v=%q, contentLength=%d", err, v, contentLength)
	}
	if startPos1 != startPos {
		t.Fatalf("unexpected startPos=%d. Expecting %d. v=%q, contentLength=%d", startPos1, startPos, v, contentLength)
	}
	if endPos1 != endPos {
		t.Fatalf("unexpected endPos=%d. Expectind %d. v=%q, contentLenght=%d", endPos1, endPos, v, contentLength)
	}
}

func TestParseByteRangeError(t *testing.T) {
	// invalid value
	testParseByteRangeError(t, "asdfasdfas", 1234)

	// invalid units
	testParseByteRangeError(t, "foobar=1-34", 600)

	// missing '-'
	testParseByteRangeError(t, "bytes=1234", 1235)

	// non-numeric range
	testParseByteRangeError(t, "bytes=foobar", 123)
	testParseByteRangeError(t, "bytes=1-foobar", 123)
	testParseByteRangeError(t, "bytes=df-344", 545)

	// multiple byte ranges
	testParseByteRangeError(t, "bytes=1-2,4-6", 123)

	// byte range exceeding contentLength
	testParseByteRangeError(t, "bytes=123-", 12)

	// startPos exceeding endPos
	testParseByteRangeError(t, "bytes=123-34", 1234)
}

func testParseByteRangeError(t *testing.T, v string, contentLength int) {
	_, _, err := ParseByteRange([]byte(v), contentLength)
	if err == nil {
		t.Fatalf("expecting error when parsing byte range %q", v)
	}
}

func TestFSCompressConcurrent(t *testing.T) {
	fs := &FS{
		Root:               ".",
		GenerateIndexPages: true,
		Compress:           true,
	}
	h := fs.NewRequestHandler()

	concurrency := 4
	ch := make(chan struct{}, concurrency)
	for i := 0; i < concurrency; i++ {
		go func() {
			for j := 0; j < 5; j++ {
				testFSCompress(t, h, "/fs.go")
				testFSCompress(t, h, "/")
				testFSCompress(t, h, "/README.md")
			}
			ch <- struct{}{}
		}()
	}

	for i := 0; i < concurrency; i++ {
		select {
		case <-ch:
		case <-time.After(time.Second):
			t.Fatalf("timeout")
		}
	}
}

func TestFSCompressSingleThread(t *testing.T) {
	fs := &FS{
		Root:               ".",
		GenerateIndexPages: true,
		Compress:           true,
	}
	h := fs.NewRequestHandler()

	testFSCompress(t, h, "/fs.go")
	testFSCompress(t, h, "/")
	testFSCompress(t, h, "/README.md")
}

func testFSCompress(t *testing.T, h RequestHandler, filePath string) {
	var ctx RequestCtx
	ctx.Init(&Request{}, nil, nil)

	// request uncompressed file
	ctx.Request.Reset()
	ctx.Request.SetRequestURI(filePath)
	h(&ctx)

	var resp Response
	s := ctx.Response.String()
	br := bufio.NewReader(bytes.NewBufferString(s))
	if err := resp.Read(br); err != nil {
		t.Fatalf("unexpected error: %s. filePath=%q", err, filePath)
	}
	if resp.StatusCode() != StatusOK {
		t.Fatalf("unexpected status code: %d. Expecting %d. filePath=%q", resp.StatusCode(), StatusOK, filePath)
	}
	ce := resp.Header.Peek("Content-Encoding")
	if string(ce) != "" {
		t.Fatalf("unexpected content-encoding %q. Expecting empty string. filePath=%q", ce, filePath)
	}
	body := string(resp.Body())

	// request compressed file
	ctx.Request.Reset()
	ctx.Request.SetRequestURI(filePath)
	ctx.Request.Header.Set("Accept-Encoding", "gzip")
	h(&ctx)
	s = ctx.Response.String()
	br = bufio.NewReader(bytes.NewBufferString(s))
	if err := resp.Read(br); err != nil {
		t.Fatalf("unexpected error: %s. filePath=%q", err, filePath)
	}
	if resp.StatusCode() != StatusOK {
		t.Fatalf("unexpected status code: %d. Expecting %d. filePath=%q", resp.StatusCode(), StatusOK, filePath)
	}
	ce = resp.Header.Peek("Content-Encoding")
	if string(ce) != "gzip" {
		t.Fatalf("unexpected content-encoding %q. Expecting %q. filePath=%q", ce, "gzip", filePath)
	}
	zbody, err := resp.BodyGunzip()
	if err != nil {
		t.Fatalf("unexpected error when gunzipping response body: %s. filePath=%q", err, filePath)
	}
	if string(zbody) != body {
		t.Fatalf("unexpected body %q. Expected %q. FilePath=%q", zbody, body, filePath)
	}
}

func TestFileLock(t *testing.T) {
	for i := 0; i < 10; i++ {
		filePath := fmt.Sprintf("foo/bar/%d.jpg", i)
		lock := getFileLock(filePath)
		lock.Lock()
		lock.Unlock()
	}

	for i := 0; i < 10; i++ {
		filePath := fmt.Sprintf("foo/bar/%d.jpg", i)
		lock := getFileLock(filePath)
		lock.Lock()
		lock.Unlock()
	}
}

func TestFSHandlerSingleThread(t *testing.T) {
	requestHandler := FSHandler(".", 0)

	f, err := os.Open(".")
	if err != nil {
		t.Fatalf("cannot open cwd: %s", err)
	}

	filenames, err := f.Readdirnames(0)
	f.Close()
	if err != nil {
		t.Fatalf("cannot read dirnames in cwd: %s", err)
	}
	sort.Sort(sort.StringSlice(filenames))

	for i := 0; i < 3; i++ {
		fsHandlerTest(t, requestHandler, filenames)
	}
}

func TestFSHandlerConcurrent(t *testing.T) {
	requestHandler := FSHandler(".", 0)

	f, err := os.Open(".")
	if err != nil {
		t.Fatalf("cannot open cwd: %s", err)
	}

	filenames, err := f.Readdirnames(0)
	f.Close()
	if err != nil {
		t.Fatalf("cannot read dirnames in cwd: %s", err)
	}
	sort.Sort(sort.StringSlice(filenames))

	concurrency := 10
	ch := make(chan struct{}, concurrency)
	for j := 0; j < concurrency; j++ {
		go func() {
			for i := 0; i < 3; i++ {
				fsHandlerTest(t, requestHandler, filenames)
			}
			ch <- struct{}{}
		}()
	}

	for j := 0; j < concurrency; j++ {
		select {
		case <-ch:
		case <-time.After(time.Second):
			t.Fatalf("timeout")
		}
	}
}

func fsHandlerTest(t *testing.T, requestHandler RequestHandler, filenames []string) {
	var ctx RequestCtx
	var req Request
	ctx.Init(&req, nil, defaultLogger)
	ctx.Request.Header.SetHost("foobar.com")

	filesTested := 0
	for _, name := range filenames {
		f, err := os.Open(name)
		if err != nil {
			t.Fatalf("cannot open file %q: %s", name, err)
		}
		stat, err := f.Stat()
		if err != nil {
			t.Fatalf("cannot get file stat %q: %s", name, err)
		}
		if stat.IsDir() {
			f.Close()
			continue
		}
		data, err := ioutil.ReadAll(f)
		f.Close()
		if err != nil {
			t.Fatalf("cannot read file contents %q: %s", name, err)
		}

		ctx.URI().Update(name)
		requestHandler(&ctx)
		if ctx.Response.bodyStream == nil {
			t.Fatalf("response body stream must be non-empty")
		}
		body, err := ioutil.ReadAll(ctx.Response.bodyStream)
		if err != nil {
			t.Fatalf("error when reading response body stream: %s", err)
		}
		if !bytes.Equal(body, data) {
			t.Fatalf("unexpected body returned: %q. Expecting %q", body, data)
		}
		filesTested++
		if filesTested >= 10 {
			break
		}
	}

	// verify index page generation
	ctx.URI().Update("/")
	requestHandler(&ctx)
	if ctx.Response.bodyStream == nil {
		t.Fatalf("response body stream must be non-empty")
	}
	body, err := ioutil.ReadAll(ctx.Response.bodyStream)
	if err != nil {
		t.Fatalf("error when reading response body stream: %s", err)
	}
	if len(body) == 0 {
		t.Fatalf("index page must be non-empty")
	}
}

func TestStripPathSlashes(t *testing.T) {
	testStripPathSlashes(t, "", 0, "")
	testStripPathSlashes(t, "", 10, "")
	testStripPathSlashes(t, "/", 0, "")
	testStripPathSlashes(t, "/", 1, "")
	testStripPathSlashes(t, "/", 10, "")
	testStripPathSlashes(t, "/foo/bar/baz", 0, "/foo/bar/baz")
	testStripPathSlashes(t, "/foo/bar/baz", 1, "/bar/baz")
	testStripPathSlashes(t, "/foo/bar/baz", 2, "/baz")
	testStripPathSlashes(t, "/foo/bar/baz", 3, "")
	testStripPathSlashes(t, "/foo/bar/baz", 10, "")

	// trailing slash
	testStripPathSlashes(t, "/foo/bar/", 0, "/foo/bar")
	testStripPathSlashes(t, "/foo/bar/", 1, "/bar")
	testStripPathSlashes(t, "/foo/bar/", 2, "")
	testStripPathSlashes(t, "/foo/bar/", 3, "")
}

func testStripPathSlashes(t *testing.T, path string, stripSlashes int, expectedPath string) {
	s := stripLeadingSlashes([]byte(path), stripSlashes)
	s = stripTrailingSlashes(s)
	if string(s) != expectedPath {
		t.Fatalf("unexpected path after stripping %q with stripSlashes=%d: %q. Expecting %q", path, stripSlashes, s, expectedPath)
	}
}

func TestFileExtension(t *testing.T) {
	testFileExtension(t, "foo.bar", false, ".bar")
	testFileExtension(t, "foobar", false, "")
	testFileExtension(t, "foo.bar.baz", false, ".baz")
	testFileExtension(t, "", false, "")
	testFileExtension(t, "/a/b/c.d/efg.jpg", false, ".jpg")

	testFileExtension(t, "foo.bar", true, ".bar")
	testFileExtension(t, "foobar.fasthttp.gz", true, "")
	testFileExtension(t, "foo.bar.baz.fasthttp.gz", true, ".baz")
	testFileExtension(t, "", true, "")
	testFileExtension(t, "/a/b/c.d/efg.jpg.fasthttp.gz", true, ".jpg")
}

func testFileExtension(t *testing.T, path string, compressed bool, expectedExt string) {
	ext := fileExtension(path, compressed)
	if ext != expectedExt {
		t.Fatalf("unexpected file extension for file %q: %q. Expecting %q", path, ext, expectedExt)
	}
}
