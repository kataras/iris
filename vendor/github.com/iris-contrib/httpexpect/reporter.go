package httpexpect

import (
	"fmt"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// AssertReporter implements Reporter interface using `testify/assert'
// package. Failures are non-fatal with this reporter.
type AssertReporter struct {
	backend *assert.Assertions
}

// NewAssertReporter returns a new AssertReporter object.
func NewAssertReporter(t assert.TestingT) *AssertReporter {
	return &AssertReporter{assert.New(t)}
}

// Errorf implements Reporter.Errorf.
func (r *AssertReporter) Errorf(message string, args ...interface{}) {
	r.backend.Fail(fmt.Sprintf(message, args...))
}

// RequireReporter implements Reporter interface using `testify/require'
// package. Failures fatal with this reporter.
type RequireReporter struct {
	backend *require.Assertions
}

// NewRequireReporter returns a new RequireReporter object.
func NewRequireReporter(t require.TestingT) *RequireReporter {
	return &RequireReporter{require.New(t)}
}

// Errorf implements Reporter.Errorf.
func (r *RequireReporter) Errorf(message string, args ...interface{}) {
	r.backend.FailNow(fmt.Sprintf(message, args...))
}
