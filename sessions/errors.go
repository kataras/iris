package sessions

import (
	"github.com/kataras/iris/errors"
)

var (
	// ErrProviderNotFound returns an error with message: 'Provider was not found. Please try to _ import one'
	ErrProviderNotFound = errors.New("Provider with name '%s' was not found. Please try to _ import this")
	// ErrProviderRegister returns an error with message: 'On provider registration. Trace: nil or empty named provider are not acceptable'
	ErrProviderRegister = errors.New("On provider registration. Trace: nil or empty named provider are not acceptable")
	// ErrProviderAlreadyExists returns an error with message: 'On provider registration. Trace: provider with name '%s' already exists, maybe you register it twice'
	ErrProviderAlreadyExists = errors.New("On provider registration. Trace: provider with name '%s' already exists, maybe you register it twice")
)
