// Copyright 2017 Γεράσιμος Μαρόπουλος. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package unis

import (
	"strings"
)

// NewSuffixRemover accepts a "suffix" and returns a new processor
// which returns the result without that "suffix".
func NewSuffixRemover(suffix string) ProcessorFunc {
	return func(original string) (result string) {
		// i.e path//
		// len(original) = 6
		// len(suffix) = 2
		// 6-2 = 4
		// [0:4] = path
		result = original

		if strings.HasSuffix(original, suffix) {
			result = original[0 : len(original)-len(suffix)]
		}

		return
	}
}

// NewAppender accepts a "suffix" and returns a new processor
// which returns the result appended with that "suffix".
func NewAppender(suffix string) ProcessorFunc {
	return func(original string) (result string) {
		result = original + suffix
		return
	}
}
