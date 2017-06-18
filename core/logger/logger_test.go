// black-box testing
package logger_test

import (
	"bytes"
	"testing"

	"github.com/cdren/iris/core/logger"
)

func TestLog(t *testing.T) {
	msg := "Hello this is me"
	l := &bytes.Buffer{}
	logger.Log(l, msg)
	if expected, got := msg, l.String(); expected != got {
		t.Fatalf("expected %s but got %s", expected, got)
	}
}
