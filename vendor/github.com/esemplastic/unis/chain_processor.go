// Copyright 2017 Γεράσιμος Μαρόπουλος. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package unis

// Processors is a list of string Processor.
type Processors []Processor

// NewChain returns a new chain of processors
// the result of the first goes to the second and so on.
func NewChain(processors ...Processor) ProcessorFunc {
	return func(original string) (result string) {
		result = original
		for _, p := range processors {
			result = p.Process(result)
		}

		return
	}
}
