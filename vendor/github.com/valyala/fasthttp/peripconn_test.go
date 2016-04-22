package fasthttp

import (
	"testing"
)

func TestIPxUint32(t *testing.T) {
	testIPxUint32(t, 0)
	testIPxUint32(t, 10)
	testIPxUint32(t, 0x12892392)
}

func testIPxUint32(t *testing.T, n uint32) {
	ip := uint322ip(n)
	nn := ip2uint32(ip)
	if n != nn {
		t.Fatalf("Unexpected value=%d for ip=%s. Expected %d", nn, ip, n)
	}
}

func TestPerIPConnCounter(t *testing.T) {
	var cc perIPConnCounter

	expectPanic(t, func() { cc.Unregister(123) })

	for i := 1; i < 100; i++ {
		if n := cc.Register(123); n != i {
			t.Fatalf("Unexpected counter value=%d. Expected %d", n, i)
		}
	}

	n := cc.Register(456)
	if n != 1 {
		t.Fatalf("Unexpected counter value=%d. Expected 1", n)
	}

	for i := 1; i < 100; i++ {
		cc.Unregister(123)
	}
	cc.Unregister(456)

	expectPanic(t, func() { cc.Unregister(123) })
	expectPanic(t, func() { cc.Unregister(456) })

	n = cc.Register(123)
	if n != 1 {
		t.Fatalf("Unexpected counter value=%d. Expected 1", n)
	}
	cc.Unregister(123)
}

func expectPanic(t *testing.T, f func()) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("Expecting panic")
		}
	}()
	f()
}
