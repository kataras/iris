package fasthttp

import (
	"bufio"
	"bytes"
	"fmt"
	"net"
	"testing"
	"time"
)

func TestAppendHTMLEscape(t *testing.T) {
	testAppendHTMLEscape(t, "", "")
	testAppendHTMLEscape(t, "<", "&lt;")
	testAppendHTMLEscape(t, "a", "a")
	testAppendHTMLEscape(t, `><"''`, "&gt;&lt;&quot;&#39;&#39;")
	testAppendHTMLEscape(t, "fo<b x='ss'>a</b>xxx", "fo&lt;b x=&#39;ss&#39;&gt;a&lt;/b&gt;xxx")
}

func testAppendHTMLEscape(t *testing.T, s, expectedS string) {
	buf := AppendHTMLEscapeBytes(nil, []byte(s))
	if string(buf) != expectedS {
		t.Fatalf("unexpected html-escaped string %q. Expecting %q. Original string %q", buf, expectedS, s)
	}
}

func TestParseIPv4(t *testing.T) {
	testParseIPv4(t, "0.0.0.0", true)
	testParseIPv4(t, "255.255.255.255", true)
	testParseIPv4(t, "123.45.67.89", true)

	// ipv6 shouldn't work
	testParseIPv4(t, "2001:4860:0:2001::68", false)

	// invalid ip
	testParseIPv4(t, "foobar", false)
	testParseIPv4(t, "1.2.3", false)
	testParseIPv4(t, "123.456.789.11", false)
}

func testParseIPv4(t *testing.T, ipStr string, isValid bool) {
	ip, err := ParseIPv4(nil, []byte(ipStr))
	if isValid {
		if err != nil {
			t.Fatalf("unexpected error when parsing ip %q: %s", ipStr, err)
		}
		s := string(AppendIPv4(nil, ip))
		if s != ipStr {
			t.Fatalf("unexpected ip parsed %q. Expecting %q", s, ipStr)
		}
	} else {
		if err == nil {
			t.Fatalf("expecting error when parsing ip %q", ipStr)
		}
	}
}

func TestAppendIPv4(t *testing.T) {
	testAppendIPv4(t, "0.0.0.0", true)
	testAppendIPv4(t, "127.0.0.1", true)
	testAppendIPv4(t, "8.8.8.8", true)
	testAppendIPv4(t, "123.45.67.89", true)

	// ipv6 shouldn't work
	testAppendIPv4(t, "2001:4860:0:2001::68", false)
}

func testAppendIPv4(t *testing.T, ipStr string, isValid bool) {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		t.Fatalf("cannot parse ip %q", ipStr)
	}
	s := string(AppendIPv4(nil, ip))
	if isValid {
		if s != ipStr {
			t.Fatalf("unepxected ip %q. Expecting %q", s, ipStr)
		}
	} else {
		ipStr = "non-v4 ip passed to AppendIPv4"
		if s != ipStr {
			t.Fatalf("unexpected ip %q. Expecting %q", s, ipStr)
		}
	}
}

func testAppendUint(t *testing.T, n int) {
	expectedS := fmt.Sprintf("%d", n)
	s := AppendUint(nil, n)
	if string(s) != expectedS {
		t.Fatalf("unexpected uint %q. Expecting %q. n=%d", s, expectedS, n)
	}
}

func testWriteHexInt(t *testing.T, n int, expectedS string) {
	var w ByteBuffer
	bw := bufio.NewWriter(&w)
	if err := writeHexInt(bw, n); err != nil {
		t.Fatalf("unexpected error when writing hex %x: %s", n, err)
	}
	if err := bw.Flush(); err != nil {
		t.Fatalf("unexpected error when flushing hex %x: %s", n, err)
	}
	s := string(w.B)
	if s != expectedS {
		t.Fatalf("unexpected hex after writing %q. Expected %q", s, expectedS)
	}
}

func TestReadHexIntError(t *testing.T) {
	testReadHexIntError(t, "")
	testReadHexIntError(t, "ZZZ")
	testReadHexIntError(t, "-123")
	testReadHexIntError(t, "+434")
}

func testReadHexIntError(t *testing.T, s string) {
	r := bytes.NewBufferString(s)
	br := bufio.NewReader(r)
	n, err := readHexInt(br)
	if err == nil {
		t.Fatalf("expecting error when reading hex int %q", s)
	}
	if n >= 0 {
		t.Fatalf("unexpected hex value read %d for hex int %q. must be negative", n, s)
	}
}

func testReadHexIntSuccess(t *testing.T, s string, expectedN int) {
	r := bytes.NewBufferString(s)
	br := bufio.NewReader(r)
	n, err := readHexInt(br)
	if err != nil {
		t.Fatalf("unexpected error: %s. s=%q", err, s)
	}
	if n != expectedN {
		t.Fatalf("unexpected hex int %d. Expected %d. s=%q", n, expectedN, s)
	}
}

func TestAppendHTTPDate(t *testing.T) {
	d := time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)
	s := string(AppendHTTPDate(nil, d))
	expectedS := "Tue, 10 Nov 2009 23:00:00 GMT"
	if s != expectedS {
		t.Fatalf("unexpected date %q. Expecting %q", s, expectedS)
	}

	b := []byte("prefix")
	s = string(AppendHTTPDate(b, d))
	if s[:len(b)] != string(b) {
		t.Fatalf("unexpected prefix %q. Expecting %q", s[:len(b)], b)
	}
	s = s[len(b):]
	if s != expectedS {
		t.Fatalf("unexpected date %q. Expecting %q", s, expectedS)
	}
}

func TestParseUintError(t *testing.T) {
	// empty string
	testParseUintError(t, "")

	// negative value
	testParseUintError(t, "-123")

	// non-num
	testParseUintError(t, "foobar234")

	// non-num chars at the end
	testParseUintError(t, "123w")

	// floating point num
	testParseUintError(t, "1234.545")

	// too big num
	testParseUintError(t, "12345678901234567890")
}

func TestParseUfloatSuccess(t *testing.T) {
	testParseUfloatSuccess(t, "0", 0)
	testParseUfloatSuccess(t, "1.", 1.)
	testParseUfloatSuccess(t, ".1", 0.1)
	testParseUfloatSuccess(t, "123.456", 123.456)
	testParseUfloatSuccess(t, "123", 123)
	testParseUfloatSuccess(t, "1234e2", 1234e2)
	testParseUfloatSuccess(t, "1234E-5", 1234E-5)
	testParseUfloatSuccess(t, "1.234e+3", 1.234e+3)
}

func TestParseUfloatError(t *testing.T) {
	// empty num
	testParseUfloatError(t, "")

	// negative num
	testParseUfloatError(t, "-123.53")

	// non-num chars
	testParseUfloatError(t, "123sdfsd")
	testParseUfloatError(t, "sdsf234")
	testParseUfloatError(t, "sdfdf")

	// non-num chars in exponent
	testParseUfloatError(t, "123e3s")
	testParseUfloatError(t, "12.3e-op")
	testParseUfloatError(t, "123E+SS5")

	// duplicate point
	testParseUfloatError(t, "1.3.4")

	// duplicate exponent
	testParseUfloatError(t, "123e5e6")

	// missing exponent
	testParseUfloatError(t, "123534e")
}

func testParseUfloatError(t *testing.T, s string) {
	n, err := ParseUfloat([]byte(s))
	if err == nil {
		t.Fatalf("Expecting error when parsing %q. obtained %f", s, n)
	}
	if n >= 0 {
		t.Fatalf("Expecting negative num instead of %f when parsing %q", n, s)
	}
}

func testParseUfloatSuccess(t *testing.T, s string, expectedF float64) {
	f, err := ParseUfloat([]byte(s))
	if err != nil {
		t.Fatalf("Unexpected error when parsing %q: %s", s, err)
	}
	delta := f - expectedF
	if delta < 0 {
		delta = -delta
	}
	if delta > expectedF*1e-10 {
		t.Fatalf("Unexpected value when parsing %q: %f. Expected %f", s, f, expectedF)
	}
}

func testParseUintError(t *testing.T, s string) {
	n, err := ParseUint([]byte(s))
	if err == nil {
		t.Fatalf("Expecting error when parsing %q. obtained %d", s, n)
	}
	if n >= 0 {
		t.Fatalf("Unexpected n=%d when parsing %q. Expected negative num", n, s)
	}
}

func testParseUintSuccess(t *testing.T, s string, expectedN int) {
	n, err := ParseUint([]byte(s))
	if err != nil {
		t.Fatalf("Unexpected error when parsing %q: %s", s, err)
	}
	if n != expectedN {
		t.Fatalf("Unexpected value %d. Expected %d. num=%q", n, expectedN, s)
	}
}
