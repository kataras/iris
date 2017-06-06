// Copyright 2017 Γεράσιμος Μαρόπουλος. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package unis

import (
	"strings"
)

// NewPrefixRemover accepts a "prefix" and returns a new processor
// which returns the result without that "prefix".
func NewPrefixRemover(prefix string) ProcessorFunc {
	return func(original string) (result string) {
		result = original
		for {
			if strings.HasPrefix(result, prefix) {
				result = result[len(prefix):]
			} else {
				break
			}
		}

		return
	}
}

// NewPrepender accepts a "prefix" and returns a new processor
// which returns the result prepended with that "prefix"
// if the "original"'s prefix != prefix.
func NewPrepender(prefix string) ProcessorFunc {
	return func(original string) string {
		if !strings.HasPrefix(original, prefix) {
			return prefix + original
		}

		return original
	}
}

// NewExclusivePrepender accepts a "prefix" and returns a new processor
// which returns the result prepended with that "prefix"
// if the "original"'s prefix != prefix.
// The difference from NewPrepender is that
// this processor will make sure that
// the prefix is that "prefix" series of characters,
// i.e:
// 1. "//path" -> NewPrepender("/") |> "//path"
//    It has a prefix already, so it doesn't prepends the "/" to the "//path",
//    but it doesn't checks if that is the correct prefix.
// 1. "//path" -> NewExclusivePrepender("/") |> "/path"
//     Checks if that is the correct prefix, if so returns as it's,
//     otherwise replace the duplications and prepend the correct prefix.
func NewExclusivePrepender(prefix string) ProcessorFunc {
	prefixRemover := NewPrefixRemover(prefix)
	prepender := NewPrepender(prefix)
	return NewChain(prefixRemover, prepender)
}
