// black-box testing
package errors_test

import (
	"testing"

	"github.com/kataras/iris/core/errors"
)

func TestReporterAdd(t *testing.T) {
	errors.Prefix = ""

	r := errors.NewReporter()

	tests := []string{"err1", "err3", "err4\nerr5"}
	for _, tt := range tests {
		r.Add(tt)
	}

	for i, e := range r.Stack() {
		tt := tests[i]
		if expected, got := tt, e.Error(); expected != got {
			t.Fatalf("[%d] expected %s but got %s", i, expected, got)
		}
	}
}
