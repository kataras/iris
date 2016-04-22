// Copyright 2015, Klaus Post, see LICENSE for details.

//+build amd64

package flate

import (
	"math/rand"
	"testing"
)

func TestCRC(t *testing.T) {
	if !useSSE42 {
		t.Skip("Skipping CRC test, no SSE 4.2 available")
	}
	for _, x := range deflateTests {
		y := x.out
		if len(y) >= minMatchLength {
			t.Logf("In: %v, Out:0x%08x", y[0:minMatchLength], crc32sse(y[0:minMatchLength]))
		}
	}
}

func TestCRCBulk(t *testing.T) {
	if !useSSE42 {
		t.Skip("Skipping CRC test, no SSE 4.2 available")
	}
	for _, x := range deflateTests {
		y := x.out
		y = append(y, y...)
		y = append(y, y...)
		y = append(y, y...)
		y = append(y, y...)
		y = append(y, y...)
		y = append(y, y...)
		if !testing.Short() {
			y = append(y, y...)
			y = append(y, y...)
		}
		y = append(y, 1)
		if len(y) >= minMatchLength {
			for j := len(y) - 1; j >= 4; j-- {

				// Create copy, so we easier detect of-of-bound reads
				test := make([]byte, j)
				test2 := make([]byte, j)
				copy(test, y[:j])
				copy(test2, y[:j])

				// We allocate one more than we need to test for unintentional overwrites
				dst := make([]hash, j-3+1)
				ref := make([]hash, j-3+1)
				for i := range dst {
					dst[i] = hash(i + 100)
					ref[i] = hash(i + 101)
				}
				// Last entry must NOT be overwritten.
				dst[j-3] = 0x1234
				ref[j-3] = 0x1234

				// Do two encodes we can compare
				crc32sseAll(test, dst)
				crc32sseAll(test2, ref)

				// Check all values
				for i, got := range dst {
					if i == j-3 {
						if dst[i] != 0x1234 {
							t.Fatalf("end of expected dst overwritten, was %08x", uint32(dst[i]))
						}
						continue
					}
					expect := crc32sse(y[i : i+4])
					if got != expect && got == hash(i)+100 {
						t.Errorf("Len:%d Index:%d, expected 0x%08x but not modified", len(y), i, uint32(expect))
					} else if got != expect {
						t.Errorf("Len:%d Index:%d, got 0x%08x expected:0x%08x", len(y), i, uint32(got), uint32(expect))
					}
					expect = ref[i]
					if got != expect {
						t.Errorf("Len:%d Index:%d, got 0x%08x expected:0x%08x", len(y), i, got, expect)
					}
				}
			}
		}
	}
}

func TestMatchLen(t *testing.T) {
	if !useSSE42 {
		t.Skip("Skipping Matchlen test, no SSE 4.2 available")
	}
	// Maximum length tested
	var maxLen = 512

	// Skips per iteration
	is, js, ks := 3, 2, 1
	if testing.Short() {
		is, js, ks = 7, 5, 3
	}

	a := make([]byte, maxLen)
	b := make([]byte, maxLen)
	bb := make([]byte, maxLen)
	rand.Seed(1)
	for i := range a {
		a[i] = byte(rand.Int63())
		b[i] = byte(rand.Int63())
	}

	// Test different lengths
	for i := 0; i < maxLen; i += is {
		// Test different dst offsets.
		for j := 0; j < maxLen-1; j += js {
			copy(bb, b)
			// Test different src offsets
			for k := i - 1; k >= 0; k -= ks {
				copy(bb[j:], a[k:i])
				maxTest := maxLen - j
				if maxTest > maxLen-k {
					maxTest = maxLen - k
				}
				got := matchLenSSE4(a[k:], bb[j:], maxTest)
				expect := matchLenReference(a[k:], bb[j:], maxTest)
				if got > maxTest || got < 0 {
					t.Fatalf("unexpected result %d (len:%d, src offset: %d, dst offset:%d)", got, maxTest, k, j)
				}
				if got != expect {
					t.Fatalf("Mismatch, expected %d, got %d", expect, got)
				}
			}
		}
	}
}

// matchLenReference is a reference matcher.
func matchLenReference(a, b []byte, max int) int {
	for i := 0; i < max; i++ {
		if a[i] != b[i] {
			return i
		}
	}
	return max
}

func TestHistogram(t *testing.T) {
	if !useSSE42 {
		t.Skip("Skipping Matchlen test, no SSE 4.2 available")
	}
	// Maximum length tested
	const maxLen = 65536
	var maxOff = 8

	// Skips per iteration
	is, js := 5, 3
	if testing.Short() {
		is, js = 9, 1
		maxOff = 1
	}

	a := make([]byte, maxLen+maxOff)
	rand.Seed(1)
	for i := range a {
		a[i] = byte(rand.Int63())
	}

	// Test different lengths
	for i := 0; i <= maxLen; i += is {
		// Test different offsets
		for j := 0; j < maxOff; j += js {
			var got [256]int32
			var reference [256]int32

			histogram(a[j:i+j], got[:])
			histogramReference(a[j:i+j], reference[:])
			for k := range got {
				if got[k] != reference[k] {
					t.Fatalf("mismatch at len:%d, offset:%d, value %d: (got) %d != %d (expected)", i, j, k, got[k], reference[k])
				}
			}
		}
	}
}

// histogramReference is a reference
func histogramReference(b []byte, h []int32) {
	if len(h) < 256 {
		panic("Histogram too small")
	}
	for _, t := range b {
		h[t]++
	}
}
