package rule

import (
	"net/http"
)

// Rule a superset of validators
type Rule interface {
	Claim(*http.Request) bool
	Valid(http.ResponseWriter, *http.Request) bool
}
