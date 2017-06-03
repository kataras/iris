// Copyright 2017 Γεράσιμος Μαρόπουλος. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package unis

// NewConditional runs the 'p' processor, if the string didn't
// changed then it assumes that that processor has being a failure
// and it returns a Chain of the 'alternative' processor(s).
func NewConditional(p Processor, alternative ...Processor) ProcessorFunc {
	chainAlternative := NewChain(alternative...)
	return func(str string) string {
		if firstTry := p.Process(str); firstTry != str {
			return firstTry
		}
		// we assume that it failed, because the
		// string didn't changed at all.
		return chainAlternative.Process(str)
	}
}
