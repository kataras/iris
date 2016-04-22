package fasthttp

import (
	"fmt"
	"testing"
	"time"
)

func TestByteBufferAcquireReleaseSerial(t *testing.T) {
	testByteBufferAcquireRelease(t)
}

func TestByteBufferAcquireReleaseConcurrent(t *testing.T) {
	concurrency := 10
	ch := make(chan struct{}, concurrency)
	for i := 0; i < concurrency; i++ {
		go func() {
			testByteBufferAcquireRelease(t)
			ch <- struct{}{}
		}()
	}

	for i := 0; i < concurrency; i++ {
		select {
		case <-ch:
		case <-time.After(time.Second):
			t.Fatalf("timeout!")
		}
	}
}

func testByteBufferAcquireRelease(t *testing.T) {
	for i := 0; i < 10; i++ {
		b := AcquireByteBuffer()
		b.B = append(b.B, "num "...)
		b.B = AppendUint(b.B, i)
		expectedS := fmt.Sprintf("num %d", i)
		if string(b.B) != expectedS {
			t.Fatalf("unexpected result: %q. Expecting %q", b.B, expectedS)
		}
		ReleaseByteBuffer(b)
	}
}
