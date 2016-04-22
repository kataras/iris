package fasthttp

import (
	"bytes"
	"io/ioutil"
	"testing"
)

func TestGzipBytes(t *testing.T) {
	testGzipBytes(t, "")
	testGzipBytes(t, "foobar")
	testGzipBytes(t, "выфаодлодл одлфываыв sd2 k34")
}

func testGzipBytes(t *testing.T, s string) {
	prefix := []byte("foobar")
	gzippedS := AppendGzipBytes(prefix, []byte(s))
	if !bytes.Equal(gzippedS[:len(prefix)], prefix) {
		t.Fatalf("unexpected prefix when compressing %q: %q. Expecting %q", s, gzippedS[:len(prefix)], prefix)
	}

	gunzippedS, err := AppendGunzipBytes(prefix, gzippedS[len(prefix):])
	if err != nil {
		t.Fatalf("unexpected error when uncompressing %q: %s", s, err)
	}
	if !bytes.Equal(gunzippedS[:len(prefix)], prefix) {
		t.Fatalf("unexpected prefix when uncompressing %q: %q. Expecting %q", s, gunzippedS[:len(prefix)], prefix)
	}
	gunzippedS = gunzippedS[len(prefix):]
	if string(gunzippedS) != s {
		t.Fatalf("unexpected uncompressed string %q. Expecting %q", gunzippedS, s)
	}
}

func TestGzipCompress(t *testing.T) {
	testGzipCompress(t, "")
	testGzipCompress(t, "foobar")
	testGzipCompress(t, "ajjnkn asdlkjfqoijfw  jfqkwj foj  eowjiq")
}

func TestFlateCompress(t *testing.T) {
	testFlateCompress(t, "")
	testFlateCompress(t, "foobar")
	testFlateCompress(t, "adf asd asd fasd fasd")
}

func testGzipCompress(t *testing.T, s string) {
	var buf bytes.Buffer
	zw := acquireGzipWriter(&buf, CompressDefaultCompression)
	if _, err := zw.Write([]byte(s)); err != nil {
		t.Fatalf("unexpected error: %s. s=%q", err, s)
	}
	releaseGzipWriter(zw)

	zr, err := acquireGzipReader(&buf)
	if err != nil {
		t.Fatalf("unexpected error: %s. s=%q", err, s)
	}
	body, err := ioutil.ReadAll(zr)
	if err != nil {
		t.Fatalf("unexpected error: %s. s=%q", err, s)
	}
	if string(body) != s {
		t.Fatalf("unexpected string after decompression: %q. Expecting %q", body, s)
	}
	releaseGzipReader(zr)
}

func testFlateCompress(t *testing.T, s string) {
	var buf bytes.Buffer
	zw := acquireFlateWriter(&buf, CompressDefaultCompression)
	if _, err := zw.Write([]byte(s)); err != nil {
		t.Fatalf("unexpected error: %s. s=%q", err, s)
	}
	releaseFlateWriter(zw)

	zr, err := acquireFlateReader(&buf)
	if err != nil {
		t.Fatalf("unexpected error: %s. s=%q", err, s)
	}
	body, err := ioutil.ReadAll(zr)
	if err != nil {
		t.Fatalf("unexpected error: %s. s=%q", err, s)
	}
	if string(body) != s {
		t.Fatalf("unexpected string after decompression: %q. Expecting %q", body, s)
	}
	releaseFlateReader(zr)
}
