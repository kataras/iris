package fasthttp

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestArgsAdd(t *testing.T) {
	var a Args
	a.Add("foo", "bar")
	a.Add("foo", "baz")
	a.Add("foo", "1")
	a.Add("ba", "23")
	if a.Len() != 4 {
		t.Fatalf("unexpected number of elements: %d. Expecting 4", a.Len())
	}
	s := a.String()
	expectedS := "foo=bar&foo=baz&foo=1&ba=23"
	if s != expectedS {
		t.Fatalf("unexpected result: %q. Expecting %q", s, expectedS)
	}

	var a1 Args
	a1.Parse(s)
	if a1.Len() != 4 {
		t.Fatalf("unexpected number of elements: %d. Expecting 4", a.Len())
	}

	var barFound, bazFound, oneFound, baFound bool
	a1.VisitAll(func(k, v []byte) {
		switch string(k) {
		case "foo":
			switch string(v) {
			case "bar":
				barFound = true
			case "baz":
				bazFound = true
			case "1":
				oneFound = true
			default:
				t.Fatalf("unexpected value %q", v)
			}
		case "ba":
			if string(v) != "23" {
				t.Fatalf("unexpected value: %q. Expecting %q", v, "23")
			}
			baFound = true
		default:
			t.Fatalf("unexpected key found %q", k)
		}
	})
	if !barFound || !bazFound || !oneFound || !baFound {
		t.Fatalf("something is missing: %v, %v, %v, %v", barFound, bazFound, oneFound, baFound)
	}
}

func TestArgsAcquireReleaseSequential(t *testing.T) {
	testArgsAcquireRelease(t)
}

func TestArgsAcquireReleaseConcurrent(t *testing.T) {
	ch := make(chan struct{}, 10)
	for i := 0; i < 10; i++ {
		go func() {
			testArgsAcquireRelease(t)
			ch <- struct{}{}
		}()
	}
	for i := 0; i < 10; i++ {
		select {
		case <-ch:
		case <-time.After(time.Second):
			t.Fatalf("timeout")
		}
	}
}

func testArgsAcquireRelease(t *testing.T) {
	a := AcquireArgs()

	for i := 0; i < 10; i++ {
		k := fmt.Sprintf("key_%d", i)
		v := fmt.Sprintf("value_%d", i*3+123)
		a.Set(k, v)
	}

	s := a.String()
	a.Reset()
	a.Parse(s)

	for i := 0; i < 10; i++ {
		k := fmt.Sprintf("key_%d", i)
		expectedV := fmt.Sprintf("value_%d", i*3+123)
		v := a.Peek(k)
		if string(v) != expectedV {
			t.Fatalf("unexpected value %q for key %q. Expecting %q", v, k, expectedV)
		}
	}

	ReleaseArgs(a)
}

func TestArgsPeekMulti(t *testing.T) {
	var a Args
	a.Parse("foo=123&bar=121&foo=321&foo=&barz=sdf")

	vv := a.PeekMulti("foo")
	expectedVV := [][]byte{
		[]byte("123"),
		[]byte("321"),
		[]byte(nil),
	}
	if !reflect.DeepEqual(vv, expectedVV) {
		t.Fatalf("unexpected vv\n%#v\nExpecting\n%#v\n", vv, expectedVV)
	}

	vv = a.PeekMulti("aaaa")
	if len(vv) > 0 {
		t.Fatalf("expecting empty result for non-existing key. Got %#v", vv)
	}

	vv = a.PeekMulti("bar")
	expectedVV = [][]byte{[]byte("121")}
	if !reflect.DeepEqual(vv, expectedVV) {
		t.Fatalf("unexpected vv\n%#v\nExpecting\n%#v\n", vv, expectedVV)
	}
}

func TestArgsEscape(t *testing.T) {
	testArgsEscape(t, "foo", "bar", "foo=bar")
	testArgsEscape(t, "f.o,1:2/4", "~`!@#$%^&*()_-=+\\|/[]{};:'\"<>,./?",
		"f.o%2C1%3A2%2F4=%7E%60%21%40%23%24%25%5E%26*%28%29_-%3D%2B%5C%7C%2F%5B%5D%7B%7D%3B%3A%27%22%3C%3E%2C.%2F%3F")
}

func testArgsEscape(t *testing.T, k, v, expectedS string) {
	var a Args
	a.Set(k, v)
	s := a.String()
	if s != expectedS {
		t.Fatalf("unexpected args %q. Expecting %q. k=%q, v=%q", s, expectedS, k, v)
	}
}

func TestArgsWriteTo(t *testing.T) {
	s := "foo=bar&baz=123&aaa=bbb"

	var a Args
	a.Parse(s)

	var w ByteBuffer
	n, err := a.WriteTo(&w)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if n != int64(len(s)) {
		t.Fatalf("unexpected n: %d. Expecting %d", n, len(s))
	}
	result := string(w.B)
	if result != s {
		t.Fatalf("unexpected result %q. Expecting %q", result, s)
	}
}

func TestArgsUint(t *testing.T) {
	var a Args
	a.SetUint("foo", 123)
	a.SetUint("bar", 0)
	a.SetUint("aaaa", 34566)

	expectedS := "foo=123&bar=0&aaaa=34566"
	s := string(a.QueryString())
	if s != expectedS {
		t.Fatalf("unexpected args %q. Expecting %q", s, expectedS)
	}

	if a.GetUintOrZero("foo") != 123 {
		t.Fatalf("unexpected arg value %d. Expecting %d", a.GetUintOrZero("foo"), 123)
	}
	if a.GetUintOrZero("bar") != 0 {
		t.Fatalf("unexpected arg value %d. Expecting %d", a.GetUintOrZero("bar"), 0)
	}
	if a.GetUintOrZero("aaaa") != 34566 {
		t.Fatalf("unexpected arg value %d. Expecting %d", a.GetUintOrZero("aaaa"), 34566)
	}

	if string(a.Peek("foo")) != "123" {
		t.Fatalf("unexpected arg value %q. Expecting %q", a.Peek("foo"), "123")
	}
	if string(a.Peek("bar")) != "0" {
		t.Fatalf("unexpected arg value %q. Expecting %q", a.Peek("bar"), "0")
	}
	if string(a.Peek("aaaa")) != "34566" {
		t.Fatalf("unexpected arg value %q. Expecting %q", a.Peek("aaaa"), "34566")
	}
}

func TestArgsCopyTo(t *testing.T) {
	var a Args

	// empty args
	testCopyTo(t, &a)

	a.Set("foo", "bar")
	testCopyTo(t, &a)

	a.Set("xxx", "yyy")
	testCopyTo(t, &a)

	a.Del("foo")
	testCopyTo(t, &a)
}

func testCopyTo(t *testing.T, a *Args) {
	keys := make(map[string]struct{})
	a.VisitAll(func(k, v []byte) {
		keys[string(k)] = struct{}{}
	})

	var b Args
	a.CopyTo(&b)

	b.VisitAll(func(k, v []byte) {
		if _, ok := keys[string(k)]; !ok {
			t.Fatalf("unexpected key %q after copying from %q", k, a.String())
		}
		delete(keys, string(k))
	})
	if len(keys) > 0 {
		t.Fatalf("missing keys %#v after copying from %q", keys, a.String())
	}
}

func TestArgsVisitAll(t *testing.T) {
	var a Args
	a.Set("foo", "bar")

	i := 0
	a.VisitAll(func(k, v []byte) {
		if string(k) != "foo" {
			t.Fatalf("unexpected key %q. Expected %q", k, "foo")
		}
		if string(v) != "bar" {
			t.Fatalf("unexpected value %q. Expected %q", v, "bar")
		}
		i++
	})
	if i != 1 {
		t.Fatalf("unexpected number of VisitAll calls: %d. Expected %d", i, 1)
	}
}

func TestArgsStringCompose(t *testing.T) {
	var a Args
	a.Set("foo", "bar")
	a.Set("aa", "bbb")
	a.Set("привет", "мир")
	a.Set("", "xxxx")
	a.Set("cvx", "")

	expectedS := "foo=bar&aa=bbb&%D0%BF%D1%80%D0%B8%D0%B2%D0%B5%D1%82=%D0%BC%D0%B8%D1%80&=xxxx&cvx"
	s := a.String()
	if s != expectedS {
		t.Fatalf("Unexpected string %q. Exected %q", s, expectedS)
	}
}

func TestArgsString(t *testing.T) {
	var a Args

	testArgsString(t, &a, "")
	testArgsString(t, &a, "foobar")
	testArgsString(t, &a, "foo=bar")
	testArgsString(t, &a, "foo=bar&baz=sss")
	testArgsString(t, &a, "")
	testArgsString(t, &a, "f%20o=x.x*-_8x%D0%BF%D1%80%D0%B8%D0%B2%D0%B5aaa&sdf=ss")
	testArgsString(t, &a, "=asdfsdf")
}

func testArgsString(t *testing.T, a *Args, s string) {
	a.Parse(s)
	s1 := a.String()
	if s != s1 {
		t.Fatalf("Unexpected args %q. Expected %q", s1, s)
	}
}

func TestArgsSetGetDel(t *testing.T) {
	var a Args

	if len(a.Peek("foo")) > 0 {
		t.Fatalf("Unexpected value: %q", a.Peek("foo"))
	}
	if len(a.Peek("")) > 0 {
		t.Fatalf("Unexpected value: %q", a.Peek(""))
	}
	a.Del("xxx")

	for j := 0; j < 3; j++ {
		for i := 0; i < 10; i++ {
			k := fmt.Sprintf("foo%d", i)
			v := fmt.Sprintf("bar_%d", i)
			a.Set(k, v)
			if string(a.Peek(k)) != v {
				t.Fatalf("Unexpected value: %q. Expected %q", a.Peek(k), v)
			}
		}
	}
	for i := 0; i < 10; i++ {
		k := fmt.Sprintf("foo%d", i)
		v := fmt.Sprintf("bar_%d", i)
		if string(a.Peek(k)) != v {
			t.Fatalf("Unexpected value: %q. Expected %q", a.Peek(k), v)
		}
		a.Del(k)
		if string(a.Peek(k)) != "" {
			t.Fatalf("Unexpected value: %q. Expected %q", a.Peek(k), "")
		}
	}

	a.Parse("aaa=xxx&bb=aa")
	if string(a.Peek("foo0")) != "" {
		t.Fatalf("Unepxected value %q", a.Peek("foo0"))
	}
	if string(a.Peek("aaa")) != "xxx" {
		t.Fatalf("Unexpected value %q. Expected %q", a.Peek("aaa"), "xxx")
	}
	if string(a.Peek("bb")) != "aa" {
		t.Fatalf("Unexpected value %q. Expected %q", a.Peek("bb"), "aa")
	}

	for i := 0; i < 10; i++ {
		k := fmt.Sprintf("xx%d", i)
		v := fmt.Sprintf("yy%d", i)
		a.Set(k, v)
		if string(a.Peek(k)) != v {
			t.Fatalf("Unexpected value: %q. Expected %q", a.Peek(k), v)
		}
	}
	for i := 5; i < 10; i++ {
		k := fmt.Sprintf("xx%d", i)
		v := fmt.Sprintf("yy%d", i)
		if string(a.Peek(k)) != v {
			t.Fatalf("Unexpected value: %q. Expected %q", a.Peek(k), v)
		}
		a.Del(k)
		if string(a.Peek(k)) != "" {
			t.Fatalf("Unexpected value: %q. Expected %q", a.Peek(k), "")
		}
	}
}

func TestArgsParse(t *testing.T) {
	var a Args

	// empty args
	testArgsParse(t, &a, "", 0, "foo=", "bar=", "=")

	// arg without value
	testArgsParse(t, &a, "foo1", 1, "foo=", "bar=", "=")

	// arg without value, but with equal sign
	testArgsParse(t, &a, "foo2=", 1, "foo=", "bar=", "=")

	// arg with value
	testArgsParse(t, &a, "foo3=bar1", 1, "foo3=bar1", "bar=", "=")

	// empty key
	testArgsParse(t, &a, "=bar2", 1, "foo=", "=bar2", "bar2=")

	// missing kv
	testArgsParse(t, &a, "&&&&", 0, "foo=", "bar=", "=")

	// multiple values with the same key
	testArgsParse(t, &a, "x=1&x=2&x=3", 3, "x=1")

	// multiple args
	testArgsParse(t, &a, "&&&qw=er&tyx=124&&&zxc_ss=2234&&", 3, "qw=er", "tyx=124", "zxc_ss=2234")

	// multiple args without values
	testArgsParse(t, &a, "&&a&&b&&bar&baz", 4, "a=", "b=", "bar=", "baz=")

	// values with '='
	testArgsParse(t, &a, "zz=1&k=v=v=a=a=s", 2, "k=v=v=a=a=s", "zz=1")

	// mixed '=' and '&'
	testArgsParse(t, &a, "sss&z=dsf=&df", 3, "sss=", "z=dsf=", "df=")

	// encoded args
	testArgsParse(t, &a, "f+o%20o=%D0%BF%D1%80%D0%B8%D0%B2%D0%B5%D1%82+test", 1, "f o o=привет test")

	// invalid percent encoding
	testArgsParse(t, &a, "f%=x&qw%z=d%0k%20p&%%20=%%%20x", 3, "f%=x", "qw%z=d%0k p", "% =%% x")

	// special chars
	testArgsParse(t, &a, "a.b,c:d/e=f.g,h:i/q", 1, "a.b,c:d/e=f.g,h:i/q")
}

func TestArgsHas(t *testing.T) {
	var a Args

	// single arg
	testArgsHas(t, &a, "foo", "foo")
	testArgsHasNot(t, &a, "foo", "bar", "baz", "")

	// multi args without values
	testArgsHas(t, &a, "foo&bar", "foo", "bar")
	testArgsHasNot(t, &a, "foo&bar", "", "aaaa")

	// multi args
	testArgsHas(t, &a, "b=xx&=aaa&c=", "b", "", "c")
	testArgsHasNot(t, &a, "b=xx&=aaa&c=", "xx", "aaa", "foo")

	// encoded args
	testArgsHas(t, &a, "a+b=c+d%20%20e", "a b")
	testArgsHasNot(t, &a, "a+b=c+d", "a+b", "c+d")
}

func testArgsHas(t *testing.T, a *Args, s string, expectedKeys ...string) {
	a.Parse(s)
	for _, key := range expectedKeys {
		if !a.Has(key) {
			t.Fatalf("Missing key %q in %q", key, s)
		}
	}
}

func testArgsHasNot(t *testing.T, a *Args, s string, unexpectedKeys ...string) {
	a.Parse(s)
	for _, key := range unexpectedKeys {
		if a.Has(key) {
			t.Fatalf("Unexpected key %q in %q", key, s)
		}
	}
}

func testArgsParse(t *testing.T, a *Args, s string, expectedLen int, expectedArgs ...string) {
	a.Parse(s)
	if a.Len() != expectedLen {
		t.Fatalf("Unexpected args len %d. Expected %d. s=%q", a.Len(), expectedLen, s)
	}
	for _, xx := range expectedArgs {
		tmp := strings.SplitN(xx, "=", 2)
		k := tmp[0]
		v := tmp[1]
		buf := a.Peek(k)
		if string(buf) != v {
			t.Fatalf("Unexpected value for key=%q: %q. Expected %q. s=%q", k, buf, v, s)
		}
	}
}
