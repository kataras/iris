// Copyright 2017 Γεράσιμος Μαρόπουλος. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package unis

// NewTargetedJoiner accepts an "expectedIndex" as int
// and a "joinerChar" as byte and returns a new processor
// which returns the result concated with that "joinerChar"
// if the "original" string[expectedIndex] != joinerChar.
//
// i.e:
// 1. "path", NewTargetedJoiner(0, '/') |> "/path"
// 2. "path/anything", NewTargetedJoiner(5, '*') |> "path/*anything".
func NewTargetedJoiner(expectedIndex int, joinerChar byte) ProcessorFunc {
	return func(original string) (result string) {
		result = original
		if expectedIndex < len(original)-1 {
			if original[expectedIndex] != joinerChar {
				strBegin := original[0:expectedIndex]
				strEnd := original[expectedIndex:]
				result = strBegin + string(joinerChar) + strEnd
			}
		}

		return
	}
}
