// Copyright (c) 2016, Gerasimos Maropoulos
// All rights reserved.
//
// Redistribution and use in source and binary forms, with or without modification,
// are permitted provided that the following conditions are met:
//
// 1. Redistributions of source code must retain the above copyright notice,
//    this list of conditions and the following disclaimer.
//
// 2. Redistributions in binary form must reproduce the above copyright notice,
//	  this list of conditions and the following disclaimer
//    in the documentation and/or other materials provided with the distribution.
//
// 3. Neither the name of the copyright holder nor the names of its contributors may be used to endorse
//    or promote products derived from this software without specific prior written permission.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
// ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
// WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
// DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER AND CONTRIBUTOR, GERASIMOS MAROPOULOS
// BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
// (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
// LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
// ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
// SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

package sessions

import (
	"fmt"
	"runtime"
)

var (
	// ErrProviderNotFound returns an error with message: 'Provider was not found. Please try to _ import one'
	ErrProviderNotFound = NewError("Provider with name '%s' was not found. Please try to _ import this")
	// ErrProviderRegister returns an error with message: 'On provider registration. Trace: nil or empty named provider are not acceptable'
	ErrProviderRegister = NewError("On provider registration. Trace: nil or empty named provider are not acceptable")
	// ErrProviderAlreadyExists returns an error with message: 'On provider registration. Trace: provider with name '%s' already exists, maybe you register it twice'
	ErrProviderAlreadyExists = NewError("On provider registration. Trace: provider with name '%s' already exists, maybe you register it twice")
)

// Error holds the error
type Error struct {
	err error
}

// Error returns the message of the actual error
func (e *Error) Error() string {
	return e.err.Error()
}

// Format returns a formatted new error based on the arguments
func (e *Error) Format(args ...interface{}) error {
	return fmt.Errorf(e.err.Error(), args...)
}

// With does the same thing as Format but it receives an error type which if it's nil it returns a nil error
func (e *Error) With(err error) error {
	if err == nil {
		return nil
	}

	return e.Format(err.Error())
}

// Return returns the actual error as it is
func (e *Error) Return() error {
	return e.err
}

// Panic output the message and after panics
func (e *Error) Panic() {
	_, fn, line, _ := runtime.Caller(1)
	errMsg := e.Error()
	errMsg = "\nCaller was: " + fmt.Sprintf("%s:%d", fn, line)
	panic(errMsg)
}

// Panicf output the formatted message and after panics
func (e *Error) Panicf(args ...interface{}) {
	_, fn, line, _ := runtime.Caller(1)
	errMsg := e.Format(args...).Error()
	errMsg = "\nCaller was: " + fmt.Sprintf("%s:%d", fn, line)
	panic(errMsg)
}

// NewError creates and returns an Error with a message
func NewError(errMsg string) *Error {
	return &Error{fmt.Errorf("\n [IRIS SESSIONS]" + "Error: " + errMsg)}
}
