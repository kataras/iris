package mvc2

import (
	"errors"
)

var (
	errNil           = errors.New("nil")
	errBad           = errors.New("bad")
	errAlreadyExists = errors.New("already exists")
)
