package reflex

import (
	"math/bits"
	"testing"
	"unsafe"
)

func TestUnsafeSet2CopiesExactWindow(t *testing.T) {
	// UnsafeSet2 must write exactly `size` bytes at `offset` and nothing else.
	dst := [16]byte{}
	for i := range dst {
		dst[i] = 0xAA
	}
	src := [4]byte{1, 2, 3, 4}

	UnsafeSet2(unsafe.Pointer(&dst), 6, unsafe.Pointer(&src), unsafe.Sizeof(src))

	want := [16]byte{
		0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA,
		1, 2, 3, 4,
		0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA,
	}
	if dst != want {
		t.Errorf("got %v, want %v", dst, want)
	}
}

func TestUnsafeCopyPartialSize(t *testing.T) {
	// A size smaller than the block must leave the tail of dst untouched.
	src := [8]byte{9, 8, 7, 6, 5, 4, 3, 2}
	dst := [8]byte{}

	UnsafeCopy(unsafe.Pointer(&dst), unsafe.Pointer(&src), 3)

	want := [8]byte{9, 8, 7}
	if dst != want {
		t.Errorf("got %v, want %v", dst, want)
	}
}

func TestUnsafeCopyZeroSize(t *testing.T) {
	src := [4]byte{1, 2, 3, 4}
	dst := [4]byte{}

	UnsafeCopy(unsafe.Pointer(&dst), unsafe.Pointer(&src), 0)

	if dst != ([4]byte{}) {
		t.Errorf("zero-size copy wrote data: %v", dst)
	}
}

func TestUnsafeCopyBeyondGigabyte(t *testing.T) {
	// The pre-Go-1.17 (*[1 << 30]byte)(ptr)[:size:size] idiom panicked with
	// "slice bounds out of range" for any size past 1GiB; unsafe.Slice has no
	// such ceiling. Costs two ~1GiB allocations, so it is skippable and only
	// meaningful where int is 64 bits wide.
	if testing.Short() {
		t.Skip("skipping ~2GiB allocation in short mode")
	}
	if bits.UintSize < 64 {
		t.Skip("needs a 64-bit address space")
	}

	const size = 1<<30 + 16
	src := make([]byte, size)
	dst := make([]byte, size)
	src[0] = 1
	src[1<<30] = 2 // just past the old ceiling
	src[size-1] = 3

	UnsafeCopy(unsafe.Pointer(&dst[0]), unsafe.Pointer(&src[0]), size)

	if dst[0] != 1 || dst[1<<30] != 2 || dst[size-1] != 3 {
		t.Errorf("sentinels: got %d, %d, %d; want 1, 2, 3", dst[0], dst[1<<30], dst[size-1])
	}
}
