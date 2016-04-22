// +build !amd64,!arm64,!ppc64

package fasthttp

import (
	"testing"
)

func TestWriteHexInt(t *testing.T) {
	testWriteHexInt(t, 0, "0")
	testWriteHexInt(t, 1, "1")
	testWriteHexInt(t, 0x123, "123")
	testWriteHexInt(t, 0x7fffffff, "7fffffff")
}

func TestAppendUint(t *testing.T) {
	testAppendUint(t, 0)
	testAppendUint(t, 123)
	testAppendUint(t, 0x7fffffff)

	for i := 0; i < 2345; i++ {
		testAppendUint(t, i)
	}
}

func TestReadHexIntSuccess(t *testing.T) {
	testReadHexIntSuccess(t, "0", 0)
	testReadHexIntSuccess(t, "fF", 0xff)
	testReadHexIntSuccess(t, "00abc", 0xabc)
	testReadHexIntSuccess(t, "7ffffff", 0x7ffffff)
	testReadHexIntSuccess(t, "000", 0)
	testReadHexIntSuccess(t, "1234ZZZ", 0x1234)
}

func TestParseUintSuccess(t *testing.T) {
	testParseUintSuccess(t, "0", 0)
	testParseUintSuccess(t, "123", 123)
	testParseUintSuccess(t, "123456789", 123456789)
}
