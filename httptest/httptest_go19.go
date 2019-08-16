// +build go1.9

package httptest

import "github.com/gavv/httpexpect"

type (
	// Request type alias.
	Request = httpexpect.Request
	// Expect type alias.
	Expect = httpexpect.Expect
)
