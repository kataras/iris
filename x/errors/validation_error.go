package errors

import (
	"strconv"
	"strings"
)

// ValidationError is an interface which IF
// it custom error types completes, then
// it can by mapped to a validation error.
//
// A validation error(s) can be given by ErrorCodeName's Validation or Err methods.
type ValidationError interface {
	error

	GetField() string
	GetValue() interface{}
	GetReason() string
}

type ValidationErrors []ValidationError

func (errs ValidationErrors) Error() string {
	var buf strings.Builder
	for i, err := range errs {
		buf.WriteByte('[')
		buf.WriteString(strconv.Itoa(i))
		buf.WriteByte(']')
		buf.WriteByte(' ')

		buf.WriteString(err.Error())

		if i < len(errs)-1 {
			buf.WriteByte(',')
			buf.WriteByte(' ')
		}
	}

	return buf.String()
}
