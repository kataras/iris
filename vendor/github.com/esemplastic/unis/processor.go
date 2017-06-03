// Copyright 2017 Γεράσιμος Μαρόπουλος. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package unis

// Processor is the most important interface of this package.
//
// It's being used to implement basic string processors.
// Users can use all these processors to build more on their packages.
//
// A Processor should change the "original" and returns its result based on that.
type Processor interface {
	// Process accepts an "original" string and returns a result based on that.
	Process(original string) (result string)
}

// ProcessorFunc same as Processor, as func. Implements the Processor.
type ProcessorFunc func(string) string

// Process accepts an "original" string and returns a result based on that.
func (p ProcessorFunc) Process(original string) (result string) {
	return p(original)
}

// OriginProcessor returns a new string processor which
// always returns the "original" string back. It does nothing.
//
// It can be used as a parameter to the library's functions.
var OriginProcessor = ProcessorFunc(func(original string) string { // Is a variable, so the user can change that too.
	return original
})

// ClearProcessor returns a new string processor which always
// returns empty string back.
//
// It can be used as a parameter to the library's functions.
var ClearProcessor = ProcessorFunc(func(string) string {
	return ""
})
